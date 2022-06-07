package test

// import (
// 	"bytes"
// 	"io/ioutil"
// 	"path"

// 	"github.com/overnest/strongdoc-go-sdk/test/testUtils"

// 	"github.com/overnest/strongdoc-go-sdk/api"
// 	"github.com/overnest/strongdoc-go-sdk/client"

// 	// cryptoKey "github.com/overnest/strongsalt-crypto-go"
// 	// cryptoKdf "github.com/overnest/strongsalt-crypto-go/kdf"
// 	// "path"
// 	"os"
// 	"testing"

// 	// "io/ioutil"
// 	"gotest.tools/assert"
// )

// /*func initTest(t *testing.T) string {
// 	// init client
// 	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
// 	assert.NilError(t, err)

// 	// register org and admin
// 	orgs, orgUsers := initData(1, 1)
// 	orgData := orgs[0]
// 	userData := orgUsers[0][0]
// 	err = RegisterOrgAndAdmin(sdc, orgData, userData)
// 	assert.NilError(t, err)
// 	// defer HardRemoveOrgs([]string{orgData.OrgID})

// 	// login
// 	adminToken, err := sdc.Login(userData.UserID, userData.Password, orgData.OrgID, userData.PasswordKeyPwd)
// 	assert.NilError(t, err)
// 	return adminToken
// }*/

// // func TestUploadDownloadE2EE {

// // }

// func testCreateUpdateDownload(t *testing.T, sdc client.StrongDocClient, uploader, downloader *testUtils.TestUser, filename1 string, cleanUp bool) string {
// 	txtBytes, err := ioutil.ReadFile(filename1)
// 	assert.NilError(t, err)

// 	//fmt.Printf("file len: %v\n", len(txtBytes))

// 	file, err := os.Open(filename1)
// 	assert.NilError(t, err)
// 	defer file.Close()

// 	err = api.Login(sdc, uploader.UserID, uploader.Password, uploader.OrgID)
// 	assert.NilError(t, err)
// 	if cleanUp {
// 		defer api.Logout(sdc)
// 	}

// 	createdDocID, err := api.CreateDocument(sdc, path.Base(filename1))
// 	assert.NilError(t, err)

// 	docVersion1, err := api.UpdateDocument(sdc, createdDocID, true, file)
// 	assert.NilError(t, err)
// 	_ = docVersion1

// 	//fmt.Println(uploadDocID)

// 	if uploader != downloader {
// 		_, err = api.Logout(sdc)
// 		assert.NilError(t, err)

// 		err := api.Login(sdc, downloader.UserID, downloader.Password, downloader.OrgID)
// 		assert.NilError(t, err)
// 	}

// 	docs, err := api.ListDocuments(sdc)
// 	assert.NilError(t, err)
// 	assert.Equal(t, len(docs), 2)

// 	downReader, err := api.DownloadDocumentStream(sdc, createdDocID)
// 	assert.NilError(t, err)
// 	plaintext, err := ioutil.ReadAll(downReader)
// 	assert.NilError(t, err)
// 	//fmt.Printf("plaintext len: %v\n", len(plaintext))
// 	assert.Assert(t, bytes.Equal(txtBytes, plaintext), "Plaintext doesn't match")

// 	if cleanUp {
// 		err = api.RemoveDocument(sdc, createdDocID)
// 		assert.NilError(t, err)

// 		docs, err = api.ListDocuments(sdc)
// 		assert.NilError(t, err)
// 		assert.Equal(t, len(docs), 0)
// 	}

// 	return createdDocID
// }

// func testUpdateDownloadAgain(t *testing.T, sdc client.StrongDocClient, docID, newFilename string) {
// 	defer api.Logout(sdc)

// 	txtBytes, err := ioutil.ReadFile(newFilename)
// 	assert.NilError(t, err)

// 	//fmt.Printf("file len: %v\n", len(txtBytes))

// 	file, err := os.Open(newFilename)
// 	assert.NilError(t, err)
// 	defer file.Close()

// 	docVersion1, err := api.UpdateDocument(sdc, docID, true, file)
// 	assert.NilError(t, err)
// 	_ = docVersion1

// 	downReader, err := api.DownloadDocumentStream(sdc, docID)
// 	assert.NilError(t, err)
// 	plaintext, err := ioutil.ReadAll(downReader)
// 	assert.NilError(t, err)
// 	//fmt.Printf("plaintext len: %v\n", len(plaintext))
// 	assert.Assert(t, bytes.Equal(txtBytes, plaintext), "Plaintext doesn't match")

// 	err = api.RemoveDocument(sdc, docID)
// 	assert.NilError(t, err)

// 	docs, err := api.ListDocuments(sdc)
// 	assert.NilError(t, err)
// 	assert.Equal(t, len(docs), 0)
// }

// /*func testE2EEUploadDownload(t *testing.T, sdc client.StrongDocClient, uploader, downloader *testUtils.TestUser, filename string) {
// 	txtBytes, err := ioutil.ReadFile(filename)
// 	assert.NilError(t, err)

// 	//fmt.Printf("file len: %v\n", len(txtBytes))

// 	file, err := os.Open(filename)
// 	assert.NilError(t, err)
// 	defer file.Close()

// 	err = api.Login(sdc, uploader.UserID, uploader.Password, uploader.OrgID)
// 	assert.NilError(t, err)
// 	defer api.Logout(sdc)

// 	uploadDocID, err := api.E2EEUploadDocument(sdc, path.Base(filename), file)
// 	assert.NilError(t, err)
// 	//fmt.Println(uploadDocID)

// 	if uploader != downloader {
// 		_, err = api.Logout(sdc)
// 		assert.NilError(t, err)

// 		err := api.Login(sdc, downloader.UserID, downloader.Password, downloader.OrgID)
// 		assert.NilError(t, err)
// 	}

// 	docs, err := api.ListDocuments(sdc)
// 	assert.NilError(t, err)
// 	assert.Equal(t, len(docs), 1)

// 	downReader, err := api.DownloadDocumentStream(sdc, uploadDocID)
// 	assert.NilError(t, err)
// 	plaintext, err := ioutil.ReadAll(downReader)
// 	assert.NilError(t, err)
// 	//fmt.Printf("plaintext len: %v\n", len(plaintext))
// 	assert.Assert(t, bytes.Equal(txtBytes, plaintext), "Plaintext doesn't match")

// 	err = api.RemoveDocument(sdc, uploadDocID)
// 	assert.NilError(t, err)

// 	docs, err = api.ListDocuments(sdc)
// 	assert.NilError(t, err)
// 	assert.Equal(t, len(docs), 0)
// }*/

// func testE2EEAdminDownload(t *testing.T, sdc client.StrongDocClient, docID string) {

// }

// /*func TestE2EEUploadDownload(t *testing.T) {
// 	sdc, orgs, registeredOrgUsers := testUtils.PrevTest(t, 1, 2)
// 	testUtils.DoRegistration(t, sdc, orgs, registeredOrgUsers)

// 	t.Run("test e2ee upload download", func(t *testing.T) {
// 		admin := registeredOrgUsers[0][0]
// 		notAdmin := registeredOrgUsers[0][1]

// 		testE2EEUploadDownload(t, sdc, admin, admin, TestDoc1)
// 		testE2EEUploadDownload(t, sdc, notAdmin, admin, TestDoc1)
// 		//testE2EEUploadDownload(t, sdc, notAdmin, admin, "../testDocuments/smallpicture.bmp")
// 	})

// }*/

// func TestCreateUpdateDownloadOnce(t *testing.T) {
// 	sdc, orgs, registeredOrgUsers := testUtils.PrevTest(t, 1, 2)
// 	testUtils.DoRegistration(t, sdc, orgs, registeredOrgUsers)

// 	t.Run("test e2ee create update download", func(t *testing.T) {
// 		admin := registeredOrgUsers[0][0]
// 		notAdmin := registeredOrgUsers[0][1]

// 		testCreateUpdateDownload(t, sdc, admin, admin, TestDoc1, true)
// 		testCreateUpdateDownload(t, sdc, notAdmin, admin, TestDoc1, true)
// 		//testE2EEUploadDownload(t, sdc, notAdmin, admin, "../testDocuments/smallpicture.bmp")
// 	})

// }

// func TestCreateUpdateDownloadTwice(t *testing.T) {
// 	sdc, orgs, registeredOrgUsers := testUtils.PrevTest(t, 1, 2)
// 	testUtils.DoRegistration(t, sdc, orgs, registeredOrgUsers)

// 	t.Run("test e2ee create update download twice", func(t *testing.T) {
// 		admin := registeredOrgUsers[0][0]
// 		notAdmin := registeredOrgUsers[0][1]

// 		docID1 := testCreateUpdateDownload(t, sdc, admin, admin, TestDoc1, false)
// 		testUpdateDownloadAgain(t, sdc, docID1, TestDoc2)
// 		docID2 := testCreateUpdateDownload(t, sdc, notAdmin, admin, TestDoc1, false)
// 		testUpdateDownloadAgain(t, sdc, docID2, TestDoc2)
// 		//testE2EEUploadDownload(t, sdc, notAdmin, admin, "../testDocuments/smallpicture.bmp")
// 	})

// }
