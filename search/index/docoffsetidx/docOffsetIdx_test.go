package docoffsetidx

import (
	"fmt"
	"io"
	"testing"
	
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestDocOffsetIdx(t *testing.T) {
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	fmt.Println("HELLO")

	_ = key
	_ = sourceFilePath

	// // Create file with encrypted content
	// idxFileName := "/tmp/docoffsetidx_test"
	// idxFile, err := os.Create(idxFileName)
	// assert.NilError(t, err)
	// defer os.Remove(idxFileName)
	// defer idxFile.Close()

	// doi, err := CreateDocOffsetIdx("docID100", 100, key, idxFile, 0)
	// assert.NilError(t, err)

	// tokenizer, err := utils.OpenFileTokenizer(sourceFilePath)
	// assert.NilError(t, err)

	// for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
	// 	adderr := doi.AddTermOffset(token, uint64(pos.Offset))
	// 	assert.NilError(t, adderr)
	// }

	// tokenizer.Close()
	// doi.Close()
	// idxFile.Close()

	// fmt.Println("-----------------------")

	// idxFile, err = os.OpenFile(idxFileName, os.O_RDWR, 0755)
	// assert.NilError(t, err)
	// defer idxFile.Close()

	// doiVersion, err := OpenDocOffsetIdx(key, idxFile, 0)
	// assert.NilError(t, err)

	// switch doiVersion.GetDoiVersion() {
	// case DOI_V1:
	// 	testDocOffsetIdxV1(t, doiVersion, sourceFilePath)
	// default:
	// 	assert.Assert(t, false, "Unsupported DOI version %v", doiVersion.GetDoiVersion())
	// }

}

func testDocOffsetIdxV1(t *testing.T, doiVersion DoiVersion, sourceFilePath string) {
	tokenizer, err := utils.OpenFileTokenizer(sourceFilePath)
	assert.NilError(t, err)
	defer tokenizer.Close()

	doi, ok := doiVersion.(*DocOffsetIdxV1)
	assert.Assert(t, ok)
	defer doi.Close()

	block, err := doi.ReadNextBlock()
	assert.NilError(t, err)

	for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
		// If we can't find the term/loc in this block, we should be
		// able to find it in the next block
		if !findTermLocationV1(block, token, uint64(pos.Offset)) {
			block, err = doi.ReadNextBlock()
			assert.NilError(t, err)
			assert.Assert(t, findTermLocationV1(block, token, uint64(pos.Offset)))
		}
	}
}

func findTermLocationV1(block *DocOffsetIdxBlkV1, term string, loc uint64) bool {
	offsetList := block.TermLoc[term]
	if len(offsetList) == 0 {
		return false
	}

	if utils.BinarySearchU64(offsetList, loc) == -1 {
		return false
	}

	return true
}
