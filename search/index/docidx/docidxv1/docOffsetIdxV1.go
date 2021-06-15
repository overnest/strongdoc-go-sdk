package docidxv1

import (
	"github.com/overnest/strongdoc-go-sdk/client"
	"io"
	"sort"

	"github.com/go-errors/errors"

	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// The format off Document Offset Index
//
// --------------------------------------------------------------------------
// |   Unencrypted    |                   Encrypted                         |
// --------------------------------------------------------------------------
// | Plaintext Header | Ciphertext Header | Block Header | .... Blocks .... |
// --------------------------------------------------------------------------

//////////////////////////////////////////////////////////////////
//
//                   Document Offset Index
//
//////////////////////////////////////////////////////////////////

// DocOffsetIdxV1 is the Document Offset Index V1
type DocOffsetIdxV1 struct {
	common.DoiVersionS
	DocID         string
	DocVer        uint64
	Key           *sscrypto.StrongSaltKey
	Nonce         []byte
	InitOffset    uint64
	PlainHdrBody  *DoiPlainHdrBodyV1
	CipherHdrBody *DoiCipherHdrBodyV1
	Writer        ssblocks.BlockListWriterV1
	Reader        ssblocks.BlockListReaderV1
	Block         *DocOffsetIdxBlkV1
	Store         interface{}
}

var initDocOffsetIdxV1 = func() interface{} {
	return &DocOffsetIdxBlkV1{}
}

// CreateDocOffsetIdxV1 creates a document offset index writer V1
func CreateDocOffsetIdxV1(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, initOffset int64) (*DocOffsetIdxV1, error) {
	var err error
	writer, err := common.OpenDocOffsetIdxWriter(sdc, docID, docVer)
	if err != nil {
		return nil, err
	}

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &DoiPlainHdrBodyV1{
		DoiVersionS: common.DoiVersionS{DoiVer: common.DOI_V1},
		KeyType:     key.Type.Name,
		DocID:       docID,
		DocVer:      docVer,
	}

	cipherHdrBody := &DoiCipherHdrBodyV1{
		BlockVersionS: common.BlockVersionS{BlockVer: common.DOI_BLOCK_V1},
	}

	if midStreamKey, ok := key.Key.(sscryptointf.KeyMidstream); ok {
		plainHdrBody.Nonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", key.Type.Name)
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
	n, err := writer.Write(plainHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(plainHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire plaintext header")
	}

	// Initialize the streaming crypto to encrypt ciphertext header and the
	// blocks after that
	streamCrypto, err := crypto.CreateStreamCrypto(key, plainHdrBody.Nonce, writer,
		initOffset+int64(n))
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
	blockWriter, err := ssblocks.NewBlockListWriterV1(streamCrypto, 0,
		uint64(initOffset+int64(len(plainHdrSerial)+len(cipherHdrSerial))))
	if err != nil {
		return nil, errors.New(err)
	}

	index := &DocOffsetIdxV1{common.DoiVersionS{DoiVer: common.DOI_V1},
		docID, docVer, key, plainHdrBody.Nonce, uint64(initOffset),
		plainHdrBody, cipherHdrBody, blockWriter, nil, nil, writer}
	return index, nil
}

// OpenDocOffsetIdxV1 opens a document offset index reader V1
func OpenDocOffsetIdxV1(key *sscrypto.StrongSaltKey, store interface{}, initOffset int64) (*DocOffsetIdxV1, error) {
	reader, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	plainHdr, parsed, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, errors.New(err)
	}

	version, err := common.DeserializeDoiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	if version.GetDoiVersion() != common.DOI_V1 {
		return nil, errors.Errorf("Document offset index is not version %v", common.DOI_V1)
	}

	// Parse plaintext header body
	plainHdrBody := &DoiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}
	return OpenDocOffsetIdxPrivV1(key, plainHdrBody, reader, initOffset+int64(parsed))
}

// OpenDocOffsetIdxPrivV1 opens a document offset index reader V1
func OpenDocOffsetIdxPrivV1(key *sscrypto.StrongSaltKey, plainHdrBody *DoiPlainHdrBodyV1,
	store interface{}, initOffset int64) (*DocOffsetIdxV1, error) {

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	_, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
	streamCrypto, err := crypto.OpenStreamCrypto(key, plainHdrBody.Nonce, store, initOffset)
	if err != nil {
		return nil, errors.New(err)
	}

	// Read the ciphertext header from storage
	cipherHdr, parsed, err := ssheaders.DeserializeCipherHdrStream(streamCrypto)
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBodyData, err := cipherHdr.GetBody()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBody := &DoiCipherHdrBodyV1{}
	cipherHdrBody, err = cipherHdrBody.deserialize(cipherHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	// Create a block list reader using the streaming crypto so the blocks will be
	// decrypted.
	reader, err := ssblocks.NewBlockListReader(streamCrypto,
		uint64(initOffset+int64(parsed)), 0, initDocOffsetIdxV1)
	if err != nil {
		return nil, errors.New(err)
	}
	blockReader, ok := reader.(ssblocks.BlockListReaderV1)
	if !ok {
		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
	}

	index := &DocOffsetIdxV1{common.DoiVersionS{DoiVer: plainHdrBody.GetDoiVersion()},
		plainHdrBody.DocID, plainHdrBody.DocVer, key, plainHdrBody.Nonce,
		uint64(initOffset), plainHdrBody, cipherHdrBody, nil, blockReader, nil, store}
	return index, nil
}

// AddTermOffset adds search term and offset to a document offset index block
func (idx *DocOffsetIdxV1) AddTermOffset(term string, offset uint64) error {
	if idx.Writer == nil {
		return errors.Errorf("The document offset index is not open for writing")
	}

	if idx.Block == nil {
		idx.Block = &DocOffsetIdxBlkV1{
			TermLoc:           make(map[string][]uint64),
			predictedJSONSize: baseDoiBlockJSONSize,
		}
	}

	idx.Block.AddTermOffset(term, offset)
	if idx.Block.predictedJSONSize > uint64(common.DOI_BLOCK_SIZE_MAX) {
		return idx.flush()
	}

	return nil
}

// ReadNextBlock returns the next document offset index block
func (idx *DocOffsetIdxV1) ReadNextBlock() (*DocOffsetIdxBlkV1, error) {
	if idx.Reader == nil {
		return nil, errors.Errorf("The document offset index is not open for reading")
	}

	blockData, jsonSize, err := idx.Reader.ReadNextBlockData()
	if err != nil {
		return nil, err
	}

	blk, ok := blockData.(*DocOffsetIdxBlkV1)
	if !ok {
		return nil, errors.Errorf("Cannot convert to DocOffsetIdxBlkV1")
	}

	blk.predictedJSONSize = uint64(jsonSize)
	return blk, nil
}

// ReadAllTermLoc returns all the term location index
func (idx *DocOffsetIdxV1) ReadAllTermLoc() ([]string, map[string][]uint64, error) {
	terms := make([]string, 0)
	termLocMap := make(map[string][]uint64)

	if idx.Reader == nil {
		return terms, termLocMap, errors.Errorf("The document offset index is not open for reading")
	}

	err := idx.Reset()
	if err != nil {
		return terms, termLocMap, err
	}

	for err == nil {
		var blk *DocOffsetIdxBlkV1
		blk, err = idx.ReadNextBlock()
		if err != nil && err != io.EOF {
			return terms, termLocMap, err
		}

		if blk != nil && len(blk.TermLoc) > 0 {
			for term, locs := range blk.TermLoc {
				rlocs := termLocMap[term]
				termLocMap[term] = append(rlocs, locs...)
			}
		}
	}

	terms = make([]string, 0, len(termLocMap))
	for term := range termLocMap {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	return terms, termLocMap, nil
}

// Reset resets the offset index for reading. Can not be done for writing
func (idx *DocOffsetIdxV1) Reset() error {
	if idx.Reader == nil {
		return errors.Errorf("The document offset index is not open for reading. Can not reset")
	}

	return idx.Reader.Reset()
}

// Close writes any residual block data to output stream
func (idx *DocOffsetIdxV1) Close() error {
	if idx.Block != nil {
		err := idx.flush()
		if err != nil {
			return err
		}
	}
	if idx.Store != nil {
		storeCloser, ok := (idx.Store).(io.Closer)
		if !ok {
			return errors.Errorf("The passed in storage does not implement io.Closer")
		}
		err := storeCloser.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (idx *DocOffsetIdxV1) flush() error {
	if idx.Block != nil {
		err := idx.Writer.WriteBlockData(idx.Block)
		if err != nil {
			return err
		}
	}
	idx.Block = nil
	return nil
}

// GetDocID gets the document ID
func (idx *DocOffsetIdxV1) GetDocID() string {
	return idx.DocID
}

// GetDocVersion gets the document version
func (idx *DocOffsetIdxV1) GetDocVersion() uint64 {
	return idx.DocVer
}
