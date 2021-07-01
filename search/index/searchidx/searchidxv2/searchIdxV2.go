package searchidxv2

import (
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

//////////////////////////////////////////////////////////////////
//
//                          Search Index
//
//////////////////////////////////////////////////////////////////

// SearchIdxV1 is the Search Index V1
type SearchIdxV1 struct {
	SearchSources map[string]*SearchTermIdxV2
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	owner         common.SearchIdxOwner
	batchMgr      *SearchTermBatchMgrV2
	delDocs       *DeletedDocsV1
}

type DeletedDocsV1 struct {
	DelDocs   []string        // List of DocIDs
	delDocMap map[string]bool // Map of DocID to boolean
}

// CreateSearchIdxWriterV1 creates a search index writer V2
func CreateSearchIdxWriterV2(owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	sources []SearchTermIdxSourceV2) (*SearchIdxV1, error) {
	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if _, ok := indexKey.Key.(sscryptointf.KeyMidstream); !ok {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	searchIdx := &SearchIdxV1{
		SearchSources: make(map[string]*SearchTermIdxV2),
		TermKey:       termKey,
		IndexKey:      indexKey,
		owner:         owner,
		batchMgr:      nil,
		delDocs:       &DeletedDocsV1{make([]string, 0), make(map[string]bool)},
	}

	var err error
	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV2(owner, sources, termKey, indexKey,
		searchIdx.delDocs)
	if err != nil {
		return nil, err
	}

	return searchIdx, nil
}

// GetUpdateIdsV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := common.CreateTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV2(sdc, owner, termHmac)
}

// GetLatestUpdateIDV1 returns the latest update IDs for a specific owner + term
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

// GetUpdateIdsHmacV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsHmacV2(sdc client.StrongDocClient, owner common.SearchIdxOwner, termHmac string) ([]string, error) {
	return common.GetUpdateIDs(sdc, owner, termHmac)
}
