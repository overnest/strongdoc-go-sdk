package searchidxv2

import (
	"fmt"
	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"hash/fnv"
	"strings"
)

const modVal = 10

func hashStringToInt(s string, modVal int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	number := h.Sum32()
	return int(number) % modVal
}
func hashTerm(term string) string {
	return fmt.Sprintf("%v", hashStringToInt(term, modVal))
}

//////////////////////////////////////////////////////////////////
//
//                   Search HashedTerm Batch Manager V2
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV2 struct {
	Owner            common.SearchIdxOwner
	SearchIdxSources []SearchTermIdxSourceV2
	TermKey          *sscrypto.StrongSaltKey
	IndexKey         *sscrypto.StrongSaltKey
	delDocs          *DeletedDocsV1
	termHeap         *binaryheap.Heap
}

type SearchTermBatchSources struct {
	Term       string // plain term
	AddSources []SearchTermIdxSourceV2
	DelSources []SearchTermIdxSourceV2
	delDocs    *DeletedDocsV1
}

type SearchTermBatchElement struct {
	HashedTerm  string
	TermKey     *sscrypto.StrongSaltKey
	IndexKey    *sscrypto.StrongSaltKey
	TermSources []*SearchTermBatchSources
}

var batchSourceComparator func(a, b interface{}) int = func(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	wa := a.(*SearchTermBatchElement)
	wb := b.(*SearchTermBatchElement)
	return strings.Compare(wa.HashedTerm, wb.HashedTerm) // not necessary to sort
}

func CreateSearchTermBatchMgrV2(owner common.SearchIdxOwner, sources []SearchTermIdxSourceV2,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermBatchMgrV2, error) {

	mgr := &SearchTermBatchMgrV2{
		Owner:    owner,
		TermKey:  termKey,
		IndexKey: indexKey,
		delDocs:  delDocs,
		termHeap: binaryheap.NewWith(batchSourceComparator)}

	termSourcesMap := make(map[string]*SearchTermBatchSources) // term -> termSources
	for _, source := range sources {
		for _, term := range source.GetAddTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
			}
			stbs.AddSources = append(stbs.AddSources, source)
		}

		for _, term := range source.GetDelTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
			}
			stbs.DelSources = append(stbs.DelSources, source)
		}
	}

	batchElementMap := make(map[string]*SearchTermBatchElement) // hashedTerm -> batchElement
	for term, termSources := range termSourcesMap {
		hashedTerm := hashTerm(term)
		batchElement := batchElementMap[hashedTerm]
		if batchElement == nil {
			batchElement = &SearchTermBatchElement{
				HashedTerm: hashedTerm,
				IndexKey:   indexKey,
				TermKey:    termKey,
			}
			batchElementMap[hashedTerm] = batchElement
			mgr.termHeap.Push(batchElement)
		}
		batchElement.TermSources = append(batchElement.TermSources, termSources)
	}
	return mgr, nil
}

//////////////////////////////////////////////////////////////////
//
//                    Search HashedTerm Batch V2
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchV2 struct {
	Owner      common.SearchIdxOwner   // owner
	SourceList []SearchTermIdxSourceV2 // all docs
	termList   []string                // all terms

	TermToWriter    map[string]*SearchTermIdxWriterV2 // hashedTerm -> term index writer
	sourceToWriters map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2
}

func (mgr *SearchTermBatchMgrV2) GetNextTermBatch(sdc client.StrongDocClient, batchSize int) (*SearchTermBatchV2, error) {

	// pop out batch elements
	var sources []*SearchTermBatchElement
	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}
		source := s.(*SearchTermBatchElement)
		sources = append(sources, source)
	}

	return CreateSearchTermBatchV2(sdc, mgr.Owner, sources)
}

func CreateSearchTermBatchV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	batchElements []*SearchTermBatchElement) (*SearchTermBatchV2, error) {
	batch := &SearchTermBatchV2{
		Owner:           owner,
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermIdxWriterV2),
		sourceToWriters: make(map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2),
		termList:        nil,
	}

	var allTerms []string
	for _, batchElement := range batchElements {
		// collect all terms
		for _, term := range batchElement.TermSources {
			allTerms = append(allTerms, term.Term)
		}
	}
	batch.termList = allTerms

	//for _, batchElement := range batchElements {
	//
	//}

	return batch, nil
}

//////////////////////////////////////////////////////////////////
//
//                 Search HashedTerm Batch Writer V2
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV2 struct {
	Blocks []*SearchTermIdxBlkV2
	Error  error
}

type SearchTermIdxWriterV2 struct {
	HashedTerm string
	Owner      common.SearchIdxOwner
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	newSti     *SearchTermIdxV2     // writer, all terms use this writer
	oldSti     common.SearchTermIdx // reader
	newStiBlk  *SearchTermIdxBlkV2
	oldStiBlk  *SearchTermIdxBlkV2
}

func CreateSearchTermIdxWriterV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, hashedTerm string,
	termKey, indexKey *sscrypto.StrongSaltKey) (*SearchTermIdxWriterV2, error) {

	var err error

	stiw := &SearchTermIdxWriterV2{
		Owner:      owner,
		HashedTerm: hashedTerm,
		TermKey:    termKey,
		IndexKey:   indexKey,
		newSti:     nil,
		oldSti:     nil,
		newStiBlk:  nil,
		oldStiBlk:  nil,
	}

	// Open previous STI if there is one
	updateID, _ := GetLatestUpdateIDV2(sdc, owner, hashedTerm, termKey)
	if updateID != "" {
		//stiw.oldSti, err = OpenSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		//err = stiw.updateHighDocVersion(sdc, owner, hashedTerm, termKey, indexKey, updateID)
		//if err != nil {
		//	return nil, err
		//}
	}

	//stiw.newSti, err = CreateSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	//stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}
