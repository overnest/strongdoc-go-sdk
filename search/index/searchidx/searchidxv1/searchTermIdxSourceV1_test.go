package searchidxv1

import (
	"testing"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"gotest.tools/assert"

	docidx "github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	didxcommon "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

func TestSearchTermIdxSourceTextFileV1(t *testing.T) {
	// ================================ Prev Test ================================
	common.EnableAllLocal()
	testClient := common.PrevTest(t)
	docID1 := "DOC1"
	docVer1 := uint64(1)
	//docID2 := "DOC2"
	docVer2 := uint64(2)

	sourceFilePath1, err := utils.FetchFileLoc("./testDocuments/doc1.txt.gz")
	assert.NilError(t, err)
	sourceFilePath2, err := utils.FetchFileLoc("./testDocuments/doc1.chg.txt.gz")
	assert.NilError(t, err)
	//sourceFilePath3, err := utils.FetchFileLoc("./testDocuments/doc2.txt.gz")
	//assert.NilError(t, err)
	//sourceFilePath4, err := utils.FetchFileLoc("./testDocuments/doc2.chg.txt.gz")
	//assert.NilError(t, err)

	defer didxcommon.RemoveDocIdxs(testClient, docID1)
	// defer didxcommon.RemoveDocIndexes(testClient, docID2)

	// ================================ Generate doc index (offset + term) ================================
	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Create doc indexes
	createDocumentIndexes(t, testClient, key, docID1, docVer1, sourceFilePath1) // doc1 ver1
	createDocumentIndexes(t, testClient, key, docID1, docVer2, sourceFilePath2) // doc1 ver2
	//createDocumentIndexes(t, testClient, key, docID2, docVer1, sourceFilePath3) // doc2 ver1
	//createDocumentIndexes(t, testClient, key, docID2, docVer2, sourceFilePath4) // doc2 ver2

	// Open doc1 ver1 index
	doi1 := openDocOffsetIndex(t, testClient, key, docID1, docVer1)
	dti1 := openDocTermIndex(t, testClient, key, docID1, docVer1)

	source1, err := SearchTermIdxSourceCreateDoc(doi1, dti1)
	assert.NilError(t, err)
	assert.Assert(t, len(source1.GetAddTerms()) > 0)
	assert.Assert(t, len(source1.GetDelTerms()) == 0)
	assert.Equal(t, source1.GetDocID(), docID1)
	assert.Equal(t, source1.GetDocVer(), docVer1)
	//closeFiles(t, source1, doi1, dti1)
	source1.Close()

	// Update existing doc
	doi2 := openDocOffsetIndex(t, testClient, key, docID1, docVer2)
	dti2 := openDocTermIndex(t, testClient, key, docID1, docVer2)
	dti1 = openDocTermIndex(t, testClient, key, docID1, docVer1)
	source2, err := SearchTermIdxSourceUpdateDoc(doi2, dti1, dti2)
	assert.NilError(t, err)
	assert.Assert(t, len(source2.GetAddTerms()) > 0)
	assert.Assert(t, len(source2.GetDelTerms()) > 0)
	assert.Equal(t, source2.GetDocID(), docID1)
	assert.Equal(t, source2.GetDocVer(), docVer2)
	source2.Close()

	// Delete existing doc
	doi2 = openDocOffsetIndex(t, testClient, key, docID1, docVer2)
	dti2 = openDocTermIndex(t, testClient, key, docID1, docVer2)
	source3, err := SearchTermIdxSourceDeleteDoc(doi2, dti2)
	assert.NilError(t, err)
	assert.Assert(t, len(source3.GetAddTerms()) == 0)
	assert.Assert(t, len(source3.GetDelTerms()) > 0)
	assert.Equal(t, source3.GetDocID(), docID1)
	assert.Equal(t, source3.GetDocVer(), docVer2)
	//closeFiles(t, source3, doi2, dti2)
	source3.Close()
}

func createDocumentIndexes(t *testing.T, sdc client.StrongDocClient, key *sscrypto.StrongSaltKey,
	docID string, docVer uint64, sourceFilepath string) {
	sourceFile, err := utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)
	defer sourceFile.Close()
	err = docidx.CreateAndSaveDocIndexes(sdc, docID, docVer, key, sourceFile)
	assert.NilError(t, err)
}

func openDocOffsetIndex(t *testing.T, sdc client.StrongDocClient, key *sscrypto.StrongSaltKey, docID string, docVer uint64) didxcommon.DocOffsetIdx {
	doi, err := docidx.OpenDocOffsetIdx(sdc, docID, docVer, key)
	assert.NilError(t, err)
	return doi
}

func openDocTermIndex(t *testing.T, sdc client.StrongDocClient, key *sscrypto.StrongSaltKey, docID string, docVer uint64) docidx.DocTermIdx {
	dti, err := docidx.OpenDocTermIdx(sdc, docID, docVer, key)
	assert.NilError(t, err)
	return dti
}
