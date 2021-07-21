package searchidxv1

import (
	"fmt"
	"io"
	"time"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

//////////////////////////////////////////////////////////////////
//
//                          Update ID
//
//////////////////////////////////////////////////////////////////

func newUpdateIDV1() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

// GetUpdateIdsV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := common.CreateTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV1(sdc, owner, termHmac)
}

// GetLatestUpdateIDV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIDV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) (string, error) {

	ids, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	if err != nil {
		return "", err
	}

	if len(ids) == 0 {
		return "", nil
	}

	return ids[0], nil
}

// GetUpdateIdsHmacV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsHmacV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac string) ([]string, error) {
	return common.GetUpdateIDs(sdc, owner, termHmac)
}

// GetLatestUpdateIdsHmacV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdsHmacV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac string) (string, error) {
	ids, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	if err != nil {
		return "", err
	}

	if len(ids) == 0 {
		return "", nil
	}

	return ids[0], nil
}

//////////////////////////////////////////////////////////////////
//
//                          Search Index
//
//////////////////////////////////////////////////////////////////

// SearchIdxV1 is the Search Index V1
type SearchIdxV1 struct {
	SearchSourcess map[string]*SearchTermIdxV1
	TermKey        *sscrypto.StrongSaltKey
	IndexKey       *sscrypto.StrongSaltKey
	owner          common.SearchIdxOwner
	batchMgr       *SearchTermBatchMgrV1
	delDocs        *DeletedDocsV1
}

type DeletedDocsV1 struct {
	DelDocs   []string        // List of DocIDs
	delDocMap map[string]bool // Map of DocID to boolean
}

// CreateSearchIdxWriterV1 creates a search index writer V1
func CreateSearchIdxWriterV1(owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	sources []SearchTermIdxSourceV1) (*SearchIdxV1, error) {

	var err error

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if _, ok := indexKey.Key.(sscryptointf.KeyMidstream); !ok {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	searchIdx := &SearchIdxV1{
		SearchSourcess: make(map[string]*SearchTermIdxV1),
		TermKey:        termKey,
		IndexKey:       indexKey,
		owner:          owner,
		batchMgr:       nil,
		delDocs:        &DeletedDocsV1{make([]string, 0), make(map[string]bool)},
	}

	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV1(owner, sources, termKey, indexKey,
		searchIdx.delDocs)
	if err != nil {
		return nil, err
	}

	return searchIdx, nil
}

var numOfTerms int = 0

func (idx *SearchIdxV1) ProcessBatchTerms(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	emptyResult := make(map[string]error)

	termBatch, err := idx.batchMgr.GetNextTermBatch(sdc, common.STI_TERM_BATCH_SIZE)
	if err != nil {
		return emptyResult, err
	}

	if termBatch.IsEmpty() {
		return emptyResult, io.EOF
	}

	numOfTerms += len(termBatch.termList)
	fmt.Println("batch: ", termBatch.termList[0], "->", termBatch.termList[len(termBatch.termList)-1], numOfTerms)

	e := utils.AddSubEvent(event, "ProcessTermBatch")
	termErrs, err := termBatch.ProcessTermBatch(sdc, e)
	utils.EndEvent(e)
	return termErrs, err
}

func (idx *SearchIdxV1) ProcessAllTerms(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
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
