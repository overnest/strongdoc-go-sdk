package searchidxv1

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sort"
	"testing"
	"time"

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
	docKeyID   = "docKey"
	termKeyID  = "termKey"
	indexKeyID = "indexKey"
)

func TestSearchTermUpdateIDsV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	idCount := 10
	updateIDs := make([]string, idCount)
	term := "myTerm"
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	// ================================ Generate updateID ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	termHmac, err := createTermHmac(term, termKey)
	assert.NilError(t, err)

	for i := 0; i < idCount; i++ {
		writer, updateID, err := common.OpenSearchTermIndexWriter(sdc, owner, termHmac)
		updateIDs[idCount-i-1] = updateID
		assert.NilError(t, err)
		err = writer.Close()
		assert.NilError(t, err)
	}

	time.Sleep(time.Second * 10)

	defer common.RemoveSearchIndex(sdc, owner, termHmac)

	resultIDs, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}

//  TODO only available for localTest
func TestSearchIdxWriterV1(t *testing.T) {
	versions := 3
	numDocs := 10
	sdc := prevTest(t)
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	var docKey, termKey, indexKey *sscrypto.StrongSaltKey = nil, nil, nil

	keys, err := loadKeys()
	if err != nil || len(keys) < 3 {
		docKey, err = sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		assert.NilError(t, err)
		termKey, err = sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
		assert.NilError(t, err)
		indexKey, err = sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		assert.NilError(t, err)

		saveKeys(map[string]*sscrypto.StrongSaltKey{
			docKeyID:   docKey,
			termKeyID:  termKey,
			indexKeyID: indexKey,
		})
	} else {
		docKey = keys[docKeyID]
		termKey = keys[termKeyID]
		indexKey = keys[indexKeyID]
		assert.Assert(t, docKey != nil && termKey != nil && indexKey != nil)
	}

	firstDocs, err := docidx.InitTestDocuments(numDocs, false)
	assert.NilError(t, err)

	docVers := make([][]*docidx.TestDocumentIdxV1, len(firstDocs))
	for i, doc := range firstDocs {
		docVers[i] = make([]*docidx.TestDocumentIdxV1, versions+1)
		docVers[i][0] = doc
	}

	testCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, nil, firstDocs)
	testValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, firstDocs)
	defer docidx.CleanAllTmpFiles()
	defer os.RemoveAll(common.GetSearchIdxPathPrefix())

	// err = cleanupSearchIndexes(owner, 1)
	// assert.NilError(t, err)

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

		testCreateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, oldDocs, newDocs)
		testValidateSearchIdxV1(t, sdc, owner, docKey, termKey, indexKey, newDocs)
	}
}

func testCreateSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
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

func testValidateSearchIdxV1(t *testing.T, sdc client.StrongDocClient,
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

		testValidateSearchTermIdxV1(t, sdc, owner, termKey, indexKey, term, docLocMap)
		testValidateSortedDocIdxV1(t, sdc, owner, termKey, indexKey, term, docLocMap)
	}

	for term, docs := range delTermsMap {
		testValidateDeletedSearchTermIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
		testValidateDeletedSortedDocIdxV1(t, sdc, owner, termKey, indexKey, term, docs)
	}
}

func testValidateSearchTermIdxV1(t *testing.T, sdc client.StrongDocClient,
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

func testValidateDeletedSearchTermIdxV1(t *testing.T, sdc client.StrongDocClient,
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

func testValidateSortedDocIdxV1(t *testing.T, sdc client.StrongDocClient,
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

func testValidateDeletedSortedDocIdxV1(t *testing.T, sdc client.StrongDocClient,
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

func cleanupSearchIndexes(owner common.SearchIdxOwner, numToKeep int) error {
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

func saveKeys(keyMap map[string]*sscrypto.StrongSaltKey) error {
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

func loadKeys() (map[string]*sscrypto.StrongSaltKey, error) {
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
	hamcTerm, err := createTermHmac(term, termKey)
	if err != nil {
		return err
	}
	return common.RemoveSearchIndex(sdc, owner, hamcTerm)
}
