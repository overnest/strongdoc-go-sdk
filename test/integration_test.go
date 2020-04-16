package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

const (
	AdminName        = "adminUserName"
	AdminPassword    = "adminUserPassword"
	AdminEmail       = "adminUser@somewhere.com"
	Organization     = "OrganizationOne"
	OrganizationAddr = ""
	Source           = "Test Active"
	SourceData       = ""
	TestDoc1         = "../testDocuments/CompanyIntro.txt"
	TestDoc2         = "../testDocuments/BedMounts.pdf"
)

func testAccounts(t *testing.T) {
	users, err := api.ListUsers()
	assert.NoError(t, err)
	assert.NotEmpty(t, users)

	account, err := api.GetAccountInfo()
	assert.NoError(t, err)
	assert.NotNil(t, account)

	user, err := api.GetUserInfo()
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func testUploadDownload(t *testing.T, fileName string) {
	txtBytes, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err)
	uploadDocID, err := api.UploadDocument(path.Base(fileName), txtBytes)
	assert.NoError(t, err)

	downBytes, err := api.DownloadDocument(uploadDocID)
	assert.NoError(t, err)
	assert.Equal(t, txtBytes, downBytes)

	docs, err := api.ListDocuments()
	assert.NoError(t, err)
	assert.Equal(t, len(docs), 1)

	hits, err := api.Search("security")
	assert.NoError(t, err)
	assert.Equal(t, len(hits), 1)

	err = api.RemoveDocument(uploadDocID)
	assert.NoError(t, err)

	docs, err = api.ListDocuments()
	assert.NoError(t, err)
	assert.Equal(t, len(docs), 0)

	downBytes, err = api.DownloadDocument(uploadDocID)
	assert.Error(t, err)

	hits, err = api.Search("security")
	assert.NoError(t, err)
	assert.Equal(t, len(hits), 0)

	file, err := os.Open(TestDoc1)
	assert.NoError(t, err)
	defer file.Close()

	uploadDocID, err = api.UploadDocumentStream(path.Base(fileName), file)
	assert.NoError(t, err)

	stream, err := api.DownloadDocumentStream(uploadDocID)
	assert.NoError(t, err)
	downBytes, err = ioutil.ReadAll(stream)
	assert.NoError(t, err)
	assert.Equal(t, txtBytes, downBytes)

	err = api.RemoveDocument(uploadDocID)
	assert.NoError(t, err)
}

func testEncryptDecrypt(t *testing.T, fileName string) {
	txtBytes, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err)
	encryptDocID, ciphertext, err := api.EncryptDocument(path.Base(fileName), txtBytes)
	assert.NoError(t, err)

	plaintext, err := api.DecryptDocument(encryptDocID, ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, txtBytes, plaintext)

	docs, err := api.ListDocuments()
	assert.NoError(t, err)
	assert.Equal(t, len(docs), 1)

	hits, err := api.Search("security")
	assert.NoError(t, err)
	assert.Equal(t, len(hits), 1)

	err = api.RemoveDocument(encryptDocID)
	assert.NoError(t, err)

	docs, err = api.ListDocuments()
	assert.NoError(t, err)
	assert.Equal(t, len(docs), 0)

	plaintext, err = api.DecryptDocument(encryptDocID, ciphertext)
	assert.Error(t, err)

	hits, err = api.Search("security")
	assert.NoError(t, err)
	assert.Equal(t, len(hits), 0)

	file, err := os.Open(TestDoc1)
	assert.NoError(t, err)
	defer file.Close()

	cipherStream, encryptDocID, err := api.EncryptDocumentStream(path.Base(fileName), file)
	assert.NoError(t, err)

	plainStream, err := api.DecryptDocumentStream(encryptDocID, cipherStream)
	assert.NoError(t, err)

	plaintext, err = ioutil.ReadAll(plainStream)
	assert.NoError(t, err)
	assert.Equal(t, txtBytes, plaintext)

	err = api.RemoveDocument(encryptDocID)
	assert.NoError(t, err)
}

func testBilling(t *testing.T) {
	bill, err := api.GetBillingDetails()
	assert.NoError(t, err)

	fmt.Println("Bill:", bill)
	fmt.Println("Bill-Documents:", bill.Documents)
	fmt.Println("Bill-Search:", bill.Search)
	fmt.Println("Bill-Traffic:", bill.Traffic)

	freqs, err := api.GetBillingFrequencyList()
	assert.NoError(t, err)
	for i, freq := range freqs {
		fmt.Printf("Get Frequency[%v]: %v\n", i, freq)
	}

	// Wait a bit
	time.Sleep(time.Second * 2)

	freq, err := api.SetNextBillingFrequency(proto.TimeInterval_MONTHLY, time.Now().AddDate(0, 2, 0))
	assert.NoError(t, err)
	fmt.Printf("Set Frequency: %v\n", freq)

	freqs, err = api.GetBillingFrequencyList()
	assert.NoError(t, err)
	for i, freq := range freqs {
		fmt.Printf("Get Frequency[%v]: %v\n", i, freq)
	}

	traffic, err := api.GetLargeTraffic(time.Now())
	assert.NoError(t, err)
	fmt.Println("Large Traffic:", traffic)
}

func TestIntegrationSmall(t *testing.T) {
	_, err := client.InitStrongDocManager(client.LOCAL, false)
	assert.NoError(t, err)

	orgID, adminID, err := api.RegisterOrganization(Organization, OrganizationAddr,
		AdminName, AdminPassword, AdminEmail, Source, SourceData)
	assert.NoError(t, err)
	assert.Equal(t, orgID, Organization)
	assert.Equal(t, adminID, AdminEmail)

	token, err := api.Login(AdminEmail, AdminPassword, Organization)
	assert.NoError(t, err)
	assert.NotNil(t, token)

	defer func() {
		success, err := api.RemoveOrganization(true)
		assert.NoError(t, err)
		assert.True(t, success)
	}()

	testAccounts(t)
	testUploadDownload(t, TestDoc1)
	testEncryptDecrypt(t, TestDoc1)
	testBilling(t)

	_, err = api.Logout()
	assert.NoError(t, err)

	_, err = api.ListDocuments()
	assert.Error(t, err)

	// Need to wait at least 1 second before logging back in
	time.Sleep(time.Second * 2)

	// Log back in so organization can be removed
	token, err = api.Login(AdminEmail, AdminPassword, Organization)
	assert.NoError(t, err)
	assert.NotNil(t, token)
}
