package api

import (
	"bytes"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

func TestDocStore(t *testing.T) {
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

	defer func() {
		_, err = RemoveOrganization(token)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	filePath, err := utils.FetchFileLoc("/testDocuments/BedMounts.pdf")
	txtBytes, err := ioutil.ReadFile(filePath)

	uploadDocID, err := UploadDocument(token, "BedMounts.pdf", txtBytes)
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	downDocBytes, err := DownloadDocument(token, uploadDocID)
	assert.Nil(t, err)

	assert.True(t, bytes.Equal(downDocBytes, txtBytes))

	err = RemoveDocument(token, uploadDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}
}

func TestDocStoreStream(t *testing.T) {
	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)
	if err != nil {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := Login(adminEmail, adminPassword, organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = RemoveOrganization(token)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	filePath, err := utils.FetchFileLoc("/testDocuments/BedMounts.pdf")
	txtBytes, err := ioutil.ReadFile(filePath)

	uploadDocID, err := UploadDocumentStream(token, "BedMounts.pdf", bytes.NewReader(txtBytes))
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	s, err := DownloadDocumentStream(token, uploadDocID)
	buf := make([]byte, 10)
	downDocBytes := make([]byte,0)
	for err == nil {
		n, readErr := s.Read(buf)
		err = readErr
		downDocBytes = append(downDocBytes, buf[:n]...)
	}
	assert.Errorf(t, err, "EOF")
	assert.True(t, bytes.Equal(downDocBytes, txtBytes))

	err = RemoveDocument(token, uploadDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}
}