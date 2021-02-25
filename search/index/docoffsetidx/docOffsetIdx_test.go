package docoffsetidx

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
	"testing"
	"time"

	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//														==doi Generator==> offset index
//													/ 									\
//		data source	==FileTokenizer==> tokenized data == 										==> search index
//													\									/
//														==dti Generator==> term index
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var testLocal = true

func TestDocOffsetIdx(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t, testLocal)

	docID := "docID100"
	docVer := uint64(100)
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilepath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	// ================================ Generate doc offset index ================================
	// Open source file
	sourceFile, err := utils.OpenLocalFile(sourceFilepath)
	assert.NilError(t, err)

	// Create encryption key
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	// Open output writer
	outputWriter, err := openOffsetIdxWriter(testLocal, sdc, docID, docVer)
	assert.NilError(t, err)

	CreateTestDocOffsetIndex(t, key, docID, docVer, sourceFile, outputWriter)

	// Close source file
	err = sourceFile.Close()
	assert.NilError(t, err)

	err = outputWriter.Close()
	assert.NilError(t, err)

	time.Sleep(5 * time.Second)

	// ================================ Open doc offset index ================================
	doiReader, err := openOffsetIdxReader(testLocal, sdc, docID, docVer)
	assert.NilError(t, err)
	defer doiReader.Close()

	doiVersion, err := OpenDocOffsetIdx(key, doiReader, 0)
	assert.NilError(t, err)

	switch doiVersion.GetDoiVersion() {
	case DOI_V1:
		sourceFile, err = utils.OpenLocalFile(sourceFilepath)
		defer sourceFile.Close()
		assert.NilError(t, err)
		testDocOffsetIdxV1(t, doiVersion, sourceFile)
	default:
		assert.Assert(t, false, "Unsupported DOI version %v", doiVersion.GetDoiVersion())
	}
}

func CreateTestDocOffsetIndex(t *testing.T, key *sscrypto.StrongSaltKey, docID string, docVer uint64,
	sourceReader, doiWriter interface{}) {
	doi, err := CreateDocOffsetIdx(docID, docVer, key, doiWriter, 0)
	assert.NilError(t, err)

	tokenizer, err := utils.OpenFileTokenizer(sourceReader)
	assert.NilError(t, err)

	for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
		adderr := doi.AddTermOffset(token, uint64(pos.Offset))
		assert.NilError(t, adderr)
	}

	doi.Close()
}

func testDocOffsetIdxV1(t *testing.T, doiVersion DocOffsetIdx, sourceFile interface{}) {
	tokenizer, err := utils.OpenFileTokenizer(sourceFile)
	assert.NilError(t, err)

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

func prevTest(t *testing.T, testLocal bool) *client.StrongDocClient {
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
	return &sdc
}

func openOffsetIdxWriter(testLocal bool, sdc *client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if testLocal {
		outputWriter, err = utils.CreateLocalFile(fmt.Sprintf("/tmp/%v_%v", docID, docVer))
	} else {
		outputWriter, err = api.NewDocIndexWriter(*sdc, docID, docVer, utils.OffsetIndex)
	}
	return
}

func openOffsetIdxReader(testLocal bool, sdc *client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if testLocal {
		reader, err = utils.OpenLocalFile(fmt.Sprintf("/tmp/%v_%v", docID, docVer))
	} else {
		reader, err = api.NewDocIndexReader(*sdc, docID, docVer, utils.OffsetIndex)
	}
	return
}
