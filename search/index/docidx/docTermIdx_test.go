package docidx

import (
	"io"
	"os"
	"sort"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestTermIdxBlockV1(t *testing.T) {
	sourceFilePath, err := utils.FetchFileLoc("./testDocuments/enwik8.txt.gz")
	assert.NilError(t, err)

	outputFileName := "/tmp/TestTermIdxBlockV1.txt"
	outfile, err := os.Create(outputFileName)
	assert.NilError(t, err)
	defer outfile.Close()
	defer os.Remove(outputFileName)

	var prevHighTerm string = ""

	for true {
		dtib := CreateDocTermIdxBlkV1(prevHighTerm, DTI_BLOCK_SIZE_MAX)

		tokenizer, err := utils.OpenFileTokenizer(sourceFilePath)
		assert.NilError(t, err)
		defer tokenizer.Close()

		for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
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

		prevHighTerm = dtib.highTerm
		assert.NilError(t, err)
	}
}

func TestTermIdxSourcesV1(t *testing.T) {
	sourceFile, err := utils.FetchFileLoc("./testDocuments/enwik8.txt.gz")
	assert.NilError(t, err)

	doiFileName := "/tmp/TestDocOffsetIdx"

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Create Document Offset Index
	createTestDocOffsetIndex(t, key, "DocID", 100, sourceFile, doiFileName)
	defer os.Remove(doiFileName)

	// Open Document Offset Index
	doiFile, err := os.Open(doiFileName)
	assert.NilError(t, err)
	doiv, err := OpenDocOffsetIdx(key, doiFile, 0)
	assert.NilError(t, err)

	// Create DOI source
	sourceDoi, err := OpenDocTermSourceDocOffsetV1(doiv)
	assert.NilError(t, err)
	defer sourceDoi.Close()
	doiTerms := gatherTermsFromSource(t, sourceDoi)

	// Create Text File source
	sourceTxt, err := OpenDocTermSourceTextFileV1(sourceFile)
	assert.NilError(t, err)
	defer sourceTxt.Close()
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

func TestTermIdxV1(t *testing.T) {
	docID := "DocID1"
	docVer := uint64(1)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	outputFileName := "/tmp/TestTermIdxV1.txt"

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	//
	// Create a document term index
	//
	terms := createTermIndexV1(t, key, sourceFilePath, outputFileName, docID, docVer)
	defer os.Remove(outputFileName)

	//
	// Open the previously written document term index
	//
	outfile, err := os.Open(outputFileName)
	assert.NilError(t, err)
	stat, err := outfile.Stat()
	assert.NilError(t, err)

	dtiv, err := OpenDocTermIdx(key, outfile, 0, uint64(stat.Size()))
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

	//
	// Search for terms in the document term index
	//
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
	err = outfile.Close()
	assert.NilError(t, err)
}

func createTermIndexV1(t *testing.T, key *sscrypto.StrongSaltKey,
	sourcefile, outputFile string, docID string, docVer uint64) (terms []string) {

	outfile, err := os.Create(outputFile)
	assert.NilError(t, err)

	source, err := OpenDocTermSourceTextFileV1(sourcefile)
	assert.NilError(t, err)
	defer source.Close()

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
	err = outfile.Close()
	assert.NilError(t, err)

	return
}

func TestTermIdxDiffV1(t *testing.T) {
	docID := "DocIDsomething"
	source1, err := utils.FetchFileLoc("./testDocuments/enwik8.uniq.txt.gz")
	assert.NilError(t, err)
	source2, err := utils.FetchFileLoc("./testDocuments/enwik8.uniq.chg.txt.gz")
	assert.NilError(t, err)

	dtiFileName1 := "/tmp/TestTermIdxV1_1.txt"
	defer os.Remove(dtiFileName1)
	dtiFileName2 := "/tmp/TestTermIdxV1_2.txt"
	defer os.Remove(dtiFileName2)

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	//
	// Create a document term index
	//
	terms1 := createTermIndexV1(t, key, source1, dtiFileName1, docID, 1)
	terms2 := createTermIndexV1(t, key, source2, dtiFileName2, docID, 2)
	addTermList, remTermList := diffTermLists(terms1, terms2)

	//
	// Open the previously written document term index
	//
	dtiFile1, dtiVer1 := openDocumentTermIndex(t, key, dtiFileName1, DTI_V1)
	defer dtiFile1.Close()
	dti1 := dtiVer1.(*DocTermIdxV1)

	dtiFile2, dtiVer2 := openDocumentTermIndex(t, key, dtiFileName2, DTI_V1)
	defer dtiFile2.Close()
	dti2 := dtiVer2.(*DocTermIdxV1)

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

func openDocumentTermIndex(t *testing.T, key *sscrypto.StrongSaltKey, filename string,
	dtiVersion uint32) (dtiFile *os.File, dtiVer DocTermIdx) {

	var err error
	dtiFile, err = os.Open(filename)
	assert.NilError(t, err)
	stat, err := dtiFile.Stat()
	assert.NilError(t, err)

	dtiVer, err = OpenDocTermIdx(key, dtiFile, 0, uint64(stat.Size()))
	assert.NilError(t, err)
	assert.Equal(t, dtiVer.GetDtiVersion(), dtiVersion)

	return
}
