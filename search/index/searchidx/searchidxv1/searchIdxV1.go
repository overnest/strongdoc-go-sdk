package searchidxv1

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
	"github.com/shengdoushi/base58"
)

//////////////////////////////////////////////////////////////////
//
//                          Update ID
//
//////////////////////////////////////////////////////////////////

func newUpdateIDV1() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

var termHmacMutex sync.Mutex

func createTermHmac(term string, termKey *sscrypto.StrongSaltKey) (string, error) {
	termHmacMutex.Lock()
	defer termHmacMutex.Unlock()

	err := termKey.MACReset()
	if err != nil {
		return "", errors.New(err)
	}

	_, err = termKey.MACWrite([]byte(term))
	if err != nil {
		return "", errors.New(err)
	}

	hmac, err := termKey.MACSum(nil)
	if err != nil {
		return "", errors.New(err)
	}

	return base58.Encode(hmac, base58.BitcoinAlphabet), nil
}

// GetUpdateIdsV1 returns the list of available update IDs for a specific owner + term in
// reverse chronological order. The most recent update ID will come first
func GetUpdateIdsV1(owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) ([]string, error) {
	termHmac, err := createTermHmac(term, termKey)
	if err != nil {
		return nil, err
	}
	return GetUpdateIdsHmacV1(owner, termHmac)
}

// GetLatestUpdateIDV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIDV1(owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) (string, error) {

	ids, err := GetUpdateIdsV1(owner, term, termKey)
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
func GetUpdateIdsHmacV1(owner common.SearchIdxOwner, termHmac string) ([]string, error) {
	path := GetSearchIdxPathV1(common.GetSearchIdxPathPrefix(), owner, termHmac, "")

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

	sort.Slice(updateIDsInt, func(i, j int) bool { return updateIDsInt[i] > updateIDsInt[j] })
	for i := 0; i < len(updateIDs); i++ {
		updateIDs[i] = fmt.Sprintf("%x", updateIDsInt[i])
	}

	return updateIDs, nil
}

// GetLatestUpdateIdsHmacV1 returns the latest update IDs for a specific owner + term
func GetLatestUpdateIdsHmacV1(owner common.SearchIdxOwner, termHmac string) (string, error) {
	ids, err := GetUpdateIdsHmacV1(owner, termHmac)
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

// GetSearchIdxPathV1 gets the base path of the search index
func GetSearchIdxPathV1(prefix string, owner common.SearchIdxOwner, term, updateID string) string {
	if len(prefix) > 0 {
		return path.Clean(fmt.Sprintf("%v/%v/sidx/%v/%v", prefix, owner, term, updateID))
	}

	return path.Clean(fmt.Sprintf("%v/sidx/%v/%v", owner, term, updateID))
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

func (idx *SearchIdxV1) ProcessBatchTerms() (map[string]error, error) {
	emptyResult := make(map[string]error)

	termBatch, err := idx.batchMgr.GetNextTermBatch(common.STI_TERM_BATCH_SIZE)
	if err != nil {
		return emptyResult, err
	}

	if termBatch.IsEmpty() {
		return emptyResult, io.EOF
	}

	// PSL DEBUG
	fmt.Println("batch: ", termBatch.termList[0], "->", termBatch.termList[len(termBatch.termList)-1])

	return termBatch.ProcessTermBatch()
}

func (idx *SearchIdxV1) ProcessAllTerms() (map[string]error, error) {
	finalResult := make(map[string]error)

	var err error = nil
	for err == nil {
		var batchResult map[string]error

		batchResult, err = idx.ProcessBatchTerms()
		if err != nil && err != io.EOF {
			return finalResult, err
		}

		for term, err := range batchResult {
			finalResult[term] = err
		}
	}

	return finalResult, nil
}
