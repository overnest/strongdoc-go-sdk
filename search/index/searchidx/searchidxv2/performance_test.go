package searchidxv2

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	common2 "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"os"
	"testing"
)

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

	common.STI_TERM_BATCH_SIZE_V2 = 10
	//event := utils.NewTimeEvent(fmt.Sprintf("test_batchSize_%d", common.STI_TERM_BATCH_SIZE_V2), output)
	testSearchIndexAllBatchesGeneration(t, sdc, owner, nil, docIDs, docVers, indexKey)
	//event.Output()

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
	siw, err := CreateSearchIdxWriterV2(owner, termKey, indexKey, sources)
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
	sources []SearchTermIdxSourceV2, err error) {
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
		var source SearchTermIdxSourceV2
		source, err = SearchTermIdxSourceCreateDoc(doi, dti)
		if err != nil {
			fmt.Println("fail to create source")
			return
		}
		sources = append(sources, source)
	}
	return
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

func prevTest(t *testing.T) client.StrongDocClient {
	if utils.TestLocal {
		return nil
	}
	// register org and admin
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	// login
	user := users[0][0]
	err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
	assert.NilError(t, err)
	return sdc
}
