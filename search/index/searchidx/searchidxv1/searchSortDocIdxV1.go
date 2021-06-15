package searchidxv1

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
	ReadNextDocInfos() ([]string, []uint64, error)
}

//
// Sorted Doc Index Source from Search Term Index
//
type ssdiSourceSTI struct {
	sti *SearchTermIdxV1
}

func openSsdiSourceFromSTI(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*ssdiSourceSTI, error) {
	sti, err := OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateID)
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
func (source *ssdiSourceSTI) ReadNextDocInfos() (docIDs []string, docVers []uint64, err error) {
	sti := source.sti
	if sti == nil {
		err = fmt.Errorf("invalid source")
		return
	}
	var stib *SearchTermIdxBlkV1 = nil
	stib, err = sti.ReadNextBlock()
	if err != nil {
		return
	}
	if stib != nil {
		for docID, verOff := range stib.DocVerOffset {
			docIDs = append(docIDs, docID)
			docVers = append(docVers, verOff.Version)
		}
	}
	return
}

//
// Sorted Doc Index Source from STI Writer
//
type ssdiSourceDocIdx struct {
	stiSources map[string]uint64
	eof        bool
}

func openSsdiSourceFromSTISource(stiSources map[string]uint64) *ssdiSourceDocIdx {
	return &ssdiSourceDocIdx{stiSources: stiSources, eof: false}
}

func (source *ssdiSourceDocIdx) Reset() error {
	source.eof = false
	return nil
}

func (source *ssdiSourceDocIdx) Close() error {
	return nil
}

func (source *ssdiSourceDocIdx) ReadNextDocInfos() (docIDs []string, docVers []uint64, err error) {
	if source.eof {
		err = io.EOF
		return
	}
	for id, ver := range source.stiSources {
		docIDs = append(docIDs, id)
		docVers = append(docVers, ver)
	}
	source.eof = true
	return
}

//////////////////////////////////////////////////////////////////
//
//                   Sorted Doc Index
//
//////////////////////////////////////////////////////////////////
// SearchSortDocIdxV1 is a structure for search sorted document index V1
type SearchSortDocIdxV1 struct {
	common.SsdiVersionS
	Owner      common.SearchIdxOwner
	Term       string
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	IndexNonce []byte
	InitOffset uint64
	termHmac   string
	updateID   string
	//sti         *SearchTermIdxV1
	source      ssdiSource
	writer      io.WriteCloser
	reader      io.ReadCloser
	bwriter     ssblocks.BlockListWriterV1
	breader     ssblocks.BlockListReaderV1
	block       *SearchSortDocIdxBlkV1
	storeSize   uint64
	totalDocIDs uint64
}

// CreateSearchSortDocIdxV1 creates a search sorted document index writer V1
func CreateSearchSortDocIdxV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term, updateID string,
	termKey, indexKey *sscrypto.StrongSaltKey, docs map[string]uint64) (*SearchSortDocIdxV1, error) {
	var err error
	ssdi := &SearchSortDocIdxV1{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: common.SSDI_V1},
		Owner:        owner,
		Term:         term,
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
		ssdi.source, err = openSsdiSourceFromSTI(sdc, owner, term, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	ssdi.termHmac, err = createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}

	_, err = ssdi.createWriter(sdc)
	if err != nil {
		return nil, err
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &SsdiPlainHdrBodyV1{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: common.SSDI_V1},
		KeyType:      ssdi.IndexKey.Type.Name,
		Nonce:        ssdi.IndexNonce,
		TermHmac:     ssdi.termHmac,
		UpdateID:     ssdi.updateID,
	}

	cipherHdrBody := &SsdiCipherHdrBodyV1{
		BlockVersionS: common.BlockVersionS{BlockVer: common.STI_BLOCK_V1},
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

// OpenSearchSortDocIdxV1 opens a search sorted document index reader V1
func OpenSearchSortDocIdxV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*SearchSortDocIdxV1, error) {
	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	termHmac, err := createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}

	reader, size, err := createSsdiReader(sdc, owner, termHmac, updateID)
	if err != nil {
		return nil, err
	}

	plainHdr, parsed, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, err
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, err
	}

	version, err := common.DeserializeSsdiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetSsdiVersion() != common.SSDI_V1 {
		return nil, errors.Errorf("Search sorted document index version %v is not %v",
			version.GetSsdiVersion(), common.SSDI_V1)
	}

	// Parse plaintext header body
	plainHdrBody := &SsdiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	ssdi := &SearchSortDocIdxV1{
		SsdiVersionS: common.SsdiVersionS{SsdiVer: plainHdrBody.GetSsdiVersion()},
		Term:         term,
		Owner:        owner,
		TermKey:      termKey,
		IndexKey:     indexKey,
		IndexNonce:   plainHdrBody.Nonce,
		InitOffset:   0,
		termHmac:     termHmac,
		updateID:     updateID,
		source:       nil,
		writer:       nil,
		reader:       reader,
		bwriter:      nil,
		breader:      nil,
		block:        nil,
		storeSize:    size,
		totalDocIDs:  0,
	}

	return openSearchSortDocIdxV1(ssdi, plainHdrBody, ssdi.InitOffset+uint64(parsed), size)
}

func openSearchSortDocIdxV1(ssdi *SearchSortDocIdxV1, plainHdrBody *SsdiPlainHdrBodyV1, plainHdrOffset, endOffset uint64) (*SearchSortDocIdxV1, error) {
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
	reader, err := ssblocks.NewBlockListReader(streamCrypto, plainHdrOffset+uint64(parsed), endOffset, initEmptySortDocIdxBlkV1)
	if err != nil {
		return nil, err
	}
	blockReader, ok := reader.(ssblocks.BlockListReaderV1)
	if !ok {
		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
	}
	ssdi.breader = blockReader
	return ssdi, nil
}

func (ssdi *SearchSortDocIdxV1) createWriter(sdc client.StrongDocClient) (io.WriteCloser, error) {
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

func (ssdi *SearchSortDocIdxV1) createReader(sdc client.StrongDocClient) (io.ReadCloser, uint64, error) {
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

func (ssdi *SearchSortDocIdxV1) WriteNextBlock() (*SearchSortDocIdxBlkV1, error) {
	if ssdi.writer == nil || ssdi.bwriter == nil {
		return nil, errors.Errorf("The search sorted document index is not open for writing")
	}

	if ssdi.block == nil {
		ssdi.block = CreateSearchSortDocIdxBlkV1("", uint64(ssdi.bwriter.GetMaxDataSize()))
	}

	err := ssdi.source.Reset()
	if err != nil {
		return nil, errors.New(err)
	}

	// Read all blocks from STI
	for {
		var docIDs []string
		var docVers []uint64
		docIDs, docVers, err = ssdi.source.ReadNextDocInfos()
		if err != nil {
			break
		}
		for i, docID := range docIDs {
			ssdi.block.AddDocVer(docID, docVers[i])
		}

	}

	if err != io.EOF {
		return nil, err
	}

	err = ssdi.flush()
	if err != nil {
		return nil, errors.New(err)
	}

	block := ssdi.block
	if block.IsFull() {
		ssdi.block = CreateSearchSortDocIdxBlkV1(block.highDocID,
			uint64(ssdi.bwriter.GetMaxDataSize()))
		return block, nil
	}

	ssdi.block = nil
	return block, io.EOF
}

func (ssdi *SearchSortDocIdxV1) ReadNextBlock() (*SearchSortDocIdxBlkV1, error) {
	if ssdi.reader == nil || ssdi.breader == nil {
		return nil, errors.Errorf("No reader available")
	}

	blockData, predictedSize, err := ssdi.breader.ReadNextBlockData()
	if err != nil {
		return nil, err
	}

	blk, ok := blockData.(*SearchSortDocIdxBlkV1)
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

func (ssdi *SearchSortDocIdxV1) ExistDocID(docID string) (bool, error) {
	if ssdi.breader == nil {
		return false, errors.Errorf("The search sorted document index is not open for reading")
	}

	blk, _, err := ssdi.breader.SearchBinary(docID, DocIDVerComparatorV1)
	if err != nil {
		return false, errors.New(err)
	}

	return (blk != nil), nil
}

func (ssdi *SearchSortDocIdxV1) FindDocID(docID string) (*DocIDVer, error) {
	if ssdi.breader == nil {
		return nil, errors.Errorf("The search sorted document index is not open for reading")
	}

	b, _, err := ssdi.breader.SearchBinary(docID, DocIDVerComparatorV1)
	if err != nil {
		return nil, errors.New(err)
	}

	if b != nil {
		blk, ok := b.(*SearchSortDocIdxBlkV1)
		if !ok {
			return nil, errors.Errorf("Cannot convert to SearchSortDocIdxBlkV1")
		}
		if docVer, exist := blk.docIDVerMap[docID]; exist {
			return &DocIDVer{docID, docVer}, nil
		}
	}

	return nil, nil
}

func (ssdi *SearchSortDocIdxV1) FindDocIDs(docIDs []string) (map[string]*DocIDVer, error) {
	if ssdi.breader == nil {
		return nil, errors.Errorf("The search sorted document index is not open for reading")
	}

	cloneDocIDs := make([]string, len(docIDs))
	copy(cloneDocIDs, docIDs)

	sort.Strings(cloneDocIDs)
	result := make(map[string]*DocIDVer)
	for _, docID := range docIDs {
		result[docID] = nil
	}

	err := ssdi.Reset()
	if err != nil {
		return nil, err
	}

	err = nil
	for i := 0; i < len(cloneDocIDs) && err == nil; {
		var blk *SearchSortDocIdxBlkV1 = nil
		blk, err = ssdi.ReadNextBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if blk != nil {
			for ; i < len(cloneDocIDs) && strings.Compare(cloneDocIDs[i], blk.highDocID) <= 0; i++ {
				if docVer, exist := blk.docIDVerMap[cloneDocIDs[i]]; exist {
					result[cloneDocIDs[i]] = &DocIDVer{cloneDocIDs[i], docVer}
				}
			}
		}
	}

	return result, nil
}

func (ssdi *SearchSortDocIdxV1) Reset() error {
	if ssdi.breader == nil {
		return errors.Errorf("The search sorted document index is not open for reading. Can not reset")
	}

	ssdi.totalDocIDs = 0
	return ssdi.breader.Reset()
}

func (ssdi *SearchSortDocIdxV1) Close() error {
	var ferr error = nil

	if ssdi.source != nil {
		ssdi.source.Close()
		ssdi.source = nil
	}

	if ssdi.writer != nil {
		if ssdi.block != nil && ssdi.block.totalDocIDs > 0 {
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

func (ssdi *SearchSortDocIdxV1) flush() error {
	if ssdi.bwriter == nil {
		return errors.Errorf("The search sorted document index is not open for writing")
	}

	block := ssdi.block.formatToBlockData()
	err := ssdi.bwriter.WriteBlockData(block)
	if err != nil {
		return err
	}
	ssdi.block = block
	ssdi.totalDocIDs += block.totalDocIDs
	return nil
}
