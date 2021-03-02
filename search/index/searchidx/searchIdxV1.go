package searchidx

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"time"

	"github.com/go-errors/errors"
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
func GetUpdateIdsV1(owner SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV1(owner, termHmac)
}

// GetLatestUpdateIdV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdV1(owner SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) (string, error) {

	ids, err := GetUpdateIdsV1(owner, term, termKey)
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
func GetUpdateIdsHmacV1(owner SearchIdxOwner, termHmac string) ([]string, error) {
	path := GetSearchIdxPathV1(GetSearchIdxPathPrefix(), owner, termHmac, "")

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.New(err)
	}

	updateIDs := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			updateIDs = append(updateIDs, file.Name())
		}
	}

	updateIDsInt := make([]int64, len(updateIDs))
	for i := 0; i < len(updateIDs); i++ {
		id, err := strconv.ParseInt(updateIDs[i], 16, 64)
		if err == nil {
			updateIDsInt[i] = id
		}
	}

	sort.Slice(updateIDsInt, func(i, j int) bool { return updateIDsInt[i] < updateIDsInt[j] })
	for i := 0; i < len(updateIDs); i++ {
		updateIDs[i] = fmt.Sprintf("%x", updateIDsInt[i])
	}

	return updateIDs, nil
}

// GetLatestUpdateIdsHmacV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdsHmacV1(owner SearchIdxOwner, termHmac string) (string, error) {
	ids, err := GetUpdateIdsHmacV1(owner, termHmac)
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

// GetSearchIdxPathV1 gets the base path of the search index
func GetSearchIdxPathV1(prefix string, owner SearchIdxOwner, term, updateID string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("%v/%v/sidx/%v/%v", prefix, owner, term, updateID)
	}

	return fmt.Sprintf("%v/sidx/%v/%v", owner, term, updateID)
}

// CreateSearchIdxV1 creates a search index writer V1
func CreateSearchIdxV1(termKey, indexKey *sscrypto.StrongSaltKey,
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

	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV1(nil, sources, termKey, indexKey,
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
