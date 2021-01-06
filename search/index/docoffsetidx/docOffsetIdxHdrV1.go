package docoffsetidx

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//               Document Offset Index Headers V1
//
//////////////////////////////////////////////////////////////////

// DoiPlainHdrBodyV1 is the plaintext header for document offset index.
type DoiPlainHdrBodyV1 struct {
	DoiVersionS
	KeyType string
	Nonce   []byte
	DocID   string
	DocVer  uint64
}

func (hdr *DoiPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DoiPlainHdrBodyV1) deserialize(data []byte) (*DoiPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// DoiCipherHdrBodyV1 is the ciphertext header for document offset index.
type DoiCipherHdrBodyV1 struct {
	BlockVersion
}

func (hdr *DoiCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DoiCipherHdrBodyV1) deserialize(data []byte) (*DoiCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}
