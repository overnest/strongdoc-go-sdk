package searchidxv1

import (
	"math/rand"
	"testing"

	docidx "github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestSearchTermUpdateIDsV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	idCount := 10
	updateIDs := make([]string, idCount)
	term := "myTerm"
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")
	defer common.RemoveSearchIndex(sdc, owner)

	// ================================ Generate updateID ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	termHmac, err := common.CreateTermHmac(term, termKey)
	assert.NilError(t, err)

	for i := 0; i < idCount; i++ {
		writer, updateID, err := common.OpenSearchTermIndexWriter(sdc, owner, termHmac)
		updateIDs[idCount-i-1] = updateID
		assert.NilError(t, err)
		err = writer.Close()
		assert.NilError(t, err)
	}

	resultIDs, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}

func TestSearchIdxWriterV1(t *testing.T) {
	// ================================ Prev Test ================================
	versions := 3
	numDocs := 10
	delDocs := 4
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	keys, err := TestGetKeys()
	assert.NilError(t, err)
	docKey, termKey, indexKey := keys[TestDocKeyID], keys[TestTermKeyID], keys[TestIndexKeyID]
	assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)

	firstDocs, err := docidx.InitTestDocuments(numDocs, false)
	assert.NilError(t, err)

	docVers := make([][]*docidx.TestDocumentIdxV1, len(firstDocs))
	for i, doc := range firstDocs {
		docVers[i] = make([]*docidx.TestDocumentIdxV1, versions+1)
		docVers[i][0] = doc
	}

	TestCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, nil, firstDocs)
	TestValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, firstDocs)
	defer docidx.CleanupTemporaryDocumentIndex()
	defer common.CleanupTemporarySearchIndex()

	// err = TestCleanupSearchIndexes(owner, 1)
	// assert.NilError(t, err)

	// ================================ Update Search Index ================================
	for v := 1; v <= versions; v++ {
		oldDocs := make([]*docidx.TestDocumentIdxV1, 0, numDocs)
		newDocs := make([]*docidx.TestDocumentIdxV1, 0, numDocs)
		for _, docVerList := range docVers {
			oldDoc := docVerList[v-1]

			addedTerms, deletedTerms := rand.Intn(99)+1, rand.Intn(99)+1
			// addedTerms, deletedTerms := 20, 10

			newDoc, err := oldDoc.CreateModifiedDoc(addedTerms, deletedTerms)
			assert.NilError(t, err)

			docVerList[v] = newDoc
			newDocs = append(newDocs, newDoc)
			oldDocs = append(oldDocs, oldDoc)
		}

		TestCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, oldDocs, newDocs)
		TestValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, newDocs)
	}

	// Test remove document
	remDocs := make([]*docidx.TestDocumentIdxV1, numDocs)
	for i, docVer := range docVers {
		remDocs[i] = docVer[len(docVer)-1]
	}

	for i := 0; i < numDocs-delDocs; i++ {
		j := rand.Intn(len(remDocs))
		remDocs[j] = remDocs[len(remDocs)-1]
		remDocs = remDocs[:len(remDocs)-1]
	}

	TestDeleteSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, remDocs)
	TestValidateDeleteSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, remDocs)
}
