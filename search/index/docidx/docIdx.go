package docidx

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
