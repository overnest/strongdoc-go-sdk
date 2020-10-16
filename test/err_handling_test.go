package test

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"gotest.tools/assert"
	"io"
	"os"
	"path"
	"testing"
)

func testStreamWithWrongDocId(t *testing.T) {
	// read and upload
	file, err := os.Open(TestDoc1)
	assert.NilError(t, err)
	defer file.Close()
	docID, err := api.UploadDocumentStream(path.Base(TestDoc1), file)
	assert.NilError(t, err)
	// download and write
	downloadFile, err := os.Create("download.txt")
	assert.NilError(t, err)
	defer downloadFile.Close()
	downloadStream, err := api.DownloadDocumentStream(docID+"wrongID")
	assert.NilError(t, err) // error here
	_, err = io.Copy(downloadFile, downloadStream)
	assert.ErrorContains(t, err, "Can not find document") // not here
}


func TestStreamErr(t *testing.T){
	_, registeredOrgUsers, orgids, err := testSetup(1, 1)
	assert.NilError(t, err)
	defer testTeardown(orgids)
	t.Run("test stream error handling", func(t *testing.T) {
		admin := registeredOrgUsers[0][0]
		// admin login
		_, err := api.Login(admin.UserID, admin.Password, admin.OrgID, admin.PasswordKeyPwd)
		assert.NilError(t, err)
		defer api.Logout()
		testStreamWithWrongDocId(t)
	})

}