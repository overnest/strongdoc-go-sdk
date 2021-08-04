package searchidxv2

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"io"
	"os"
	"sort"
	"testing"
)

const (
	TestDocKeyID   = "docKey"
	TestTermKeyID  = "termKey"
	TestIndexKeyID = "indexKey"
)

// create search index v2
func TestCreateSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner,
	docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	oldDocs []*docidx.TestDocumentIdxV1,
	newDocs []*docidx.TestDocumentIdxV1) {

	sources := make([]SearchTermIdxSourceV2, 0, len(newDocs))
	for i, newDoc := range newDocs {
		// create doi & dti
		assert.NilError(t, newDoc.CreateDoiAndDti(sdc, docKey))
		newDoi, err := newDoc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)
		defer newDoi.Close()
		newDti, err := newDoc.OpenDti(sdc, docKey)
		assert.NilError(t, err)
		defer newDti.Close()

		// create SearchTermIdxSourceV2
		if oldDocs == nil {
			source, err := SearchTermIdxSourceCreateDoc(newDoi, newDti)
			assert.NilError(t, err)
			sources = append(sources, source)
		} else {
			oldDoc := oldDocs[i]
			oldDti, err := oldDoc.OpenDti(sdc, docKey)
			assert.NilError(t, err)
			defer oldDti.Close()

			source, err := SearchTermIdxSourceUpdateDoc(newDoi, oldDti, newDti)
			assert.NilError(t, err)
			sources = append(sources, source)
		}
	}

	// create search index writer
	siw, err := CreateSearchIdxWriterV2(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	// process all terms
	termErr, err := siw.ProcessAllTerms(sdc, nil)
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
	}
	assert.NilError(t, err)

	for term, err := range termErr {
		if err != nil {
			fmt.Println(term, err.(*errors.Error).ErrorStack())
		}
		assert.NilError(t, err)
	}
}

// validate created search index
func TestValidateSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	newDocs []*docidx.TestDocumentIdxV1) {

	docTermsMap := make(map[string]bool)                                     // term -> true
	docTermLocMap := make(map[*docidx.TestDocumentIdxV1]map[string][]uint64) // term -> []locations
	delTermsMap := make(map[string][]*docidx.TestDocumentIdxV1)              // term -> []testDocs

	// open document term index
	for _, doc := range newDocs {
		doi, err := doc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)

		terms, termloc, err := doi.(*docidxv1.DocOffsetIdxV1).ReadAllTermLoc()
		assert.NilError(t, err)

		docTermLocMap[doc] = termloc
		for _, term := range terms {
			docTermsMap[term] = true
		}

		err = doi.Close()
		assert.NilError(t, err)

		for term := range doc.DeletedTerms {
			delTermsMap[term] = append(delTermsMap[term], doc)
		}
	}

	docTerms := make([]string, 0, len(docTermsMap))
	for term := range docTermsMap {
		docTerms = append(docTerms, term)
	}
	sort.Strings(docTerms)

	for _, term := range docTerms {
		docLocMap := make(map[*docidx.TestDocumentIdxV1][]uint64) // doc -> locations
		for doc, termLocMap := range docTermLocMap {
			if locs, exist := termLocMap[term]; exist {
				docLocMap[doc] = locs
			}
		}

		//fmt.Println("validate term", term, "hashedTerm", common.HashTerm(term))
		TestValidateSearchTermIdxV2(t, sdc, owner, termKey, indexKey, term, docLocMap)
		TestValidateSortedDocIdxV2(t, sdc, owner, termKey, indexKey, term, docLocMap)
	}

	for term, docs := range delTermsMap {
		TestValidateDeletedSearchTermIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
		TestValidateDeletedSortedDocIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
	}
}

// validate search term index
func TestValidateSearchTermIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, docLocMap map[*docidx.TestDocumentIdxV1][]uint64) {
	hashedTerm := common.HashTerm(term)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, hashedTerm, termKey)
	assert.NilError(t, err)

	// open term index
	sti, err := OpenSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
	assert.NilError(t, err)

	for err == nil {
		var blk *SearchTermIdxBlkV2
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		if _, exists := blk.TermDocVerOffset[term]; !exists {
			continue
		}
		//fmt.Println("check the block")

		for doc, locs := range docLocMap {
			verOff := blk.TermDocVerOffset[term][doc.DocID]
			if verOff == nil {
				continue
			}

			//fmt.Println("check docID", doc.DocID)
			//fmt.Println("locs", locs)
			//fmt.Println("verOff.Offsets", verOff.Offsets)

			assert.Equal(t, doc.DocVer, verOff.Version)
			assert.Assert(t, len(locs) >= len(verOff.Offsets))
			assert.DeepEqual(t, verOff.Offsets, locs[:len(verOff.Offsets)])
			docLocMap[doc] = locs[len(verOff.Offsets):]
		}
	}

	err = sti.Close()
	assert.NilError(t, err)

	for _, locs := range docLocMap {
		assert.Equal(t, len(locs), 0)
	}
}

// validate deleted search term index
func TestValidateDeletedSearchTermIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {
	hashedTerm := common.HashTerm(term)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, hashedTerm, termKey)
	assert.NilError(t, err)

	sti, err := OpenSearchTermIdxV2(sdc, owner, term, termKey, indexKey, updateID)
	assert.NilError(t, err)

	for err == nil {
		var blk *SearchTermIdxBlkV2
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		if _, exists := blk.TermDocVerOffset[term]; !exists {
			continue
		}

		for _, delDoc := range delDocs {
			assert.Assert(t, blk.TermDocVerOffset[term][delDoc.DocID] == nil)
		}
	}

	err = sti.Close()
	assert.NilError(t, err)
}

// validate sorted doc index
func TestValidateSortedDocIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, docLocMap map[*docidx.TestDocumentIdxV1][]uint64) {
	hashedTerm := common.HashTerm(term)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, hashedTerm, termKey)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
	assert.NilError(t, err)

	docIDs := make([]string, 0, len(docLocMap))
	for doc := range docLocMap {
		docIDs = append(docIDs, doc.DocID)
	}

	docIdVerMap, err := ssdi.FindDocIDs(term, docIDs)
	assert.NilError(t, err)

	for doc := range docLocMap {
		docIDVer := docIdVerMap[doc.DocID]
		assert.Assert(t, docIDVer != nil)
		if docIDVer != nil {
			assert.Equal(t, doc.DocVer, docIDVer.DocVer)
		}
	}

	err = ssdi.Close()
	assert.NilError(t, err)
}

// validate deleted sorted doc index
func TestValidateDeletedSortedDocIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {

	hashedTerm := common.HashTerm(term)
	updateIDs, err := GetUpdateIdsV2(sdc, owner, hashedTerm, termKey)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateIDs[0])
	if err != nil {
		assert.Equal(t, err, os.ErrNotExist)
		return
	}

	docIDs := make([]string, 0, len(delDocs))
	for _, doc := range delDocs {
		docIDs = append(docIDs, doc.DocID)
	}

	docIdVerMap, err := ssdi.FindDocIDs(term, docIDs)
	assert.NilError(t, err)

	for _, doc := range delDocs {
		assert.Assert(t, docIdVerMap[doc.DocID] == nil)
	}

	err = ssdi.Close()
	assert.NilError(t, err)
}

//func TestDeleteSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
//	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
//	delDocs []*docidx.TestDocumentIdxV1) {
//
//	sources := make([]SearchTermIdxSourceV2, 0, len(delDocs))
//	for _, delDoc := range delDocs {
//		assert.NilError(t, delDoc.CreateDoiAndDti(sdc, docKey))
//		delDoi, err := delDoc.OpenDoi(sdc, docKey)
//		assert.NilError(t, err)
//		defer delDoi.Close()
//		delDti, err := delDoc.OpenDti(sdc, docKey)
//		assert.NilError(t, err)
//		defer delDti.Close()
//
//		source, err := SearchTermIdxSourceDeleteDoc(delDoi, delDti)
//		assert.NilError(t, err)
//		sources = append(sources, source)
//	}
//
//	siw, err := CreateSearchIdxWriterV2(owner, termKey, indexKey, sources)
//	assert.NilError(t, err)
//
//	termErr, err := siw.ProcessAllTerms(sdc, nil)
//	if err != nil {
//		fmt.Println(err.(*errors.Error).ErrorStack())
//	}
//	assert.NilError(t, err)
//
//	for term, err := range termErr {
//		if err != nil {
//			fmt.Println(term, err.(*errors.Error).ErrorStack())
//		}
//		assert.NilError(t, err)
//	}
//}
//
//func TestValidateDeleteSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
//	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
//	delDocs []*docidx.TestDocumentIdxV1) {
//
//	delTermsMap := make(map[string][]*docidx.TestDocumentIdxV1) // term -> []testDocs
//
//	for _, doc := range delDocs {
//		dti, err := doc.OpenDti(sdc, docKey)
//		assert.NilError(t, err)
//
//		terms, _, err := dti.(*docidxv1.DocTermIdxV1).ReadAllTerms()
//		assert.NilError(t, err)
//
//		for _, term := range terms {
//			delTermsMap[term] = append(delTermsMap[term], doc)
//		}
//
//		err = dti.Close()
//		assert.NilError(t, err)
//
//	}
//
//	for term, docs := range delTermsMap {
//		TestValidateDeletedSearchTermIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
//		TestValidateDeletedSortedDocIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
//	}
//}
//
//func TestCleanupSearchIndexes(owner common.SearchIdxOwner, numToKeep int) error {
//	searchIdxPath := path.Clean(fmt.Sprintf("%v/%v/sidx", common.GetSearchIdxPathPrefix(), owner))
//
//	termDirInfos, err := ioutil.ReadDir(searchIdxPath)
//	if err != nil {
//		return err
//	}
//
//	for _, termDirInfo := range termDirInfos {
//		if !termDirInfo.IsDir() {
//			continue
//		}
//
//		termDirPath := path.Clean(fmt.Sprintf("%v/%v", searchIdxPath, termDirInfo.Name()))
//		updateIdInfos, err := ioutil.ReadDir(termDirPath)
//		if err != nil {
//			return err
//		}
//
//		// Sort by update time
//		sort.Slice(updateIdInfos, func(i, j int) bool {
//			return updateIdInfos[i].ModTime().Before(updateIdInfos[j].ModTime())
//		})
//
//		keep := utils.Min(numToKeep, len(updateIdInfos))
//		removeDirInfos := updateIdInfos[len(updateIdInfos)-keep:]
//		for _, removeDirInfo := range removeDirInfos {
//			removeDirPath := path.Clean(fmt.Sprintf("%v/%v", termDirPath, removeDirInfo.Name()))
//			err := os.RemoveAll(removeDirPath)
//			if err != nil {
//				return err
//			}
//		}
//	}
//
//	return nil
//}
//
//func TestSaveKeys(keyMap map[string]*sscrypto.StrongSaltKey) error {
//	return searchidxv1.TestSaveKeys(keyMap)
//}
//
//func TestLoadKeys() (map[string]*sscrypto.StrongSaltKey, error) {
//	return searchidxv1.TestLoadKeys()
//}
//
//func TestGetKeys() (map[string]*sscrypto.StrongSaltKey, error) {
//	return searchidxv1.TestGetKeys()
//}