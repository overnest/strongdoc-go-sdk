package searchidx

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//               Search Term Index Headers V1
//
//////////////////////////////////////////////////////////////////

// StiPlainHdrBodyV1 is the plaintext header for search term index.
type StiPlainHdrBodyV1 struct {
	StiVersionS
	KeyType  string
	Nonce    []byte
	TermHmac string
	UpdateID string
}

func (hdr *StiPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *StiPlainHdrBodyV1) deserialize(data []byte) (*StiPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// StiCipherHdrBodyV1 is the ciphertext header for search term index.
type StiCipherHdrBodyV1 struct {
	BlockVersion
}

func (hdr *StiCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *StiCipherHdrBodyV1) deserialize(data []byte) (*StiCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}
