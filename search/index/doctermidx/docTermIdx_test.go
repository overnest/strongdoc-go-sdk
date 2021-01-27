package doctermidx

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestTermIdxBlockV1(t *testing.T) {
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)

	outputFileName := "/tmp/TestTermIdxBlockV1.txt"
	outfile, err := os.Create(outputFileName)
	assert.NilError(t, err)
	defer outfile.Close()
	defer os.Remove(outputFileName)

	var prevHighTerm string = ""

	for true {
		dtib := CreateDockTermIdxBlkV1(prevHighTerm, DTI_BLOCK_SIZE_MAX)

		tokenizer, err := utils.OpenFileTokenizer(sourceFilePath)
		assert.NilError(t, err)
		defer tokenizer.Close()

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

func TestTermIdxV1(t *testing.T) {
	docID := "DocID1"
	docVer := uint64(1)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	outputFileName := "/tmp/TestTermIdxV1.txt"

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	outfile, err := os.Create(outputFileName)
	assert.NilError(t, err)

	source, err := OpenDocTermSourceTextFileV1(sourceFilePath)
	assert.NilError(t, err)

	//
	// Create a document term index
	//
	dti, err := CreateDocTermIdxV1(docID, docVer, key, source, outfile, 0)
	assert.NilError(t, err)

	terms := make([]string, 0)

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

	//
	// Open the previously written document term index
	//
	outfile, err = os.Open(outputFileName)
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
