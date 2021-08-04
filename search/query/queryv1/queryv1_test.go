package queryv1

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv2"
	"io"
	"path/filepath"
	"sort"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv1"
	qcommon "github.com/overnest/strongdoc-go-sdk/search/query/common"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

//
// Known term locations
//
// "sclerites"    : A Text-book of Entomology
// "murexide"     : A Text-book of Entomology
// "paarige"      : A Test-book of Entomology
// "holsters"     : A Tale Of Two Cities, The Count Of Monte Cristo
// "ironmongery"  : A Christmas Carol, David Copperfield, The Slang Dictionary, Ulysses
// "irradiated"   : Anne of Green Gables, Frankenstein
// "cashregister" : Ulysses
// "membrane"     : A Text-book of Entomology, Essays of Michel de Montaigne, Leviathan, Moby Dick, Ulysses, Walden
// "beetles"      : A Text-book of Entomology, Crime and Punishment, Great Expectations, Pillar of Fire, Pygmalion,
//                  The Great Gatsby, The Happy Prince, Jungle Book, Picture of Dorian Gray, Secret Garden,
//                  The Sign Of The Four, The Strong Case Of Dr Jekyll and Mr Hyde, Ulysses, War and Peace

const searchIndexTestVersion = common.STI_V2

//  TODO only available for localTest
func TestPhraseSearchV1(t *testing.T) {
	sdc, owner, _, termKey, indexKey, docs := setup(t, searchIndexTestVersion)
	defer cleanup()

	phrases := [][]string{
		{"almost", "no", "restrictions"},
		{"almost", "no", "restrictions", "doesnotexist"},
		{"world", "more", "ridiculous", "than"},
		{"world", "more", "doesnotexist", "ridiculous", "than"},
	}

	for _, phrase := range phrases {
		fmt.Println("phrase", phrase)
		// Do Search
		reader, err := OpenPhraseSearchV1(sdc, owner, phrase, termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)

		var searchResult *PhraseSearchResultV1 = nil
		for err == nil {
			var newResult *PhraseSearchResultV1 = nil
			newResult, err = reader.GetNextResult()
			//fmt.Println("newResult=", newResult, "err=", err)
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}

			searchResult = searchResult.Merge(newResult)
		}

		reader.Close()
		//fmt.Println("searchResult", searchResult)

		// Do Grep
		grepResult := filterGrep(t, phrase, docs)
		fmt.Println("grepResult", grepResult)

		// Compare Results
		for _, doc := range docs {
			searchDoc := searchResult.GetMatchDocMap()[doc.DocFileName]
			grepDoc := grepResult[doc.DocFileName]

			if searchDoc == nil || len(grepDoc) == 0 {
				assert.Assert(t, searchDoc == nil)
				assert.Equal(t, len(grepDoc), 0)
			} else {
				assert.Equal(t, len(searchDoc.PhraseOffsets), len(grepDoc), doc.DocFileName)
			}
		}
	}
}

func TestTermSearchV1(t *testing.T) {
	sdc, owner, _, termKey, indexKey, docs := setup(t, searchIndexTestVersion)
	defer cleanup()

	termList := [][]string{
		{"restrictions", "doesnotexist", "sclerites", "paarige"},
	}

	for _, terms := range termList {
		// Do Search
		reader, err := OpenTermSearchV1(sdc, owner, terms, termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)

		var searchResult *TermSearchResultV1 = nil
		for err == nil {
			var newResult *TermSearchResultV1 = nil
			newResult, err = reader.GetNextResult()
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}

			searchResult = searchResult.Merge(newResult)
		}

		reader.Close()

		for _, term := range terms {
			// Do Grep
			grepResult := filterGrep(t, []string{term}, docs)

			// Compare Results
			searchDocIDs := searchResult.TermDocIDs[term]

			grepDocIDs := make([]string, 0, len(grepResult))
			for docID, offsets := range grepResult {
				if len(offsets) > 0 {
					grepDocIDs = append(grepDocIDs, docID)
				}
			}
			sort.Strings(grepDocIDs)

			if searchDocIDs == nil {
				searchDocIDs = []string{}
			}
			assert.DeepEqual(t, searchDocIDs, grepDocIDs)
		} // for _, term := range terms
	} // for _, terms := range termList
}

func TestQueryTermsAndV1(t *testing.T) {
	sdc, owner, _, termKey, indexKey, docs := setup(t, searchIndexTestVersion)
	defer cleanup()

	termList := [][]string{
		{"restrictions", "sclerites"},
		{"paarige", "restrictions"},
		{"restrictions", "doesnotexist", "sclerites"},
		{"doesnotexist", "paarige", "restrictions"},
	}

	for _, terms := range termList {
		queryResult, err := QueryTermsAndV1(sdc, owner, terms, termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)

		queryResultMap := make(map[string]bool)
		for _, queryDoc := range queryResult.GetDocVers() {
			queryResultMap[queryDoc.GetDocID()] = true
		}

		grepResultMap := make(map[string]bool)
		firstTerm := true
		for _, term := range terms {
			grepResult := filterGrep(t, []string{term}, docs)
			if firstTerm {
				for docID := range grepResult {
					grepResultMap[docID] = true
				}
			} else {
				for docID := range grepResultMap {
					if _, exist := grepResult[docID]; !exist {
						delete(grepResultMap, docID)
					}
				}
			}
			firstTerm = false
		}

		assert.DeepEqual(t, queryResultMap, grepResultMap)
	}
}

func TestQueryTermsOrV1(t *testing.T) {
	sdc, owner, _, termKey, indexKey, docs := setup(t, common.STI_V1)
	defer cleanup()

	termList := [][]string{
		{"restrictions", "sclerites"},
		{"paarige", "restrictions"},
		{"restrictions", "doesnotexist", "sclerites"},
		{"doesnotexist", "paarige", "restrictions"},
		{"doesnotexist", "doesnotexisteither"},
	}

	for _, terms := range termList {
		queryResult, err := QueryTermsOrV1(sdc, owner, terms, termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)

		queryResultMap := make(map[string]bool)
		for _, queryDoc := range queryResult.GetDocVers() {
			queryResultMap[queryDoc.GetDocID()] = true
		}

		grepResultMap := make(map[string]bool)
		for _, term := range terms {
			grepResult := filterGrep(t, []string{term}, docs)
			for docID := range grepResult {
				grepResultMap[docID] = true
			}
		}

		assert.DeepEqual(t, queryResultMap, grepResultMap)
	}
}

func TestQueryResultsAnd(t *testing.T) {
	sdc, owner, _, termKey, indexKey, _ := setup(t, searchIndexTestVersion)
	defer cleanup()

	queryList := []QueryCompV1{
		NewQueryCompV1(qcommon.TermAnd, []string{"membrane"}, nil),
		NewQueryCompV1(qcommon.TermAnd, []string{"sclerites", "murexide"}, nil),
	}

	for _, queryComp := range queryList {
		result := performSearchOp(t, sdc, owner, queryComp, termKey, indexKey)
		queryComp.SetResult(result)
	}

	queryAnd := NewQueryCompV1(qcommon.QueryAnd, nil, queryList)
	resultAnd := performSearchOp(t, sdc, owner, queryAnd, termKey, indexKey)
	assert.Equal(t, len(resultAnd.GetDocVers()), 1)

	queryOr := NewQueryCompV1(qcommon.QueryOr, nil, queryList)
	resultOr := performSearchOp(t, sdc, owner, queryOr, termKey, indexKey)
	assert.Equal(t, len(resultOr.GetDocVers()), 1)
}

func performSearchOp(t *testing.T, sdc client.StrongDocClient, owner common.SearchIdxOwner,
	queryComp QueryCompV1, termKey, indexKey *sscrypto.StrongSaltKey) QueryResultV1 {

	switch queryComp.GetOp() {
	case qcommon.TermAnd:
		queryResult, err := QueryTermsAndV1(sdc, owner, queryComp.GetTerms(), termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)
		return queryResult
	case qcommon.TermOr:
		queryResult, err := QueryTermsOrV1(sdc, owner, queryComp.GetTerms(), termKey, indexKey, searchIndexTestVersion)
		assert.NilError(t, err)
		return queryResult
	case qcommon.QueryAnd:
		queryResult, err := QueryResultsAndV1(queryComp.GetChildResults())
		assert.NilError(t, err)
		return queryResult
	case qcommon.QueryOr:
		queryResult, err := QueryResultsOr(queryComp.GetChildResults())
		assert.NilError(t, err)
		return queryResult
	default:
		assert.Assert(t, false, "Search operation %v not supported", queryComp.GetOp())
		return nil
	}
}

func filterGrep(t *testing.T, terms []string, docs []*docidx.TestDocumentIdxV1) map[string][]uint64 { // Document -> TermOffsets in bytes
	grepResult, err := utils.Grep(utils.GetInitialTestDocumentDir(), terms, true)
	assert.NilError(t, err)

	result := make(map[string][]uint64)
	for _, doc := range docs {
		if len(grepResult[doc.DocFilePath]) > 0 {
			result[filepath.Base(doc.DocFilePath)] = grepResult[doc.DocFilePath]
		}
	}

	return result
}

func setup(t *testing.T, ver uint32) (sdc client.StrongDocClient, owner common.SearchIdxOwner,
	docKey, termKey, indexKey *sscrypto.StrongSaltKey, docs []*docidx.TestDocumentIdxV1) {
	defer fmt.Println("========================= finish set up =========================")
	numDocs := 10
	sdc = prevTest(t)
	owner = common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	keys, err := searchidxv1.TestGetKeys()
	assert.NilError(t, err)
	docKey, termKey, indexKey = keys[searchidxv1.TestDocKeyID], keys[searchidxv1.TestTermKeyID],
		keys[searchidxv1.TestIndexKeyID]
	assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)

	docs, err = docidx.InitTestDocuments(numDocs, false)
	assert.NilError(t, err)

	common.RemoveSearchIndex(sdc, owner)
	switch ver {
	case common.STI_V1:
		searchidxv1.TestCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, nil, docs)
		searchidxv1.TestValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, docs)
	case common.STI_V2:
		searchidxv2.TestCreateSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, nil, docs)
		searchidxv2.TestValidateSearchIdxV2(t, sdc, owner, docKey, termKey, indexKey, docs)
	default:
		err = fmt.Errorf("invalid version")
		return
	}
	return
}

func cleanup() {
	docidx.CleanupTemporaryDocumentIndex()
	common.CleanupTemporarySearchIndex()
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
