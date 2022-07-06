package documentv1

import (
	"encoding/json"
	"fmt"
	"io"

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
	common.DocFormatVer
	KeyID   string
	KeyType string
	Nonce   []byte
	DocID   string
	DocVer  string
}

func (hdr *DocPlainHdrBodyV1) Serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocPlainHdrBodyV1) Write(writer io.Writer) (n int, err error) {
	serial, err := hdr.Serialize()
	if err != nil {
		return 0, err
	}

	size := common.DocHeaderLen(len(serial))
	n, err = size.Write(writer)
	if err != nil {
		return
	}

	m, err := writer.Write(serial)
	n = n + m
	if err != nil {
		return
	}

	if m != len(serial) {
		return n, fmt.Errorf("could not write entire plain text header")
	}

	return
}

func (hdr *DocPlainHdrBodyV1) Read(reader io.Reader) (err error) {
	hdrLen, err := common.DocHeaderLenRead(reader)
	if err != nil {
		return
	}

	buf := make([]byte, hdrLen)
	n, err := reader.Read(buf)
	if err != nil {
		return
	}

	if common.DocHeaderLen(n) != hdrLen {
		return fmt.Errorf("could not read entire plain text header")
	}

	_, err = hdr.Deserialize(buf)
	return
}

func (hdr *DocPlainHdrBodyV1) Deserialize(data []byte) (*DocPlainHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

func DocPlainHdrBodyV1Deserialize(data []byte) (*DocPlainHdrBodyV1, error) {
	plainHdrBody := &DocPlainHdrBodyV1{}
	return plainHdrBody.Deserialize(data)
}

// DocCipherHdrBodyV1 is the ciphertext header for document
type DocCipherHdrBodyV1 struct {
	common.DocFormatVer
	DocName string
}

func (hdr *DocCipherHdrBodyV1) Serialize() ([]byte, error) {
	b, err := json.Marshal(hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

func (hdr *DocCipherHdrBodyV1) Write(writer io.Writer) (n int, err error) {
	serial, err := hdr.Serialize()
	if err != nil {
		return 0, err
	}

	size := common.DocHeaderLen(len(serial))
	n, err = size.Write(writer)
	if err != nil {
		return
	}

	m, err := writer.Write(serial)
	n = n + m
	if err != nil {
		return
	}

	if m != len(serial) {
		return n, fmt.Errorf("could not write entire cipher text header")
	}

	return
}

func (hdr *DocCipherHdrBodyV1) Read(reader io.Reader) (err error) {
	hdrLen, err := common.DocHeaderLenRead(reader)
	if err != nil {
		return
	}

	buf := make([]byte, hdrLen)
	n, err := reader.Read(buf)
	if err != nil {
		return
	}

	if common.DocHeaderLen(n) != hdrLen {
		return fmt.Errorf("could not read entire cipher text header")
	}

	_, err = hdr.Deserialize(buf)
	return
}

func (hdr *DocCipherHdrBodyV1) Deserialize(data []byte) (*DocCipherHdrBodyV1, error) {
	err := json.Unmarshal(data, hdr)
	if err != nil {
		return nil, errors.New(err)
	}
	return hdr, nil
}

func DocCipherHdrBodyV1Deserialize(data []byte) (*DocCipherHdrBodyV1, error) {
	cipherHdrBody := &DocCipherHdrBodyV1{}
	return cipherHdrBody.Deserialize(data)
}
