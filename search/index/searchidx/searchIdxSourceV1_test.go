package searchidx

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"io"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/search/index/docoffsetidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/doctermidx"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

var testLocal = true

func TestSearchSourceTextFileV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID1 := "DOC1"
	docVer1 := uint64(1)
	docID2 := "DOC2"
	docVer2 := uint64(2)

	dataSource1, err := utils.FetchFileLoc("./testDocuments/doc1.txt.gz")
	assert.NilError(t, err)
	dataSource2, err := utils.FetchFileLoc("./testDocuments/doc1.chg.txt.gz")
	assert.NilError(t, err)
	dataSource3, err := utils.FetchFileLoc("./testDocuments/doc2.txt.gz")
	assert.NilError(t, err)
	dataSource4, err := utils.FetchFileLoc("./testDocuments/doc2.chg.txt.gz")
	assert.NilError(t, err)

	// ================================ Generate doc offset index & term index ================================
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	createDocumentIndexes(t, testLocal, testClient, key, docID1, docVer1, dataSource1)
	createDocumentIndexes(t, testLocal, testClient, key, docID1, docVer2, dataSource2)
	createDocumentIndexes(t, testLocal, testClient, key, docID2, docVer1, dataSource3)
	createDocumentIndexes(t, testLocal, testClient, key, docID2, docVer2, dataSource4)

	// Create new doc
	// open doc offset index, create doi
	doiFileOld1, doiOld1 := openDocOffsetIndex(t, testLocal, testClient, key, docID1, docVer1)
	dtiFileOld1, dtiOld1 := openDocTermIndex(t, testLocal, testClient, key, docID1, docVer1)
	searchSource1, err := SearchSourceCreateDoc(doiOld1, dtiOld1)
	assert.NilError(t, err)
	assert.Assert(t, len(searchSource1.GetAddTerms()) > 0)
	assert.Assert(t, len(searchSource1.GetDelTerms()) == 0)
	assert.Equal(t, searchSource1.GetDocID(), docID1)
	assert.Equal(t, searchSource1.GetDocVer(), docVer1)
	closeFiles(t, searchSource1, dtiOld1, dtiFileOld1, doiOld1, doiFileOld1)

	// Update existing doc
	doiFileNew1, doiNew1 := openDocOffsetIndex(t, testLocal, testClient, key, docID1, docVer2)
	dtiFileNew1, dtiNew1 := openDocTermIndex(t, testLocal, testClient, key, docID1, docVer2)
	dtiFileOld1, dtiOld1 = openDocTermIndex(t, testLocal, testClient, key, docID1, docVer1)
	searchSource2, err := SearchSourceUpdateDoc(doiNew1, dtiOld1, dtiNew1)
	assert.NilError(t, err)
	assert.Assert(t, len(searchSource2.GetAddTerms()) > 0)
	assert.Assert(t, len(searchSource2.GetDelTerms()) > 0)
	assert.Equal(t, searchSource2.GetDocID(), docID1)
	assert.Equal(t, searchSource2.GetDocVer(), docVer2)
	closeFiles(t, searchSource2, dtiOld1, dtiFileOld1, dtiNew1, dtiFileNew1, doiNew1, doiFileNew1)

	// Delete existing doc
	doiFileNew1, doiNew1 = openDocOffsetIndex(t, testLocal, testClient, key, docID1, docVer2)
	dtiFileNew1, dtiNew1 = openDocTermIndex(t, testLocal, testClient, key, docID1, docVer2)
	source3, err := SearchSourceDeleteDoc(doiNew1, dtiNew1)
	assert.NilError(t, err)
	assert.Assert(t, len(source3.GetAddTerms()) == 0)
	assert.Assert(t, len(source3.GetDelTerms()) > 0)
	assert.Equal(t, source3.GetDocID(), docID1)
	assert.Equal(t, source3.GetDocVer(), docVer2)
	closeFiles(t, source3, dtiNew1, dtiFileNew1, doiNew1, doiFileNew1)

	testUtils.RemoveDocIndexes(testLocal, testClient, docID1, docVer1)
	testUtils.RemoveDocIndexes(testLocal, testClient, docID1, docVer2)
	testUtils.RemoveDocIndexes(testLocal, testClient, docID2, docVer1)
	testUtils.RemoveDocIndexes(testLocal, testClient, docID2, docVer2)

}

func createDocumentIndexes(t *testing.T,
	testLocal bool, testClient client.StrongDocClient,
	key *sscrypto.StrongSaltKey, docID string, docVer uint64, sourceFilepath string) {

	// ========================== generate doc offset index ==========================
	sourceFile, err := utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)
	doiWriter, err := testUtils.OpenOffsetIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)
	createDocOffsetIndex(t, key, docID, docVer, sourceFile, doiWriter)
	err = sourceFile.Close()
	assert.NilError(t, err)
	err = doiWriter.Close()
	assert.NilError(t, err)

	// ========================== generate doc term index ==========================
	// open doc offset index, create doi
	doiReader, err := testUtils.OpenOffsetIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)

	doi, err := docoffsetidx.OpenDocOffsetIdxV1(key, doiReader, 0)
	assert.NilError(t, err)

	defer doiReader.Close()
	defer doi.Close()

	// create dti, using doi
	dtiWriter, err := testUtils.OpenTermIdxWriter(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)
	createDocTermIndex(t, key, docID, docVer, doi, dtiWriter)
	err = dtiWriter.Close()
	assert.NilError(t, err)
	return
}

func createDocOffsetIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
	sourceReader, doiWriter interface{}) {
	doi, err := docoffsetidx.CreateDocOffsetIdx(docID, docVer, key, doiWriter, 0)
	assert.NilError(t, err)

	tokenizer, err := utils.OpenFileTokenizer(sourceReader)
	assert.NilError(t, err)

	for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
		adderr := doi.AddTermOffset(token, uint64(pos.Offset))
		assert.NilError(t, adderr)
	}

	doi.Close()
}

func openDocOffsetIndex(t *testing.T,
	testLocal bool, testClient client.StrongDocClient,
	key *sscrypto.StrongSaltKey, docID string, docVer uint64) (doiFile io.ReadCloser, doi *docoffsetidx.DocOffsetIdxV1) {
	var err error

	doiFile, err = testUtils.OpenOffsetIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)

	doi, err = docoffsetidx.OpenDocOffsetIdxV1(key, doiFile, 0)
	assert.NilError(t, err)

	return
}

func createDocTermIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
	doi docoffsetidx.DocOffsetIdx, dtiWriter interface{}) {
	source, err := doctermidx.OpenDocTermSourceDocOffsetV1(doi)
	assert.NilError(t, err)

	//
	// Create a document term index
	//
	dti, err := doctermidx.CreateDocTermIdxV1(docID, docVer, key, source, dtiWriter, 0)
	assert.NilError(t, err)

	err = nil
	for err == nil {
		_, err = dti.WriteNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}
	}

	err = dti.Close()
	assert.NilError(t, err)

	return
}

func openDocTermIndex(t *testing.T,
	testLocal bool, testClient client.StrongDocClient,
	key *sscrypto.StrongSaltKey, docID string, docVer uint64) (dtiFile io.ReadCloser, dti *doctermidx.DocTermIdxV1) {
	var err error

	dtiFile, err = testUtils.OpenTermIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)
	fileSize, err := testUtils.GetFileSize(testLocal, dtiFile)
	assert.NilError(t, err)

	dti, err = doctermidx.OpenDocTermIdxV1(key, dtiFile, 0, fileSize)
	assert.NilError(t, err)

	return
}

func closeFiles(t *testing.T, objs ...io.Closer) {
	for _, obj := range objs {
		err := obj.Close()
		assert.NilError(t, err)
	}
}

func prevTest(t *testing.T) client.StrongDocClient {
	if testLocal {
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
