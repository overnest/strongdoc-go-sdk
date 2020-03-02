package api

import (
	"bytes"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

func TestShare(t *testing.T) {

	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Failed to register organization: %s", err)
		return
	}
	token, err := Login(adminEmail, adminPassword, organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	//defer func() {
	//	_, err = RemoveOrganization(token, true)
	//	if err != nil {
	//		log.Printf("Failed to log in: %s", err)
	//		return
	//	}
	//}()

	filePath, err := utils.FetchFileLoc("/testDocuments/BedMounts.pdf")
	txtBytes, err := ioutil.ReadFile(filePath)

	docID, err := UploadDocument(token, "BedMounts.pdf", txtBytes)
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	user := "otherUser"
	pass := "pass"
	email := "email@email.com"

	// create user
	otherUserID, err := RegisterUser(token, user, pass, email, false)
	if err != nil && !strings.Contains(err.Error(), "already belongs") {
		fmt.Printf("otherUserID [%s] err: [%v]\n", otherUserID, err)
		return
	}
	defer RemoveUser(token, otherUserID)
	// share to him
	ok, err := ShareDocument(token, docID, email)
	fmt.Printf("success: [%t]\n", ok)
	assert.True(t, ok)
	// log out
	_, err = Logout(token)
	// log in as other user
	token, err = Login(email, pass, organization)
	if err != nil {
		fmt.Printf("token [%s] err: [%v]\n", token, err)
		return
	}
	// download document
	plaintext, err := DownloadDocument(token, docID)
	// if success, then pass
	assert.True(t, bytes.Equal(plaintext, txtBytes))
	assert.Nil(t, err)
}
