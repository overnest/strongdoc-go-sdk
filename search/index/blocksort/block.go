package blocksort

import (
	"encoding/json"
	"strings"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/go-errors/errors"
)

const (
	DTI_V1  = uint32(1)
	DTI_VER = DTI_V1

	DTI_BLOCK_V1  = uint32(1)
	DTI_BLOCK_VER = DTI_BLOCK_V1

	DTI_BLOCK_SIZE_MAX       = int(1024 * 1024 * 10) // 10MB
	DTI_BLOCK_MARGIN_PERCENT = int(10)               // 10% margin
)

var debug bool = false

//////////////////////////////////////////////////////////////////
//
//                Document HashedTerm Index Block
//
//////////////////////////////////////////////////////////////////

// DocTermIdxBlkV1 is the Document HashedTerm Index Block V1
type DocTermIdxBlkV1 struct {
	Terms             []string
	termMap           map[string]uint32 `json:"-"`
	termTree          *rbt.Tree         `json:"-"`
	totalTerms        uint64            `json:"-"`
	lowTerm           string            `json:"-"`
	highTerm          string            `json:"-"`
	prevHighTerm      string            `json:"-"`
	prevHighTermCount uint32            `json:"-"`
	predictedJSONSize uint64            `json:"-"`
	isFull            bool              `json:"-"`
}

var baseBlockJSONSize uint64

func init() {
	base, _ := CreateDockTermIdxBlkV1("", 0).Serialize()
	baseBlockJSONSize = uint64(len(base))
}

func CreateDockTermIdxBlkV1(prevHighTerm string, prevHighTermCount uint32) *DocTermIdxBlkV1 {
	return &DocTermIdxBlkV1{
		Terms:             []string{},
		termMap:           make(map[string]uint32),
		termTree:          rbt.NewWithStringComparator(),
		totalTerms:        0,
		lowTerm:           "",
		highTerm:          "",
		prevHighTerm:      prevHighTerm,
		prevHighTermCount: prevHighTermCount,
		predictedJSONSize: baseBlockJSONSize,
		isFull:            false,
	}
}

// AddTerm adds a term to the block
func (blk *DocTermIdxBlkV1) AddTerm(term string) {
	compPrevHigh := strings.Compare(term, blk.prevHighTerm)

	if compPrevHigh < 0 {
		return
	}

	// The term is already covered in the previous block
	if compPrevHigh == 0 && blk.prevHighTermCount > 0 {
		blk.prevHighTermCount--
		return
	}

	newSize := blk.newSize(term)
	if newSize > uint64(DTI_BLOCK_SIZE_MAX+(DTI_BLOCK_SIZE_MAX/100*DTI_BLOCK_MARGIN_PERCENT)) {
		blk.isFull = true
	}

	if !blk.isFull {
		// We still have room in the block
		blk.addTerm(term)
	} else {
		// No more room in the block
		if strings.Compare(term, blk.highTerm) < 0 {
			// The current term comes before the high term, need to make room
			blk.removeHighTerm()
			blk.addTerm(term)
		} else {
			// The current term is either:
			//   1. comes after the high term
			//   2. equal the high term
			// In either case, discard
		}
	}
}

func (blk *DocTermIdxBlkV1) newSize(term string) uint64 {
	if len(blk.termMap) > 0 {
		// Added "Terms":["aaa","bbb","<term>"]
		// 2 double quotes, 1 comma
		return blk.predictedJSONSize + uint64(len(term)+3)
	}

	// Added "Terms":["<term>"]
	// 2 double quotes, 0 comma
	return blk.predictedJSONSize + uint64(len(term)+2)
}

func (blk *DocTermIdxBlkV1) addTerm(term string) {
	newSize := blk.newSize(term)
	blk.termMap[term] = blk.termMap[term] + 1
	blk.termTree.Put(term, true)

	if blk.lowTerm == "" && blk.highTerm == "" {
		blk.lowTerm = term
		blk.highTerm = term
	} else if strings.Compare(term, blk.lowTerm) < 0 {
		blk.lowTerm = term
	} else if strings.Compare(term, blk.highTerm) > 0 {
		blk.highTerm = term
	}

	// PSL debug
	// if debug {
	// 	if strings.Compare(term, "has") <= 0 {
	// 		fmt.Println("added", term)
	// 	}
	// }

	blk.predictedJSONSize = newSize
	blk.totalTerms++
}

func (blk *DocTermIdxBlkV1) removeHighTerm() {
	// sizeWithNewTerm := blk.newSize(term)

	// // Start removing the high terms until it can fit the new term into the target size
	// for sizeWithNewTerm > targetSize {

	var removeSize uint64
	if len(blk.termMap) > 1 || blk.termMap[blk.lowTerm] > 1 {
		// There is more than 1 entry left. Will be deleting 2 double quotes + 1 comma
		removeSize = uint64(len(blk.highTerm) + 3)
	} else {
		// There is only 1 entry left, Will be deleting 2 double quotes + 0 comma
		removeSize = uint64(len(blk.highTerm) + 2)
	}

	// sizeWithNewTerm -= removeSize
	blk.predictedJSONSize -= removeSize

	blk.termMap[blk.highTerm] = blk.termMap[blk.highTerm] - 1
	blk.totalTerms--
	if blk.termMap[blk.highTerm] == 0 {
		delete(blk.termMap, blk.highTerm)
		blk.termTree.Remove(blk.highTerm)

		// // PSL debug
		// if debug {
		// 	if strings.Compare(blk.highTerm, "has") <= 0 {
		// 		fmt.Println("removed", blk.highTerm, "to add", term)
		// 	}
		// }

		max := blk.termTree.Right()
		if max == nil {
			blk.highTerm = ""
			blk.lowTerm = ""
		} else {
			blk.highTerm = max.Key.(string)
		}
	}
	// }
}

// Serialize the block
func (blk *DocTermIdxBlkV1) Serialize() ([]byte, error) {
	for blk.predictedJSONSize > uint64(DTI_BLOCK_SIZE_MAX) {
		blk.removeHighTerm()
	}

	terms := blk.termTree.Keys()
	blk.Terms = make([]string, blk.totalTerms)
	i := uint32(0)
	for _, t := range terms {
		term := t.(string)
		for j := uint32(0); j < blk.termMap[term]; j++ {
			blk.Terms[i] = term
			i++
		}
	}

	b, err := json.Marshal(blk)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize the block
func (blk *DocTermIdxBlkV1) Deserialize(data []byte) (*DocTermIdxBlkV1, error) {
	err := json.Unmarshal(data, blk)
	if err != nil {
		return nil, errors.New(err)
	}

	blk.termMap = make(map[string]uint32)

	if len(blk.Terms) > 0 {
		blk.lowTerm = blk.Terms[0]
	}

	for _, term := range blk.Terms {
		blk.highTerm = term
		blk.termMap[term] = blk.termMap[term] + 1
		blk.termTree.Put(term, true)
	}

	return blk, nil
}

func (blk *DocTermIdxBlkV1) GetLowTermCount() (uint32, error) {
	if blk.lowTerm == "" || blk.termMap[blk.lowTerm] == 0 {
		return 0, errors.Errorf("The low term %v does not exist", blk.lowTerm)
	}

	return blk.termMap[blk.lowTerm], nil
}

func (blk *DocTermIdxBlkV1) GetHighTermCount() (uint32, error) {
	if blk.highTerm == "" || blk.termMap[blk.highTerm] == 0 {
		return 0, errors.Errorf("The high term %v does not exist", blk.highTerm)
	}

	return blk.termMap[blk.highTerm], nil
}

func (blk *DocTermIdxBlkV1) IsFull() bool {
	return blk.isFull
}
