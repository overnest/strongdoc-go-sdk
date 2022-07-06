package common

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/api/apidef"
)

const (
	DOC_META_V1  = int64(1)
	DOC_META_CUR = DOC_META_V1

	DOC_FORMAT_V1  = int64(1)
	DOC_FORMAT_CUR = DOC_FORMAT_V1

	DOC_HEADER_SIZE = 4
)

type DocHeaderLen int32

func (l DocHeaderLen) Serialize() []byte {
	b := make([]byte, DOC_HEADER_SIZE)
	binary.BigEndian.PutUint32(b, uint32(l))
	return b
}

func (l DocHeaderLen) Write(writer io.Writer) (n int, err error) {
	return writer.Write(l.Serialize())
}

func DocHeaderLenDeserialize(b []byte) DocHeaderLen {
	if len(b) < DOC_HEADER_SIZE {
		return 0
	}
	return DocHeaderLen(binary.BigEndian.Uint32(b[:DOC_HEADER_SIZE]))
}

func DocHeaderLenRead(reader io.Reader) (l DocHeaderLen, err error) {
	buf := make([]byte, DOC_HEADER_SIZE)
	n, err := reader.Read(buf)
	if err != nil {
		return
	}

	if n != DOC_HEADER_SIZE {
		return 0, fmt.Errorf("can not read document header length(%v)", n)
	}

	l = DocHeaderLenDeserialize(buf)
	return
}

//////////////////////////////////////////////////////////////////
//
//                    Document Format Version
//
//////////////////////////////////////////////////////////////////
type DocFormatVer struct {
	DocFormatVer int64
}

// GetDocFormatVer retrieves the document format version
func (h *DocFormatVer) GetDocFormatVer() int64 {
	return h.DocFormatVer
}

// Deserialize deserializes the data into version number object
func (h *DocFormatVer) Deserialize(data []byte) (*DocFormatVer, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDocVer deserializes the data into version number object
func DeserializeDocVer(data []byte) (*DocFormatVer, error) {
	h := &DocFormatVer{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                    DocWriter Interface
//
//////////////////////////////////////////////////////////////////
type DocWriter interface {
	DocID() string
	DocVer() string
	DocName() string
	DocKeyID() string
	DocKeyType() string
	DocKey() *apidef.Key
	Write(p []byte) (n int, err error)
	Close() error
}

type DocReader interface {
	DocID() string
	DocVer() string
	DocName() string
	DocKeyID() string
	DocKeyType() string
	DocKey() *apidef.Key
	Read(p []byte) (n int, err error)
	Close() error
}
