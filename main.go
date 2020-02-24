package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/overnest/strongdoc-go/api"
)

func main() {

	fmt.Printf("Hello!")

	flag.Parse()

	orgID, adminID, err := api.RegisterOrganization(api.Organization, "", api.AdminName,
		api.AdminPassword, api.AdminEmail)
	if err != nil {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := api.Login(api.AdminEmail, api.AdminPassword, api.Organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = api.RemoveOrganization(token)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	txtBytes, err := ioutil.ReadFile("/Users/jonathan/strongdoc-go/testDocuments/CompanyIntro.txt")
	fmt.Printf("Printing txtBytes: [%v]", txtBytes)

	uploadDocID, err := api.UploadDocumentStream(token, "CompanyIntro.txt", bytes.NewReader(txtBytes))
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	hits, err := api.Search(token, "security")
	if err != nil {
		log.Printf("Can not search documents: %s", err)
		return
	}

	if len(hits) != 1 {
		log.Printf("Incorrect search results. Expecting 1, getting %v", len(hits))
		return
	}

	downBytes, err := api.DownloadDocument(token, uploadDocID)
	if err != nil {
		return
	}

	//downStream, err := api.DownloadDocumentStream(token, uploadDocID)
	//if err != nil {
	//	log.Printf("Can not download document: %s", err)
	//	return
	//}
	//p := make([]byte, 100)
	//var readingErr error = nil
	//downBytes := make([]byte, 0)
	//for readingErr != io.EOF {
	//	if readingErr != nil {
	//		return
	//	}
	//	_, readingErr = downStream.Read(p)
	//	p = append(downBytes, p...)
	//}

	if !bytes.Equal(txtBytes, downBytes) {
		log.Printf("The downloaded content is different from uploaded")
		return
	}

	err = api.RemoveDocument(token, uploadDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}

	pdfBytes, err := ioutil.ReadFile("/Users/jonathan/strongdoc-go/testDocuments/BedMounts.pdf")
	fmt.Printf("Printing pdfBytes: [%v]", pdfBytes)
	encryptDocID, ciphertext, err := api.EncryptDocument(token, "BedMounts.pdf", pdfBytes)
	if err != nil {
		log.Printf("Can not encrypt document: %s", err)
		return
	}

	//hits, err = api.Search(token, "bed mounts")
	if err != nil {
		log.Printf("Can not search documents: %s", err)
		return
	}

	decryptBytes, err := api.DecryptDocument(token, encryptDocID, ciphertext)
	if err != nil {
		log.Printf("Can not decrypt document: %s", err)
		return
	}

	if !bytes.Equal(pdfBytes, decryptBytes) {
		log.Printf("The decrypted content is different from uploaded")
		return
	}

	err = api.RemoveDocument(token, encryptDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}
}