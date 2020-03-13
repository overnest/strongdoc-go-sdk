package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely
// identifies the document.
func UploadDocument(token string, docName string, plaintext []byte) (docID string, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	stream, err := authClient.UploadDocumentStream(context.Background())
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
		err = fmt.Errorf("UploadDocument response is nil: [%v]", err)
		return
	}

	docID = res.GetDocID()
	return
}

// UploadDocumentStream encrypts a document with Strongdoc
// and stores it in Strongdoc-provided storage.
// It accepts an io.Reader (which should
// contain the plaintext) and the document name.
//
// It then returns a docID that uniquely
// identifies the document.
func UploadDocumentStream(token string, docName string, plainStream io.Reader) (docID string, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	stream, err := authClient.UploadDocumentStream(context.Background())
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

	for {
		block := make([]byte, blockSize)
		n, err := plainStream.Read(block)
		if err != nil && err == io.EOF {
			dataReq := &proto.UploadDocStreamReq{
				NameOrData: &proto.UploadDocStreamReq_Plaintext{
					Plaintext: block[:n]}}
			if err = stream.Send(dataReq); err != nil {
				return "", err
			}
			break
		} else if err != nil {
			return "", err
		} else {
			dataReq := &proto.UploadDocStreamReq{
				NameOrData: &proto.UploadDocStreamReq_Plaintext{
					Plaintext: block[:n]}}
			if err = stream.Send(dataReq); err != nil {
				return "", err
			}
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
func DownloadDocument(token string, docID string) (plaintext []byte, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.DownloadDocStreamReq{DocID: docID}
	stream, err := authClient.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}

	for err == nil {
		res, rcverr := stream.Recv()
		if res != nil && docID != res.GetDocID() {
			err = fmt.Errorf("Requested document %v, received document %v instead",
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
// the requested document.
func DownloadDocumentStream(token string, docID string) (plainStream io.Reader, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	//defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.DownloadDocStreamReq{DocID: docID}
	stream, err := authClient.DownloadDocumentStream(context.Background(), req)
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
		return 0, fmt.Errorf("Requested document %v, received document %v instead",
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
func RemoveDocument(token, docID string) error {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return err
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.RemoveDocumentRequest{DocID: docID}
	_, err = authClient.RemoveDocument(context.Background(), req)
	return err
}

// ShareDocument shares the document with other users.
// Note that the user that you are sharing with be be in an organization
// that has been declared available for sharing with the Add Sharable Organizations function.
func ShareDocument(token, docId, userId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.ShareDocumentRequest{
		DocID:  docId,
		UserID: userId,
	}
	res, err := authClient.ShareDocument(context.Background(), req)
	if err != nil {
		fmt.Printf("Failed to share: %v", err)
		return
	}
	success = res.Success
	return
}

// UnshareDocument rescinds permission granted earlier, removing other users'
// access to those documents.
func UnshareDocument(token, docId, userId string) (count int64, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.UnshareDocumentRequest{
		DocID:  docId,
		UserID: userId,
	}
	res, err := authClient.UnshareDocument(context.Background(), req)

	count = res.Count
	return
}

type Document struct {
	DocName string
	DocID string
	Size uint64
}

// convertListDocumentsResponse_Document strips
func convertListDocumentsResponse_Document(protoDocs []*proto.ListDocumentsResponse_Document) (docs []Document) {
	for _, protoDoc := range protoDocs {
		doc := Document{
			DocName: protoDoc.DocName,
			DocID:   protoDoc.DocID,
			Size:    protoDoc.Size,
		}
		docs = append(docs, doc)
	}
	return docs
}

// ListDocuments returns a slice of Document objects, representing
// the documents accessible by the user.
func ListDocuments(token string) (docs []Document, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.ListDocumentsRequest{}
	res, err := authClient.ListDocuments(context.Background(), req)
	if err != nil {
		return
	}
	docs = convertListDocumentsResponse_Document(res.Documents)
	return
}

// GetDocumentsSize returns the size of all the documents of the organization.
func GetDocumentsSize(token string) (size uint64, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.GetDocumentsSizeRequest{}
	res, err := authClient.GetDocumentsSize(context.Background(), req)
	if err != nil {
		return
	}

	size = res.Size
	return
}

// GetIndexSize returns the size of all the indexes of the organization.
func GetIndexSize(token string) (size int64, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.GetIndexSizeRequest{}
	res, err := authClient.GetIndexSize(context.Background(), req)

	size = res.Size
	return
}
