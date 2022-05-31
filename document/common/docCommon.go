package common

import (
	"encoding/json"
	"io"

	"github.com/go-errors/errors"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const (
	DOC_FORMAT_V1  = uint32(1)
	DOC_FORMAT_VER = DOC_FORMAT_V1
)

//////////////////////////////////////////////////////////////////
//
//                    Document Format Version
//
//////////////////////////////////////////////////////////////////
type DocFormatVersionS struct {
	DocFormatVer uint32
}

// GetDocFormatVersion retrieves the document format version
func (h *DocFormatVersionS) GetDocFormatVersion() uint32 {
	return h.DocFormatVer
}

// Deserialize deserializes the data into version number object
func (h *DocFormatVersionS) Deserialize(data []byte) (*DocFormatVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDocFormatVersion deserializes the data into version number object
func DeserializeDocFormatVersion(data []byte) (*DocFormatVersionS, error) {
	h := &DocFormatVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                    Document Interface
//
//////////////////////////////////////////////////////////////////
type Document interface {
	DocID() string
	DocVer() string
	DocName() string
	DocPath() string
	DocKeyID() string
	DocKey() *sscrypto.StrongSaltKey
	Encrypt() ([]byte, error)
	EncrytStream() io.Reader
	Decrypt() ([]byte, error)
	DecryptStream() io.Reader
}
