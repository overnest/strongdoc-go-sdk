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

// TODO: optimization try TermDocIDVers => map[string]map[string]uint64
// SearchSortDocIdxBlkV2 is the Search Sorted Document Index Block V2
type SearchSortDocIdxBlkV2 struct {
	TermDocIDVers []*DocIDVerV2

	//docIDVerMap map[string]uint64 `json:"-"`

	termToDocIDVer map[string]map[string]uint64 `json:"-"` // term -> (docID -> docVer)

	termToDocIDTree    map[string]*rbt.Tree `json:"-"`
	totalTermDocIdVers uint64               `json:"-"`

	termToLowDocID  map[string]string `json:"-"`
	termToHighDocID map[string]string `json:"-"`

	termTree *rbt.Tree `json:"-"`
	lowTerm  string    `json:"-"`
	highTerm string    `json:"-"`

	prevHighTerm  string `json:"-"`
	prevHighDocID string `json:"-"`

	isFull            bool   `json:"-"`
	predictedJSONSize uint64 `json:"-"`
	maxDataSize       uint64 `json:"-"`
}

// DocIDVerV2 stores the DocID and DocVer
type DocIDVerV2 struct {
	Term   string
	DocID  string
	DocVer uint64
}

func (termidver *DocIDVerV2) String() string {
	return fmt.Sprintf("%v:%v:%v", termidver.Term, termidver.DocID, termidver.DocVer)
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

	docIDVer := &DocIDVerV2{"", "", 0}
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

//TODO: testing
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

	// term < lowTerm
	if strings.Compare(term, blk.lowTerm) < 0 {
		return -1, nil // value < block
	}
	// term > highTerm
	if strings.Compare(term, blk.highTerm) > 0 {
		return 2, nil // value > block
	}

	if _, exist := blk.termToDocIDVer[term]; exist { // lowTerm <= term <= highTerm
		if _, exist := blk.termToDocIDVer[term][docID]; exist {
			return 1, nil // value in block
		}

		// term == lowTerm && docID < lowDocID
		if strings.Compare(docID, blk.termToLowDocID[term]) < 0 &&
			strings.Compare(term, blk.lowTerm) == 0 {
			return -1, nil // value < block
		}

		// term == highTerm && docID > highDocID
		if strings.Compare(docID, blk.termToHighDocID[term]) > 0 &&
			strings.Compare(term, blk.highTerm) == 0 {
			return 2, nil // value > block
		}

	}
	return 0, nil
}

// CreateSearchSortDocIdxBlkV2 creates a new Search Index Block V2
func CreateSearchSortDocIdxBlkV2(prevHighTerm, prevHighDocID string, maxDataSize uint64) *SearchSortDocIdxBlkV2 {
	return &SearchSortDocIdxBlkV2{
		TermDocIDVers:      make([]*DocIDVerV2, 0, 100),
		termToDocIDVer:     make(map[string]map[string]uint64),
		termToDocIDTree:    make(map[string]*rbt.Tree),
		totalTermDocIdVers: 0,

		termToLowDocID:  make(map[string]string),
		termToHighDocID: make(map[string]string),

		termTree:          rbt.NewWithStringComparator(),
		prevHighTerm:      prevHighTerm,
		prevHighDocID:     prevHighDocID,
		isFull:            false,
		predictedJSONSize: baseSsdiBlockJSONSize,
		maxDataSize:       maxDataSize,
	}
}

// AddTermDocVer adds (term, documentID, version)
func (blk *SearchSortDocIdxBlkV2) AddTermDocVer(term, docID string, docVer uint64) {
	// The term or docID is already covered in the previous block
	compareTerm := strings.Compare(term, blk.prevHighTerm)
	if compareTerm < 0 || (compareTerm == 0 && strings.Compare(docID, blk.prevHighDocID) <= 0) {
		return
	}

	// The <term, docID> is unprocessed
	// term > prevHighTerm || (term == prevHighTerm && docID > prevHighDocID)

	newSize := blk.newSize(term, docID, docVer)
	// Yes we are storing more than the max data size. We'll remove the
	// extra data during serialization time
	if newSize > uint64(blk.maxDataSize+
		(blk.maxDataSize/uint64(100)*uint64(common.SSDI_BLOCK_MARGIN_PERCENT))) {
		// predicted new size
		blk.isFull = true
	}

	// We still have room in the block
	if !blk.isFull {
		blk.addDocVer(term, docID, docVer)
	} else {
		// The current docID comes before the high docID and doesn't already exist,
		// need to make room

		// If currentTerm < highTerm
		// If currentTerm == highTerm && docID < highDocID
		if strings.Compare(term, blk.highTerm) < 0 ||
			(strings.Compare(term, blk.highTerm) == 0 && strings.Compare(docID, blk.termToHighDocID[term]) < 0) {
			blk.removeLastEntry()
			blk.addDocVer(term, docID, docVer)
		}

		//else {
		// The current term is either:
		//   1. comes after the high docID
		//   2. equal high term but the docID comes after the high docID
		// In either case, discard
		//}
	}
}

func (blk *SearchSortDocIdxBlkV2) newSize(term, docID string, docVer uint64) uint64 {
	// Added "TermDocIDVers":[{"<docID>",<docVer>}]
	//   1. New baseSsdiDocIDVerJSONSize
	//	 2. New Term
	//   3. New docID
	//   4. New docVer
	//   5. No comma
	newLen := blk.predictedJSONSize + blk.entrySize(term, docID, docVer)

	// Added "TermDocIDVers":[{"term1", "aaa",1},{"term2","bbb",1},{"term","<docID>",<docVer>}]
	// 1 extra comma
	if len(blk.termToDocIDVer) > 0 {
		newLen++
	}

	return newLen
}

func (blk *SearchSortDocIdxBlkV2) entrySize(term, docID string, docVer uint64) uint64 {
	return (baseSsdiDocIDVerJSONSize + uint64(len(term)) + uint64(len(docID)+len(fmt.Sprintf("%v", docVer))))
}

// add <term, docID, docVer> entry
func (blk *SearchSortDocIdxBlkV2) addDocVer(term, docID string, docVer uint64) {
	// The DocID already exists. Skip
	if _, exist := blk.termToDocIDVer[term]; exist {
		if _, exist := blk.termToDocIDVer[docID]; exist { //TODO check twice
			return
		}
	}

	newSize := blk.newSize(term, docID, docVer)

	if _, exists := blk.termToDocIDVer[term]; !exists {
		// new term
		blk.termToDocIDVer[term] = make(map[string]uint64)
		blk.termTree.Put(term, "")
		blk.termToLowDocID[term] = docID
		blk.termToHighDocID[term] = docID
		blk.termToDocIDTree[term] = rbt.NewWithStringComparator()

	}

	blk.termToDocIDVer[term][docID] = docVer
	blk.termToDocIDTree[term].Put(docID, docVer)

	if strings.Compare(docID, blk.termToLowDocID[term]) < 0 {
		blk.termToLowDocID[term] = docID
	} else if strings.Compare(docID, blk.termToHighDocID[term]) > 0 {
		blk.termToHighDocID[term] = docID
	}

	blk.predictedJSONSize = newSize
	blk.totalTermDocIdVers++
	blk.lowTerm = blk.termTree.Left().Key.(string)
	blk.highTerm = blk.termTree.Right().Key.(string)

}

func (blk *SearchSortDocIdxBlkV2) removeLastEntry() error {
	highTerm := blk.highTerm
	highDocID := blk.termToHighDocID[highTerm]
	highDocVer := blk.termToDocIDVer[highTerm][highDocID]
	removeSize := blk.entrySize(highTerm, highDocID, highDocVer)

	if len(blk.termToDocIDVer) > 1 {
		// There is more than 1 entry left. Will be deleting entry + 1 comma
		removeSize++
	}

	blk.predictedJSONSize -= removeSize
	blk.totalTermDocIdVers--

	delete(blk.termToDocIDVer[highTerm], highDocID)

	blk.termToDocIDTree[highTerm].Remove(highDocID)

	if len(blk.termToDocIDVer[highTerm]) == 0 {
		// highTerm only includes one entry, which has been removed
		delete(blk.termToDocIDVer, highTerm)
		delete(blk.termToDocIDTree, highTerm)
		blk.termTree.Remove(highTerm)
		delete(blk.termToLowDocID, highTerm)
		delete(blk.termToHighDocID, highTerm)

	} else {
		max := blk.termToDocIDTree[highTerm].Right()
		blk.termToHighDocID[highTerm] = max.Key.(string)
	}

	if len(blk.termTree.Keys()) > 0 {
		blk.lowTerm = blk.termTree.Left().Key.(string)
		blk.highTerm = blk.termTree.Right().Key.(string)
	} else {
		blk.lowTerm = ""
		blk.highTerm = ""
	}

	return nil
}

func (blk *SearchSortDocIdxBlkV2) formatToBlockData() *SearchSortDocIdxBlkV2 {
	for blk.predictedJSONSize > blk.maxDataSize {
		blk.removeLastEntry()
		blk.isFull = true
	}

	var termDocIDVers []*DocIDVerV2

	for _, t := range blk.termTree.Keys() {
		term := t.(string)
		docIDs := blk.termToDocIDTree[term].Keys()
		for _, id := range docIDs {
			docID := id.(string)
			docVer := blk.termToDocIDVer[term][docID]
			termDocIDVers = append(termDocIDVers, &DocIDVerV2{Term: term, DocID: docID, DocVer: docVer})
		}

	}
	blk.TermDocIDVers = termDocIDVers
	return blk
}

func (blk *SearchSortDocIdxBlkV2) formatFromBlockData() (*SearchSortDocIdxBlkV2, error) {
	if blk.termToDocIDVer == nil {
		blk.termToDocIDVer = make(map[string]map[string]uint64)
	}
	if blk.termToDocIDTree == nil {
		blk.termToDocIDTree = make(map[string]*rbt.Tree)
	}

	blk.totalTermDocIdVers = uint64(len(blk.TermDocIDVers))
	if blk.totalTermDocIdVers > 0 {
		blk.lowTerm = blk.TermDocIDVers[0].Term
		blk.highTerm = blk.TermDocIDVers[blk.totalTermDocIdVers-1].Term
	}

	for _, termDocIDVer := range blk.TermDocIDVers {
		term := termDocIDVer.Term
		docID := termDocIDVer.DocID
		docVer := termDocIDVer.DocVer

		if _, exists := blk.termToDocIDVer[term]; !exists {
			blk.termToDocIDVer[term] = make(map[string]uint64)
			blk.termToDocIDTree[term] = rbt.NewWithStringComparator()
		}
		blk.termToDocIDVer[term][docID] = docVer
		blk.termToDocIDTree[term].Put(docID, docVer)
	}

	//TODO necessary
	for term, docIDTree := range blk.termToDocIDTree {
		blk.termToLowDocID[term] = docIDTree.Left().Key.(string)
		blk.termToHighDocID[term] = docIDTree.Right().Key.(string)

		blk.termTree.Put(term, "")
	}

	return blk, nil
}

// IsFull shows whether the block is full
func (blk *SearchSortDocIdxBlkV2) IsFull() bool {
	return blk.isFull
}

// GetLowDocID returns the lowest docID in the block
func (blk *SearchSortDocIdxBlkV2) GetLowDocID(term string) string {
	if _, ok := blk.termToLowDocID[term]; !ok {
		return ""
	}
	return blk.termToLowDocID[term]
}

// GetHighDocID returns the highest docID in the block
func (blk *SearchSortDocIdxBlkV2) GetHighDocID(term string) string {
	if _, ok := blk.termToHighDocID[term]; !ok {
		return ""
	}
	return blk.termToHighDocID[term]
}
