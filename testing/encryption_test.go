package testing

import (
	"bytes"
	"fmt"
	"github.com/overnest/strongdoc-go/api"

	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
)

func TestEncrypt(t *testing.T) {

	//_, _, err := RegisterOrganization(Organization, "", AdminName,
	//	AdminPassword, AdminEmail)
	//if err != nil {
	//	log.Printf("Failed to register organization: %s", err)
	//	return
	//}

	token, err := api.Login(adminEmail, adminPassword, organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}
	docName := "BedMounts.pdf"
	pdfBytes, err := ioutil.ReadFile("/Users/jonathan/strongdoc-go/testDocuments/BedMounts.pdf")

	ecs, err := api.EncryptDocumentStream(token, docName)
	if err != nil {
		log.Printf("Can not encrypt document: %s", err)
		return
	}
	n, err := ecs.Write(pdfBytes)
	if err != nil {
		return
	}
	fmt.Printf("Wrote %d bytes to encryptStream\n", n)

	blockSize := 10000
	buf := make([]byte, blockSize)
	encryptedBytes := make([]byte,0)
	for len(encryptedBytes) % blockSize == 0 { // a quick hack, fails if fileSize % blockSize == 0
		n, readErr := ecs.Read(buf)
		err = readErr
		encryptedBytes = append(encryptedBytes, buf[:n]...)
	}
	encryptDocID := ecs.DocId()

	//decryptedBytes, err := api.DecryptDocument(token, encryptDocID, encryptedBytes)
	//if err != nil {
	//	log.Printf("Can not decrypt document: %s", err)
	//	return
	//}

	dcs, err := api.DecryptDocumentStream(token, encryptDocID)
	if err != nil {
		log.Printf("Can not decrypt document: %s", err)
		return
	}
	n, err = dcs.Write(encryptedBytes)
	fmt.Printf("Wrote %d bytes to decryptStream\n", n)
	if err != nil {
		return
	}
	decryptedBytes := make([]byte,0)
	fmt.Printf("len(decryptedBytes) mod blockSize is [%d]\n", len(decryptedBytes) % blockSize)
	n, readErr := dcs.Read(buf)
	err = readErr
	decryptedBytes = append(decryptedBytes, buf[:n]...)
	for len(decryptedBytes) % blockSize == 9902 { // a quick hack, fails if fileSize % blockSize == 0
		n, readErr := dcs.Read(buf)
		err = readErr
		decryptedBytes = append(decryptedBytes, buf[:n]...)
		fmt.Printf("len(decryptedBytes) mod blockSize is [%d]\n", len(decryptedBytes) % blockSize)
	}
	fmt.Printf("len of pdfBytes      : [%d]\n", len(pdfBytes))
	fmt.Printf("len of decryptedBytes: [%d]\n", len(decryptedBytes))
	fmt.Printf("first 50 btyes of pdfBytes      : [%v]\n", pdfBytes)
	fmt.Printf("first 50 btyes of decryptedBytes: [%v]\n", decryptedBytes)
	fmt.Printf("last 50 btyes of pdfBytes       : [%v]\n", pdfBytes[len(pdfBytes)-50:])
	fmt.Printf("last 50 btyes of decryptedBytes : [%v]\n", decryptedBytes[len(decryptedBytes)-50:])
	assert.True(t, bytes.Equal(pdfBytes, decryptedBytes))

	err = api.RemoveDocument(token, encryptDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}
}
