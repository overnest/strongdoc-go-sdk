package searchidx

import (
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

//////////////////////////////////////////////////////////////////
//
//                          Update ID
//
//////////////////////////////////////////////////////////////////

// GetUpdateIdsV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV1(sdc client.StrongDocClient, owner SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV1(sdc, owner, termHmac)
}

// GetLatestUpdateIdV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdV1(sdc client.StrongDocClient, owner SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) (string, error) {

	ids, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	if err != nil {
		return "", err
	}

	if ids == nil || len(ids) == 0 {
		return "", nil
	}

	return ids[0], nil
}

// GetUpdateIdsHmacV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsHmacV1(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string) ([]string, error) {
	return getUpdateIDs(sdc, owner, termHmac)
}

// GetLatestUpdateIdsHmacV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdsHmacV1(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string) (string, error) {
	ids, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	if err != nil {
		return "", err
	}

	if ids == nil || len(ids) == 0 {
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
	batchMgr       *SearchTermBatchMgrV1
	delDocs        *DeletedDocsV1
}

type DeletedDocsV1 struct {
	DelDocs   []string        // List of DocIDs
	delDocMap map[string]bool // Map of DocID to boolean
}

// CreateSearchIdxV1 creates a search index writer V1
func CreateSearchIdxV1(sdc client.StrongDocClient, termKey, indexKey *sscrypto.StrongSaltKey,
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
		batchMgr:       nil,
		delDocs:        &DeletedDocsV1{make([]string, 0), make(map[string]bool)},
	}

	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV1(sdc, nil, sources, termKey, indexKey,
		searchIdx.delDocs)
	if err != nil {
		return nil, err
	}

	return searchIdx, nil
}

func (idx *SearchIdxV1) DoBatch() error {
	termBatch, err := idx.batchMgr.GetNextTermBatch(STI_TERM_BATCH_SIZE)
	if err != nil {
		return err
	}

	termBatch.ProcessBatch()
	return nil
}
