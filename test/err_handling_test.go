package test

import (
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"os"
	"path"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"gotest.tools/assert"
)

func testStreamWithWrongDocId(t *testing.T, sdc client.StrongDocClient) {
	// read and upload
	file, err := os.Open(TestDoc1)
	assert.NilError(t, err)
	defer file.Close()
	docID, err := api.UploadDocumentStream(sdc, path.Base(TestDoc1), file)
	assert.NilError(t, err)
	// download and write
	downloadFile, err := os.Create("download.txt")
	assert.NilError(t, err)
	defer downloadFile.Close()
	downloadStream, err := api.DownloadDocumentStream(sdc, docID+"wrongID")
	assert.ErrorContains(t, err, "Cannot find document") // error here
	assert.Equal(t, nil, downloadStream)
	//_, err = io.Copy(downloadFile, downloadStream)
	//assert.NilError(t, err) // not here
}

func TestStreamErr(t *testing.T) {
	sdc, orgs, registeredOrgUsers := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, registeredOrgUsers)
	t.Run("test stream error handling", func(t *testing.T) {
		admin := registeredOrgUsers[0][0]
		// admin login
		err := api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
		assert.NilError(t, err)
		defer api.Logout(sdc)
		testStreamWithWrongDocId(t, sdc)
	})

}
