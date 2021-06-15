package docidxv1

import (
	"fmt"
	"github.com/overnest/strongsalt-common-go/blocks"
	"log"
	"strings"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
)

//////////////////////////////////////////////////////////////////
//
//                Document Term Index Block
//
//////////////////////////////////////////////////////////////////

// DocTermIdxBlkV1 is the Document Term Index Block V1
type DocTermIdxBlkV1 struct {
	Terms             []string
	termMap           map[string]bool `json:"-"`
	termTree          *rbt.Tree       `json:"-"`
	totalTerms        uint64          `json:"-"`
	lowTerm           string          `json:"-"`
	highTerm          string          `json:"-"`
	prevHighTerm      string          `json:"-"`
	predictedJSONSize uint64          `json:"-"`
	isFull            bool            `json:"-"`
	maxDataSize       uint64          `json:"-"`
}

var baseDtiBlockJSONSize uint64

var initEmptyDocTermIdxBlkV1 = func() interface{} {
	return CreateDocTermIdxBlkV1("", 0)
}

func init() {
	blk, _ := initEmptyDocTermIdxBlkV1().(*DocTermIdxBlkV1)
	blk = blk.formatToBlockData()
	predictSize, err := blocks.GetPredictedJSONSize(blk)
	if err != nil {
		log.Fatal(err)
	}
	baseDtiBlockJSONSize = uint64(predictSize)
}

// DocTermComparatorV1 is a comparator function definition.
// Returns:
//   < 0      , if value < block
//   1        , if value is in block
//   0        , if value not in block
//   > 1      , if value > block
func DocTermComparatorV1(value interface{}, blockData interface{}) (int, error) {
	term, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("cannot convert value to string")
	}

	blk, ok := blockData.(*DocTermIdxBlkV1)
	if !ok {
		return 0, fmt.Errorf("cannot convert blockData to DocTermIdxBlkV1")
	}
	blk = blk.formatFromBlockData()

	if blk.termMap[term] {

		return 1, nil
	}

	if strings.Compare(term, blk.lowTerm) < 0 {
		return -1, nil
	}

	if strings.Compare(term, blk.highTerm) > 0 {
		return 2, nil
	}

	return 0, nil
}

func CreateDocTermIdxBlkV1(prevHighTerm string, maxDataSize uint64) *DocTermIdxBlkV1 {
	return &DocTermIdxBlkV1{
		Terms:             []string{},
		termMap:           make(map[string]bool),
		termTree:          rbt.NewWithStringComparator(),
		totalTerms:        0,
		lowTerm:           "",
		highTerm:          "",
		prevHighTerm:      prevHighTerm,
		predictedJSONSize: baseDtiBlockJSONSize,
		isFull:            false,
		maxDataSize:       maxDataSize,
	}
}

// AddTerm adds a term to the block
func (blk *DocTermIdxBlkV1) AddTerm(term string) {
	// The term is already covered in the previous block
	if strings.Compare(term, blk.prevHighTerm) <= 0 {
		return
	}

	newSize := blk.newSize(term)
	// Yes we are storing more than the max data size. We'll remove the
	// extra data during serialization time
	if newSize > uint64(blk.maxDataSize+
		(blk.maxDataSize/uint64(100)*uint64(common.DTI_BLOCK_MARGIN_PERCENT))) {
		blk.isFull = true
	}

	// We still have room in the block
	if !blk.isFull {
		blk.addTerm(term)
	} else {
		// The current term comes before the high term and doesn't already exist,
		// need to make room
		if strings.Compare(term, blk.highTerm) < 0 && !blk.termMap[term] {
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
	// The term already exists. Skip
	if blk.termMap[term] {
		return
	}

	newSize := blk.newSize(term)
	blk.termMap[term] = true
	blk.termTree.Put(term, true)

	if blk.lowTerm == "" && blk.highTerm == "" {
		blk.lowTerm = term
		blk.highTerm = term
	} else if strings.Compare(term, blk.lowTerm) < 0 {
		blk.lowTerm = term
	} else if strings.Compare(term, blk.highTerm) > 0 {
		blk.highTerm = term
	}

	blk.predictedJSONSize = newSize
	blk.totalTerms++
}

func (blk *DocTermIdxBlkV1) removeHighTerm() {
	var removeSize uint64
	if len(blk.termMap) > 1 {
		// There is more than 1 entry left. Will be deleting 2 double quotes + 1 comma
		removeSize = uint64(len(blk.highTerm) + 3)
	} else {
		// There is only 1 entry left, Will be deleting 2 double quotes + 0 comma
		removeSize = uint64(len(blk.highTerm) + 2)
	}

	blk.predictedJSONSize -= removeSize
	blk.totalTerms--

	delete(blk.termMap, blk.highTerm)
	blk.termTree.Remove(blk.highTerm)
	max := blk.termTree.Right()
	if max == nil {
		blk.highTerm = ""
		blk.lowTerm = ""
	} else {
		blk.highTerm = max.Key.(string)
	}
}

func (blk *DocTermIdxBlkV1) formatToBlockData() *DocTermIdxBlkV1 {
	for blk.predictedJSONSize > blk.maxDataSize {
		blk.removeHighTerm()
	}

	terms := blk.termTree.Keys()
	blk.Terms = make([]string, len(terms))
	for i, t := range terms {
		blk.Terms[i] = t.(string)
	}
	return blk
}

func (blk *DocTermIdxBlkV1) formatFromBlockData() *DocTermIdxBlkV1 {
	if blk.termMap == nil {
		blk.termMap = make(map[string]bool)
	}
	if blk.termTree == nil {
		blk.termTree = rbt.NewWithStringComparator()
	}

	blk.totalTerms = uint64(len(blk.Terms))
	if blk.totalTerms > 0 {
		blk.lowTerm = blk.Terms[0]
	}

	for _, term := range blk.Terms {
		blk.highTerm = term
		blk.termMap[term] = true
		blk.termTree.Put(term, true)
	}

	return blk
}

// IsFull shows whether the block is full
func (blk *DocTermIdxBlkV1) IsFull() bool {
	return blk.isFull
}

// GetLowTerm returns the lowest term in the sorted term list
func (blk *DocTermIdxBlkV1) GetLowTerm() string {
	return blk.lowTerm
}

// GetHighTerm returns the highest term in the sorted term list
func (blk *DocTermIdxBlkV1) GetHighTerm() string {
	return blk.highTerm
}

// GetTotalTerms returns the total terms in the sorted term list
func (blk *DocTermIdxBlkV1) GetTotalTerms() uint64 {
	return blk.totalTerms
}
