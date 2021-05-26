package searchidxv1

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	docidx "github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

const (
	TestDocKeyID   = "docKey"
	TestTermKeyID  = "termKey"
	TestIndexKeyID = "indexKey"
)

func TestCreateSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner,
	docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	oldDocs []*docidx.TestDocumentIdxV1,
	newDocs []*docidx.TestDocumentIdxV1) {

	sources := make([]SearchTermIdxSourceV1, 0, len(newDocs))
	for i, newDoc := range newDocs {
		assert.NilError(t, newDoc.CreateDoiAndDti(sdc, docKey))
		newDoi, err := newDoc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)
		defer newDoi.Close()
		newDti, err := newDoc.OpenDti(sdc, docKey)
		assert.NilError(t, err)
		defer newDti.Close()

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

	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	termErr, err := siw.ProcessAllTerms(sdc)
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

func TestValidateSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	newDocs []*docidx.TestDocumentIdxV1) {

	docTermsMap := make(map[string]bool)
	docTermLocMap := make(map[*docidx.TestDocumentIdxV1]map[string][]uint64) // term -> []locations
	delTermsMap := make(map[string][]*docidx.TestDocumentIdxV1)              // term -> []testDocs

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

		TestValidateSearchTermIdxV1(t, sdc, owner, termKey, indexKey, term, docLocMap)
		TestValidateSortedDocIdxV1(t, sdc, owner, termKey, indexKey, term, docLocMap)
	}

	for term, docs := range delTermsMap {
		TestValidateDeletedSearchTermIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
		TestValidateDeletedSortedDocIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
	}
}

func TestValidateSearchTermIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, docLocMap map[*docidx.TestDocumentIdxV1][]uint64) {

	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	sti, err := OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	assert.NilError(t, err)

	for err == nil {
		var blk *SearchTermIdxBlkV1
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		for doc, locs := range docLocMap {
			verOff := blk.DocVerOffset[doc.DocID]
			if verOff == nil {
				continue
			}
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

func TestValidateDeletedSearchTermIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {

	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	sti, err := OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	assert.NilError(t, err)

	for err == nil {
		var blk *SearchTermIdxBlkV1
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		for _, delDoc := range delDocs {
			assert.Assert(t, blk.DocVerOffset[delDoc.DocID] == nil)
		}
	}

	err = sti.Close()
	assert.NilError(t, err)
}

func TestValidateSortedDocIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, docLocMap map[*docidx.TestDocumentIdxV1][]uint64) {

	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	assert.NilError(t, err)

	docIDs := make([]string, 0, len(docLocMap))
	for doc := range docLocMap {
		docIDs = append(docIDs, doc.DocID)
	}

	docIdVerMap, err := ssdi.FindDocIDs(docIDs)
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

func TestValidateDeletedSortedDocIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {

	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	if err != nil {
		assert.Equal(t, err, os.ErrNotExist)
		return
	}

	docIDs := make([]string, 0, len(delDocs))
	for _, doc := range delDocs {
		docIDs = append(docIDs, doc.DocID)
	}

	docIdVerMap, err := ssdi.FindDocIDs(docIDs)
	assert.NilError(t, err)

	for _, doc := range delDocs {
		assert.Assert(t, docIdVerMap[doc.DocID] == nil)
	}

	err = ssdi.Close()
	assert.NilError(t, err)
}

func TestDeleteSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	delDocs []*docidx.TestDocumentIdxV1) {

	sources := make([]SearchTermIdxSourceV1, 0, len(delDocs))
	for _, delDoc := range delDocs {
		assert.NilError(t, delDoc.CreateDoiAndDti(sdc, docKey))
		delDoi, err := delDoc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)
		defer delDoi.Close()
		delDti, err := delDoc.OpenDti(sdc, docKey)
		assert.NilError(t, err)
		defer delDti.Close()

		source, err := SearchTermIdxSourceDeleteDoc(delDoi, delDti)
		assert.NilError(t, err)
		sources = append(sources, source)
	}

	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	termErr, err := siw.ProcessAllTerms(sdc)
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

func TestValidateDeleteSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	delDocs []*docidx.TestDocumentIdxV1) {

	delTermsMap := make(map[string][]*docidx.TestDocumentIdxV1) // term -> []testDocs

	for _, doc := range delDocs {
		dti, err := doc.OpenDti(sdc, docKey)
		assert.NilError(t, err)

		terms, _, err := dti.(*docidxv1.DocTermIdxV1).ReadAllTerms()
		assert.NilError(t, err)

		for _, term := range terms {
			delTermsMap[term] = append(delTermsMap[term], doc)
		}

		err = dti.Close()
		assert.NilError(t, err)

	}

	for term, docs := range delTermsMap {
		TestValidateDeletedSearchTermIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
		TestValidateDeletedSortedDocIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
	}
}

func TestCleanupSearchIndexes(owner common.SearchIdxOwner, numToKeep int) error {
	searchIdxPath := path.Clean(fmt.Sprintf("%v/%v/sidx", common.GetSearchIdxPathPrefix(), owner))

	termDirInfos, err := ioutil.ReadDir(searchIdxPath)
	if err != nil {
		return err
	}

	for _, termDirInfo := range termDirInfos {
		if !termDirInfo.IsDir() {
			continue
		}

		termDirPath := path.Clean(fmt.Sprintf("%v/%v", searchIdxPath, termDirInfo.Name()))
		updateIdInfos, err := ioutil.ReadDir(termDirPath)
		if err != nil {
			return err
		}

		// Sort by update time
		sort.Slice(updateIdInfos, func(i, j int) bool {
			return updateIdInfos[i].ModTime().Before(updateIdInfos[j].ModTime())
		})

		keep := utils.Min(numToKeep, len(updateIdInfos))
		removeDirInfos := updateIdInfos[len(updateIdInfos)-keep:]
		for _, removeDirInfo := range removeDirInfos {
			removeDirPath := path.Clean(fmt.Sprintf("%v/%v", termDirPath, removeDirInfo.Name()))
			err := os.RemoveAll(removeDirPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var savedKeyPath string = path.Clean("/tmp/savedKeys")

func TestSaveKeys(keyMap map[string]*sscrypto.StrongSaltKey) error {
	if err := os.MkdirAll(savedKeyPath, 0770); err != nil {
		return err
	}

	for name, key := range keyMap {
		data, err := key.Serialize()
		if err != nil {
			return err
		}

		file, err := os.Create(path.Clean(fmt.Sprintf("%v/%v", savedKeyPath, name)))
		if err != nil {
			return err
		}
		n, err := file.Write(data)
		if err != nil {
			return err
		}
		if n != len(data) {
			return errors.Errorf("Written %v bytes instead of %v bytes", n, len(data))
		}
		file.Close()
	}

	return nil
}

func TestLoadKeys() (map[string]*sscrypto.StrongSaltKey, error) {
	keyMap := make(map[string]*sscrypto.StrongSaltKey)

	keyInfos, err := ioutil.ReadDir(savedKeyPath)
	if err != nil {
		return keyMap, err
	}

	for _, keyInfo := range keyInfos {
		if keyInfo.IsDir() {
			continue
		}

		file, err := os.Open(path.Clean(fmt.Sprintf("%v/%v", savedKeyPath, keyInfo.Name())))
		if err != nil {
			return keyMap, err
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return keyMap, err
		}

		key, err := sscrypto.DeserializeKey(data)
		if err != nil {
			return keyMap, err
		}

		keyMap[keyInfo.Name()] = key
	}

	return keyMap, err
}

func TestGetKeys() (map[string]*sscrypto.StrongSaltKey, error) {
	keys, err := TestLoadKeys()
	if err != nil || len(keys) < 3 {
		docKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		if err != nil {
			return nil, err
		}
		termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
		if err != nil {
			return nil, err
		}
		indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		if err != nil {
			return nil, err
		}

		keys = map[string]*sscrypto.StrongSaltKey{
			TestDocKeyID:   docKey,
			TestTermKeyID:  termKey,
			TestIndexKeyID: indexKey,
		}
		TestSaveKeys(keys)
		return keys, nil
	}

	return keys, err
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

func generateTermHmacAndRemoveSearchIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) error {
	hamcTerm, err := common.CreateTermHmac(term, termKey)
	if err != nil {
		return err
	}
	return common.RemoveSearchIndex(sdc, owner, hamcTerm)
}