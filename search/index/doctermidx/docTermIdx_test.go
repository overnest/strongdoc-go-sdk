package doctermidx

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docoffsetidx"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"time"

	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
	"os"
	"sort"
	"testing"

	"gotest.tools/assert"
)

// run all tests $go test -run TestTermIdx -timeout=20m
// run single test $go test -run TestXX -timeout=15m

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//
//	data source	==FileTokenizer==> tokenized data ==doi Generator==> offset index ==dti Generator==> term index ==> search index
//
//  step1: tokenize data using FileTokenizer
//	step2: generate Document Offset Index(doi) from tokenized data
//	step3: generate Document Term Index(dti) from tokenized data or doi
//  step4: generate Org/User Search Index(si) from  doi and dti(optional)
//
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var testLocal = false

// test create term index block
func TestTermIdxBlockV1(t *testing.T) {
	sourceFilePath, err := utils.FetchFileLoc("./testDocuments/enwik8.txt.gz")
	assert.NilError(t, err)

	outputFileName := "/tmp/TestTermIdxBlockV1.txt"
	outfile, err := os.Create(outputFileName)
	assert.NilError(t, err)
	defer outfile.Close()
	defer os.Remove(outputFileName)

	var prevHighTerm = ""

	sourceFile, err := os.Open(sourceFilePath)
	assert.NilError(t, err)
	defer sourceFile.Close()

	for true {
		sourceFile.Seek(0, utils.SeekSet)

		dtib := CreateDocTermIdxBlkV1(prevHighTerm, DTI_BLOCK_SIZE_MAX)

		tokenizer, err := utils.OpenFileTokenizer(sourceFile)
		assert.NilError(t, err)

		for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
			dtib.AddTerm(token)
			_ = pos
		}

		s, err := dtib.Serialize()
		assert.NilError(t, err)

		fmt.Println(dtib.lowTerm, dtib.highTerm, len(dtib.Terms), dtib.IsFull())
		fmt.Println(len(s), dtib.predictedJSONSize, DTI_BLOCK_SIZE_MAX)

		for _, t := range dtib.Terms {
			outfile.WriteString(t)
			outfile.WriteString("\n")
		}

		if !dtib.IsFull() {
			break
		}

		prevHighTerm = dtib.highTerm
		assert.NilError(t, err)
	}
}

// test create term source
func TestTermIdxSourcesV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "docID100"
	docVer := uint64(100)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilepath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	// ================================ Generate doc term index ================================
	// Open source file
	sourceFile, err := utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)

	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Open output writer
	outputWriter, err := testUtils.OpenOffsetIdxWriter(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)

	CreateTestDocOffsetIndex(t, key, docID, docVer, sourceFile, outputWriter)

	// Close source file
	err = sourceFile.Close()
	assert.NilError(t, err)

	err = outputWriter.Close()
	assert.NilError(t, err)

	time.Sleep(100 * time.Second)

	// ================================ Open doc offset index ================================
	doiReader, err := testUtils.OpenOffsetIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)
	defer doiReader.Close()

	doiv, err := docoffsetidx.OpenDocOffsetIdx(key, doiReader, 0)
	assert.NilError(t, err)

	// Create DOI source
	sourceDoi, err := OpenDocTermSourceDocOffsetV1(doiv)
	assert.NilError(t, err)
	doiTerms := gatherTermsFromSource(t, sourceDoi)

	// Create Text File source
	sourceFile, err = utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)
	sourceTxt, err := OpenDocTermSourceTextFileV1(sourceFile)
	assert.NilError(t, err)
	defer sourceFile.Close()
	txtTerms := gatherTermsFromSource(t, sourceTxt)
	assert.DeepEqual(t, doiTerms, txtTerms)

	// Reset Text File Source
	err = sourceTxt.Reset()
	assert.NilError(t, err)
	txtTermsNew := gatherTermsFromSource(t, sourceTxt)
	assert.DeepEqual(t, txtTerms, txtTermsNew)

	// Reset DOI Source
	err = sourceDoi.Reset()
	assert.NilError(t, err)
	doiTermsNew := gatherTermsFromSource(t, sourceDoi)
	assert.DeepEqual(t, doiTerms, doiTermsNew)

	// ================================ Delete doc offset index ================================
	testUtils.RemoveOffsetIndexFile(testLocal, testClient, docID, docVer)
}

// test create term index
func TestTermIdxV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "DocID1"
	docVer := uint64(1)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilepath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	// ================================ Generate doc term index ================================
	// Open source file
	sourceFile, err := os.Open(sourceFilepath)
	assert.NilError(t, err)

	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	dtiWriter, err := testUtils.OpenTermIdxWriter(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)

	terms := testCreateTermIndexV1(t, key, sourceFile, dtiWriter, docID, docVer)

	err = dtiWriter.Close()
	assert.NilError(t, err)

	err = sourceFile.Close()
	assert.NilError(t, err)

	time.Sleep(30 * time.Second)

	// ================================ Open doc term index ================================
	// Open the previously written document term index
	dtiReader, err := testUtils.OpenTermIdxReader(testLocal, testClient, docID, docVer)
	assert.NilError(t, err)

	size, err := testUtils.GetFileSize(testLocal, dtiReader)
	assert.NilError(t, err)

	dtiv, err := OpenDocTermIdx(key, dtiReader, 0, size)
	assert.NilError(t, err)
	assert.Equal(t, dtiv.GetDtiVersion(), uint32(1))

	dti, ok := dtiv.(*DocTermIdxV1)
	assert.Assert(t, ok)

	err = nil
	for err == nil {
		var blk *DocTermIdxBlkV1 = nil
		blk, err = dti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}
		if blk != nil {
			// fmt.Println(blk.Terms)
			for _, term := range blk.Terms {
				assert.Equal(t, term, terms[0])
				terms = terms[1:]
			}
		}
	}
	assert.Equal(t, len(terms), 0)

	// Search for terms in the document term index
	positiveMatches := []string{"against", "early", "working", "class", "radicals", "including",
		"the", "diggers", "of", "english", "revolution", "and", "sans", "culottes", "french",
		"whilst", "term", "is", "still", "used", "in", "a", "pejorative"}
	for _, term := range positiveMatches {
		found, err := dti.FindTerm(term)
		assert.NilError(t, err)
		assert.Assert(t, found, "term %v should have been found", term)
	}

	negativeMatches := []string{"againstagainst", "earlyearly", "workingworking", "classclass",
		"radicalsradicals", "includingincluding", "thethe", "diggersdiggers", "ofof",
		"englishenglish", "revolutionrevolution", "andand", "sanssans", "culottesculottes",
		"frenchfrench", "whilstwhilst", "termterm", "stillstill", "usedused", "inin",
		"pejorativepejorative"}
	for _, term := range negativeMatches {
		found, err := dti.FindTerm(term)
		assert.NilError(t, err)
		assert.Assert(t, !found, "term %v should not have been found", term)
	}

	err = dti.Close()
	assert.NilError(t, err)
	err = dtiReader.Close()
	assert.NilError(t, err)

	// ================================ Delete doc offset, term index ================================
	testUtils.RemoveTermIndexFile(testLocal, testClient, docID, docVer)
}

func testCreateTermIndexV1(t *testing.T, key *sscrypto.StrongSaltKey,
	sourcefile, outfile interface{}, docID string, docVer uint64) (terms []string) {

	source, err := OpenDocTermSourceTextFileV1(sourcefile)
	assert.NilError(t, err)

	//
	// Create a document term index
	//
	dti, err := CreateDocTermIdxV1(docID, docVer, key, source, outfile, 0)
	assert.NilError(t, err)

	terms = make([]string, 0, 1000)

	err = nil
	for err == nil {
		var blk *DocTermIdxBlkV1 = nil
		blk, err = dti.WriteNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		terms = append(terms, blk.Terms...)
	}

	err = dti.Close()
	assert.NilError(t, err)

	return
}

// test term index diff
func TestTermIdxDiffV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "DocID1"
	docVer1 := uint64(1)
	docVer2 := uint64(2)

	sourcefilepath1, err := utils.FetchFileLoc("./testDocuments/enwik8.uniq.txt.gz")
	assert.NilError(t, err)
	sourcefilepath2, err := utils.FetchFileLoc("./testDocuments/enwik8.uniq.chg.txt.gz")
	assert.NilError(t, err)

	// ================================ Generate doc term index ================================
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Create a document term index
	source1, err := utils.OpenLocalFile(sourcefilepath1)
	assert.NilError(t, err)

	source2, err := utils.OpenLocalFile(sourcefilepath2)

	assert.NilError(t, err)

	output1, err := testUtils.OpenTermIdxWriter(testLocal, testClient, docID, docVer1)
	assert.NilError(t, err)

	output2, err := testUtils.OpenTermIdxWriter(testLocal, testClient, docID, docVer2)
	assert.NilError(t, err)

	terms1 := testCreateTermIndexV1(t, key, source1, output1, docID, docVer1)
	terms2 := testCreateTermIndexV1(t, key, source2, output2, docID, docVer2)

	err = source1.Close()
	assert.NilError(t, err)

	err = source2.Close()
	assert.NilError(t, err)

	err = output1.Close()
	assert.NilError(t, err)

	err = output2.Close()
	assert.NilError(t, err)

	termap1 := make(map[string]bool)
	termap2 := make(map[string]bool)
	addedList := make([]string, 0, 1000)
	deletedList := make([]string, 0, 1000)

	for _, term := range terms1 {
		termap1[term] = true
	}
	for _, term := range terms2 {
		termap2[term] = true
	}

	for term := range termap1 {
		if !termap2[term] {
			deletedList = append(deletedList, term)
		}
	}

	for term := range termap2 {
		if !termap1[term] {
			addedList = append(addedList, term)
		}
	}

	sort.Strings(addedList)
	sort.Strings(deletedList)

	time.Sleep(20 * time.Second)

	// Open the previously written document term index
	outfile1, err := testUtils.OpenTermIdxReader(testLocal, testClient, docID, docVer1)
	assert.NilError(t, err)
	defer outfile1.Close()
	size1, err := testUtils.GetFileSize(testLocal, outfile1)
	assert.NilError(t, err)

	dtiv1, err := OpenDocTermIdx(key, outfile1, 0, size1)
	assert.NilError(t, err)
	assert.Equal(t, dtiv1.GetDtiVersion(), uint32(1))

	outfile2, err := testUtils.OpenTermIdxReader(testLocal, testClient, docID, docVer2)
	assert.NilError(t, err)
	defer outfile2.Close()
	size2, err := testUtils.GetFileSize(testLocal, outfile2)
	assert.NilError(t, err)

	dtiv2, err := OpenDocTermIdx(key, outfile2, 0, size2)
	assert.NilError(t, err)
	assert.Equal(t, dtiv2.GetDtiVersion(), uint32(1))

	added, deleted, err := DiffDocTermIdx(dtiv1, dtiv2)
	assert.NilError(t, err)

	assert.DeepEqual(t, addedList, added)
	assert.DeepEqual(t, deletedList, deleted)

	// ================================ Delete doc offset, term index ================================
	testUtils.RemoveTermIndexFile(testLocal, testClient, docID, docVer1)
	testUtils.RemoveTermIndexFile(testLocal, testClient, docID, docVer2)
}

func CreateTestDocOffsetIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
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

func gatherTermsFromSource(t *testing.T, source DocTermSourceV1) []string {
	terms := make([]string, 0, 10000000)
	for true {
		term, _, err := source.GetNextTerm()
		if term != "" {
			terms = append(terms, term)
		}
		if err != nil {
			assert.Equal(t, err, io.EOF)
			break
		}
	}
	sort.Strings(terms)
	return terms
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
