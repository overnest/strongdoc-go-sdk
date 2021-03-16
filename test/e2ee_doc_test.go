package test

import (
	"bytes"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"io/ioutil"
	"path"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"

	// cryptoKey "github.com/overnest/strongsalt-crypto-go"
	// cryptoKdf "github.com/overnest/strongsalt-crypto-go/kdf"
	// "path"
	"os"
	"testing"

	// "io/ioutil"
	"gotest.tools/assert"
)

/*func initTest(t *testing.T) string {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	orgs, orgUsers := initData(1, 1)
	orgData := orgs[0]
	userData := orgUsers[0][0]
	err = RegisterOrgAndAdmin(sdc, orgData, userData)
	assert.NilError(t, err)
	// defer HardRemoveOrgs([]string{orgData.OrgID})

	// login
	adminToken, err := sdc.Login(userData.UserID, userData.Password, orgData.OrgID, userData.PasswordKeyPwd)
	assert.NilError(t, err)
	return adminToken
}*/

// func TestUploadDownloadE2EE {

// }

func testE2EEUploadDownload(t *testing.T, sdc client.StrongDocClient, uploader, downloader *testUtils.TestUser, filename string) {
	txtBytes, err := ioutil.ReadFile(filename)
	assert.NilError(t, err)

	//fmt.Printf("file len: %v\n", len(txtBytes))

	file, err := os.Open(filename)
	assert.NilError(t, err)
	defer file.Close()

	err = api.Login(sdc, uploader.UserID, uploader.Password, uploader.OrgID)
	assert.NilError(t, err)
	defer api.Logout(sdc)

	uploadDocID, err := api.E2EEUploadDocument(sdc, path.Base(filename), file)
	assert.NilError(t, err)
	//fmt.Println(uploadDocID)

	if uploader != downloader {
		_, err = api.Logout(sdc)
		assert.NilError(t, err)

		err := api.Login(sdc, downloader.UserID, downloader.Password, downloader.OrgID)
		assert.NilError(t, err)
	}

	docs, err := api.ListDocuments(sdc)
	assert.NilError(t, err)
	assert.Equal(t, len(docs), 1)

	downReader, err := api.DownloadDocumentStream(sdc, uploadDocID)
	assert.NilError(t, err)
	plaintext, err := ioutil.ReadAll(downReader)
	assert.NilError(t, err)
	//fmt.Printf("plaintext len: %v\n", len(plaintext))
	assert.Assert(t, bytes.Equal(txtBytes, plaintext), "Plaintext doesn't match")

	err = api.RemoveDocument(sdc, uploadDocID)
	assert.NilError(t, err)

	docs, err = api.ListDocuments(sdc)
	assert.NilError(t, err)
	assert.Equal(t, len(docs), 0)
}

func testE2EEAdminDownload(t *testing.T, sdc client.StrongDocClient, docID string) {

}

func TestE2EEUploadDownload(t *testing.T) {
	sdc, orgs, registeredOrgUsers := testUtils.PrevTest(t, 1, 2)
	testUtils.DoRegistration(t, sdc, orgs, registeredOrgUsers)

	t.Run("test e2ee upload download", func(t *testing.T) {
		admin := registeredOrgUsers[0][0]
		notAdmin := registeredOrgUsers[0][1]

		testE2EEUploadDownload(t, sdc, admin, admin, TestDoc1)
		testE2EEUploadDownload(t, sdc, notAdmin, admin, TestDoc1)
		//testE2EEUploadDownload(t, sdc, notAdmin, admin, "../testDocuments/smallpicture.bmp")
	})

}
