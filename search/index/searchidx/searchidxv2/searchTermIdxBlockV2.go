package searchidxv2

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-errors/errors"
	"github.com/overnest/strongsalt-common-go/blocks"
)

//////////////////////////////////////////////////////////////////
//
//                  Search Term Index Block
//
//////////////////////////////////////////////////////////////////

// SearchTermIdxBlkV1 is the Search Index Block V2
type SearchTermIdxBlkV2 struct {
	TermDocVerOffset  map[string](map[string]*VersionOffsetV2) // term -> (docID -> versionOffsets)
	predictedJSONSize uint64                                   `json:"-"`
	maxDataSize       uint64                                   `json:"-"`
}

// VersionOffsetV2 stores the document version and associated offsets
type VersionOffsetV2 struct {
	Version uint64
	Offsets []uint64
}

var baseStiBlockJSONSize uint64
var baseStiTermDocVersionOffsetJSONSize uint64
var baseStiDocVersionOffsetJSONSize uint64
var baseStiVersionOffsetJSONSize uint64

var initSearchTermIdxBlkV2 = func() interface{} {
	return CreateSearchTermIdxBlkV2(0)
}

func init() {
	blk := initSearchTermIdxBlkV2()
	predictSize, err := blocks.GetPredictedJSONSize(blk)
	if err != nil {
		log.Fatal(err)
	}
	baseStiBlockJSONSize = uint64(predictSize)

	verOffset := &VersionOffsetV2{0, []uint64{}}
	if b, err := json.Marshal(verOffset); err == nil {
		baseStiVersionOffsetJSONSize = uint64(len(b)) - 1 // Remove the 0 version
	}

	docVerOffset := make(map[string]*VersionOffsetV2)
	docVerOffset[""] = verOffset
	base2, err := blocks.GetPredictedJSONSize(docVerOffset)
	if err != nil {
		log.Fatal(err)
	}
	baseStiDocVersionOffsetJSONSize = uint64(base2) - 1

	termDocVerOffset := make(map[string]map[string]*VersionOffsetV2)
	termDocVerOffset[""] = docVerOffset
	base1, err := blocks.GetPredictedJSONSize(termDocVerOffset)
	if err != nil {
		log.Fatal(err)
	}
	baseStiTermDocVersionOffsetJSONSize = uint64(base1) - 1

}

// CreateSearchTermIdxBlkV1 creates a new Search Index Block V2
func CreateSearchTermIdxBlkV2(maxDataSize uint64) *SearchTermIdxBlkV2 {
	return &SearchTermIdxBlkV2{
		TermDocVerOffset:  make(map[string]map[string]*VersionOffsetV2),
		predictedJSONSize: baseStiBlockJSONSize,
		maxDataSize:       maxDataSize,
	}
}

// AddDocOffset adds a new offset to the block
func (blk *SearchTermIdxBlkV2) AddDocOffset(term, docID string, docVer uint64, offset uint64) error {
	return blk.AddDocOffsets(term, docID, docVer, []uint64{offset})
}

// AddDocOffsets adds new term offsets to the block
func (blk *SearchTermIdxBlkV2) AddDocOffsets(term, docID string, docVer uint64, offsets []uint64) error {
	if len(offsets) == 0 {
		return fmt.Errorf("invalid offset")
	}
	// Delete existing DocID entry if it's older than the incoming one
	docVerOffset := blk.TermDocVerOffset[term]
	newSize := blk.predictedJSONSize
	if docVerOffset == nil {
		// Need to add new term
		//{"<Term>":
		//	{"<docID>":
		//		{"Version":<docVer>,"Offsets":[<o1>,<o2>,...], }
		//	}
		//}

		newSize += uint64(len(term)) +
			uint64(len(docID)) +
			uint64(len(fmt.Sprintf("%v", docVer))) +
			// 2 double quotes, 1 colon
			baseStiDocVersionOffsetJSONSize + 3

		if len(blk.TermDocVerOffset) != 0 {
			newSize += 1 // 1 comma
		}

		docVerOffset = make(map[string]*VersionOffsetV2)
		docVerOffset[docID] = &VersionOffsetV2{
			Version: docVer,
			Offsets: make([]uint64, 0, len(offsets)),
		}
		blk.TermDocVerOffset[term] = docVerOffset

	} else { // existing term
		verOffset := docVerOffset[docID]
		if verOffset == nil {
			// Need to add new doc
			//	{"<docID1>":
			//		{"Version":<docVer>,"Offsets":[<o1>,<o2>,...], },
			//	"<docID2>":
			//		{"Version":<docVer>,"Offsets":[<o1>,<o2>,...], }
			//	}
			newSize += uint64(len(docID)) +
				baseStiDocVersionOffsetJSONSize +
				uint64(len(fmt.Sprintf("%v", docVer)))
			docVerOffset[docID] = &VersionOffsetV2{
				Version: docVer,
				Offsets: make([]uint64, 0, len(offsets)),
			}

			blk.TermDocVerOffset[term][docID] = &VersionOffsetV2{
				Version: docVer,
				Offsets: make([]uint64, 0, len(offsets)),
			}
		} else if verOffset.Version < docVer {
			// Update doc version
			//{"<Term>":
			//	{"<docID>":
			//		{"Version":<docVer>,"Offsets":[<o1>,<o2>,...], }} --> update version & offsets
			//}
			oldVer := verOffset.Version
			oldOffsets := verOffset.Offsets
			newSize -= uint64(len(fmt.Sprintf("%v", oldVer)))
			for _, oldOffset := range oldOffsets {
				newSize -= uint64(len(fmt.Sprintf("%v", oldOffset)) + 1)
			}
			newSize += 1

			newSize += uint64(len(fmt.Sprintf("%v", docVer)))
			verOffset.Offsets = make([]uint64, 0, len(offsets))
			verOffset.Version = docVer
		}

	}
	// append offsets to existing Term & DocID
	versionOffsets := blk.TermDocVerOffset[term][docID]
	for _, offset := range offsets {
		newSize += uint64(len(fmt.Sprintf("%v", offset)) + 1) // +1 is for comma
	}
	if len(versionOffsets.Offsets) == 0 {
		newSize--
	}
	versionOffsets.Offsets = append(versionOffsets.Offsets, offsets...)

	blk.predictedJSONSize = newSize

	return nil
}

// IsEmpty shows whether the block is empty
func (blk *SearchTermIdxBlkV2) IsEmpty() bool {
	return (len(blk.TermDocVerOffset) == 0)
}

// Serialize the version offset struct
func (vo *VersionOffsetV2) Serialize() ([]byte, error) {
	b, err := json.Marshal(vo)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize the version offset struct
func (vo *VersionOffsetV2) Deserialize(data []byte) (*VersionOffsetV2, error) {
	err := json.Unmarshal(data, vo)
	if err != nil {
		return nil, errors.New(err)
	}

	return vo, nil
}
