package doctermidx

import (
	"encoding/json"
	"io"

	"github.com/go-errors/errors"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const (
	DTI_V1  = uint32(1)
	DTI_VER = DTI_V1

	DTI_BLOCK_V1  = uint32(1)
	DTI_BLOCK_VER = DTI_BLOCK_V1

	DTI_BLOCK_SIZE_MAX       = uint64(1024 * 1024) // 10MB
	DTI_BLOCK_MARGIN_PERCENT = uint64(10)          // 10% margin
)

//////////////////////////////////////////////////////////////////
//
//                Document Term Index Version
//
//////////////////////////////////////////////////////////////////

// DtiVersion store document term index version
type DtiVersion interface {
	GetDtiVersion() uint32
}

// DtiVersionS is structure used to store document term index version
type DtiVersionS struct {
	DtiVer uint32
}

// GetDtiVersion retrieves the document term index version number
func (h *DtiVersionS) GetDtiVersion() uint32 {
	return h.DtiVer
}

// Deserialize deserializes the data into version number object
func (h *DtiVersionS) Deserialize(data []byte) (*DtiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDtiVersion deserializes the data into version number object
func DeserializeDtiVersion(data []byte) (*DtiVersionS, error) {
	h := &DtiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//              Document Term Index Block Version
//
//////////////////////////////////////////////////////////////////

// BlockVersion is structure used to store document term index block version
type BlockVersion struct {
	BlockVer uint32
}

// GetBlockVersion retrieves the document term index version block number
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
//                   Document Term Index
//
//////////////////////////////////////////////////////////////////

// CreateDocTermIdx creates a new document term index for writing
func CreateDocTermIdx(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	source DocTermSourceV1, store interface{}, initOffset int64) (*DocTermIdxV1, error) {

	return CreateDocTermIdxV1(docID, docVer, key, source, store, initOffset)
}

// OpenDocTermIdx opens a document term index for reading
func OpenDocTermIdx(key *sscrypto.StrongSaltKey, store interface{}, initOffset uint64, endOffset uint64) (DtiVersion, error) {
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

	version, err := DeserializeDtiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDtiVersion() {
	case DTI_V1:
		// Parse plaintext header body
		plainHdrBody := &DtiPlainHdrBodyV1{}
		plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return OpenDocTermIdxV1(key, plainHdrBody, reader, initOffset, endOffset, initOffset+uint64(parsed))
	default:
		return nil, errors.Errorf("Document term index version %v is not supported",
			version.GetDtiVersion())
	}
}
