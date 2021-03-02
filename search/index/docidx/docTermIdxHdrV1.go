package docidx

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//               Document Term Index Headers V1
//
//////////////////////////////////////////////////////////////////

// DtiPlainHdrBodyV1 is the plaintext header for document term index.
type DtiPlainHdrBodyV1 struct {
	DtiVersionS
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

// DtiCipherHdrBodyV1 is the ciphertext header for document term index.
type DtiCipherHdrBodyV1 struct {
	BlockVersion
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
