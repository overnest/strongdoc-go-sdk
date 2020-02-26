package api

import (
	"bytes"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestEncryptStream(t *testing.T) {
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

	fileName := "CompanyIntro.txt"
	filePath, err := utils.FetchFileLoc("/testDocuments/CompanyIntro.txt")

	pdf, err := os.Open(filePath)
	pdfBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to open file: %s", err)
		return
	}

	// encrypting plainText
	eds, docId, err := EncryptDocumentStream(token, fileName, pdf)
	if err != nil {
		log.Printf("could not create EncryptStream object: %s", err)
		return
	}
	blockSize := 10000
	buf := make([]byte, blockSize)
	encryptedBytes := make([]byte,0)
	for err != io.EOF { // a quick hack, fails if fileSize % blockSize == 0
		n, readErr := eds.Read(buf)
		err = readErr
		encryptedBytes = append(encryptedBytes, buf[:n]...)
	}

	// decrypting cipherText
	dds, err := DecryptDocumentStream(token, docId, bytes.NewReader(encryptedBytes))
	if err != nil {
		log.Printf("Can not decrypt document: %s", err)
		return
	}
	decryptedBytes := make([]byte,0)
	for err != io.EOF { // a quick hack, fails if fileSize % blockSize == 0
		n, readErr := dds.Read(buf)
		decryptedBytes = append(decryptedBytes, buf[:n]...)
		if readErr != nil && readErr != io.EOF {
			return
		}
		err = readErr
	}

	// prints and asserts

	fmt.Printf("first 20 btyes of pdfBytes      : [%v]\n", pdfBytes[:20])
	fmt.Printf("first 20 btyes of decryptedBytes: [%v]\n", decryptedBytes[:20])
	fmt.Printf("last 20 btyes of pdfBytes       : [%v]\n", pdfBytes[len(pdfBytes)-20:])
	fmt.Printf("last 20 btyes of decryptedBytes : [%v]\n", decryptedBytes[len(decryptedBytes)-20:])
	assert.True(t, bytes.Equal(pdfBytes, decryptedBytes))

	err = RemoveDocument(token, docId)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}
}
