package searchidxv1

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	common2 "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"os"
	"testing"
)

// Experiment1
// test single file search one-batch terms index generation processing time with different batch size
// x : batch size
// y : processing time
func TestSIDExp1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "ownerID123")
	output := "result1.txt"
	os.Remove(output)

	// ================================ Generate Doc index ================================
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	docIDs, docVers := testDocIndexGeneration(t, sdc, indexKey, 10)

	// ================================ Generate Search index ================================

	common.STI_TERM_BATCH_SIZE = 200
	event := utils.NewTimeEvent("processAllBatches", output)
	testSearchIndexAllBatchesGeneration(t, sdc, owner, event, docIDs, docVers, indexKey)
	event.Output()

}

// Experiment2
// test multiple files search index all terms generation processing time with different batch size
// x : batch size
// y1 : single file processing time
// y2 : two files processing time
func TestSIDExp2(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "ownerID123")
	output := "result2.txt"
	os.Remove(output)

	// ================================ Generate Doc index ================================
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	docIDs, docVers := testDocIndexGeneration(t, sdc, indexKey, 2)

	// ================================ Generate Search index ================================
	for i := 1; i <= 25; i++ {
		common.STI_TERM_BATCH_SIZE = i

		// single file
		event := utils.NewTimeEvent(fmt.Sprintf("test %v", i), output)
		testSearchIndexAllBatchesGeneration(t, sdc, owner, event, []string{docIDs[0]}, []uint64{docVers[0]}, indexKey)
		event.Output()

		// two files
		event = utils.NewTimeEvent(fmt.Sprintf("test %v", i), output)
		testSearchIndexAllBatchesGeneration(t, sdc, owner, event, docIDs, docVers, indexKey)
		event.Output()
	}

}

// Experiment3
// test search index all terms generation processing time with different batch size
func TestSIDExp3(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "ownerID123")

	output := "result3.txt"
	os.Remove(output)

	// ================================ Generate Doc index ================================
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	docIDs, docVers := testDocIndexGeneration(t, sdc, indexKey, 1)

	// ================================ Generate Search index ================================
	for i := 151; i <= 300; i += 10 {
		common.STI_TERM_BATCH_SIZE = i
		event := utils.NewTimeEvent(fmt.Sprintf("test %d", i), output)
		testSearchIndexAllBatchesGeneration(t, sdc, owner, event, docIDs, docVers, indexKey)
		event.Output()
	}

}

func testDocIndexGeneration(t *testing.T, sdc client.StrongDocClient, indexKey *sscrypto.StrongSaltKey, num int) (docIDs []string, docVers []uint64) {
	// remove existing doc indexes
	docs, err := docidx.InitTestDocuments(num, false)
	for _, doc := range docs {
		common2.RemoveDocIndexes(sdc, doc.DocID)
	}

	// generate doc indexes
	for _, doc := range docs {
		docIDs = append(docIDs, doc.DocID)
		docVers = append(docVers, doc.DocVer)
		err = doc.CreateDoiAndDti(sdc, indexKey)
		assert.NilError(t, err)
	}
	return
}

func testSearchIndexAllBatchesGeneration(t *testing.T, sdc client.StrongDocClient, owner common.SearchIdxOwner, event *utils.TimeEvent,
	docIDs []string, docVers []uint64, indexKey *sscrypto.StrongSaltKey) {
	testSearchIndexGeneration(t, sdc, owner, false, event, docIDs, docVers, indexKey)
}

func testSearchIndexOneBatchGeneration(t *testing.T, sdc client.StrongDocClient, owner common.SearchIdxOwner, event *utils.TimeEvent,
	docIDs []string, docVers []uint64, indexKey *sscrypto.StrongSaltKey) {
	testSearchIndexGeneration(t, sdc, owner, true, event, docIDs, docVers, indexKey)
}

func testSearchIndexGeneration(t *testing.T, sdc client.StrongDocClient, owner common.SearchIdxOwner, oneBatch bool, event *utils.TimeEvent,
	docIDs []string, docVers []uint64, indexKey *sscrypto.StrongSaltKey) {
	// remove existing search index
	defer common.RemoveSearchIndex(sdc, owner)

	// create search index sources
	e1 := utils.AddSubEvent(event, "createSearchIndexSources")
	sources, err := createSearchIndexSources(sdc, docIDs, docVers, indexKey)
	utils.EndEvent(e1)
	assert.NilError(t, err)

	// create search index writer
	e2 := utils.AddSubEvent(event, "CreateSearchIdxWriterV1")
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, sources)
	utils.EndEvent(e2)
	assert.NilError(t, err)

	// create search index
	var e3 *utils.TimeEvent
	if oneBatch {
		e3 = utils.AddSubEvent(event, "ProcessBatchTerms")
		_, err = siw.ProcessBatchTerms(sdc, e3)

	} else {
		e3 = utils.AddSubEvent(event, "ProcessAllTerms")
		_, err = siw.ProcessAllTerms(sdc, e3)
	}
	utils.EndEvent(e3)
	assert.NilError(t, err)

	// close sources
	for _, source := range sources {
		err = source.Close()
		assert.NilError(t, err)
	}
}

func createSearchIndexSources(sdc client.StrongDocClient, docIDs []string, docVers []uint64, indexKey *sscrypto.StrongSaltKey) (
	sources []SearchTermIdxSourceV1, err error) {
	for idx, docID := range docIDs {
		var doi common2.DocOffsetIdx
		doi, err = docidx.OpenDocOffsetIdx(sdc, docID, docVers[idx], indexKey)
		if err != nil {
			fmt.Println("fail to open offset index ")
			return
		}
		var dti common2.DocTermIdx
		dti, err = docidx.OpenDocTermIdx(sdc, docID, docVers[idx], indexKey)
		if err != nil {
			fmt.Println("fail to open term index ")
			return
		}
		var source SearchTermIdxSourceV1
		source, err = SearchTermIdxSourceCreateDoc(doi, dti)
		if err != nil {
			fmt.Println("fail to create source")
			return
		}
		sources = append(sources, source)
	}
	return
}
