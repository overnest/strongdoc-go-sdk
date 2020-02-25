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

func TestRcv(t *testing.T) {

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


	txtBytes, err := ioutil.ReadFile("/Users/jonathan/strongdoc-go/testDocuments/CompanyIntro.txt")
	fmt.Printf("Printing txtBytes: [%v]\n", txtBytes)

	uploadDocID, err := api.UploadDocumentStream(token, "CompanyIntro.txt", bytes.NewReader(txtBytes))
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	downDocBytesNoStream, err := api.DownloadDocument(token, uploadDocID)
	assert.Nil(t, err)
	fmt.Printf("%s\n", string(downDocBytesNoStream))

	s, err := api.DownloadDocumentStream(token, uploadDocID)
	buf := make([]byte, 10)
	downDocBytesStream := make([]byte,0)
	for err == nil {
		n, readErr := s.Read(buf)
		err = readErr
		downDocBytesStream = append(downDocBytesStream, buf[:n]...)
	}
	assert.Errorf(t, err, "EOF")
	fmt.Printf("%s\n", string(downDocBytesStream))


	assert.True(t, bytes.Equal(downDocBytesStream, txtBytes))
	assert.True(t, bytes.Equal(downDocBytesStream, downDocBytesNoStream))

	err = api.RemoveDocument(token, uploadDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}

}
