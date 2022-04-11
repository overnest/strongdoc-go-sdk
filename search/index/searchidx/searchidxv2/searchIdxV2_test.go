package searchidxv2

import (
	"testing"

	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
)

func TestCreateSearchIdx(t *testing.T) {
	common.EnableAllLocal()
	sdc := common.PrevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")
	numDocs := 10
	keys, err := common.TestGetKeys()
	assert.NilError(t, err)
	docKey, termKey, indexKey := keys[common.TestDocKeyID], keys[common.TestTermKeyID],
		keys[common.TestIndexKeyID]
	assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)

	docs, err := docidx.InitTestDocumentIdx(numDocs, false)
	assert.NilError(t, err)
	defer docidx.RemoveTestDocumentIdxs(sdc, docs)
	defer common.RemoveSearchIndex(sdc, owner)
	TestCreateDocIndexAndSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, nil, docs)
	TestValidateSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, docs)
}

func TestUpdateSearchIdx(t *testing.T) {
	common.EnableAllLocal()
	sdc := common.PrevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	keys, err := common.TestGetKeys()
	assert.NilError(t, err)
	docKey, termKey, indexKey := keys[common.TestDocKeyID], keys[common.TestTermKeyID],
		keys[common.TestIndexKeyID]
	assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)

	numDocs := 10
	addTerms := 10
	deleteTerms := 10

	oldDocs, err := docidx.InitTestDocumentIdx(numDocs, false)
	assert.NilError(t, err)
	defer docidx.RemoveTestDocumentIdxs(sdc, oldDocs)

	var newDocs []*docidx.TestDocumentIdxV1
	for _, doc := range oldDocs {
		newDoc, err := doc.CreateModifiedDoc(addTerms, deleteTerms)
		assert.NilError(t, err)
		newDocs = append(newDocs, newDoc)
	}
	defer docidx.CleanupTestDocumentsTmpFiles()

	defer common.RemoveSearchIndex(sdc, owner)
	TestCreateDocIndexAndSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, nil, oldDocs)
	TestValidateSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, oldDocs)
	TestCreateDocIndexAndSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, oldDocs, newDocs)
	TestValidateSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, newDocs)
}
