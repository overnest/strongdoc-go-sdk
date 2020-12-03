package test

import (
	"fmt"
	"io/ioutil"

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
	err = registerOrgAndAdmin(sdc, orgData, userData)
	assert.NilError(t, err)
	// defer HardRemoveOrgs([]string{orgData.OrgID})

	// login
	adminToken, err := sdc.Login(userData.UserID, userData.Password, orgData.OrgID, userData.PasswordKeyPwd)
	assert.NilError(t, err)
	return adminToken
}*/

// func TestUploadDownloadE2EE {

// }

func testE2EEUpload(t *testing.T, sdc client.StrongDocClient) {
	// txtBytes, err := ioutil.ReadFile(fileName)
	// assert.NilError(t, err)

	file, err := os.Open(TestDoc1)
	assert.NilError(t, err)
	defer file.Close()

	uploadDocID, err := api.E2EEUploadDocument(sdc, "TestDoc1", file)
	assert.NilError(t, err)
	fmt.Println(uploadDocID)

	// downBytes, err := api.E2EEDownloadDocument(uploadDocID)
	// assert.NilError(t, err)
	// assert.Equal(t, txtBytes, downBytes)

	docs, err := api.ListDocuments(sdc)
	assert.NilError(t, err)
	assert.Equal(t, len(docs), 1)

	downReader, err := api.DownloadDocumentStream(sdc, uploadDocID)
	assert.NilError(t, err)
	downBytes, err := ioutil.ReadAll(downReader)
	assert.NilError(t, err)
	print(string(downBytes))
	print(string([]byte("Testing Testing 123")))

	err = api.RemoveDocument(sdc, uploadDocID)
	assert.NilError(t, err)

	// docs, err = api.ListDocuments()
	// assert.NoError(t, err)
	// assert.Equal(t, len(docs), 0)

	// file, err := os.Open(TestDoc1)
	// assert.NoError(t, err)
	// defer file.Close()
}

func TestE2EEUpload(t *testing.T) {
	sdc, _, registeredOrgUsers, orgids, err := testSetup(1, 1)
	assert.NilError(t, err)
	defer testTeardown(orgids)
	t.Run("test e2ee upload", func(t *testing.T) {
		admin := registeredOrgUsers[0][0]
		// admin login
		_, err := api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
		assert.NilError(t, err)
		defer api.Logout(sdc)
		testE2EEUpload(t, sdc)
	})

}
