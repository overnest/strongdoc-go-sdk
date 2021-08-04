package searchidx

import (
	"io"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	docidx "github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	dicommon "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv1"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"

	"gotest.tools/assert"
)

//  TODO only available for localTest
func TestSearchIdxWriterV1(t *testing.T) {
	numDocs := 10
	numTerms := 5
	// delDocs := 4
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	keys, err := searchidxv1.TestGetKeys()
	assert.NilError(t, err)
	docKey, termKey, indexKey := keys[searchidxv1.TestDocKeyID], keys[searchidxv1.TestTermKeyID],
		keys[searchidxv1.TestIndexKeyID]
	assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)

	docs, err := docidx.InitTestDocuments(numDocs, false)
	assert.NilError(t, err)

	searchidxv1.TestCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, nil, docs)
	searchidxv1.TestValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, docs)
	defer docidx.CleanupTemporaryDocumentIndex()
	defer common.CleanupTemporarySearchIndex()

	origTermDocs := make(map[string]map[string]uint64) // term -> (docID->docVer)
	terms := make([]string, 0, numTerms)
	for _, doc := range docs {
		dti, err := docidx.OpenDocTermIdx(sdc, doc.DocID, doc.DocVer, docKey)
		assert.NilError(t, err)

		switch dti.GetDtiVersion() {
		case dicommon.DTI_V1:
			dtiv1 := dti.(*docidxv1.DocTermIdxV1)
			err = nil
			for err == nil {
				var blk *docidxv1.DocTermIdxBlkV1
				blk, err = dtiv1.ReadNextBlock()
				if err != nil {
					assert.Equal(t, err, io.EOF)
				}

				if blk != nil {
					for _, term := range blk.Terms {
						docIDVer := origTermDocs[term]
						if docIDVer == nil {
							docIDVer = make(map[string]uint64)
							origTermDocs[term] = docIDVer
						}
						docIDVer[doc.DocID] = doc.DocVer
					}
				}
			}
		default:
			assert.Assert(t, false, "DTI version %v unsupported", dti.GetDtiVersion())
		}

		assert.NilError(t, dti.Close())
	} // for _, doc := range docs

	for term := range origTermDocs {
		terms = append(terms, term)
		if len(terms) >= numTerms {
			break
		}
	}

	//
	// Test STI
	//
	stiReader, err := OpenSearchTermIndex(sdc, owner, terms, termKey, indexKey, common.STI_V1)
	assert.NilError(t, err)
	defer stiReader.Close()

	stiSearchTermDocs := make(map[string]map[string]uint64) // term -> (docID->docVer)
	for err == nil {
		var stiData StiData
		stiData, err = stiReader.ReadNextData()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if stiData != nil {
			for _, term := range terms {
				docOffset := stiData.GetAllDocOffsets()[term]
				if docOffset != nil {
					docIDVer := stiSearchTermDocs[term]
					if docIDVer == nil {
						docIDVer = make(map[string]uint64)
						stiSearchTermDocs[term] = docIDVer
					}
					for _, docID := range docOffset.GetDocIDs() {
						docIDVer[docID] = docOffset.GetDocVer(docID)
					}
				}
			}
		}
	}

	for term, docIDVer := range stiSearchTermDocs {
		assert.DeepEqual(t, origTermDocs[term], docIDVer)
	}

	//
	// Test SSDI
	//
	ssdiReader, err := OpenSearchSortedDocIndex(sdc, owner, terms, termKey, indexKey, common.STI_V1)
	assert.NilError(t, err)
	defer ssdiReader.Close()

	ssdiSearchTermDocs := make(map[string]map[string]uint64) // term -> (docID->docVer)
	for err == nil {
		var ssdiData SsdiData
		ssdiData, err = ssdiReader.ReadNextData()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if ssdiData != nil {
			for _, term := range terms {
				docs := ssdiData.GetDocs(term)
				if docs != nil {
					docIDVer := ssdiSearchTermDocs[term]
					if docIDVer == nil {
						docIDVer = make(map[string]uint64)
						ssdiSearchTermDocs[term] = docIDVer
					}
					for _, doc := range docs {
						docIDVer[doc.GetDocID()] = doc.GetDocVer()
					}
				}
			}
		}
	}

	for term, docIDVer := range ssdiSearchTermDocs {
		assert.DeepEqual(t, origTermDocs[term], docIDVer)
	}

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
