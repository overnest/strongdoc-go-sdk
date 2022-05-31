package documentv1

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/document/common"
)

//////////////////////////////////////////////////////////////////
//
//                    Document Headers V1
//
//////////////////////////////////////////////////////////////////

// DocPlainHdrBodyV1 is the plaintext header for document.
type DocPlainHdrBodyV1 struct {
	common.DocFormatVersionS
	KeyID   string
	KeyType string
	Nonce   []byte
	DocID   string
	DocVer  uint64
}

func (hdr *DocPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocPlainHdrBodyV1) deserialize(data []byte) (*DocPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

func DocPlainHdrBodyV1Deserialize(data []byte) (*DocPlainHdrBodyV1, error) {
	plainHdrBody := &DocPlainHdrBodyV1{}
	return plainHdrBody.deserialize(data)
}

// DocCipherHdrBodyV1 is the ciphertext header for document
type DocCipherHdrBodyV1 struct {
	common.DocFormatVersionS
	DocName string
	DocPath string
}

func (hdr *DocCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocCipherHdrBodyV1) deserialize(data []byte) (*DocCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

func DocCipherHdrBodyV1Deserialize(data []byte) (*DocCipherHdrBodyV1, error) {
	cipherHdrBody := &DocCipherHdrBodyV1{}
	return cipherHdrBody.deserialize(data)
}
