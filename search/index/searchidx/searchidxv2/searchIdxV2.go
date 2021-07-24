package searchidxv2

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
	"io"
)

//////////////////////////////////////////////////////////////////
//
//                          Search Index
//
//////////////////////////////////////////////////////////////////

// SearchIdxV2 is the Search Index V2
type SearchIdxV2 struct {
	SearchSources map[string]*SearchTermIdxV2
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	owner         common.SearchIdxOwner
	batchMgr      *SearchTermBatchMgrV2
	delDocs       *DeletedDocsV2
}

type DeletedDocsV2 struct {
	DelDocs   []string        // List of DocIDs
	delDocMap map[string]bool // Map of DocID to boolean
}

// CreateSearchIdxWriterV1 creates a search index writer V2
func CreateSearchIdxWriterV2(owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	sources []SearchTermIdxSourceV2) (*SearchIdxV2, error) {
	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if _, ok := indexKey.Key.(sscryptointf.KeyMidstream); !ok {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	searchIdx := &SearchIdxV2{
		SearchSources: make(map[string]*SearchTermIdxV2),
		TermKey:       termKey,
		IndexKey:      indexKey,
		owner:         owner,
		batchMgr:      nil,
		delDocs:       &DeletedDocsV2{make([]string, 0), make(map[string]bool)},
	}

	var err error
	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV2(owner, sources, termKey, indexKey,
		searchIdx.delDocs)
	if err != nil {
		return nil, err
	}

	return searchIdx, nil
}

var batchNum int = 0
var numOfTerms int = 0

// process one batch
func (idx *SearchIdxV2) ProcessBatchTerms(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {

	if event == nil {
		event = utils.NewTimeEvent("ProcessBatchTerms", "result1.txt")
		defer event.Output()
	}
	emptyResult := make(map[string]error) // term -> error

	termBatch, err := idx.batchMgr.GetNextTermBatch(sdc, common.STI_TERM_BATCH_SIZE_V2)

	if err != nil {
		return emptyResult, err
	}

	if termBatch.IsEmpty() {
		return emptyResult, io.EOF
	}
	batchNum++
	numOfTerms += len(termBatch.termList)
	fmt.Println("batch", batchNum, "process", len(termBatch.termList), "terms,", numOfTerms, "in total")

	//fmt.Println(termBatch.termList)
	e := utils.AddSubEvent(event, "ProcessTermBatch")
	termErrs, err := termBatch.ProcessTermBatch(sdc, e)
	utils.EndEvent(e)
	return termErrs, err
}

func (idx *SearchIdxV2) ProcessAllTerms(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	finalResult := make(map[string]error)

	var err error = nil
	for err == nil {
		var batchResult map[string]error

		batchResult, err = idx.ProcessBatchTerms(sdc, event)
		if err != nil && err != io.EOF {
			return finalResult, err
		}

		for term, err := range batchResult {
			finalResult[term] = err
		}
	}

	return finalResult, nil
}

// GetUpdateIdsV2 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := common.CreateTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV2(sdc, owner, termHmac)
}

// GetLatestUpdateIDV2 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIDV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) (string, error) {
	ids, err := GetUpdateIdsV2(sdc, owner, term, termKey)
	if err != nil {
		return "", err
	}

	if len(ids) == 0 {
		return "", nil
	}

	return ids[0], nil
}

// GetUpdateIdsHmacV2 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsHmacV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac string) ([]string, error) {
	return common.GetUpdateIDs(sdc, owner, termHmac)
}
