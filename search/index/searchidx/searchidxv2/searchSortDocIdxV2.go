package searchidxv2

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
	"io"
	"sort"
	"strings"
)

//////////////////////////////////////////////////////////////////
//
//                   Sorted Doc Index Source
//
//////////////////////////////////////////////////////////////////
type ssdiSource interface {
	Reset() error
	Close() error
	ReadNextDocInfos() (map[string]map[string]uint64, error)
}

//
// Sorted Doc Index Source from Search HashedTerm Index
//
type ssdiSourceSTI struct {
	sti *SearchTermIdxV2
}

func openSsdiSourceFromSTI(sdc client.StrongDocClient, owner common.SearchIdxOwner, hashedTerm string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*ssdiSourceSTI, error) {
	sti, err := OpenSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
	if err != nil {
		return nil, err
	}
	return &ssdiSourceSTI{sti}, nil
}

func (source *ssdiSourceSTI) Reset() error {
	if source.sti == nil {
		return nil
	}
	return source.sti.Reset()
}

func (source *ssdiSourceSTI) Close() error {
	if source.sti == nil {
		return nil
	}
	err := source.sti.Close()
	source.sti = nil
	return err
}

// Read Block from STI
// return [term -> [docID -> docVer]]
func (source *ssdiSourceSTI) ReadNextDocInfos() (termDocVer map[string]map[string]uint64, err error) {
	termDocVer = make(map[string]map[string]uint64)
	sti := source.sti
	if sti == nil {
		err = fmt.Errorf("invalid source")
		return
	}
	var stib *SearchTermIdxBlkV2 = nil
	stib, err = sti.ReadNextBlock()
	if err != nil {
		return
	}
	if stib != nil {
		for term, docVerOff := range stib.TermDocVerOffset {
			if len(docVerOff) > 0 {
				if _, ok := termDocVer[term]; !ok {
					termDocVer[term] = make(map[string]uint64)
				}
				for docID, verOff := range docVerOff {
					if _, ok := termDocVer[term][docID]; ok {
						if verOff.Version > termDocVer[term][docID] { // higher version
							termDocVer[term][docID] = verOff.Version
						}
					} else {
						termDocVer[term][docID] = verOff.Version
					}

				}
			}

		}
	}
	return
}

//
// Sorted Doc Index Source from STI Writer
//
type ssdiSourceDocIdx struct {
	stiSources map[string]map[string]uint64
	eof        bool
}

func openSsdiSourceFromSTISource(stiSources map[string]map[string]uint64) *ssdiSourceDocIdx {
	return &ssdiSourceDocIdx{stiSources: stiSources, eof: false}
}

func (source *ssdiSourceDocIdx) Reset() error {
	source.eof = false
	return nil
}

func (source *ssdiSourceDocIdx) Close() error {
	return nil
}

func (source *ssdiSourceDocIdx) ReadNextDocInfos() (termDocVer map[string]map[string]uint64, err error) {
	if source.eof {
		err = io.EOF
		return
	}
	termDocVer = source.stiSources

	source.eof = true
	return
}

//////////////////////////////////////////////////////////////////
//
//                   Sorted Doc Index
//
//////////////////////////////////////////////////////////////////
// SearchSortDocIdxV2 is a structure for search sorted document index V1
type SearchSortDocIdxV2 struct {
	common.SsdiVersionS
	Owner       common.SearchIdxOwner
	HashedTerm  string
	TermKey     *sscrypto.StrongSaltKey
	IndexKey    *sscrypto.StrongSaltKey
	IndexNonce  []byte
	InitOffset  uint64
	termHmac    string
	updateID    string
	source      ssdiSource
	writer      io.WriteCloser
	reader      io.ReadCloser
	bwriter     ssblocks.BlockListWriterV1
	breader     ssblocks.BlockListReaderV1
	block       *SearchSortDocIdxBlkV2
	storeSize   uint64
	totalDocIDs uint64
}

// CreateSearchSortDocIdxV2 creates a search sorted document index writer V1
func CreateSearchSortDocIdxV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, hashedTerm, updateID string,
	termKey, indexKey *sscrypto.StrongSaltKey, docs map[string]map[string]uint64) (*SearchSortDocIdxV2, error) {

	var err error
	ssdi := &SearchSortDocIdxV2{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: common.SSDI_V2},
		Owner:        owner,
		HashedTerm:   hashedTerm,
		TermKey:      termKey,
		IndexKey:     indexKey,
		IndexNonce:   nil,
		InitOffset:   0,
		termHmac:     "",
		updateID:     updateID,
		source:       nil,
		writer:       nil,
		reader:       nil,
		bwriter:      nil,
		breader:      nil,
		block:        nil,
		storeSize:    0,
		totalDocIDs:  0,
	}

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if midStreamKey, ok := indexKey.Key.(sscryptointf.KeyMidstream); ok {
		ssdi.IndexNonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	if docs != nil {
		ssdi.source = openSsdiSourceFromSTISource(docs)
	} else {
		ssdi.source, err = openSsdiSourceFromSTI(sdc, owner, hashedTerm, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	ssdi.termHmac, err = common.CreateTermHmac(hashedTerm, termKey)
	if err != nil {
		return nil, err
	}

	//fmt.Println("hashedTerm", hashedTerm, "termHmac", ssdi.termHmac)
	_, err = ssdi.createWriter(sdc)
	if err != nil {
		return nil, err
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &SsdiPlainHdrBodyV1{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: common.SSDI_V2},
		KeyType:      ssdi.IndexKey.Type.Name,
		Nonce:        ssdi.IndexNonce,
		TermHmac:     ssdi.termHmac,
		UpdateID:     ssdi.updateID,
	}

	cipherHdrBody := &SsdiCipherHdrBodyV1{
		BlockVersionS: common.BlockVersionS{BlockVer: common.STI_BLOCK_V2},
	}

	plainHdrBodySerial, err := plainHdrBody.serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdr := ssheaders.CreatePlainHdr(ssheaders.HeaderTypeJSONGzip, plainHdrBodySerial)
	plainHdrSerial, err := plainHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBodySerial, err := cipherHdrBody.serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdr := ssheaders.CreateCipherHdr(ssheaders.HeaderTypeJSONGzip, cipherHdrBodySerial)
	cipherHdrSerial, err := cipherHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	// Write the plaintext header to storage
	n, err := ssdi.writer.Write(plainHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(plainHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire plaintext header")
	}

	// Initialize the streaming crypto to encrypt ciphertext header and the
	// blocks after that
	streamCrypto, err := crypto.CreateStreamCrypto(ssdi.IndexKey, plainHdrBody.Nonce,
		ssdi.writer, int64(ssdi.InitOffset)+int64(n))
	if err != nil {
		return nil, errors.New(err)
	}

	// Write the ciphertext header to storage
	n, err = streamCrypto.Write(cipherHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(cipherHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire ciphertext header")
	}

	// Create a block list writer using the streaming crypto so the blocks will be
	// encrypted.
	ssdi.bwriter, err = ssblocks.NewBlockListWriterV1(streamCrypto, uint32(common.SSDI_BLOCK_SIZE_MAX),
		ssdi.InitOffset+uint64(len(plainHdrSerial)+len(cipherHdrSerial)))
	if err != nil {
		return nil, errors.New(err)
	}

	return ssdi, nil
}

// OpenSearchSortDocIdxV2 opens a search sorted document index reader V1
func OpenSearchSortDocIdxV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, hashedTerm string, termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*SearchSortDocIdxV2, error) {
	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	termID, err := common.CreateTermHmac(hashedTerm, termKey)
	if err != nil {
		return nil, err
	}

	reader, size, err := createSsdiReader(sdc, owner, termID, updateID)
	if err != nil {
		return nil, err
	}

	plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, err
	}

	return OpenSearchSortDocIdxPrivV2(owner, hashedTerm, termID, updateID, termKey, indexKey,
		reader, plainHdr, 0, uint64(plainHdrSize), size)
}

func OpenSearchSortDocIdxPrivV2(owner common.SearchIdxOwner, hashedTerm, termID, updateID string,
	termKey, indexKey *sscrypto.StrongSaltKey, reader io.ReadCloser, plainHdr ssheaders.Header,
	initOffset, plainHdrSize, endOffset uint64) (*SearchSortDocIdxV2, error) {

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, err
	}

	version, err := common.DeserializeSsdiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetSsdiVersion() != common.SSDI_V2 {
		return nil, errors.Errorf("Search sorted document index version %v is not %v",
			version.GetSsdiVersion(), common.SSDI_V2)
	}

	// Parse plaintext header body
	plainHdrBody := &SsdiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	ssdi := &SearchSortDocIdxV2{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: plainHdrBody.GetSsdiVersion()},
		HashedTerm:   hashedTerm,
		Owner:        owner,
		TermKey:      termKey,
		IndexKey:     indexKey,
		IndexNonce:   plainHdrBody.Nonce,
		InitOffset:   0,
		termHmac:     termID,
		updateID:     updateID,
		source:       nil,
		writer:       nil,
		reader:       reader,
		bwriter:      nil,
		breader:      nil,
		block:        nil,
		storeSize:    endOffset,
		totalDocIDs:  0,
	}

	plainHdrOffset := initOffset + plainHdrSize

	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
	streamCrypto, err := crypto.OpenStreamCrypto(ssdi.IndexKey, ssdi.IndexNonce, ssdi.reader, int64(plainHdrOffset))
	if err != nil {
		return nil, errors.New(err)
	}

	// Read the ciphertext header from storage
	cipherHdr, parsed, err := ssheaders.DeserializeCipherHdrStream(streamCrypto)
	if err != nil {
		return nil, err
	}

	cipherHdrBodyData, err := cipherHdr.GetBody()
	if err != nil {
		return nil, err
	}

	cipherHdrBody := &SsdiCipherHdrBodyV1{}
	_, err = cipherHdrBody.deserialize(cipherHdrBodyData)
	if err != nil {
		return nil, err
	}

	// Create a block list reader using the streaming crypto so the blocks will be
	// decrypted.
	bReader, err := ssblocks.NewBlockListReader(streamCrypto, plainHdrOffset+uint64(parsed), endOffset, initEmptySortDocIdxBlkV1)
	if err != nil {
		return nil, err
	}
	blockReader, ok := bReader.(ssblocks.BlockListReaderV1)
	if !ok {
		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
	}
	ssdi.breader = blockReader
	return ssdi, nil
}

func (ssdi *SearchSortDocIdxV2) createWriter(sdc client.StrongDocClient) (io.WriteCloser, error) {
	if ssdi.writer != nil {
		return ssdi.writer, nil
	}

	writer, err := common.OpenSearchSortDocIndexWriter(sdc, ssdi.Owner, ssdi.termHmac, ssdi.updateID)
	if err != nil {
		return nil, err
	}
	ssdi.writer = writer

	return ssdi.writer, nil
}

func (ssdi *SearchSortDocIdxV2) createReader(sdc client.StrongDocClient) (io.ReadCloser, uint64, error) {
	if ssdi.reader != nil {
		return ssdi.reader, ssdi.storeSize, nil
	}

	return createSsdiReader(sdc, ssdi.Owner, ssdi.termHmac, ssdi.updateID)
}

func createSsdiReader(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac, updateID string) (io.ReadCloser, uint64, error) {
	reader, size, err := common.OpenSearchSortDocIndexReader(sdc, owner, termHmac, updateID)
	if err != nil {
		return nil, 0, err
	}
	return reader, size, nil
}

func (ssdi *SearchSortDocIdxV2) WriteNextBlock() (*SearchSortDocIdxBlkV2, error) {
	if ssdi.writer == nil || ssdi.bwriter == nil {
		return nil, errors.Errorf("The search sorted document index is not open for writing")
	}

	if ssdi.block == nil {
		ssdi.block = CreateSearchSortDocIdxBlkV2("", "", uint64(ssdi.bwriter.GetMaxDataSize()))
	}

	err := ssdi.source.Reset()
	if err != nil {
		return nil, errors.New(err)
	}

	for {
		var termDocIDVers map[string]map[string]uint64
		termDocIDVers, err = ssdi.source.ReadNextDocInfos()
		if err != nil {
			break
		}

		for term, docIDVers := range termDocIDVers {
			//fmt.Println("term", term, "len(docIDVers)", len(docIDVers))
			for docID, ver := range docIDVers {
				ssdi.block.AddTermDocVer(term, docID, ver)
			}

		}
	}

	if err != io.EOF {
		return nil, err
	}

	err = ssdi.flush() // send to server
	if err != nil {
		return nil, errors.New(err)
	}

	block := ssdi.block
	if block.IsFull() {
		ssdi.block = CreateSearchSortDocIdxBlkV2(block.highTerm, block.termToHighDocID[block.highTerm],
			uint64(ssdi.bwriter.GetMaxDataSize()))

		return block, nil

	}

	// block is not full, no more unprocessed data, already reach the end, return EOF

	ssdi.block = nil
	return block, io.EOF
}

func (ssdi *SearchSortDocIdxV2) ReadNextBlock() (*SearchSortDocIdxBlkV2, error) {
	if ssdi.reader == nil || ssdi.breader == nil {
		return nil, errors.Errorf("No reader available")
	}

	blockData, predictedSize, err := ssdi.breader.ReadNextBlockData()
	if err != nil {
		return nil, err
	}

	blk, ok := blockData.(*SearchSortDocIdxBlkV2)
	if !ok {
		return nil, errors.Errorf("Cannot convert to DocOffsetIdxBlkV1")
	}

	blk.predictedJSONSize = uint64(predictedSize)
	blk, err = blk.formatFromBlockData()
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (ssdi *SearchSortDocIdxV2) ExistDocID(term, docID string) (bool, error) {
	if ssdi.breader == nil {
		return false, errors.Errorf("The search sorted document index is not open for reading")
	}

	blk, _, err := ssdi.breader.SearchBinary(TermDoc{Term: term, DocID: docID}, DocIDVerComparatorV2)
	if err != nil {
		return false, errors.New(err)
	}

	return (blk != nil), nil
}

// binary search O(logN)
func (ssdi *SearchSortDocIdxV2) FindDocID(term, docID string) (*DocIDVerV2, error) {
	if ssdi.breader == nil {
		return nil, errors.Errorf("The search sorted document index is not open for reading")
	}

	b, _, err := ssdi.breader.SearchBinary(TermDoc{Term: term, DocID: docID}, DocIDVerComparatorV2)
	if err != nil {
		return nil, errors.New(err)
	}

	if b != nil {
		blk, ok := b.(*SearchSortDocIdxBlkV2)
		if !ok {
			return nil, errors.Errorf("Cannot convert to SearchSortDocIdxBlkV1")
		}
		if termDocVer, exists := blk.termToDocIDVer[term]; exists {
			if docVer, exist := termDocVer[docID]; exist {
				return &DocIDVerV2{term, docID, docVer}, nil
			}
		}

	}

	return nil, nil
}

// go through ssdi blocks, O(N)
func (ssdi *SearchSortDocIdxV2) FindDocIDs(term string, docIDs []string) (map[string]*DocIDVerV2, error) {
	if ssdi.breader == nil {
		return nil, errors.Errorf("The search sorted document index is not open for reading")
	}

	sortedCloneDocIDs := make([]string, len(docIDs))
	copy(sortedCloneDocIDs, docIDs)

	sort.Strings(sortedCloneDocIDs)
	result := make(map[string]*DocIDVerV2)
	for _, docID := range docIDs {
		result[docID] = nil
	}

	err := ssdi.Reset()
	if err != nil {
		return nil, err
	}

	err = nil
	for i := 0; i < len(sortedCloneDocIDs) && err == nil; {
		var blk *SearchSortDocIdxBlkV2 = nil
		blk, err = ssdi.ReadNextBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if blk != nil {
			if _, ok := blk.termToDocIDVer[term]; !ok {
				continue
			}
			for ; i < len(sortedCloneDocIDs) && strings.Compare(sortedCloneDocIDs[i], blk.termToHighDocID[term]) <= 0; i++ {
				if docVer, exist := blk.termToDocIDVer[term][sortedCloneDocIDs[i]]; exist {
					result[sortedCloneDocIDs[i]] = &DocIDVerV2{term, sortedCloneDocIDs[i], docVer}
				}
			}
		}
	}

	return result, nil
}

func (ssdi *SearchSortDocIdxV2) Reset() error {
	if ssdi.breader == nil {
		return errors.Errorf("The search sorted document index is not open for reading. Can not reset")
	}

	ssdi.totalDocIDs = 0
	return ssdi.breader.Reset()
}

func (ssdi *SearchSortDocIdxV2) Close() error {
	var ferr error = nil

	if ssdi.source != nil {
		ssdi.source.Close()
		ssdi.source = nil
	}

	if ssdi.writer != nil {
		if ssdi.block != nil && ssdi.block.totalTermDocIdVers > 0 {
			ferr = utils.FirstError(ferr, ssdi.flush())
		}
		ssdi.block = nil

		ferr = utils.FirstError(ferr, ssdi.writer.Close())
		ssdi.writer = nil
	}

	if ssdi.reader != nil {
		ferr = utils.FirstError(ferr, ssdi.reader.Close())
		ssdi.reader = nil
	}

	ssdi.totalDocIDs = 0
	return ferr
}

func (ssdi *SearchSortDocIdxV2) flush() error {
	if ssdi.bwriter == nil {
		return errors.Errorf("The search sorted document index is not open for writing")
	}

	block := ssdi.block.formatToBlockData()
	err := ssdi.bwriter.WriteBlockData(block)
	if err != nil {
		return err
	}
	ssdi.block = block
	ssdi.totalDocIDs += block.totalTermDocIdVers
	return nil
}
