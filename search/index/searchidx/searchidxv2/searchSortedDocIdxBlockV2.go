package searchidxv2

import (
	"fmt"
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongsalt-common-go/blocks"
	"github.com/pkg/errors"
	"log"
	"strings"
)

//////////////////////////////////////////////////////////////////
//
//              Search Sorted Document Index Block
//
//////////////////////////////////////////////////////////////////

// SearchSortDocIdxBlkV2 is the Search Sorted Document Index Block V2
type SearchSortDocIdxBlkV2 struct {
	Term              string
	DocIDVers         []*DocIDVerV2
	docIDVerMap       map[string]uint64 `json:"-"`
	docIDTree         *rbt.Tree         `json:"-"`
	totalDocIDs       uint64            `json:"-"`
	lowDocID          string            `json:"-"`
	highDocID         string            `json:"-"`
	prevHighDocID     string            `json:"-"`
	isFull            bool              `json:"-"`
	predictedJSONSize uint64            `json:"-"`
	maxDataSize       uint64            `json:"-"`
}

// DocIDVerV2 stores the DocID and DocVer
type DocIDVerV2 struct {
	DocID  string
	DocVer uint64
}

func (idver *DocIDVerV2) String() string {
	return fmt.Sprintf("%v:%v", idver.DocID, idver.DocVer)
}

var baseSsdiBlockJSONSize uint64
var baseSsdiDocIDVerJSONSize uint64

var initEmptySortDocIdxBlkV1 = func() interface{} {
	return CreateSearchSortDocIdxBlkV2("", "", 0)
}

func init() {
	blk, _ := initEmptySortDocIdxBlkV1().(*SearchSortDocIdxBlkV2)
	blk = blk.formatToBlockData()
	predictSize, err := blocks.GetPredictedJSONSize(blk)
	if err != nil {
		log.Fatal(err)
	}
	baseSsdiBlockJSONSize = uint64(predictSize)

	docIDVer := &DocIDVerV2{"", 0}
	if size, err := blocks.GetPredictedJSONSize(docIDVer); err == nil {
		baseSsdiDocIDVerJSONSize = uint64(size) - 1 // Remove the 0 version
	}
}

// DocIDVerComparatorV1 is a comparator function definition.
// Returns:
//   < 0      , if value < block
//   1        , if value is in block
//   0        , if value not in block
//   > 1      , if value > block
type TermDoc struct {
	Term  string
	DocID string
}

func DocIDVerComparatorV2(value interface{}, blockData interface{}) (int, error) {
	termDoc, _ := value.(TermDoc)
	docID := termDoc.DocID
	term := termDoc.Term
	blk, ok := blockData.(*SearchSortDocIdxBlkV2)
	if !ok {
		return 0, errors.Errorf("Cannot convert to SearchSortDocIdxBlkV2")
	}
	blk, err := blk.formatFromBlockData()
	if err != nil {
		return 0, err
	}

	if compareTerm := strings.Compare(term, blk.Term); compareTerm != 0 {
		return compareTerm, nil
	}

	// same term

	if _, exist := blk.docIDVerMap[docID]; exist {
		return 1, nil
	}

	if strings.Compare(docID, blk.lowDocID) < 0 {
		return -1, nil
	}

	if strings.Compare(docID, blk.highDocID) > 0 {
		return 2, nil
	}

	return 0, nil
}

// CreateSearchSortDocIdxBlkV2 creates a new Search Index Block V2
func CreateSearchSortDocIdxBlkV2(term, prevHighDocID string, maxDataSize uint64) *SearchSortDocIdxBlkV2 {
	return &SearchSortDocIdxBlkV2{
		Term:              term,
		DocIDVers:         make([]*DocIDVerV2, 0, 100),
		docIDVerMap:       make(map[string]uint64),
		docIDTree:         rbt.NewWithStringComparator(),
		totalDocIDs:       0,
		lowDocID:          "",
		highDocID:         "",
		prevHighDocID:     prevHighDocID,
		isFull:            false,
		predictedJSONSize: baseSsdiBlockJSONSize + uint64(len(term)),
		maxDataSize:       maxDataSize,
	}
}

// AddDocVer adds a new document ID and version
func (blk *SearchSortDocIdxBlkV2) AddDocVer(docID string, docVer uint64) {
	// The docID is already covered in the previous block
	if strings.Compare(docID, blk.prevHighDocID) <= 0 {
		return
	}

	newSize := blk.newSize(docID, docVer)
	// Yes we are storing more than the max data size. We'll remove the
	// extra data during serialization time
	if newSize > uint64(blk.maxDataSize+
		(blk.maxDataSize/uint64(100)*uint64(common.SSDI_BLOCK_MARGIN_PERCENT))) {
		blk.isFull = true
	}

	// We still have room in the block
	if !blk.isFull {
		blk.addDocVer(docID, docVer)
	} else {
		// The current docID comes before the high docID and doesn't already exist,
		// need to make room
		if strings.Compare(docID, blk.highDocID) < 0 {
			if _, exist := blk.docIDVerMap[docID]; !exist {
				blk.removeHighTerm()
				blk.addDocVer(docID, docVer)
			}
		} //else {
		// The current term is either:
		//   1. comes after the high docID
		//   2. equal the high docID
		// In either case, discard
		//}
	}
}

func (blk *SearchSortDocIdxBlkV2) newSize(docID string, docVer uint64) uint64 {
	// Added "DocIDVers":[{"<docID>",<docVer>}]
	//   1. New baseSsdiDocIDVerJSONSize
	//   2. New docID
	//   3. New docVer
	//   4. No comma
	newLen := blk.predictedJSONSize + blk.entrySize(docID, docVer)

	// Added "DocIDVers":[{"aaa",1},{"bbb",1},{"<docID>",<docVer>}]
	// 1 extra comma
	if len(blk.docIDVerMap) > 0 {
		newLen++
	}

	return newLen
}

func (blk *SearchSortDocIdxBlkV2) entrySize(docID string, docVer uint64) uint64 {
	return (baseSsdiDocIDVerJSONSize + uint64(len(docID)+len(fmt.Sprintf("%v", docVer))))
}

func (blk *SearchSortDocIdxBlkV2) addDocVer(docID string, docVer uint64) {
	// The DocID already exists. Skip
	if _, exist := blk.docIDVerMap[docID]; exist {
		return
	}

	newSize := blk.newSize(docID, docVer)
	blk.docIDVerMap[docID] = docVer
	blk.docIDTree.Put(docID, docVer)

	if blk.lowDocID == "" && blk.highDocID == "" {
		blk.lowDocID = docID
		blk.highDocID = docID
	} else if strings.Compare(docID, blk.lowDocID) < 0 {
		blk.lowDocID = docID
	} else if strings.Compare(docID, blk.highDocID) > 0 {
		blk.highDocID = docID
	}

	blk.predictedJSONSize = newSize
	blk.totalDocIDs++
}

func (blk *SearchSortDocIdxBlkV2) removeHighTerm() {
	removeSize := blk.entrySize(blk.highDocID, blk.docIDVerMap[blk.highDocID])

	if len(blk.docIDVerMap) > 1 {
		// There is more than 1 entry left. Will be deleting entry + 1 comma
		removeSize++
	}

	blk.predictedJSONSize -= removeSize
	blk.totalDocIDs--

	delete(blk.docIDVerMap, blk.highDocID)
	blk.docIDTree.Remove(blk.highDocID)
	max := blk.docIDTree.Right()
	if max == nil {
		blk.highDocID = ""
		blk.lowDocID = ""
	} else {
		blk.highDocID = max.Key.(string)
	}
}

func (blk *SearchSortDocIdxBlkV2) formatToBlockData() *SearchSortDocIdxBlkV2 {
	for blk.predictedJSONSize > blk.maxDataSize {
		blk.removeHighTerm()
		blk.isFull = true
	}

	docIDs := blk.docIDTree.Keys()
	blk.DocIDVers = make([]*DocIDVerV2, len(docIDs))
	for i, id := range docIDs {
		docID := id.(string)
		blk.DocIDVers[i] = &DocIDVerV2{docID, blk.docIDVerMap[docID]}
	}
	return blk
}

func (blk *SearchSortDocIdxBlkV2) formatFromBlockData() (*SearchSortDocIdxBlkV2, error) {
	if blk.docIDVerMap == nil {
		blk.docIDVerMap = make(map[string]uint64)
	}
	if blk.docIDTree == nil {
		blk.docIDTree = rbt.NewWithStringComparator()
	}

	blk.totalDocIDs = uint64(len(blk.DocIDVers))
	if blk.totalDocIDs > 0 {
		blk.lowDocID = blk.DocIDVers[0].DocID
		blk.highDocID = blk.DocIDVers[blk.totalDocIDs-1].DocID
	}

	for _, docIDVer := range blk.DocIDVers {
		blk.docIDVerMap[docIDVer.DocID] = docIDVer.DocVer
	}

	for _, docIDVer := range blk.DocIDVers {
		blk.docIDTree.Put(docIDVer.DocID, docIDVer.DocVer)
	}

	return blk, nil
}

// IsFull shows whether the block is full
func (blk *SearchSortDocIdxBlkV2) IsFull() bool {
	return blk.isFull
}

// GetLowDocID returns the lowest docID in the sorted list
func (blk *SearchSortDocIdxBlkV2) GetLowDocID() string {
	return blk.lowDocID
}

// GetHighDocID returns the highest docID in the sorted list
func (blk *SearchSortDocIdxBlkV2) GetHighDocID() string {
	return blk.highDocID
}
