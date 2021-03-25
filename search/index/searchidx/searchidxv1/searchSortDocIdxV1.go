package searchidxv1

import (
	"github.com/overnest/strongdoc-go-sdk/client"
	"io"
	"sort"
	"strings"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// SearchSortDocIdxV1 is a structure for search sorted document index V1
type SearchSortDocIdxV1 struct {
	common.SsdiVersionS
	Owner       common.SearchIdxOwner
	Term        string
	TermKey     *sscrypto.StrongSaltKey
	IndexKey    *sscrypto.StrongSaltKey
	IndexNonce  []byte
	InitOffset  uint64
	termHmac    string
	updateID    string
	sti         *SearchTermIdxV1
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
	termKey, indexKey *sscrypto.StrongSaltKey) (*SearchSortDocIdxV1, error) {

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
		sti:          nil,
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

	//t1 := time.Now()
	ssdi.sti, err = OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateID)
	if err != nil {
		return nil, err
	}
	//t2 := time.Now()
	//fmt.Println("open search term index", t2.Sub(t1).Milliseconds(), "ms")

	ssdi.termHmac, err = createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}

	//t3 := time.Now()
	_, err = ssdi.createWriter(sdc)
	if err != nil {
		return nil, err
	}
	//t4 := time.Now()

	//fmt.Println("createWriter", t4.Sub(t3).Milliseconds(), "ms")

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
		sti:          nil,
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
	reader, err := ssblocks.NewBlockListReader(streamCrypto, plainHdrOffset+uint64(parsed), endOffset)
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

	err := ssdi.sti.Reset()
	if err != nil {
		return nil, errors.New(err)
	}

	// Read all blocks from STI
	for err == nil {
		var stib *SearchTermIdxBlkV1 = nil
		stib, err = ssdi.sti.ReadNextBlock()
		if stib != nil {
			for docID, verOff := range stib.DocVerOffset {
				ssdi.block.AddDocVer(docID, verOff.Version)
			}
		}
	}

	if err != io.EOF {
		return nil, err
	}

	block := ssdi.block
	err = ssdi.flush(block)
	if err != nil {
		return nil, errors.New(err)
	}

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

	b, err := ssdi.breader.ReadNextBlock()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if b != nil && len(b.GetData()) > 0 {
		ssdib := CreateSearchSortDocIdxBlkV1("", 0)
		return ssdib.Deserialize(b.GetData())
	}

	return nil, err
}

func (ssdi *SearchSortDocIdxV1) ExistDocID(docID string) (bool, error) {
	if ssdi.breader == nil {
		return false, errors.Errorf("The search sorted document index is not open for reading")
	}

	blk, err := ssdi.breader.SearchBinary(docID, DocIDVerComparatorV1)
	if err != nil {
		return false, errors.New(err)
	}

	return (blk != nil), nil
}

func (ssdi *SearchSortDocIdxV1) FindDocID(docID string) (*DocIDVer, error) {
	if ssdi.breader == nil {
		return nil, errors.Errorf("The search sorted document index is not open for reading")
	}

	b, err := ssdi.breader.SearchBinary(docID, DocIDVerComparatorV1)
	if err != nil {
		return nil, errors.New(err)
	}

	if b != nil {
		blk := CreateSearchSortDocIdxBlkV1("", 0)
		blk, err = blk.Deserialize(b.GetData())
		if err != nil {
			return nil, err
		}

		if blk != nil {
			if docVer, exist := blk.docIDVerMap[docID]; exist {
				return &DocIDVer{docID, docVer}, nil
			}
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

	if ssdi.sti != nil {
		ssdi.sti.Close()
		ssdi.sti = nil
	}

	if ssdi.writer != nil {
		if ssdi.block != nil && ssdi.block.totalDocIDs > 0 {
			ferr = utils.FirstError(ferr, ssdi.flush(ssdi.block))
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

func (ssdi *SearchSortDocIdxV1) flush(block *SearchSortDocIdxBlkV1) error {
	if ssdi.bwriter == nil {
		return errors.Errorf("The search sorted document index is not open for writing")
	}

	serial, err := ssdi.block.Serialize()
	if err != nil {
		return err
	}

	_, err = ssdi.bwriter.WriteBlockData(serial)
	if err != nil {
		return err
	}

	ssdi.totalDocIDs += block.totalDocIDs
	return nil
}
