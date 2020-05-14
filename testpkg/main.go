package main

import (
	"io"
	"log"
	"os"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/udhos/equalfile"
)

const (
	adminName         = "adminUserName"
	adminPassword     = "adminUserPassword"
	adminEmail        = "adminUser@somewhere.com"
	organization      = "OrganizationOne"
	organizationAddr  = ""
	organizationEmail = "info@organizationone.com"
	source            = "Test Active"
	sourceData        = ""
	testDoc1          = "./testdocs/CompanyIntro.txt"
	testDoc2          = "./testdocs/BedMounts.pdf"
	tempFileName1     = "/tmp/tempstrongdoc1"
	tempFileName2     = "/tmp/tempstrongdoc2"
)

func testEncryptDecryptStream() {
	var plaintextFileName string = testDoc1
	var ciphertextFileName string = tempFileName1
	var decrypttextFileName string = tempFileName2

	plainTextFile, err := os.Open(plaintextFileName)
	if err != nil {
		log.Printf("Can not open file: %s", err)
		os.Exit(1)
	}
	defer plainTextFile.Close()

	cipherTextStream, docID, err := api.EncryptDocumentStream(plaintextFileName, plainTextFile)
	if err != nil {
		log.Printf("Can not encrypt document: %s", err)
		os.Exit(1)
	}

	cipherTextFile, err := os.Create(ciphertextFileName)
	if err != nil {
		log.Printf("Can not create file: %s", err)
		os.Exit(1)
	}
	defer os.Remove(ciphertextFileName)
	defer cipherTextFile.Close()

	_, err = io.Copy(cipherTextFile, cipherTextStream)
	if err != nil {
		log.Printf("Can not copy stream: %s", err)
		os.Exit(1)
	}
	cipherTextFile.Close()

	cipherTextFile, err = os.Open(ciphertextFileName)
	if err != nil {
		log.Printf("Can not open file: %s", err)
		os.Exit(1)
	}

	plainTextStream, err := api.DecryptDocumentStream(docID, cipherTextFile)
	if err != nil {
		log.Printf("Can not decrypt document: %s", err)
		os.Exit(1)
	}

	decryptTextFile, err := os.Create(decrypttextFileName)
	if err != nil {
		log.Printf("Can not create file: %s", err)
		os.Exit(1)
	}
	defer os.Remove(decrypttextFileName)
	defer decryptTextFile.Close()

	_, err = io.Copy(decryptTextFile, plainTextStream)
	if err != nil {
		log.Printf("Can not copy stream: %s", err)
		os.Exit(1)
	}
	decryptTextFile.Close()

	equal, err := equalfile.New(nil, equalfile.Options{}).CompareFile(plaintextFileName, decrypttextFileName)
	if err != nil {
		log.Printf("Can not compare files: %s", err)
		os.Exit(1)
	}

	if !equal {
		log.Printf("Decrypted file is not the same as original file")
		os.Exit(1)
	}

	err = api.RemoveDocument(docID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		os.Exit(1)
	}
}

func main() {
	_, err := client.InitStrongDocManager(client.LOCAL, false)
	if err != nil {
		log.Printf("Can not initialize StrongDoc manager: %s", err)
		os.Exit(1)
	}

	orgID, adminID, err := api.RegisterOrganization(organization, organizationAddr,
		organizationEmail, adminName, adminPassword, adminEmail, source, sourceData)
	if err != nil {
		log.Printf("RegisterOrganization error: %s", err)
		os.Exit(1)
	}
	if orgID != organization || adminID != adminEmail {
		log.Printf("RegisterOrganization results incorrect.")
		os.Exit(1)
	}

	_, err = api.Login(adminEmail, adminPassword, organization)
	if err != nil {
		log.Printf("Can not login: %s", err)
		os.Exit(1)
	}

	defer func() {
		_, err := api.RemoveOrganization(true)
		if err != nil {
			log.Printf("Can not remove organization: %s", err)
		}
	}()

	testEncryptDecryptStream()
}
