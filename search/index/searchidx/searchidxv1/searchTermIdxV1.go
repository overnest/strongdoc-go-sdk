package searchidxv1

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// SearchTermIdxV1 is a structure for search term index V1
type SearchTermIdxV1 struct {
	common.StiVersionS
	Term       string
	Owner      common.SearchIdxOwner
	OldSti     common.SearchTermIdx
	DelDocs    *DeletedDocsV1
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	IndexNonce []byte
	InitOffset uint64
	termHmac   string
	updateID   string
	writer     io.WriteCloser
	reader     io.ReadCloser
	bwriter    ssblocks.BlockListWriterV1
	breader    ssblocks.BlockListReaderV1
	storeSize  uint64
}

// CreateSearchTermIdxV1 creates a search term index writer V1
func CreateSearchTermIdxV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string,
	termKey, indexKey *sscrypto.StrongSaltKey, oldSti common.SearchTermIdx,
	delDocs *DeletedDocsV1) (*SearchTermIdxV1, error) {

	var err error
	sti := &SearchTermIdxV1{
		StiVersionS: common.StiVersionS{StiVer: common.STI_V1},
		Term:        term,
		Owner:       owner,
		OldSti:      oldSti,
		DelDocs:     delDocs,
		TermKey:     termKey,
		IndexKey:    indexKey,
		IndexNonce:  nil,
		InitOffset:  0,
		termHmac:    "",
		updateID:    "",
		writer:      nil,
		reader:      nil,
		bwriter:     nil,
		breader:     nil,
		storeSize:   0,
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

	sti.termHmac, err = sti.createTermHmac()
	if err != nil {
		return nil, err
	}

	err = sti.createWriter(sdc)
	if err != nil {
		return nil, err
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &StiPlainHdrBodyV1{
		StiVersionS: common.StiVersionS{StiVer: common.STI_V1},
		KeyType:     sti.IndexKey.Type.Name,
		Nonce:       sti.IndexNonce,
		TermHmac:    sti.termHmac,
		UpdateID:    sti.updateID,
	}

	cipherHdrBody := &StiCipherHdrBodyV1{
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
	sti.bwriter, err = ssblocks.NewBlockListWriterV1(streamCrypto, uint32(common.STI_BLOCK_SIZE_MAX),
		sti.InitOffset+uint64(len(plainHdrSerial)+len(cipherHdrSerial)))
	if err != nil {
		return nil, errors.New(err)
	}

	return sti, nil
}

// OpenSearchTermIdxV1 opens a search term index reader V1
func OpenSearchTermIdxV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*SearchTermIdxV1, error) {
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

	reader, size, err := createStiReader(sdc, owner, termHmac, updateID)
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

	version, err := common.DeserializeStiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetStiVersion() != common.STI_V1 {
		return nil, errors.Errorf("Search term index version %v is not %v",
			version.GetStiVersion(), common.STI_V1)
	}

	// Parse plaintext header body
	plainHdrBody := &StiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	sti := &SearchTermIdxV1{
		StiVersionS: common.StiVersionS{StiVer: plainHdrBody.GetStiVersion()},
		Term:        term,
		Owner:       owner,
		OldSti:      nil,
		DelDocs:     nil,
		TermKey:     termKey,
		IndexKey:    indexKey,
		IndexNonce:  plainHdrBody.Nonce,
		InitOffset:  0,
		termHmac:    termHmac,
		updateID:    updateID,
		writer:      nil,
		reader:      reader,
		bwriter:     nil,
		breader:     nil,
		storeSize:   size,
	}

	return openSearchTermIdxV1(sti, plainHdrBody, sti.InitOffset+uint64(parsed), size)
}

func openSearchTermIdxV1(sti *SearchTermIdxV1, plainHdrBody *StiPlainHdrBodyV1, plainHdrOffset, endOffset uint64) (*SearchTermIdxV1, error) {
	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
	streamCrypto, err := crypto.OpenStreamCrypto(sti.IndexKey, sti.IndexNonce, sti.reader, int64(plainHdrOffset))
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

	cipherHdrBody := &StiCipherHdrBodyV1{}
	cipherHdrBody, err = cipherHdrBody.deserialize(cipherHdrBodyData)
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
	sti.breader = blockReader
	return sti, nil
}

func (sti *SearchTermIdxV1) createTermHmac() (string, error) {
	return createTermHmac(sti.Term, sti.TermKey)
}

func (sti *SearchTermIdxV1) createWriter(sdc client.StrongDocClient) error {
	if sti.writer != nil {
		return nil
	}

	writer, updateID, err := common.OpenSearchTermIndexWriter(sdc, sti.Owner, sti.termHmac)
	if err != nil {
		return err
	}
	sti.writer = writer
	sti.updateID = updateID
	return nil
}

func (sti *SearchTermIdxV1) createReader(sdc client.StrongDocClient) (io.ReadCloser, uint64, error) {
	if sti.reader != nil {
		return sti.reader, sti.storeSize, nil
	}

	return createStiReader(sdc, sti.Owner, sti.termHmac, sti.updateID)
}

func createStiReader(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac, updateID string) (io.ReadCloser, uint64, error) {
	return common.OpenSearchTermIndexReader(sdc, owner, termHmac, updateID)
}

func (sti *SearchTermIdxV1) GetMaxBlockDataSize() uint64 {
	if sti.bwriter != nil {
		return uint64(sti.bwriter.GetMaxDataSize())
	}

	return common.STI_BLOCK_SIZE_MAX
}

func (sti *SearchTermIdxV1) WriteNextBlock(block *SearchTermIdxBlkV1) error {
	if sti.writer == nil || sti.bwriter == nil {
		return errors.Errorf("No writer available")
	}

	if block != nil {
		b, err := block.Serialize()
		if err != nil {
			return err
		}
		_, err = sti.bwriter.WriteBlockData(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sti *SearchTermIdxV1) ReadNextBlock() (*SearchTermIdxBlkV1, error) {
	if sti.reader == nil || sti.breader == nil {
		return nil, errors.Errorf("No reader available")
	}

	b, err := sti.breader.ReadNextBlock()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if b != nil && len(b.GetData()) > 0 {
		stib := CreateSearchTermIdxBlkV1(0)
		return stib.Deserialize(b.GetData())
	}

	return nil, err
}

func (sti *SearchTermIdxV1) Reset() error {
	if sti.breader == nil {
		return errors.Errorf("The search term index is not open for reading. Can not reset")
	}

	return sti.breader.Reset()
}

func (sti *SearchTermIdxV1) Close() error {
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
