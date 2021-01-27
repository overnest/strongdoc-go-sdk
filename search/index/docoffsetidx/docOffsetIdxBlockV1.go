package docoffsetidx

import (
	"encoding/json"
	"fmt"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//                Document Offset Index Block
//
//////////////////////////////////////////////////////////////////

// DocOffsetIdxBlkV1 is the Document Offset Index Block V1
type DocOffsetIdxBlkV1 struct {
	TermLoc           map[string][]uint64
	predictedJSONSize uint64 `json:"-"`
}

var baseBlockJSONSize uint64

func init() {
	base, _ := (&DocOffsetIdxBlkV1{TermLoc: make(map[string][]uint64)}).Serialize()
	baseBlockJSONSize = uint64(len(base))
}

// AddTermOffset adds a term + offset pair to the block
func (blk *DocOffsetIdxBlkV1) AddTermOffset(term string, offset uint64) {
	_, ok := blk.TermLoc[term]
	if !ok {
		// Added "<term>":[<offset>],
		// 2 double quotes, 1 colon, 1 left bracket, 1 right bracket
		blk.predictedJSONSize += uint64(len(term) + 2 + 1 + 2 + len(fmt.Sprintf("%v", offset)))
		if len(blk.TermLoc) > 0 { // There would be a comma at the end
			blk.predictedJSONSize++
		}
		blk.TermLoc[term] = []uint64{offset}
	} else {
		// Added "xxxxx":[yyy,zzz,<offset>], 1 comma + size of offset
		blk.predictedJSONSize += uint64(1 + len(fmt.Sprintf("%v", offset)))
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
