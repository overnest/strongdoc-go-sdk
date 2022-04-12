package query

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/utils"

	scom "github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"gotest.tools/assert"
)

func TestQuery(t *testing.T) {
	scom.EnableAllLocal()
	cred := OpenCredentials()
	defer Cleanup(cred)

	testDocs, err := docidx.InitTestDocumentIdx(10, false)
	assert.NilError(t, err)

	uploadDocs := make([]string, len(testDocs))
	for i, testDoc := range testDocs {
		uploadDocs[i] = testDoc.DocFilePath
	}

	docs, err := UploadDocument(cred, uploadDocs)
	assert.NilError(t, err)

	fmt.Println(utils.JsonPrint(docs))

	phrases := []string{
		"almost no restrictions",
		"almost no restrictions doesnotexist",
		"world more ridiculous than",
		"world more doesnotexist ridiculous than",
	}

	for _, phrase := range phrases {
		result, err := Search(cred, phrase)
		assert.NilError(t, err)

		fmt.Printf("\"%v\" hits(%v): %v\n", phrase, len(result), utils.JsonPrint(result))
	}
}
