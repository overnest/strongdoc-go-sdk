package searchidxv1

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/overnest/strongsalt-common-go/blocks"

	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

func TestSearchSortDocIdxBlockV1(t *testing.T) {
	maxSize := uint64(10000)
	idCount := 100000

	docIDs := make([]string, idCount)
	sortedDocIDs := make([]string, idCount)
	usedDocIDs := make(map[string]bool)
	for i := 0; i < idCount; i++ {
		docID := fmt.Sprintf("DocID_%v", rand.Intn(1000000))
		for usedDocIDs[docID] {
			docID = fmt.Sprintf("DocID_%v", rand.Intn(1000000))
		}
		docIDs[i] = docID
		sortedDocIDs[i] = docID
		usedDocIDs[docID] = true
	}
	sort.Strings(sortedDocIDs)
	// fmt.Println(docIDs)

	totalDocIDs := uint64(0)
	highDocID := ""

	for len(sortedDocIDs) > 0 {
		ssdib := CreateSearchSortDocIdxBlkV1(highDocID, maxSize)
		for i, docID := range docIDs {
			ssdib.AddDocVer(docID, uint64(i))
		}

		validateSsdibSize(t, ssdib)
		highDocID = ssdib.highDocID

		for i := uint64(0); i < ssdib.totalDocIDs; i++ {
			// fmt.Println(sortedDocIDs[i], ssdib.DocIDVers[i].DocID)
			assert.Equal(t, sortedDocIDs[i], ssdib.DocIDVers[i].DocID)
		}

		sortedDocIDs = sortedDocIDs[ssdib.totalDocIDs:]
		totalDocIDs += ssdib.totalDocIDs
	}

	assert.Equal(t, totalDocIDs, uint64(len(docIDs)))
}

func validateSsdibSize(t *testing.T, stib *SearchSortDocIdxBlkV1) {
	stib = stib.formatToBlockData()
	predictSize, err := blocks.GetPredictedJSONSize(stib)
	assert.NilError(t, err)
	// fmt.Println("predictSize", predictSize, "pred", stib.predictedJSONSize, "max", stib.maxDataSize, string(b))
	assert.Equal(t, uint64(predictSize), stib.predictedJSONSize)
	assert.Assert(t, stib.predictedJSONSize <= stib.maxDataSize)
}

func TestSearchSortDocIdxSimpleV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")
	term := "term1"
	maxDocID := 2000
	maxOffsetCount := 30
	defer common.RemoveSearchIndex(sdc, owner)

	// ================================ Generate Search HashedTerm Index ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	//
	// Create STI
	//
	_, stiBlocks := createSearchTermIdxSimpleV1(t, sdc, owner, term,
		termKey, indexKey, maxDocID, maxOffsetCount)

	// Convert STI blocks to sorted SSDI DocIDVer list
	docIDVers := make([]*DocIDVerV1, 0, 1000)
	docIDVerMap := make(map[string]uint64)
	for _, stiBlock := range stiBlocks {
		for docID, verOff := range stiBlock.DocVerOffset {
			if _, exist := docIDVerMap[docID]; !exist {
				docIDVerMap[docID] = verOff.Version
				docIDVers = append(docIDVers, &DocIDVerV1{docID, verOff.Version})
			}
		}
	}
	sort.Slice(docIDVers, func(i, j int) bool {
		return (strings.Compare(docIDVers[i].DocID, docIDVers[j].DocID) < 0)
	})

	// ================================ Get UpdateID ================================
	//
	// Create SSDI from STI
	//
	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	// ================================ Generate Search Sorted Doc Index ================================
	ssdi, err := CreateSearchSortDocIdxV1(sdc, owner, term, updateIDs[0], termKey, indexKey, nil)
	assert.NilError(t, err)

	err = nil
	ssdiBlocks := make([]*SearchSortDocIdxBlkV1, 0, 1000)
	for err == nil {
		var blk *SearchSortDocIdxBlkV1 = nil
		blk, err = ssdi.WriteNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}
		if blk != nil {

			//fmt.Println(blk.DocIDVers)

			ssdiBlocks = append(ssdiBlocks, blk)
		}
	}

	err = ssdi.Close()
	assert.NilError(t, err)

	//
	// Validate the written SSDI blocks
	//
	validateSsdiBlocks(t, docIDVers, ssdiBlocks)

	// ================================ Open Search Sorted Doc Index ================================
	//
	// Open SSDI for reading
	//
	ssdi, err = OpenSearchSortDocIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	assert.NilError(t, err)

	err = nil
	ssdiBlocks = ssdiBlocks[:0]
	for err == nil {
		var blk *SearchSortDocIdxBlkV1 = nil
		blk, err = ssdi.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}
		if blk != nil {
			ssdiBlocks = append(ssdiBlocks, blk)
		}
	}

	//
	// Test search results
	//
	searches := 50
	searchPositiveDocs := make([]*DocIDVerV1, searches)
	searchPositiveDocIDs := make([]string, searches)
	searchMixDocs := make([]*DocIDVerV1, searches)
	searchMixDocIDs := make([]string, searches)
	indexMap := make(map[int]bool)
	for i := 0; i < searches; i++ {
		var idx int
		for {
			idx = rand.Intn(len(docIDVers))
			_, ok := indexMap[idx]
			if !ok {
				indexMap[idx] = true
				break
			}
		}
		docID, docVer := docIDVers[idx].DocID, docIDVers[idx].DocVer

		searchPositiveDocs[i] = docIDVers[idx]
		searchPositiveDocIDs[i] = docID

		neg := rand.Intn(2)
		if neg == 1 {
			docID = fmt.Sprintf("%v_NEG", docID)
		}
		searchMixDocs[i] = &DocIDVerV1{docID, docVer}
		searchMixDocIDs[i] = docID
	}

	t1 := time.Now()

	// Validate positive single searche result
	for _, searchDoc := range searchPositiveDocs {
		docVer, err := ssdi.FindDocID(searchDoc.DocID)
		assert.NilError(t, err)
		assert.Equal(t, searchDoc.DocVer, docVer.DocVer)
	}

	t2 := time.Now()

	// Validate positive batch search result
	searchResult, err := ssdi.FindDocIDs(searchPositiveDocIDs)
	assert.NilError(t, err)
	assert.Equal(t, len(searchResult), len(searchPositiveDocIDs))
	for i, searchDocID := range searchPositiveDocIDs {
		docIDVer, exist := searchResult[searchDocID]
		assert.Assert(t, exist)
		assert.DeepEqual(t, docIDVer, searchPositiveDocs[i])
	}

	t3 := time.Now()

	// Validate mixed single search result
	for _, searchDoc := range searchMixDocs {
		docVer, err := ssdi.FindDocID(searchDoc.DocID)
		assert.NilError(t, err)
		if strings.HasSuffix(searchDoc.DocID, "_NEG") {
			assert.Assert(t, docVer == nil)
		} else {
			assert.Equal(t, searchDoc.DocVer, docVer.DocVer)
		}
	}

	t4 := time.Now()

	// Validate mixed batch search result
	searchResult, err = ssdi.FindDocIDs(searchMixDocIDs)
	assert.NilError(t, err)
	assert.Equal(t, len(searchResult), len(searchMixDocIDs))
	for i, searchDocID := range searchMixDocIDs {
		docIDVer, exist := searchResult[searchDocID]
		assert.Assert(t, exist)

		if strings.HasSuffix(searchDocID, "_NEG") {
			assert.Assert(t, docIDVer == nil)
		} else {
			assert.DeepEqual(t, docIDVer, searchMixDocs[i])
		}
	}

	t5 := time.Now()
	fmt.Println("single positive search ms", t2.Sub(t1).Milliseconds())
	fmt.Println("batch positive search ms", t3.Sub(t2).Milliseconds())
	fmt.Println("single mix search ms", t4.Sub(t3).Milliseconds())
	fmt.Println("batch mix search ms", t5.Sub(t4).Milliseconds())

	//
	// Validate the read SSDI blocks
	//
	validateSsdiBlocks(t, docIDVers, ssdiBlocks)

	err = ssdi.Close()
	assert.NilError(t, err)
}

func validateSsdiBlocks(t *testing.T, expectedDocIDVers []*DocIDVerV1, ssdiBlocks []*SearchSortDocIdxBlkV1) {
	for _, docIDVer := range expectedDocIDVers {
		// Get rid of empty SSDI blocks
		for len(ssdiBlocks) > 0 && len(ssdiBlocks[0].DocIDVers) == 0 {
			ssdiBlocks = ssdiBlocks[1:]
		}

		assert.Assert(t, len(ssdiBlocks) > 0)
		ssdiBlock := ssdiBlocks[0]
		// fmt.Println(ssdiBlock.DocIDVers[0], docIDVer)
		assert.DeepEqual(t, ssdiBlock.DocIDVers[0], docIDVer)
		ssdiBlock.DocIDVers = ssdiBlock.DocIDVers[1:]
	}

	for len(ssdiBlocks) > 0 && len(ssdiBlocks[0].DocIDVers) == 0 {
		ssdiBlocks = ssdiBlocks[1:]
	}

	assert.Equal(t, len(ssdiBlocks), 0)
}
