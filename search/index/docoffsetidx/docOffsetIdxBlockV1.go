package docoffsetidx

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//                Document Offset Index Block
//
//////////////////////////////////////////////////////////////////

// DocOffsetIdxBlkV1 is the Document Offset Index Block V1
type DocOffsetIdxBlkV1 struct {
	TermLoc map[string][]uint64
}

// AddTermOffset adds a term + offset pair to the block
func (blk *DocOffsetIdxBlkV1) AddTermOffset(term string, offset uint64) {
	_, ok := blk.TermLoc[term]
	if !ok {
		blk.TermLoc[term] = []uint64{offset}
	} else {
		blk.TermLoc[term] = append(blk.TermLoc[term], offset)
	}
}

// Serialize the block
func (blk *DocOffsetIdxBlkV1) Serialize() ([]byte, error) {
	b, err := json.Marshal(blk)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize the block
func (blk *DocOffsetIdxBlkV1) Deserialize(data []byte) (*DocOffsetIdxBlkV1, error) {
	err := json.Unmarshal(data, blk)
	if err != nil {
		return nil, errors.New(err)
	}
	return blk, nil
}
