package docidxv1

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
)

//////////////////////////////////////////////////////////////////
//
//               Document Term Index Headers V1
//
//////////////////////////////////////////////////////////////////

// DtiPlainHdrBodyV1 is the plaintext header for document term index.
type DtiPlainHdrBodyV1 struct {
	common.DtiVersionS
	KeyType string
	Nonce   []byte
	DocID   string
	DocVer  uint64
}

func (hdr *DtiPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DtiPlainHdrBodyV1) deserialize(data []byte) (*DtiPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

func DtiPlainHdrBodyV1Deserialize(data []byte) (*DtiPlainHdrBodyV1, error) {
	plainHdrBody := &DtiPlainHdrBodyV1{}
	return plainHdrBody.deserialize(data)
}

// DtiCipherHdrBodyV1 is the ciphertext header for document term index.
type DtiCipherHdrBodyV1 struct {
	common.BlockVersionS
}

func (hdr *DtiCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DtiCipherHdrBodyV1) deserialize(data []byte) (*DtiCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}
