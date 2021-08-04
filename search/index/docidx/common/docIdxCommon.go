package common

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

const (
	DOI_V1  = uint32(1)
	DOI_VER = DOI_V1

	DOI_BLOCK_V1  = uint32(1)
	DOI_BLOCK_VER = DOI_BLOCK_V1

	DOI_BLOCK_SIZE_MAX = int(1024 * 1024 * 5) // 5MB
	// DOI_BLOCK_MARGIN   = int(200)             // 200 byte margin

	DTI_V1  = uint32(1)
	DTI_VER = DTI_V1

	DTI_BLOCK_V1  = uint32(1)
	DTI_BLOCK_VER = DTI_BLOCK_V1

	DTI_BLOCK_SIZE_MAX       = uint64(1024 * 1024) // 1MB
	DTI_BLOCK_MARGIN_PERCENT = uint64(10)          // 10% margin
)

//////////////////////////////////////////////////////////////////
//
//                Document Index Block Version
//
//////////////////////////////////////////////////////////////////

// BlockVersion is the interface used to store a block of any version
type BlockVersion interface {
	GetBlockVersion() uint32
}

// BlockVersionS is structure used to store document offset index block version
type BlockVersionS struct {
	BlockVer uint32
}

// GetBlockVersion retrieves the document offset index version block number
func (h *BlockVersionS) GetBlockVersion() uint32 {
	return h.BlockVer
}

// Deserialize deserializes the data into version number object
func (h *BlockVersionS) Deserialize(data []byte) (*BlockVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeBlockVersion deserializes the data into version number object
func DeserializeBlockVersion(data []byte) (*BlockVersionS, error) {
	h := &BlockVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                Document Offset Index Version
//
//////////////////////////////////////////////////////////////////

// DocOffsetIdx store document offset index version
type DocOffsetIdx interface {
	GetDoiVersion() uint32
	GetDocID() string
	GetDocVersion() uint64
	Close() error
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
//                Document HashedTerm Index Version
//
//////////////////////////////////////////////////////////////////

// DocTermIdx store document term index version
type DocTermIdx interface {
	GetDtiVersion() uint32
	GetDocID() string
	GetDocVersion() uint64
	Close() error
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
