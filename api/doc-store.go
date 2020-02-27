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
		return
	}

	docNameReq := &proto.UploadDocStreamReq{
		NameOrData: &proto.UploadDocStreamReq_DocName{
			DocName: docName},
	}
	if err = stream.Send(docNameReq); err != nil {
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
			return
		}
	}

	if err = stream.CloseSend(); err != nil {
		return
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return
	}

	docID = resp.GetDocID()
	return
}

// UploadDocumentStream encrypts a document with Strongdoc
// and stores it in Strongdoc-provided storage.
// It accepts an io.Reader (which should
// contain the plaintext) and the document name.
//
// It then returns a docId that uniquely
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
// storage. You must provide it with its docId.
func DownloadDocument(token string, docId string) (plaintext []byte, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.DownloadDocStreamReq{DocID: docId}
	stream, err := authClient.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}

	for err == nil {
		resp, rcverr := stream.Recv()
		if resp != nil && docId != resp.GetDocID() {
			err = fmt.Errorf("Requested document %v, received document %v instead",
				docId, resp.GetDocID())
			return
		}
		plaintext = append(plaintext, resp.GetPlaintext()...)
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
func DownloadDocumentStream(token string, docId string) (downloadStream io.Reader, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	//defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.DownloadDocStreamReq{DocID: docId}
	stream, err := authClient.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}
	downloadStream = &storeStream{
		svc:    &stream,
		buffer: new(bytes.Buffer),
		docID:  docId,
	}
	return
}

type storeStream struct {
	svc *proto.StrongDocService_DownloadDocumentStreamClient
	buffer *bytes.Buffer
	docID string
}

// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
func (stream *storeStream) Read(p []byte) (n int, err error) {
	docID := stream.docID
	nPreRcvBytes, bufReadErr := stream.buffer.Read(p) // drain buffer first
	if bufReadErr != nil && bufReadErr != io.EOF {
		return 0, fmt.Errorf("buffer read err: [%v]", err)
	}
	nPostRcvBytes := 0
	if nPreRcvBytes == len(p) { // p runs out
		return nPreRcvBytes, nil
	} else { // if buffer has been completely drained
		svc := *stream.svc
		res, rcvErr := svc.Recv() // refill it from stream
		if res != nil && docID != res.GetDocID() {
			rcvErr = fmt.Errorf("requested document %v, received document %v instead",
				docID, res.GetDocID())
		}
		if rcvErr != nil && rcvErr != io.EOF {
			return 0, fmt.Errorf("error reading from rpc connection: [%v]", rcvErr)
		} else if rcvErr == io.EOF && bufReadErr == io.EOF {
			return 0, io.EOF
		} else if rcvErr == io.EOF {
			return nPreRcvBytes, nil
		}
		plaintext := res.GetPlaintext()
		stream.buffer.Write(plaintext)
		nPostRcvBytes, err = stream.buffer.Read(p[nPreRcvBytes:]) // read remaining part of buffer.
		if err != nil {
			return 0, fmt.Errorf("buffer read err: [%v]", err)
		}
		return nPreRcvBytes + nPostRcvBytes, err
	}
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
func ShareDocument(token, docId, userId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.ShareDocumentRequest{
		DocID:                docId,
		UserID:               userId,
	}
	res, err := authClient.ShareDocument(context.Background(), req)

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
		DocID:                docId,
		UserID:               userId,
	}
	res, err := authClient.UnshareDocument(context.Background(), req)

	count = res.Count
	return
}

type Documents struct {
	protoDocs *[]*proto.ListDocumentsResponse_Document
}

// ListDocuments returns a slice of Document objects, representing
// the documents accessible by the user.
func ListDocuments(token string) (docs Documents, err error) {
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

	docs = Documents{&res.Documents}
	return
}

// AddSharableOrg adds a sharable Organization.
func AddSharableOrg(token, orgId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.AddSharableOrgRequest{
		NewOrgID:             orgId,
	}
	res, err := authClient.AddSharableOrg(context.Background(), req)

	success = res.Success
	return
}

// RemoveSharableOrg removes a sharable Organization.
func RemoveSharableOrg(token, orgId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.RemoveSharableOrgRequest{
		RemoveOrgID:          orgId,
	}
	res, err := authClient.RemoveSharableOrg(context.Background(), req)

	success = res.Success
	return
}

// SetMultiLevelSharing sets MultiLevel Sharing.
func SetMultiLevelSharing(token string, enable bool) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.SetMultiLevelSharingRequest{
		Enable:               enable,
	}
	res, err := authClient.SetMultiLevelSharing(context.Background(), req)

	success = res.Success
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