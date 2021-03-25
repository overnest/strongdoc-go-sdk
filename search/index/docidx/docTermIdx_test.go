package docidx

import (
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"

	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"io"
	"os"
	"sort"
	"testing"
	"time"
)

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

		dtib := docidxv1.CreateDocTermIdxBlkV1(prevHighTerm, common.DTI_BLOCK_SIZE_MAX)

		tokenizer, err := utils.OpenFileTokenizer(sourceFile)
		assert.NilError(t, err)

		for token, pos, _, err := tokenizer.NextToken(); err != io.EOF; token, pos, _, err = tokenizer.NextToken() {
			dtib.AddTerm(token)
			_ = pos
		}

		_, err = dtib.Serialize()
		assert.NilError(t, err)

		// fmt.Println(dtib.lowTerm, dtib.highTerm, len(dtib.Terms), dtib.IsFull())
		// fmt.Println(len(s), dtib.predictedJSONSize, DTI_BLOCK_SIZE_MAX)

		for _, t := range dtib.Terms {
			outfile.WriteString(t)
			outfile.WriteString("\n")
		}

		if !dtib.IsFull() {
			break
		}

		prevHighTerm = dtib.GetHighTerm()
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

	defer common.RemoveDocIndexes(testClient, docID)
	// ================================ Generate doc term index ================================
	// Open source file
	sourceFile, err := utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)

	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Generate doc offset index, writes output
	err = CreateAndSaveDocOffsetIdx(testClient, docID, docVer, key, sourceFile)
	assert.NilError(t, err)

	// Close source file
	err = sourceFile.Close()
	assert.NilError(t, err)

	time.Sleep(100 * time.Second)

	// ================================ Open doc offset index ================================
	doiv, err := OpenDocOffsetIdx(testClient, docID, docVer, key)
	assert.NilError(t, err)
	defer doiv.Close()

	// Create DOI source
	sourceDoi, err := docidxv1.OpenDocTermSourceDocOffsetV1(doiv)
	assert.NilError(t, err)
	doiTerms := gatherTermsFromSource(t, sourceDoi)

	// Create Text File source
	sourceFile, err = utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)
	sourceTxt, err := docidxv1.OpenDocTermSourceTextFileV1(sourceFile)
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
}

func gatherTermsFromSource(t *testing.T, source docidxv1.DocTermSourceV1) []string {
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

// test create term index
func TestTermIdxV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "DocID1"
	docVer := uint64(1)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilepath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	defer common.RemoveDocIndexes(testClient, docID)

	// ================================ Generate doc term index ================================
	// Open source file
	sourceFile, err := os.Open(sourceFilepath)
	assert.NilError(t, err)

	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	terms, err := createTermIndexAndGetTerms(testClient, docID, docVer, key, sourceFile)
	assert.NilError(t, err)

	err = sourceFile.Close()
	assert.NilError(t, err)

	time.Sleep(30 * time.Second)

	// ================================ Open doc term index ================================
	// Open the previously written document term index
	dtiv, err := OpenDocTermIdx(testClient, docID, docVer, key)
	assert.NilError(t, err)
	assert.Equal(t, dtiv.GetDtiVersion(), uint32(1))

	dti, ok := dtiv.(*docidxv1.DocTermIdxV1)
	assert.Assert(t, ok)

	err = nil
	for err == nil {
		var blk *docidxv1.DocTermIdxBlkV1 = nil
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
}

func createTermIndexAndGetTerms(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourcefile utils.Storage) ([]string, error) {
	source, err := docidxv1.OpenDocTermSourceTextFileV1(sourcefile)
	if err != nil {
		return nil, err
	}

	// Create a document term index
	dti, err := docidxv1.CreateDocTermIdxV1(sdc, docID, docVer, key, source, 0)
	if err != nil {
		return nil, err
	}

	var terms []string

	for err == nil {
		var blk *docidxv1.DocTermIdxBlkV1
		blk, err = dti.WriteNextBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
		terms = append(terms, blk.Terms...)

	}

	err = dti.Close()
	if err != nil {
		return nil, err
	}
	return terms, nil
}

// test term index diff
func TestTermIdxDiffV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "DocIDsomething"
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

	terms1, err := createTermIndexAndGetTerms(testClient, docID, docVer1, key, source1)
	assert.NilError(t, err)
	terms2, err := createTermIndexAndGetTerms(testClient, docID, docVer2, key, source2)
	assert.NilError(t, err)
	addTermList, remTermList := diffTermLists(terms1, terms2)

	err = source1.Close()
	assert.NilError(t, err)

	err = source2.Close()
	assert.NilError(t, err)

	time.Sleep(20 * time.Second)

	// ================================ Open doc term index ================================
	// Open the previously written document term index
	dtiVer1, err := OpenDocTermIdx(testClient, docID, docVer1, key)
	assert.NilError(t, err)
	dti1 := dtiVer1.(*docidxv1.DocTermIdxV1)

	dtiVer2, err := OpenDocTermIdx(testClient, docID, docVer2, key)
	assert.NilError(t, err)
	dti2 := dtiVer2.(*docidxv1.DocTermIdxV1)

	//
	// Diff nil, dti1
	//
	added, deleted, err := DiffDocTermIdx(nil, dtiVer1)
	assert.NilError(t, err)
	assert.DeepEqual(t, added, terms1)
	assert.DeepEqual(t, deleted, []string{})

	err = dti1.Reset()
	assert.NilError(t, err)

	//
	// Diff dti1, dti2
	//
	added, deleted, err = DiffDocTermIdx(dtiVer1, dtiVer2)
	assert.NilError(t, err)
	assert.DeepEqual(t, added, addTermList)
	assert.DeepEqual(t, deleted, remTermList)

	err = dti1.Reset()
	assert.NilError(t, err)
	err = dti2.Reset()
	assert.NilError(t, err)

	//
	// Diff dti2, nil
	//
	added, deleted, err = DiffDocTermIdx(dtiVer2, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, added, []string{})
	assert.DeepEqual(t, deleted, terms2)

	dti1.Close()
	dti2.Close()
}

func diffTermLists(oldTermList, newTermList []string) (addTermList, delTermList []string) {
	oldTermMap := make(map[string]bool)
	newTermMap := make(map[string]bool)
	addTermList = make([]string, 0, 1000)
	delTermList = make([]string, 0, 1000)

	for _, term := range oldTermList {
		oldTermMap[term] = true
	}
	for _, term := range newTermList {
		newTermMap[term] = true
	}

	for term := range oldTermMap {
		if !newTermMap[term] {
			delTermList = append(delTermList, term)
		}
	}

	for term := range newTermMap {
		if !oldTermMap[term] {
			addTermList = append(addTermList, term)
		}
	}

	sort.Strings(addTermList)
	sort.Strings(delTermList)

	return
}
