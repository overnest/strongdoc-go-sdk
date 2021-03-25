package searchidxv1

import (
	"fmt"
	"github.com/go-errors/errors"
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

var docID1 = "doc1"
var docID2 = "doc2"
var docVer1 uint64 = 1
var docVer2 uint64 = 1
var owner = common.CreateSearchIdxOwner(utils.OwnerUser, "ownerID123")

var filename = "./testDocuments/CompanyIntro.txt"

//var filename = "./testDocuments/books/A Christmas Carol in Prose.txt.gz"

func createSearchIndexSource(sdc client.StrongDocClient, sourceFilePath string,
	docID string, docVer uint64, indexKey *sscrypto.StrongSaltKey) (source SearchTermIdxSourceV1, err error) {
	// open source file
	var sourceFile *os.File
	sourceFile, err = utils.OpenLocalFile(sourceFilePath)
	defer sourceFile.Close()
	// create doi and dti
	err = docidx.CreateAndSaveDocIndexes(sdc, docID, docVer, indexKey, sourceFile)
	if err != nil {
		return
	}
	// open doi and dti, create search source
	var doi common2.DocOffsetIdx
	doi, err = docidx.OpenDocOffsetIdx(sdc, docID, docVer, indexKey)
	if err != nil {
		return
	}
	var dti common2.DocTermIdx
	dti, err = docidx.OpenDocTermIdx(sdc, docID, docVer, indexKey)
	if err != nil {
		return
	}
	source, err = SearchTermIdxSourceCreateDoc(doi, dti)
	return
}

func TestBatchCreation(t *testing.T) {
	sdc := prevTest(t)
	defer common2.RemoveDocIndexes(sdc, docID1)
	defer common2.RemoveDocIndexes(sdc, docID2)

	// get file path, generate index key
	sourceFilePath, err := utils.FetchFileLoc(filename)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	// create search index source
	source1, err := createSearchIndexSource(sdc, sourceFilePath, docID1, docVer1, indexKey)
	assert.NilError(t, err)
	defer source1.Close()
	//source2, err := createSearchIndexSource(sdc, sourceFilePath, docID2, docVer2, indexKey)
	//assert.NilError(t, err)
	//defer source2.Close()

	// create search index writer
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	//siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, []SearchTermIdxSourceV1{source1, source2})
	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, []SearchTermIdxSourceV1{source1})

	// Process
	//t1 := time.Now()
	termErr, err := siw.ProcessAllTerms(sdc)
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
	}
	assert.NilError(t, err)
	//t2 := time.Now()
	//fmt.Println("Process All terms time ", t2.Sub(t1).Seconds(), "s")

	for term, err := range termErr {
		if err != nil {
			fmt.Println(term, err.(*errors.Error).ErrorStack())
		}
		assert.NilError(t, err)
	}
}
