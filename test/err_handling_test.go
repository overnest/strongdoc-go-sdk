package test

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	assert "github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func testStreamWithWrongDocId(t *testing.T) {
	file, err := os.Open(TestDoc1)
	assert.NoError(t, err)
	defer file.Close()
	docID, err := api.UploadDocumentStream(path.Base(TestDoc1), file)
	assert.NoError(t, err)
	fmt.Println(docID)
}

// test with command line $go test -run TestStreamErr -dev
func TestStreamErr(t *testing.T){
	// login in as org1 admin
	_, err := api.Login(ORG1_Admin_ID, ORG1_Admin_Pwd, ORG1)
	assert.NoError(t, err)
	testStreamWithWrongDocId(t)
	// log out
	_, err = api.Logout()
	assert.NoError(t, err)
}