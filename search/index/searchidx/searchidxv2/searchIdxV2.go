package searchidxv2

import (
	"github.com/go-errors/errors"
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
