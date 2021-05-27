package docidx

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"io"
	"testing"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//
//	data source	==FileTokenizer==> tokenized data ==doi Generator==> offset index ==dti Generator==> term index ==> search index
//
//  step1: tokenize data using FileTokenizer
//	step2: generate Document Offset Index(doi) from tokenized data
//	step3: generate Document Term Index(dti) from tokenized source data or doi(preferred)
//
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestDocOffsetIdx(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	docID := "docID100"
	docVer := uint64(100)
	defer common.RemoveDocIndexes(testClient, docID)

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

	// Generate doc offset index, writes output
	err = CreateAndSaveDocOffsetIdx(testClient, docID, docVer, key, sourceFile)
	assert.NilError(t, err)

	// Close source file
	err = sourceFile.Close()
	assert.NilError(t, err)

	// ================================ Open doc offset index ================================
	doiVersion, err := OpenDocOffsetIdx(testClient, docID, docVer, key)
	assert.NilError(t, err)
	defer doiVersion.Close()

	switch doiVersion.GetDoiVersion() {
	case common.DOI_V1:
		sourceFile, err = utils.OpenLocalFile(sourceFilepath)
		defer sourceFile.Close()
		assert.NilError(t, err)
		testDocOffsetIdxV1(t, doiVersion, sourceFile)
	default:
		assert.Assert(t, false, "Unsupported DOI version %v", doiVersion.GetDoiVersion())
	}
}

func testDocOffsetIdxV1(t *testing.T, doiVersion common.DocOffsetIdx, sourceFile utils.Source) {
	tokenizer, err := utils.OpenBleveTokenizer(sourceFile)
	assert.NilError(t, err)

	doi, ok := doiVersion.(*docidxv1.DocOffsetIdxV1)
	assert.Assert(t, ok)
	defer doi.Close()

	block, err := doi.ReadNextBlock()
	assert.NilError(t, err)

	for token, wordCounter, err := tokenizer.NextToken(); err != io.EOF; token, wordCounter, err = tokenizer.NextToken() {
		// If we can't find the term/loc in this block, we should be
		// able to find it in the next block
		if !findTermLocationV1(block, token, wordCounter) {
			block, err = doi.ReadNextBlock()
			assert.NilError(t, err)
			assert.Assert(t, findTermLocationV1(block, token, wordCounter))
		}
	}

	err = tokenizer.Close()
	assert.NilError(t, err)
}

func findTermLocationV1(block *docidxv1.DocOffsetIdxBlkV1, term string, loc uint64) bool {
	offsetList := block.TermLoc[term]
	if len(offsetList) == 0 {
		return false
	}

	if utils.BinarySearchU64(offsetList, loc) == -1 {
		return false
	}

	return true
}

func prevTest(t *testing.T) client.StrongDocClient {
	if utils.TestLocal {
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
