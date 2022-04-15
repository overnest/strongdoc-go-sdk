package searchidxv2

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
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
		common.STI_TERM_BATCH_SIZE_V2, common.STI_TERM_BUCKET_COUNT, searchIdx.delDocs)
	if err != nil {
		return nil, err
	}

	return searchIdx, nil
}

// process one batch
func (idx *SearchIdxV2) ProcessBatchTerms(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	emptyResult := make(map[string]error) // term -> error

	termBatch, err := idx.batchMgr.GetNextTermBatch(sdc)
	if err != nil {
		return emptyResult, err
	}

	if termBatch.IsEmpty() {
		return emptyResult, io.EOF
	}

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

// GetLatestUpdateIDV2 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIDV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, bucketID string) (string, error) {
	ids, err := GetUpdateIdsV2(sdc, owner, bucketID)
	if err != nil {
		return "", err
	}

	if len(ids) == 0 {
		return "", nil
	}

	return ids[0], nil
}

// GetUpdateIdsV2 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, bucketID string) ([]string, error) {
	return common.GetUpdateIDs(sdc, owner, bucketID)
}
