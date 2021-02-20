package searchidx

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/search/index/docoffsetidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/doctermidx"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

func TestSearchSourceTextFileV1(t *testing.T) {
	docID1 := "DOC1"
	docVer1 := uint64(1)
	docID2 := "DOC2"
	docVer2 := uint64(2)

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	docFileNameOld1, docFileNameNew1, docFileNameOld2, docFileNameNew2 := createDocumentFileName(t)
	doiFileNameOld1, dtiFileNameOld1 := createDocumentIndexes(t, key, docID1, docVer1, docFileNameOld1)
	doiFileNameNew1, dtiFileNameNew1 := createDocumentIndexes(t, key, docID1, docVer2, docFileNameNew1)
	doiFileNameOld2, dtiFileNameOld2 := createDocumentIndexes(t, key, docID2, docVer1, docFileNameOld2)
	doiFileNameNew2, dtiFileNameNew2 := createDocumentIndexes(t, key, docID2, docVer2, docFileNameNew2)

	// Create new doc
	doiFileOld1, doiOld1 := openDocOffsetIndex(t, key, doiFileNameOld1)
	dtiFileOld1, dtiOld1 := openDocTermIndex(t, key, dtiFileNameOld1)
	source1, err := SearchSourceCreateDoc(doiOld1, dtiOld1)
	assert.NilError(t, err)
	assert.Assert(t, len(source1.GetAddTerms()) > 0)
	assert.Assert(t, len(source1.GetDelTerms()) == 0)
	assert.Equal(t, source1.GetDocID(), docID1)
	assert.Equal(t, source1.GetDocVer(), docVer1)
	closeFiles(t, source1, dtiOld1, dtiFileOld1, doiOld1, doiFileOld1)

	// Update existing doc
	doiFileNew1, doiNew1 := openDocOffsetIndex(t, key, doiFileNameNew1)
	dtiFileNew1, dtiNew1 := openDocTermIndex(t, key, dtiFileNameNew1)
	dtiFileOld1, dtiOld1 = openDocTermIndex(t, key, dtiFileNameOld1)
	source2, err := SearchSourceUpdateDoc(doiNew1, dtiOld1, dtiNew1)
	assert.NilError(t, err)
	assert.Assert(t, len(source2.GetAddTerms()) > 0)
	assert.Assert(t, len(source2.GetDelTerms()) > 0)
	assert.Equal(t, source2.GetDocID(), docID1)
	assert.Equal(t, source2.GetDocVer(), docVer2)
	closeFiles(t, source2, dtiOld1, dtiFileOld1, dtiNew1, dtiFileNew1, doiNew1, doiFileNew1)

	// Delete existing doc
	doiFileNew1, doiNew1 = openDocOffsetIndex(t, key, doiFileNameNew1)
	dtiFileNew1, dtiNew1 = openDocTermIndex(t, key, dtiFileNameNew1)
	source3, err := SearchSourceDeleteDoc(doiNew1, dtiNew1)
	assert.NilError(t, err)
	assert.Assert(t, len(source3.GetAddTerms()) == 0)
	assert.Assert(t, len(source3.GetDelTerms()) > 0)
	assert.Equal(t, source3.GetDocID(), docID1)
	assert.Equal(t, source3.GetDocVer(), docVer2)
	closeFiles(t, source3, dtiNew1, dtiFileNew1, doiNew1, doiFileNew1)

	removeFiles(t, doiFileNameOld1, dtiFileNameOld1, doiFileNameNew1,
		dtiFileNameNew1, doiFileNameOld2, dtiFileNameOld2, doiFileNameNew2,
		dtiFileNameNew2)
}

func createDocumentFileName(t *testing.T) (string, string, string, string) {
	docFileNameOld1, err := utils.FetchFileLoc("./testDocuments/doc1.txt.gz")
	assert.NilError(t, err)
	docFileNameNew1, err := utils.FetchFileLoc("./testDocuments/doc1.chg.txt.gz")
	assert.NilError(t, err)
	docFileNameOld2, err := utils.FetchFileLoc("./testDocuments/doc2.txt.gz")
	assert.NilError(t, err)
	docFileNameNew2, err := utils.FetchFileLoc("./testDocuments/doc2.chg.txt.gz")
	assert.NilError(t, err)
	return docFileNameOld1, docFileNameNew1, docFileNameOld2, docFileNameNew2
}

func createDocumentIndexes(t *testing.T, key *sscrypto.StrongSaltKey,
	docID string, docVer uint64, sourceFile string) (doiFileName, dtiFileName string) {

	doiFileName = fmt.Sprintf("/tmp/doi_%v_%v.idx", docID, docVer)
	dtiFileName = fmt.Sprintf("/tmp/dti_%v_%v.idx", docID, docVer)

	createDocOffsetIndex(t, key, docID, docVer, sourceFile, doiFileName)

	doiFile, doi := openDocOffsetIndex(t, key, doiFileName)
	defer doiFile.Close()
	defer doi.Close()

	createDocTermIndex(t, key, docID, docVer, doi, dtiFileName)
	return
}

func createDocOffsetIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
	sourceFile, outputFile string) {

	idxFile, err := os.Create(outputFile)
	assert.NilError(t, err)
	defer idxFile.Close()

	doi, err := docoffsetidx.CreateDocOffsetIdx(docID, docVer, key, idxFile, 0)
	assert.NilError(t, err)
	defer doi.Close()

	tokenizer, err := utils.OpenFileTokenizer(sourceFile)
	assert.NilError(t, err)
	defer tokenizer.Close()

	for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
		adderr := doi.AddTermOffset(token, uint64(pos.Offset))
		assert.NilError(t, adderr)
	}
}

func openDocOffsetIndex(t *testing.T, key *sscrypto.StrongSaltKey, fileName string) (doiFile *os.File, doi *docoffsetidx.DocOffsetIdxV1) {
	var err error

	doiFile, err = os.Open(fileName)
	assert.NilError(t, err)

	doi, err = docoffsetidx.OpenDocOffsetIdxV1(key, doiFile, 0)
	assert.NilError(t, err)

	return
}

func createDocTermIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
	doi docoffsetidx.DocOffsetIdx, outputFile string) {

	outfile, err := os.Create(outputFile)
	assert.NilError(t, err)

	source, err := doctermidx.OpenDocTermSourceDocOffsetV1(doi)
	assert.NilError(t, err)
	defer source.Close()

	//
	// Create a document term index
	//
	dti, err := doctermidx.CreateDocTermIdxV1(docID, docVer, key, source, outfile, 0)
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
	err = outfile.Close()
	assert.NilError(t, err)

	return
}

func openDocTermIndex(t *testing.T, key *sscrypto.StrongSaltKey, fileName string) (dtiFile *os.File, dti *doctermidx.DocTermIdxV1) {
	var err error

	dtiFile, err = os.Open(fileName)
	assert.NilError(t, err)
	info, err := os.Stat(fileName)
	assert.NilError(t, err)

	dti, err = doctermidx.OpenDocTermIdxV1(key, dtiFile, 0, uint64(info.Size()))
	assert.NilError(t, err)

	return
}

func closeFiles(t *testing.T, objs ...io.Closer) {
	for _, obj := range objs {
		err := obj.Close()
		assert.NilError(t, err)
	}
}

func removeFiles(t *testing.T, filenames ...string) {
	for _, filename := range filenames {
		os.Remove(filename)
	}
}
