package api

import (
	"bytes"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
)

func TestEncrypt(t *testing.T) {
	//_, _, err := RegisterOrganization(organization, "", adminName,
	//	adminPassword, adminEmail)
	//if err != nil {
	//	log.Printf("Failed to register organization: %s", err)
	//	return
	//}

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

	pdfBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to open file: %s", err)
		return
	}

	// encrypting plainText
	docId, encryptedBytes, err := EncryptDocument(token, fileName, pdfBytes)
	if err != nil {
		log.Printf("could not create EncryptStream object: %s", err)
		return
	}

	// decrypting cipherText
	decryptedBytes, err := DecryptDocument(token, docId, encryptedBytes)
	if err != nil {
		log.Printf("Can not decrypt document: %s", err)
		return
	}
	if err != nil {
		return
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
