package searchidxv2

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
)

//////////////////////////////////////////////////////////////////
//
//                Search HashedTerm Index Headers V2
//
//////////////////////////////////////////////////////////////////

// StiPlainHdrBodyV2 is the plaintext header for search term index.
type StiPlainHdrBodyV2 struct {
	common.StiVersionS
	KeyType  string
	Nonce    []byte
	TermID   string
	UpdateID string
}

func (hdr *StiPlainHdrBodyV2) Serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *StiPlainHdrBodyV2) Deserialize(data []byte) (*StiPlainHdrBodyV2, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// StiCipherHdrBodyV2 is the ciphertext header for search term index.
type StiCipherHdrBodyV2 struct {
	common.BlockVersionS
}

func (hdr *StiCipherHdrBodyV2) Serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *StiCipherHdrBodyV2) Deserialize(data []byte) (*StiCipherHdrBodyV2, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}
