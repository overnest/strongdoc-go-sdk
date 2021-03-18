package common

import (
	"encoding/json"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io"
	"path"

	"github.com/go-errors/errors"
)

const (
	// SI_V1  = uint32(1)
	// SI_VER = SI_V1

	SI_OWNER_ORG = SearchIdxOwnerType("ORG")
	SI_OWNER_USR = SearchIdxOwnerType("USR")

	STI_BLOCK_V1  = uint32(1)
	STI_BLOCK_VER = STI_BLOCK_V1

	STI_BLOCK_SIZE_MAX = uint64(1024 * 1024 * 5) // 5MB
	// STI_BLOCK_MARGIN_PERCENT = uint64(10)              // 10% margin

	STI_V1              = uint32(1)
	STI_VER             = STI_V1
	STI_TERM_BATCH_SIZE = 1000 // Process terms in batches of 1000

	SSDI_BLOCK_SIZE_MAX       = uint64(1024 * 1) //1024 * 5) // 5MB
	SSDI_BLOCK_MARGIN_PERCENT = uint64(10)       // 10% margin

	SSDI_V1  = uint32(1)
	SSDI_VER = SSDI_V1
)

// GetSearchIdxPathPrefix gets the search index path prefix
func GetSearchIdxPathPrefix() string {
	return path.Clean("/tmp/search")
}

//////////////////////////////////////////////////////////////////
//
//                   Search Index Owner
//
//////////////////////////////////////////////////////////////////

// common.SearchIdxOwnerType is the owner type of the search index
type SearchIdxOwnerType string

// common.SearchIdxOwner is the search index owner interface
type SearchIdxOwner interface {
	GetOwnerType() utils.OwnerType
	GetOwnerID() string
	fmt.Stringer
}

type searchIdxOwner struct {
	ownerType utils.OwnerType
	ownerID   string
}

func (sio *searchIdxOwner) GetOwnerType() utils.OwnerType {
	return sio.ownerType
}

func (sio *searchIdxOwner) GetOwnerID() string {
	return sio.ownerID
}

func (sio *searchIdxOwner) String() string {
	return fmt.Sprintf("%v_%v", sio.GetOwnerType(), sio.GetOwnerID())
}

// common.SearchIdxOwner creates a new searchh index owner
func CreateSearchIdxOwner(ownerType utils.OwnerType, ownerID string) SearchIdxOwner {
	return &searchIdxOwner{ownerType, ownerID}
}

//////////////////////////////////////////////////////////////////
//
//                     Search Term Index
//
//////////////////////////////////////////////////////////////////

// SearchTermIdx store search term index version
type SearchTermIdx interface {
	GetStiVersion() uint32
	io.Closer
}

// StiVersionS is structure used to store search term index version
type StiVersionS struct {
	StiVer uint32
}

// GetStiVersion retrieves the search term index version number
func (h *StiVersionS) GetStiVersion() uint32 {
	return h.StiVer
}

// Deserialize deserializes the data into version number object
func (h *StiVersionS) Deserialize(data []byte) (*StiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeStiVersion deserializes the data into version number object
func DeserializeStiVersion(data []byte) (*StiVersionS, error) {
	h := &StiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                 Search Sorted Document Index
//
//////////////////////////////////////////////////////////////////

// SearchSortDocIdx store search sorted docuemtn index version
type SearchSortDocIdx interface {
	GetSsdiVersion() uint32
	io.Closer
}

// SsdiVersionS is structure used to store search term index version
type SsdiVersionS struct {
	SsdiVer uint32
}

// GetSsdiVersion retrieves the search term index version number
func (h *SsdiVersionS) GetSsdiVersion() uint32 {
	return h.SsdiVer
}

// Deserialize deserializes the data into version number object
func (h *SsdiVersionS) Deserialize(data []byte) (*SsdiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeSsdiVersion deserializes the data into version number object
func DeserializeSsdiVersion(data []byte) (*SsdiVersionS, error) {
	h := &SsdiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                          Search Block
//
//////////////////////////////////////////////////////////////////

// BlockVersion is the interface used to store a block of any version
type BlockVersion interface {
	GetBlockVersion() uint32
}

// BlockVersionS is structure used to store search index block version
type BlockVersionS struct {
	BlockVer uint32
}

// GetBlockVersion retrieves the search index version block number
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
