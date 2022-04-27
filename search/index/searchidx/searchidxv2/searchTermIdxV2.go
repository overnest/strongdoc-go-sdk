package searchidxv2

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// SearchTermIdxV2 is a structure for search term index V2
type SearchTermIdxV2 struct {
	common.StiVersionS
	TermID        string
	Owner         common.SearchIdxOwner
	OldSti        common.SearchTermIdx
	DelDocs       *DeletedDocsV2
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	IndexNonce    []byte
	InitOffset    uint64
	TokenizerType tokenizer.TokenizerType
	updateID      string
	writer        io.WriteCloser
	reader        io.ReadCloser
	bwriter       ssblocks.BlockListWriterV1
	breader       ssblocks.BlockListReaderV1
	storeSize     uint64
}

// CreateSearchTermIdxV2 creates a search term index writer V2
func CreateSearchTermIdxV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, termID string,
	termKey, indexKey *sscrypto.StrongSaltKey, oldSti common.SearchTermIdx,
	delDocs *DeletedDocsV2, tokenizerType tokenizer.TokenizerType) (*SearchTermIdxV2, error) {

	var err error
	sti := &SearchTermIdxV2{
		StiVersionS:   common.StiVersionS{StiVer: common.STI_V2},
		TermID:        termID,
		Owner:         owner,
		OldSti:        oldSti,
		DelDocs:       delDocs,
		TermKey:       termKey,
		IndexKey:      indexKey,
		IndexNonce:    nil,
		InitOffset:    0,
		TokenizerType: tokenizerType,
		updateID:      "",
		writer:        nil,
		reader:        nil,
		bwriter:       nil,
		breader:       nil,
		storeSize:     0,
	}

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if midStreamKey, ok := indexKey.Key.(sscryptointf.KeyMidstream); ok {
		sti.IndexNonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	err = sti.createWriter(sdc)
	if err != nil {
		return nil, err
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &StiPlainHdrBodyV2{
		StiVersionS: common.StiVersionS{StiVer: common.STI_V2},
		KeyType:     sti.IndexKey.Type.Name,
		Nonce:       sti.IndexNonce,
		TermID:      sti.TermID,
		UpdateID:    sti.updateID,
	}

	cipherHdrBody := &StiCipherHdrBodyV2{
		BlockVersionS: common.BlockVersionS{BlockVer: common.STI_BLOCK_V2},
		TokenizerType: tokenizerType,
	}

	plainHdrBodySerial, err := plainHdrBody.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdr := ssheaders.CreatePlainHdr(ssheaders.HeaderTypeJSONGzip, plainHdrBodySerial)
	plainHdrSerial, err := plainHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBodySerial, err := cipherHdrBody.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdr := ssheaders.CreateCipherHdr(ssheaders.HeaderTypeJSONGzip, cipherHdrBodySerial)
	cipherHdrSerial, err := cipherHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	// Write the plaintext header to storage
	n, err := sti.writer.Write(plainHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(plainHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire plaintext header")
	}

	// Initialize the streaming crypto to encrypt ciphertext header and the
	// blocks after that
	streamCrypto, err := crypto.CreateStreamCrypto(sti.IndexKey, plainHdrBody.Nonce,
		sti.writer, int64(sti.InitOffset)+int64(n))
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
	sti.bwriter, err = ssblocks.NewBlockListWriterV1(streamCrypto, 0,
		sti.InitOffset+uint64(len(plainHdrSerial)+len(cipherHdrSerial)))
	if err != nil {
		return nil, errors.New(err)
	}

	return sti, nil
}

// OpenSearchTermIdxV2 opens a search term index reader V2
func OpenSearchTermIdxV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, termID string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*SearchTermIdxV2, error) {

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	reader, size, err := createStiReader(sdc, owner, termID, updateID)
	if err != nil {
		return nil, err
	}

	plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, err
	}

	return OpenSearchTermIdxPrivV2(owner, termID, updateID, termKey, indexKey,
		reader, plainHdr, 0, uint64(plainHdrSize), size)
}

func OpenSearchTermIdxPrivV2(owner common.SearchIdxOwner, termID, updateID string,
	termKey, indexKey *sscrypto.StrongSaltKey, reader io.ReadCloser, plainHdr ssheaders.Header,
	initOffset, plainHdrSize, endOffset uint64) (*SearchTermIdxV2, error) {

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, err
	}

	version, err := common.DeserializeStiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetStiVersion() != common.STI_V2 {
		return nil, errors.Errorf("Search term index version %v is not %v",
			version.GetStiVersion(), common.STI_V2)
	}

	// Parse plaintext header body
	plainHdrBody := &StiPlainHdrBodyV2{}
	plainHdrBody, err = plainHdrBody.Deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	sti := &SearchTermIdxV2{
		StiVersionS: common.StiVersionS{StiVer: plainHdrBody.GetStiVersion()},
		TermID:      termID,
		Owner:       owner,
		OldSti:      nil,
		DelDocs:     nil,
		TermKey:     termKey,
		IndexKey:    indexKey,
		IndexNonce:  plainHdrBody.Nonce,
		InitOffset:  0,
		updateID:    updateID,
		writer:      nil,
		reader:      reader,
		bwriter:     nil,
		breader:     nil,
		storeSize:   endOffset,
	}

	plainHdrOffset := initOffset + plainHdrSize

	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
	streamCrypto, err := crypto.OpenStreamCrypto(sti.IndexKey, sti.IndexNonce, sti.reader, int64(plainHdrOffset))
	if err != nil {
		return nil, errors.New(err)
	}

	// Read the ciphertext header from storage
	cipherHdr, cipherHdrSize, err := ssheaders.DeserializeCipherHdrStream(streamCrypto)
	if err != nil {
		return nil, err
	}

	cipherHdrBodyData, err := cipherHdr.GetBody()
	if err != nil {
		return nil, err
	}

	cipherHdrBody := &StiCipherHdrBodyV2{}
	cipherHdrBody, err = cipherHdrBody.Deserialize(cipherHdrBodyData)
	if err != nil {
		return nil, err
	}
	sti.TokenizerType = cipherHdrBody.TokenizerType

	// Create a block list reader using the streaming crypto so the blocks will be
	// decrypted.
	bReader, err := ssblocks.NewBlockListReader(streamCrypto, plainHdrOffset+uint64(cipherHdrSize), endOffset, func() interface{} {
		return CreateSearchTermIdxBlkV2(0)
	})
	if err != nil {
		return nil, err
	}
	blockReader, ok := bReader.(ssblocks.BlockListReaderV1)
	if !ok {
		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
	}
	sti.breader = blockReader
	return sti, nil
}

func (sti *SearchTermIdxV2) createWriter(sdc client.StrongDocClient) error {
	if sti.writer != nil {
		return nil
	}

	writer, updateID, err := common.OpenSearchTermIndexWriter(sdc, sti.Owner, sti.TermID)
	if err != nil {
		return err
	}
	sti.writer = writer
	sti.updateID = updateID
	return nil
}

func (sti *SearchTermIdxV2) createReader(sdc client.StrongDocClient) (io.ReadCloser, uint64, error) {
	if sti.reader != nil {
		return sti.reader, sti.storeSize, nil
	}

	return createStiReader(sdc, sti.Owner, sti.TermID, sti.updateID)
}

func createStiReader(sdc client.StrongDocClient, owner common.SearchIdxOwner, termID, updateID string) (io.ReadCloser, uint64, error) {
	return common.OpenSearchTermIndexReader(sdc, owner, termID, updateID)
}

func (sti *SearchTermIdxV2) GetMaxBlockDataSize() uint64 {
	if sti.bwriter != nil {
		return uint64(sti.bwriter.GetMaxDataSize())
	}

	return common.STI_BLOCK_SIZE_MAX
}

func (sti *SearchTermIdxV2) WriteNextBlock(block *SearchTermIdxBlkV2) error {
	if sti.writer == nil || sti.bwriter == nil {
		return errors.Errorf("No writer available")
	}

	if block != nil {
		return sti.bwriter.WriteBlockData(block)
	}

	return nil
}

func (sti *SearchTermIdxV2) ReadNextBlock() (*SearchTermIdxBlkV2, error) {
	if sti.reader == nil || sti.breader == nil {
		return nil, errors.Errorf("No reader available")
	}

	blockData, jsonSize, err := sti.breader.ReadNextBlockData()
	if err != nil {
		return nil, err
	}
	stib, _ := blockData.(*SearchTermIdxBlkV2)

	stib.predictedJSONSize = uint64((jsonSize))
	return stib, err
}

func (sti *SearchTermIdxV2) Reset() error {
	if sti.breader == nil {
		return errors.Errorf("The search term index is not open for reading. Can not reset")
	}

	return sti.breader.Reset()
}

func (sti *SearchTermIdxV2) Close() error {
	var werr, rerr error = nil, nil

	if sti.writer != nil {
		werr = sti.writer.Close()
	}

	if sti.reader != nil {
		rerr = sti.reader.Close()
	}

	if werr != nil {
		return werr
	}

	if rerr != nil {
		return rerr
	}

	return nil
}
