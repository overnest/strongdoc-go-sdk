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
//                    Search Term Batch V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchV2 struct {
	Owner           common.SearchIdxOwner
	SourceList      []SearchTermIdxSourceV2
	TermToWriter    map[string]*SearchTermIdxWriterV2 // hashedTerm -> SearchTermIdxWriterV1
	sourceToWriters map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2
	termList        []string
}

//////////////////////////////////////////////////////////////////
//
//                   Search Term Batch Manager V2
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

func (mgr *SearchTermBatchMgrV2) GetNextTermBatch(sdc client.StrongDocClient, batchSize int) (*SearchTermBatchV2, error) {

	// pop out batch elements
	var sources []*SearchTermBatchElement
	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {

		}
		source := s.(*SearchTermBatchElement)
		sources = append(sources, source)
	}

	return CreateSearchTermBatchV2(sdc, mgr.Owner, sources)
}

func CreateSearchTermBatchV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	sources []*SearchTermBatchElement) (*SearchTermBatchV2, error) {
	batch := &SearchTermBatchV2{
		Owner:           owner,
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermIdxWriterV2),
		sourceToWriters: make(map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2),
		termList:        nil,
	}

	var allTerms []string
	for _, source := range sources {
		// open term index writer
		for _, term := range source.TermSources {
			allTerms = append(allTerms, term.Term)

		}
	}

	return batch, nil
}

//////////////////////////////////////////////////////////////////
//
//                 Search Term Batch Writer V1
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV2 struct {
	Blocks []*SearchTermIdxBlkV2
	Error  error
}

type SearchTermIdxWriterV2 struct {
	Term            string
	Owner           common.SearchIdxOwner
	AddSources      []SearchTermIdxSourceV2
	DelSources      []SearchTermIdxSourceV2
	TermKey         *sscrypto.StrongSaltKey
	IndexKey        *sscrypto.StrongSaltKey
	updateID        string
	delDocMap       map[string]bool   // DocID -> boolean
	highDocVerMap   map[string]uint64 // DocID -> DocVer
	newSti          *SearchTermIdxV2
	oldSti          common.SearchTermIdx
	newStiBlk       *SearchTermIdxBlkV2
	oldStiBlk       *SearchTermIdxBlkV2
	oldStiBlkDocIDs []string
}

func CreateSearchTermIdxWriterV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string,
	addSource, delSource []SearchTermIdxSourceV2,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermIdxWriterV2, error) {

	var err error

	if addSource == nil {
		addSource = make([]SearchTermIdxSourceV2, 0, 10)
	}
	if delSource == nil {
		delSource = make([]SearchTermIdxSourceV2, 0, 10)
	}

	stiw := &SearchTermIdxWriterV2{
		Owner:           owner,
		Term:            term,
		AddSources:      addSource,
		DelSources:      delSource,
		TermKey:         termKey,
		IndexKey:        indexKey,
		updateID:        "",
		delDocMap:       make(map[string]bool),
		highDocVerMap:   make(map[string]uint64),
		newSti:          nil,
		oldSti:          nil,
		newStiBlk:       nil,
		oldStiBlk:       nil,
		oldStiBlkDocIDs: make([]string, 0),
	}

	// Record the DocIDs to be deleted
	for _, ds := range delSource {
		stiw.delDocMap[ds.GetDocID()] = true
	}

	if delDocs != nil {
		for _, docID := range delDocs.DelDocs {
			stiw.delDocMap[docID] = true
		}
	}

	// Find the highest version of each DocID. This is so we can remove
	// any DocID with lower version
	for _, stis := range addSource {
		if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}
	for _, stis := range delSource {
		if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}

	// Open previous STI if there is one
	updateID, _ := GetLatestUpdateIDV2(sdc, owner, term, termKey)
	if updateID != "" {
		stiw.oldSti, err = OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		err = stiw.updateHighDocVersion(sdc, owner, term, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	stiw.newSti, err = CreateSearchTermIdxV1(sdc, owner, term, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}
