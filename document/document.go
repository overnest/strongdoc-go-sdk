package document

import (
	"github.com/overnest/strongdoc-go-sdk/api/apidef"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/document/common"
	"github.com/overnest/strongdoc-go-sdk/document/documentv1"
)

func CreateDocument(sdc client.StrongDocClient, docName string, docKey *apidef.Key) (common.DocWriter, error) {
	return documentv1.CreateDocumentV1(sdc, docName, docKey)
}

func OpenDocument(sdc client.StrongDocClient, docID string, docKey *apidef.Key) (common.DocReader, error) {
	return documentv1.OpenDocumentV1(sdc, docID, docKey)
}
