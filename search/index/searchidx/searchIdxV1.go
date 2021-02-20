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
	sources []SearchIdxSourceV1) (*SearchIdxV1, error) {

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

	searchIdx.batchMgr, err = CreateSearchTermBatchMgrV1(sources, termKey, indexKey)
	if err != nil {
		return nil, err
	}

	//
	// Figure out all document IDs that need to be deleted from the old search index
	//
	// if delDocs != nil {
	// 	for _, docID := range delDocs {
	// 		searchIdx.delDocs.delDocMap[docID] = true
	// 	}
	// }

	// If any of the document versions are newer than the one in the old search index,
	// they need to be deleted
	// if oldSortedDocIdx != nil {
	// 	// Comparison is much easier if there is sorted doc index

	// } else if oldSearchIdx != nil {
	// 	// If there isn't sorted doc index, then it can also be done with old search index
	// }

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

// // ReadNextBlock returns the next document term index block
// func (idx *SearchIdxV1) ReadNextBlock() (*SearchIdxBlkV1, error) {
// 	if idx.Reader == nil {
// 		return nil, errors.Errorf("The document term index is not open for reading")
// 	}

// 	b, err := idx.Reader.ReadNextBlock()
// 	if err != nil && err != io.EOF {
// 		return nil, errors.New(err)
// 	}

// 	if b != nil && len(b.GetData()) > 0 {
// 		block := CreateDockTermIdxBlkV1("", 0)
// 		blk, derr := block.Deserialize(b.GetData())
// 		if derr != nil {
// 			return nil, errors.New(derr)
// 		}

// 		return blk, err
// 	}

// 	return nil, err
// }

// // FindTerm attempts to find the specified term in the term index
// func (idx *SearchIdxV1) FindTerm(term string) (bool, error) {
// 	if idx.Reader == nil {
// 		return false, errors.Errorf("The document term index is not open for reading")
// 	}

// 	blk, err := idx.Reader.SearchBinary(term, SearchComparatorV1)
// 	if err != nil {
// 		return false, errors.New(err)
// 	}

// 	return (blk != nil), nil
// }

// // Close writes any residual block data to output stream
// func (idx *SearchIdxV1) Close() error {
// 	if idx.Block != nil && idx.Block.totalTerms > 0 {
// 		serial, err := idx.Block.Serialize()
// 		if err != nil {
// 			return errors.New(err)
// 		}
// 		return idx.flush(serial)
// 	}
// 	return nil
// }

// func (idx *SearchIdxV1) flush(data []byte) error {
// 	if idx.Writer == nil {
// 		return errors.Errorf("The document term index is not open for writing")
// 	}

// 	_, err := idx.Writer.WriteBlockData(data)
// 	if err != nil {
// 		return errors.New(err)
// 	}

// 	idx.Block = CreateDocTermIdxBlkV1(idx.Block.highTerm, uint64(idx.Writer.GetMaxDataSize()))
// 	return nil
// }
