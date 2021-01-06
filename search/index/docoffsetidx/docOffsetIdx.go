package docoffsetidx

import (
	"encoding/json"
	"io"

	"github.com/go-errors/errors"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const (
	DOI_V1  = uint32(1)
	DOI_VER = DOI_V1

	DOI_BLOCK_V1  = uint32(1)
	DOI_BLOCK_VER = DOI_BLOCK_V1

	DOI_BLOCK_SIZE_MAX = int(1024 * 1024 * 5) // 5MB
	DOI_BLOCK_MARGIN   = int(200)             // 200 byte margin
)

//////////////////////////////////////////////////////////////////
//
//                Document Offset Index Version
//
//////////////////////////////////////////////////////////////////

// DoiVersion store document offset index version
type DoiVersion interface {
	GetDoiVersion() uint32
}

// DoiVersionS is structure used to store document offset index version
type DoiVersionS struct {
	DoiVer uint32
}

// GetDoiVersion retrieves the document offset index version number
func (h *DoiVersionS) GetDoiVersion() uint32 {
	return h.DoiVer
}

// Deserialize deserializes the data into version number object
func (h *DoiVersionS) Deserialize(data []byte) (*DoiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDoiVersion deserializes the data into version number object
func DeserializeDoiVersion(data []byte) (*DoiVersionS, error) {
	h := &DoiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//              Document Offset Index Block Version
//
//////////////////////////////////////////////////////////////////

// BlockVersion is structure used to store document offset index block version
type BlockVersion struct {
	BlockVer uint32
}

// GetBlockVersion retrieves the document offset index version block number
func (h *BlockVersion) GetBlockVersion() uint32 {
	return h.BlockVer
}

// Deserialize deserializes the data into version number object
func (h *BlockVersion) Deserialize(data []byte) (*BlockVersion, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeBlockVersion deserializes the data into version number object
func DeserializeBlockVersion(data []byte) (*BlockVersion, error) {
	h := &BlockVersion{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                   Document Offset Index
//
//////////////////////////////////////////////////////////////////

// CreateDocOffsetIdx creates a new document offset index for writing
func CreateDocOffsetIdx(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	store interface{}, initOffset int64) (*DocOffsetIdxV1, error) {

	return CreateDocOffsetIdxV1(docID, docVer, key, store, initOffset)
}

// OpenDocOffsetIdx opens a document offset index for reading
func OpenDocOffsetIdx(key *sscrypto.StrongSaltKey, store interface{}, initOffset int64) (DoiVersion, error) {
	reader, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	plainHdr, parsed, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, errors.New(err)
	}

	version, err := DeserializeDoiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDoiVersion() {
	case DOI_V1:
		// Parse plaintext header body
		plainHdrBody := &DoiPlainHdrBodyV1{}
		plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return OpenDocOffsetIdxV1(key, plainHdrBody, reader, initOffset+int64(parsed))
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			version.GetDoiVersion())
	}
}
