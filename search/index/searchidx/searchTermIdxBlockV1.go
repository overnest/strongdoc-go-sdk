package searchidx

import (
	"encoding/json"
	"fmt"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//                  Search Term Index Block
//
//////////////////////////////////////////////////////////////////

// SearchTermIdxBlkV1 is the Search Index Block V1
type SearchTermIdxBlkV1 struct {
	DocVerOffset      map[string]*VersionOffsetV1 // DocID->VersionOffset
	predictedJSONSize uint64                      `json:"-"`
	maxDataSize       uint64                      `json:"-"`
}

// VersionOffsetV1 stores the document version and associated offsets
type VersionOffsetV1 struct {
	Version uint64
	Offsets []uint64
}

var baseStiBlockJSONSize uint64
var baseStiVersionOffsetJSONSize uint64

func init() {
	base, _ := CreateSearchTermIdxBlkV1(0).Serialize()
	baseStiBlockJSONSize = uint64(len(base))

	verOffset := &VersionOffsetV1{0, []uint64{}}
	if b, err := json.Marshal(verOffset); err == nil {
		baseStiVersionOffsetJSONSize = uint64(len(b)) - 1 // Remove the 0 version
	}
}

// CreateSearchTermIdxBlkV1 creates a new Search Index Block V1
func CreateSearchTermIdxBlkV1(maxDataSize uint64) *SearchTermIdxBlkV1 {
	return &SearchTermIdxBlkV1{
		DocVerOffset:      make(map[string]*VersionOffsetV1),
		predictedJSONSize: baseStiBlockJSONSize,
		maxDataSize:       maxDataSize,
	}
}

// AddDocOffset adds a new offset to the block
func (blk *SearchTermIdxBlkV1) AddDocOffset(docID string, docVer uint64, offset uint64) error {
	return blk.AddDocOffsets(docID, docVer, []uint64{offset})
}

// AddDocOffsets adds new offsets to the block
func (blk *SearchTermIdxBlkV1) AddDocOffsets(docID string, docVer uint64, offsets []uint64) error {
	// Delete existing DocID entry if it's older than the incoming one
	versionOffset := blk.DocVerOffset[docID]
	if versionOffset != nil && versionOffset.Version < docVer {
		blk.DocRemove(docID)
	}

	mapSize := len(blk.DocVerOffset)
	newSize := blk.predictedJSONSize
	versionOffset = blk.DocVerOffset[docID]
	if versionOffset == nil {
		// Need to add new docID:
		// {"DocVerOffset":{
		//    "<docID>":{"Version":<docVer>,"Offsets":[<o1>,<o2>,...]},
		// }}
		newSize += (uint64(len(docID)+3) +
			uint64(baseStiVersionOffsetJSONSize) +
			uint64(len(fmt.Sprintf("%v", docVer))))
		if mapSize > 0 {
			newSize++ // Need to add a comma at the end if there are already keys in the map
		}

		versionOffset = &VersionOffsetV1{
			Version: docVer,
			Offsets: make([]uint64, 0, len(offsets)),
		}
	}

	for _, offset := range offsets {
		newSize += uint64(len(fmt.Sprintf("%v", offset)) + 1) // +1 is for comma
	}

	if len(versionOffset.Offsets) == 0 {
		// If the list was empty before we add new values, we should have 1 less comma
		newSize--
	}

	if newSize > blk.maxDataSize {
		return errors.Errorf("The new size %v is bigger than the maximum allowed size %v",
			newSize, blk.maxDataSize)
	}

	versionOffset.Offsets = append(versionOffset.Offsets, offsets...)
	blk.DocVerOffset[docID] = versionOffset
	blk.predictedJSONSize = newSize
	return nil
}

// DocRemove removes a document from the Search Index Block
func (blk *SearchTermIdxBlkV1) DocRemove(docID string) {
	verOffset, ok := blk.DocVerOffset[docID]
	delete(blk.DocVerOffset, docID)

	if ok {
		// Need to remove docID:
		// {"DocVerOffset":{
		//    "<docID>":{"Version":<docVer>,"Offsets":[<o1>,<o2>,...]}, <--- THIS
		// }}

		// 2 double quotes, 1 colon, 1 comma
		// (comma only present if there is more than 1 DocID entry)
		blk.predictedJSONSize -= uint64(len(docID) + 3 + 1)
		if verOffset == nil {
			blk.predictedJSONSize -= uint64(len("null"))
		} else {
			b, _ := verOffset.Serialize()
			if b != nil {
				blk.predictedJSONSize -= uint64(len(b))
			}
		}

		// If there are no entries left, it means we removed a comma that wasn't there.
		// Add one byte back
		if len(blk.DocVerOffset) == 0 {
			blk.predictedJSONSize++
		}
	}
}

// IsEmpty shows whether the block is empty
func (blk *SearchTermIdxBlkV1) IsEmpty() bool {
	return (len(blk.DocVerOffset) == 0)
}

// Serialize the block
func (blk *SearchTermIdxBlkV1) Serialize() ([]byte, error) {
	b, err := json.Marshal(blk)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize the block
func (blk *SearchTermIdxBlkV1) Deserialize(data []byte) (*SearchTermIdxBlkV1, error) {
	err := json.Unmarshal(data, blk)
	if err != nil {
		return nil, errors.New(err)
	}

	blk.predictedJSONSize = uint64(len(data))
	return blk, nil
}

// Serialize the version offset struct
func (vo *VersionOffsetV1) Serialize() ([]byte, error) {
	b, err := json.Marshal(vo)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize the version offset struct
func (vo *VersionOffsetV1) Deserialize(data []byte) (*VersionOffsetV1, error) {
	err := json.Unmarshal(data, vo)
	if err != nil {
		return nil, errors.New(err)
	}

	return vo, nil
}
