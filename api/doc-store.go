package api

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely identifies the document.
func UploadDocument(docName string, plaintext []byte) (docID string, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	stream, err := sdc.UploadDocumentStream(context.Background())
	if err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	docNameReq := &proto.UploadDocStreamReq{
		NameOrData: &proto.UploadDocStreamReq_DocName{
			DocName: docName},
	}
	if err = stream.Send(docNameReq); err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	for i := 0; i < len(plaintext); i += blockSize {
		var block []byte
		if i+blockSize < len(plaintext) {
			block = plaintext[i : i+blockSize]
		} else {
			block = plaintext[i:]
		}

		dataReq := &proto.UploadDocStreamReq{
			NameOrData: &proto.UploadDocStreamReq_Plaintext{
				Plaintext: block}}
		if err = stream.Send(dataReq); err != nil {
			err = fmt.Errorf("UploadDocument err: [%v]", err)
			return
		}
	}

	if err = stream.CloseSend(); err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}
	if res == nil {
		err = fmt.Errorf("UploadDocument Resp is nil: [%v]", err)
		return
	}

	docID = res.GetDocID()
	return
}

// UploadDocumentStream encrypts a document with Strongdoc and stores it in Strongdoc-provided storage.
// It accepts an io.Reader (which should contain the plaintext) and the document name.
//
// It then returns a docID that uniquely identifies the document.
func UploadDocumentStream(docName string, plainStream io.Reader) (docID string, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	stream, err := sdc.UploadDocumentStream(context.Background())
	if err != nil {
		return
	}

	docNameReq := &proto.UploadDocStreamReq{
		NameOrData: &proto.UploadDocStreamReq_DocName{
			DocName: docName},
	}
	if err = stream.Send(docNameReq); err != nil {
		return
	}

	block := make([]byte, blockSize)
	for {
		n, inerr := plainStream.Read(block)
		if inerr == nil || inerr == io.EOF {
			if n > 0 {
				dataReq := &proto.UploadDocStreamReq{
					NameOrData: &proto.UploadDocStreamReq_Plaintext{
						Plaintext: block[:n]}}
				if err = stream.Send(dataReq); err != nil {
					return "", err
				}
			}
			if inerr == io.EOF {
				break
			}
		} else {
			err = inerr
			return
		}
	}

	if err = stream.CloseSend(); err != nil {
		return "", err
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return
	}

	docID = resp.GetDocID()
	return
}

// DownloadDocument downloads a document stored in Strongdoc-provided
// storage. You must provide it with its docID.
func DownloadDocument(docID string) (plaintext []byte, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.DownloadDocStreamReq{DocID: docID}
	stream, err := sdc.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}

	for err == nil {
		res, rcverr := stream.Recv()
		if res != nil && docID != res.GetDocID() {
			err = fmt.Errorf("Reqed document %v, received document %v instead",
				docID, res.GetDocID())
			return
		}
		plaintext = append(plaintext, res.GetPlaintext()...)
		err = rcverr
	}

	if err == io.EOF {
		err = nil
	}

	return
}

// DownloadDocumentStream decrypts any document previously stored on
// Strongdoc-provided storage, and makes the plaintext available via an io.Reader interface.
// You must also pass in its docId.
//
// It then returns an io.Reader object that contains the plaintext of
// the Reqed document.
func DownloadDocumentStream(docID string) (plainStream io.Reader, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.DownloadDocStreamReq{DocID: docID}
	stream, err := sdc.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}
	plainStream = &downloadStream{
		grpcStream: &stream,
		grpcEOF:    false,
		buffer:     new(bytes.Buffer),
		docID:      docID,
	}
	return
}

type downloadStream struct {
	grpcStream *proto.StrongDocService_DownloadDocumentStreamClient
	grpcEOF    bool
	buffer     *bytes.Buffer
	docID      string
}

// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
func (stream *downloadStream) Read(p []byte) (n int, err error) {
	if stream.buffer.Len() > 0 {
		return stream.buffer.Read(p)
	}

	if stream.grpcEOF {
		return 0, io.EOF
	}

	resp, err := (*stream.grpcStream).Recv()
	if resp != nil && stream.docID != resp.GetDocID() {
		return 0, fmt.Errorf("Reqed document %v, received document %v instead",
			stream.docID, resp.GetDocID())
	}

	if err != nil {
		if err != io.EOF {
			return 0, err
		}
		stream.grpcEOF = true
	}

	if resp != nil {
		plaintext := resp.GetPlaintext()
		copied := copy(p, plaintext)
		if copied < len(plaintext) {
			stream.buffer.Write(plaintext[copied:])
		}
		return copied, nil
	}

	return 0, nil
}

// RemoveDocument deletes a document from Strongdoc-provided
// storage. If you are a regular user, you may only remove
// a document that belongs to you. If you are an administrator,
// you can remove all the documents of the organization
// for which you are an administrator.
func RemoveDocument(docID string) error {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return err
	}

	req := &proto.RemoveDocumentReq{DocID: docID}
	_, err = sdc.RemoveDocument(context.Background(), req)
	return err
}

// ShareDocument shares the document with other users.
// Note that the user that you are sharing with be be in an organization
// that has been declared available for sharing with the Add Sharable Organizations function.
func ShareDocument(docID, userID string) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return false, err
	}

	req := &proto.ShareDocumentReq{
		DocID:  docID,
		UserID: userID,
	}
	res, err := sdc.ShareDocument(context.Background(), req)
	if err != nil {
		fmt.Printf("Failed to share: %v", err)
		return
	}
	success = res.Success
	return
}

// UnshareDocument rescinds permission granted earlier, removing other users'
// access to those documents.
func UnshareDocument(docID, userID string) (count int64, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return 0, err
	}

	req := &proto.UnshareDocumentReq{
		DocID:  docID,
		UserID: userID,
	}
	res, err := sdc.UnshareDocument(context.Background(), req)

	count = res.Count
	return
}

// Document contains the document information
type Document struct {
	// The document ID.
	DocID string
	// The document name.
	DocName string
	// The document size.
	Size uint64
}

// ListDocuments returns a slice of Document objects, representing
// the documents accessible by the user.
func ListDocuments() (docs []Document, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	req := &proto.ListDocumentsReq{}
	res, err := sdc.ListDocuments(context.Background(), req)
	if err != nil {
		return nil, err
	}

	documents, err := utils.ConvertStruct(res.GetDocuments(), []Document{})
	if err != nil {
		return nil, err
	}

	return *documents.(*[]Document), nil
}
