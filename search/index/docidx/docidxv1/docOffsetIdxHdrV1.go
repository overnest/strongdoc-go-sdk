package docidxv1

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
)

//////////////////////////////////////////////////////////////////
//
//               Document Offset Index Headers V1
//
//////////////////////////////////////////////////////////////////

// DoiPlainHdrBodyV1 is the plaintext header for document offset index.
type DoiPlainHdrBodyV1 struct {
	common.DoiVersionS
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

// DoiPlainHdrBodyV1Deserialize deserializes DoiPlainHdrBodyV1
func DoiPlainHdrBodyV1Deserialize(data []byte) (*DoiPlainHdrBodyV1, error) {
	plainHdrBody := &DoiPlainHdrBodyV1{}
	return plainHdrBody.deserialize(data)
}

// DoiCipherHdrBodyV1 is the ciphertext header for document offset index.
type DoiCipherHdrBodyV1 struct {
	common.BlockVersionS
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
