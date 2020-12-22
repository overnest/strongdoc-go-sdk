package v1

import (
	"encoding/json"

	"github.com/go-errors/errors"

	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

var a ssblocks.Block
var b ssheaders.PlainHdrV1

// DocOffsetPlainHdrBodyV1 is the plaintext header for document offset index.
type DocOffsetPlainHdrBodyV1 struct {
	Version uint32
	KeyType string
	Nonce   []byte
	DocID   string
	DocVer  uint64
}

func (hdr *DocOffsetPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocOffsetPlainHdrBodyV1) deserialize(data []byte) (*DocOffsetPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// DocOffsetCipherHdrBodyV1 is the ciphertext header for document offset index.
type DocOffsetCipherHdrBodyV1 struct {
	Version uint32
}

func (hdr *DocOffsetCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocOffsetCipherHdrBodyV1) deserialize(data []byte) (*DocOffsetCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// DocOffsetIdxV1 is the Document Offset Index V1
type DocOffsetIdxV1 struct {
	DocID  string
	DocVer uint64
	Key    sscrypto.StrongSaltKey
	Offset uint64
	Writer ssblocks.BlockListWriterV1
	Reader ssblocks.BlockListReaderV1
}

// DocOffsetIdxBlkV1 is the Document Offset Index Block V1
type DocOffsetIdxBlkV1 struct {
	TermLoc map[string][]uint32
}

// NewDocOffsetIdxWriterV1 creates a document offset index writer V1
func NewDocOffsetIdxWriterV1(docID string, docVer uint64, key sscrypto.StrongSaltKey, store interface{}) (*DocOffsetIdxV1, error) {
	blockWriter, err := ssblocks.NewBlockListWriterV1(store, 0, 0) // Offset index is not padded
	if err != nil {
		return nil, errors.New(err)
	}

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	plainHdrBody := &DocOffsetPlainHdrBodyV1{
		Version: INDEX_PLAIN_HDR_BODY_V1,
		KeyType: key.Type.Name,
		DocID:   docID,
		DocVer:  docVer,
	}

	if midStreamKey, ok := key.Key.(sscryptointf.KeyMidstream); ok {
		plainHdrBody.Nonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", key.Type.Name)
	}

	index := &DocOffsetIdxV1{docID, docVer, key, 0, blockWriter, nil}
	return index, nil
}
