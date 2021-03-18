package docidxv1

import (
	"io"

	"github.com/go-errors/errors"

	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// The format off Document Term Index
//
// --------------------------------------------------------------------------
// |   Unencrypted    |                   Encrypted                         |
// --------------------------------------------------------------------------
// | Plaintext Header | Ciphertext Header | Block Header | .... Blocks .... |
// --------------------------------------------------------------------------

//////////////////////////////////////////////////////////////////
//
//                   Document Term Index
//
//////////////////////////////////////////////////////////////////

// DocTermIdxV1 is the Document Term Index V1
type DocTermIdxV1 struct {
	common.DtiVersionS
	DocID         string
	DocVer        uint64
	Key           *sscrypto.StrongSaltKey
	Nonce         []byte
	InitOffset    uint64
	PlainHdrBody  *DtiPlainHdrBodyV1
	CipherHdrBody *DtiCipherHdrBodyV1
	Writer        ssblocks.BlockListWriterV1
	Reader        ssblocks.BlockListReaderV1
	Block         *DocTermIdxBlkV1
	Source        DocTermSourceV1
	Store         interface{}
}

// CreateDocTermIdxV1 creates a document term index writer V1
func CreateDocTermIdxV1(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	source DocTermSourceV1, store interface{}, initOffset int64) (*DocTermIdxV1, error) {

	var err error
	writer, ok := store.(io.Writer)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Writer")
	}

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	if source == nil {
		return nil, errors.Errorf("A source is required")
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &DtiPlainHdrBodyV1{
		DtiVersionS: common.DtiVersionS{DtiVer: common.DTI_V1},
		KeyType:     key.Type.Name,
		DocID:       docID,
		DocVer:      docVer,
	}

	cipherHdrBody := &DtiCipherHdrBodyV1{
		BlockVersionS: common.BlockVersionS{BlockVer: common.DTI_BLOCK_V1},
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
	streamCrypto, err := crypto.CreateStreamCrypto(key, plainHdrBody.Nonce, store,
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
	blockWriter, err := ssblocks.NewBlockListWriterV1(streamCrypto, uint32(common.DTI_BLOCK_SIZE_MAX),
		uint64(initOffset+int64(len(plainHdrSerial)+len(cipherHdrSerial))))
	if err != nil {
		return nil, errors.New(err)
	}

	index := &DocTermIdxV1{common.DtiVersionS{DtiVer: common.DTI_V1},
		docID, docVer, key, plainHdrBody.Nonce, uint64(initOffset),
		plainHdrBody, cipherHdrBody, blockWriter, nil, nil, source, store}
	return index, nil
}

// OpenDocTermIdxV1 opens a document offset index reader V1
func OpenDocTermIdxV1(key *sscrypto.StrongSaltKey, store interface{}, initOffset uint64, endOffset uint64) (*DocTermIdxV1, error) {
	reader, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	plainHdr, parsed, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, err
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, err
	}

	version, err := common.DeserializeDtiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetDtiVersion() != common.DTI_V1 {
		return nil, errors.Errorf("Document term index version is not %v", common.DTI_V1)
	}

	// Parse plaintext header body
	plainHdrBody := &DtiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, err
	}
	return OpenDocTermIdxPrivV1(key, plainHdrBody, reader, initOffset, endOffset, initOffset+uint64(parsed))
}

// OpenDocTermIdxPrivV1 opens a document offset index reader V1
func OpenDocTermIdxPrivV1(key *sscrypto.StrongSaltKey, plainHdrBody *DtiPlainHdrBodyV1,
	store interface{}, initOffset uint64, endOffset uint64, plainHdrOffset uint64) (*DocTermIdxV1, error) {

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	_, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
	streamCrypto, err := crypto.OpenStreamCrypto(key, plainHdrBody.Nonce, store, int64(plainHdrOffset))
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

	cipherHdrBody := &DtiCipherHdrBodyV1{}
	cipherHdrBody, err = cipherHdrBody.deserialize(cipherHdrBodyData)
	if err != nil {
		return nil, err
	}

	// Create a block list reader using the streaming crypto so the blocks will be
	// decrypted.
	reader, err := ssblocks.NewBlockListReader(streamCrypto,
		plainHdrOffset+uint64(parsed), endOffset)
	if err != nil {
		return nil, err
	}
	blockReader, ok := reader.(ssblocks.BlockListReaderV1)
	if !ok {
		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
	}

	index := &DocTermIdxV1{common.DtiVersionS{DtiVer: plainHdrBody.GetDtiVersion()},
		plainHdrBody.DocID, plainHdrBody.DocVer, key, plainHdrBody.Nonce,
		uint64(initOffset), plainHdrBody, cipherHdrBody, nil, blockReader, nil,
		nil, store}
	return index, nil
}

// WriteNextBlock writes the next document term index block, and returns the written block.
// Returns io.EOF when last block is written.
func (idx *DocTermIdxV1) WriteNextBlock() (*DocTermIdxBlkV1, error) {
	if idx.Writer == nil {
		return nil, errors.Errorf("The document term index is not open for writing")
	}

	if idx.Block == nil {
		idx.Block = CreateDocTermIdxBlkV1("", uint64(idx.Writer.GetMaxDataSize()))
	}

	err := idx.Source.Reset()
	if err != nil {
		return nil, errors.New(err)
	}

	for err == nil {
		var term string
		term, _, err = idx.Source.GetNextTerm()
		if len(term) > 0 {
			idx.Block.AddTerm(term)
		}
	}

	if err == io.EOF {
		block := idx.Block

		serial, err := idx.Block.Serialize()
		if err != nil {
			return nil, errors.New(err)
		}

		err = idx.flush(serial)
		if err != nil {
			return nil, errors.New(err)
		}

		if block.IsFull() {
			return block, nil
		}

		return block, io.EOF
	}

	return nil, errors.New(err)
}

// ReadNextBlock returns the next document term index block
func (idx *DocTermIdxV1) ReadNextBlock() (*DocTermIdxBlkV1, error) {
	if idx.Reader == nil {
		return nil, errors.Errorf("The document term index is not open for reading")
	}

	b, err := idx.Reader.ReadNextBlock()
	if err != nil && err != io.EOF {
		return nil, errors.New(err)
	}

	if b != nil && len(b.GetData()) > 0 {
		block := CreateDocTermIdxBlkV1("", 0)
		blk, derr := block.Deserialize(b.GetData())
		if derr != nil {
			return nil, errors.New(derr)
		}

		return blk, err
	}

	return nil, err
}

// FindTerm attempts to find the specified term in the term index
func (idx *DocTermIdxV1) FindTerm(term string) (bool, error) {
	if idx.Reader == nil {
		return false, errors.Errorf("The document term index is not open for reading")
	}

	blk, err := idx.Reader.SearchBinary(term, DocTermComparatorV1)
	if err != nil {
		return false, errors.New(err)
	}

	return (blk != nil), nil
}

// ReadAllTerms reads all the terms in the index
func (idx *DocTermIdxV1) ReadAllTerms() ([]string, map[string]bool, error) {
	var err error = nil
	termList := make([]string, 0, 1000)
	termMap := make(map[string]bool)

	err = idx.Reset()
	if err != nil {
		return termList, termMap, err
	}

	for err == nil {
		var blk *DocTermIdxBlkV1 = nil
		blk, err = idx.ReadNextBlock()
		if err != nil && err != io.EOF {
			return termList, termMap, err
		}
		if blk != nil {
			for _, term := range blk.Terms {
				termList = append(termList, term)
				termMap[term] = true
			}
		}
	}

	return termList, termMap, nil
}

// Reset resets the term index for reading. Can not be done for writing
func (idx *DocTermIdxV1) Reset() error {
	if idx.Reader == nil {
		return errors.Errorf("The document term index is not open for reading. Can not reset")
	}

	return idx.Reader.Reset()
}

// Close writes any residual block data to output stream
func (idx *DocTermIdxV1) Close() error {
	if idx.Block != nil && idx.Block.totalTerms > 0 {
		serial, err := idx.Block.Serialize()
		if err != nil {
			return errors.New(err)
		}
		err = idx.flush(serial)
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

func (idx *DocTermIdxV1) flush(data []byte) error {
	if idx.Writer == nil {
		return errors.Errorf("The document term index is not open for writing")
	}

	_, err := idx.Writer.WriteBlockData(data)
	if err != nil {
		return errors.New(err)
	}

	idx.Block = CreateDocTermIdxBlkV1(idx.Block.highTerm, uint64(idx.Writer.GetMaxDataSize()))
	return nil
}

// GetDocID gets the document ID
func (idx *DocTermIdxV1) GetDocID() string {
	return idx.DocID
}

// GetDocVersion gets the document version
func (idx *DocTermIdxV1) GetDocVersion() uint64 {
	return idx.DocVer
}
