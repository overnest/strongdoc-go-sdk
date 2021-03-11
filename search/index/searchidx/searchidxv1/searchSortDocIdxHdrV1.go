package searchidxv1

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
)

//////////////////////////////////////////////////////////////////
//
//           Search Sorted Document Index Headers V1
//
//////////////////////////////////////////////////////////////////

// SsdiPlainHdrBodyV1 is the plaintext header for search sorted document index.
type SsdiPlainHdrBodyV1 struct {
	common.SsdiVersionS
	KeyType  string
	Nonce    []byte
	TermHmac string
	UpdateID string
}

func (hdr *SsdiPlainHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *SsdiPlainHdrBodyV1) deserialize(data []byte) (*SsdiPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

// SsdiCipherHdrBodyV1 is the ciphertext header for search sorted document index.
type SsdiCipherHdrBodyV1 struct {
	common.BlockVersionS
}

func (hdr *SsdiCipherHdrBodyV1) serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *SsdiCipherHdrBodyV1) deserialize(data []byte) (*SsdiCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}
