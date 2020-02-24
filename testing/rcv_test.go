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

	token, err := api.Login(api.AdminEmail, api.AdminPassword, api.Organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}


	txtBytes, err := ioutil.ReadFile("/Users/jonathan/strongdoc-go/testDocuments/CompanyIntro.txt")
	fmt.Printf("Printing txtBytes: [%v]", txtBytes)

	uploadDocID, err := api.UploadDocumentStream(token, "CompanyIntro.txt", bytes.NewReader(txtBytes))
	if err != nil {
		log.Printf("Can not upload document: %s", err)
		return
	}

	downDocBytesNoStream, err := api.DownloadDocument(token, uploadDocID)
	assert.NotNil(t, err)
	fmt.Printf("%s", string(downDocBytesNoStream))

	s, err := api.DownloadDocumentStream(token, uploadDocID)
	downDocBytesStream := make([]byte, 1000)
	_, err = s.Read(downDocBytesStream)
	assert.NotNil(t, err)
	fmt.Printf("%s", string(downDocBytesStream))


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
	//
	//if !bytes.Equal(txtBytes, downBytes) {
	//	log.Printf("The downloaded content is different from uploaded")
	//	return
	//}

	err = api.RemoveDocument(token, uploadDocID)
	if err != nil {
		log.Printf("Can not remove document: %s", err)
		return
	}

}
