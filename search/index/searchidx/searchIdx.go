package searchidx

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-errors/errors"
)

const (
	// SI_V1  = uint32(1)
	// SI_VER = SI_V1

	SI_OWNER_ORG = SearchIdxOwnerType("ORG")
	SI_OWNER_USR = SearchIdxOwnerType("USR")

	STI_BLOCK_V1  = uint32(1)
	STI_BLOCK_VER = STI_BLOCK_V1

	STI_BLOCK_SIZE_MAX       = uint64(1024 * 1024 * 5) // 5MB
	STI_BLOCK_MARGIN_PERCENT = uint64(10)              // 10% margin

	STI_V1              = uint32(1)
	STI_VER             = STI_V1
	STI_TERM_BATCH_SIZE = 100 // Process terms in batches of 100
)

// GetSearchIdxPathPrefix gets the search index path prefix
func GetSearchIdxPathPrefix() string {
	return "/tmp/search"
}

//////////////////////////////////////////////////////////////////
//
//                   Search Index Owner
//
//////////////////////////////////////////////////////////////////

// SearchIdxOwnerType is the owner type of the search index
type SearchIdxOwnerType string

// SearchIdxOwner is the search index owner interface
type SearchIdxOwner interface {
	GetOwnerType() SearchIdxOwnerType
	GetOwnerID() string
	fmt.Stringer
}

type searchIdxOwner struct {
	ownerType SearchIdxOwnerType
	ownerID   string
}

func (sio *searchIdxOwner) GetOwnerType() SearchIdxOwnerType {
	return sio.ownerType
}

func (sio *searchIdxOwner) GetOwnerID() string {
	return sio.ownerID
}

func (sio *searchIdxOwner) String() string {
	return fmt.Sprintf("%v_%v", sio.GetOwnerType(), sio.GetOwnerID())
}

// CreateSearchIdxOwner creates a new searchh index owner
func CreateSearchIdxOwner(ownerType SearchIdxOwnerType, ownerID string) SearchIdxOwner {
	return &searchIdxOwner{ownerType, ownerID}
}

// //////////////////////////////////////////////////////////////////
// //
// //                       Search Index
// //
// //////////////////////////////////////////////////////////////////

// // SearchIdx store search index version
// type SearchIdx interface {
// 	GetSiVersion() uint32
// }

// // SiVersionS is structure used to store search index version
// type SiVersionS struct {
// 	SiVer uint32
// }

// // GetSiVersion retrieves the search index version number
// func (h *SiVersionS) GetSiVersion() uint32 {
// 	return h.SiVer
// }

// // Deserialize deserializes the data into version number object
// func (h *SiVersionS) Deserialize(data []byte) (*SiVersionS, error) {
// 	err := json.Unmarshal(data, h)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	return h, nil
// }

// // DeserializeSiVersion deserializes the data into version number object
// func DeserializeSiVersion(data []byte) (*SiVersionS, error) {
// 	h := &SiVersionS{}
// 	return h.Deserialize(data)
// }

//////////////////////////////////////////////////////////////////
//
//                     Search Term Index
//
//////////////////////////////////////////////////////////////////

// SearchTermIdx store search term index version
type SearchTermIdx interface {
	GetStiVersion() uint32
	io.Closer
}

// StiVersionS is structure used to store search term index version
type StiVersionS struct {
	StiVer uint32
}

// GetStiVersion retrieves the search term index version number
func (h *StiVersionS) GetStiVersion() uint32 {
	return h.StiVer
}

// Deserialize deserializes the data into version number object
func (h *StiVersionS) Deserialize(data []byte) (*StiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeStiVersion deserializes the data into version number object
func DeserializeStiVersion(data []byte) (*StiVersionS, error) {
	h := &StiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                 Search Term Index Block
//
//////////////////////////////////////////////////////////////////

// BlockVersion is structure used to store search index block version
type BlockVersion struct {
	BlockVer uint32
}

// GetBlockVersion retrieves the search index version block number
func (h *BlockVersion) GetBlockVersion() uint32 {
	return h.BlockVer
}

// Deserialize deserializes the data into version number object
func (h *BlockVersion) Deserialize(data []byte) (*BlockVersion, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeBlockVersion deserializes the data into version number object
func DeserializeBlockVersion(data []byte) (*BlockVersion, error) {
	h := &BlockVersion{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                          Search Index
//
//////////////////////////////////////////////////////////////////

// // SearchIdxV1 is the Search Index V1
// type SearchIdxV1 struct {
// 	DtiVersionS
// 	DocID         string
// 	DocVer        uint64
// 	Key           *sscrypto.StrongSaltKey
// 	Nonce         []byte
// 	InitOffset    uint64
// 	PlainHdrBody  *DtiPlainHdrBodyV1
// 	CipherHdrBody *DtiCipherHdrBodyV1
// 	Writer        ssblocks.BlockListWriterV1
// 	Reader        ssblocks.BlockListReaderV1
// 	Block         *SearchIdxBlkV1
// 	Source        SearchSourceV1
// }

// // CreateSearchIdxV1 creates a search index writer V1
// func CreateSearchIdxV1(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
// 	source SearchSourceV1, store interface{}, initOffset int64) (*SearchIdxV1, error) {

// 	var err error
// 	writer, ok := store.(io.Writer)
// 	if !ok {
// 		return nil, errors.Errorf("The passed in storage does not implement io.Writer")
// 	}

// 	if key.Type != sscrypto.Type_XChaCha20 {
// 		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
// 			key.Type.Name, sscrypto.Type_XChaCha20.Name)
// 	}

// 	if source == nil {
// 		return nil, errors.Errorf("A source is required")
// 	}

// 	// Create plaintext and ciphertext headers
// 	plainHdrBody := &DtiPlainHdrBodyV1{
// 		DtiVersionS: DtiVersionS{DtiVer: DTI_V1},
// 		KeyType:     key.Type.Name,
// 		DocID:       docID,
// 		DocVer:      docVer,
// 	}

// 	cipherHdrBody := &DtiCipherHdrBodyV1{
// 		BlockVersion: BlockVersion{BlockVer: DTI_BLOCK_V1},
// 	}

// 	if midStreamKey, ok := key.Key.(sscryptointf.KeyMidstream); ok {
// 		plainHdrBody.Nonce, err = midStreamKey.GenerateNonce()
// 		if err != nil {
// 			return nil, errors.New(err)
// 		}
// 	} else {
// 		return nil, errors.Errorf("The key type %v is not a midstream key", key.Type.Name)
// 	}

// 	plainHdrBodySerial, err := plainHdrBody.serialize()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	plainHdr := ssheaders.CreatePlainHdr(ssheaders.HeaderTypeJSONGzip, plainHdrBodySerial)
// 	plainHdrSerial, err := plainHdr.Serialize()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	cipherHdrBodySerial, err := cipherHdrBody.serialize()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	cipherHdr := ssheaders.CreateCipherHdr(ssheaders.HeaderTypeJSONGzip, cipherHdrBodySerial)
// 	cipherHdrSerial, err := cipherHdr.Serialize()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	// Write the plaintext header to storage
// 	n, err := writer.Write(plainHdrSerial)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	if n != len(plainHdrSerial) {
// 		return nil, errors.Errorf("Failed to write the entire plaintext header")
// 	}

// 	// Initialize the streaming crypto to encrypt ciphertext header and the
// 	// blocks after that
// 	streamCrypto, err := crypto.CreateStreamCrypto(key, plainHdrBody.Nonce, store,
// 		initOffset+int64(n))
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	// Write the ciphertext header to storage
// 	n, err = streamCrypto.Write(cipherHdrSerial)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	if n != len(cipherHdrSerial) {
// 		return nil, errors.Errorf("Failed to write the entire ciphertext header")
// 	}

// 	// Create a block list writer using the streaming crypto so the blocks will be
// 	// encrypted.
// 	blockWriter, err := ssblocks.NewBlockListWriterV1(streamCrypto, uint32(DTI_BLOCK_SIZE_MAX),
// 		uint64(initOffset+int64(len(plainHdrSerial)+len(cipherHdrSerial))))
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	index := &SearchIdxV1{DtiVersionS{DtiVer: DTI_V1},
// 		docID, docVer, key, plainHdrBody.Nonce, uint64(initOffset),
// 		plainHdrBody, cipherHdrBody, blockWriter, nil, nil, source}
// 	return index, nil
// }

// // OpenSearchIdxV1 opens a document offset index reader V1
// func OpenSearchIdxV1(key *sscrypto.StrongSaltKey, plainHdrBody *DtiPlainHdrBodyV1,
// 	store interface{}, initOffset uint64, endOffset uint64, plainHdrOffset uint64) (*SearchIdxV1, error) {

// 	if key.Type != sscrypto.Type_XChaCha20 {
// 		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
// 			key.Type.Name, sscrypto.Type_XChaCha20.Name)
// 	}

// 	_, ok := store.(io.Reader)
// 	if !ok {
// 		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
// 	}

// 	// Initialize the streaming crypto to decrypt ciphertext header and the blocks after that
// 	streamCrypto, err := crypto.CreateStreamCrypto(key, plainHdrBody.Nonce, store, int64(plainHdrOffset))
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	// Read the ciphertext header from storage
// 	cipherHdr, parsed, err := ssheaders.DeserializeCipherHdrStream(streamCrypto)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	cipherHdrBodyData, err := cipherHdr.GetBody()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	cipherHdrBody := &DtiCipherHdrBodyV1{}
// 	cipherHdrBody, err = cipherHdrBody.deserialize(cipherHdrBodyData)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	// Create a block list reader using the streaming crypto so the blocks will be
// 	// decrypted.
// 	reader, err := ssblocks.NewBlockListReader(streamCrypto,
// 		plainHdrOffset+uint64(parsed), endOffset)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	blockReader, ok := reader.(ssblocks.BlockListReaderV1)
// 	if !ok {
// 		return nil, errors.Errorf("Block list reader is not BlockListReaderV1")
// 	}

// 	index := &SearchIdxV1{DtiVersionS{DtiVer: plainHdrBody.GetDtiVersion()},
// 		plainHdrBody.DocID, plainHdrBody.DocVer, key, plainHdrBody.Nonce,
// 		uint64(initOffset), plainHdrBody, cipherHdrBody, nil, blockReader, nil,
// 		nil}
// 	return index, nil
// }

// // WriteNextBlock writes the next search index block, and returns the written block.
// // Returns io.EOF when last block is written.
// func (idx *SearchIdxV1) WriteNextBlock() (*SearchIdxBlkV1, error) {
// 	if idx.Writer == nil {
// 		return nil, errors.Errorf("The search index is not open for writing")
// 	}

// 	if idx.Block == nil {
// 		idx.Block = CreateDockTermIdxBlkV1("", uint64(idx.Writer.GetMaxDataSize()))
// 	}

// 	err := idx.Source.Reset()
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}

// 	for err == nil {
// 		var term string
// 		term, _, err = idx.Source.GetNextTerm()
// 		if len(term) > 0 {
// 			idx.Block.AddTerm(term)
// 		}
// 	}

// 	if err == io.EOF {
// 		block := idx.Block

// 		serial, err := idx.Block.Serialize()
// 		if err != nil {
// 			return nil, errors.New(err)
// 		}

// 		err = idx.flush(serial)
// 		if err != nil {
// 			return nil, errors.New(err)
// 		}

// 		if block.IsFull() {
// 			return block, nil
// 		}

// 		return block, io.EOF
// 	}

// 	return nil, errors.New(err)
// }

// // ReadNextBlock returns the next search index block
// func (idx *SearchIdxV1) ReadNextBlock() (*SearchIdxBlkV1, error) {
// 	if idx.Reader == nil {
// 		return nil, errors.Errorf("The search index is not open for reading")
// 	}

// 	b, err := idx.Reader.ReadNextBlock()
// 	if err != nil && err != io.EOF {
// 		return nil, errors.New(err)
// 	}

// 	if b != nil && len(b.GetData()) > 0 {
// 		block := CreateDockTermIdxBlkV1("", 0)
// 		blk, derr := block.Deserialize(b.GetData())
// 		if derr != nil {
// 			return nil, errors.New(derr)
// 		}

// 		return blk, err
// 	}

// 	return nil, err
// }

// // FindTerm attempts to find the specified term in the term index
// func (idx *SearchIdxV1) FindTerm(term string) (bool, error) {
// 	if idx.Reader == nil {
// 		return false, errors.Errorf("The search index is not open for reading")
// 	}

// 	blk, err := idx.Reader.SearchBinary(term, SearchComparatorV1)
// 	if err != nil {
// 		return false, errors.New(err)
// 	}

// 	return (blk != nil), nil
// }

// // Close writes any residual block data to output stream
// func (idx *SearchIdxV1) Close() error {
// 	if idx.Block != nil && idx.Block.totalTerms > 0 {
// 		serial, err := idx.Block.Serialize()
// 		if err != nil {
// 			return errors.New(err)
// 		}
// 		return idx.flush(serial)
// 	}
// 	return nil
// }

// func (idx *SearchIdxV1) flush(data []byte) error {
// 	if idx.Writer == nil {
// 		return errors.Errorf("The search index is not open for writing")
// 	}

// 	_, err := idx.Writer.WriteBlockData(data)
// 	if err != nil {
// 		return errors.New(err)
// 	}

// 	idx.Block = CreateSearchIdxBlkV1(idx.Block.highTerm, uint64(idx.Writer.GetMaxDataSize()))
// 	return nil
// }
