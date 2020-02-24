package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// UploadDocument uploads a document to Strongdoc-provided storage.
// Returns a docID used to perform queries on it.
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

	var blockSize int = 10000
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

// UploadDocumentStream accepts a reader and reads from it until it
// ends. Incomplete downloads will not be completed.
func UploadDocumentStream(token string, docName string, reader *bytes.Reader) (docID string, err error) {
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

	var blockSize = 10000
	for {
		block := make([]byte, blockSize)
		n, err := reader.Read(block)
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
		time.Sleep(time.Second)
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
// storage.
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
		resp, rcverr := stream.Recv()
		if resp != nil && docID != resp.GetDocID() {
			err = fmt.Errorf("Requested document %v, received document %v instead",
				docID, resp.GetDocID())
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

type S3Stream struct {
	svc *proto.StrongDocService_DownloadDocumentStreamClient
	buffer *bytes.Buffer
	docID string
}

// TODO: handle empty p
// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
func (stream *S3Stream) Read(p []byte) (n int, err error) {
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

// DownloadDocumentStream generates an S3Stream object, which implements
// the io.Reader interface. It contains a client which maintains a
// connection to Strongdoc-provided storage. Returns 0, EOF iff
// the stream has been exhausted.
func DownloadDocumentStream(token string, docID string) (s3stream S3Stream, err error) {
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
	s3stream = S3Stream{
		svc: &stream,
		buffer: new(bytes.Buffer),
		docID: docID,
	}

	return s3stream, nil
}

// EncryptDocument encrypts a document with strongdoc, but does not store the actual document
func EncryptDocument(token string, docName string, plaintext []byte) (docID string, ciphertext []byte, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	stream, err := authClient.EncryptDocumentStream(context.Background())
	if err != nil {
		return
	}

	docNameReq := &proto.EncryptDocStreamReq{
		NameOrData: &proto.EncryptDocStreamReq_DocName{
			DocName: docName}}
	if err = stream.Send(docNameReq); err != nil {
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		return
	}

	docID = resp.GetDocID()
	var blockSize int = 10000
	for i := 0; i < len(plaintext); i += blockSize {
		var block []byte
		if i+blockSize < len(plaintext) {
			block = plaintext[i : i+blockSize]
		} else {
			block = plaintext[i:]
		}

		dataReq := &proto.EncryptDocStreamReq{
			NameOrData: &proto.EncryptDocStreamReq_Plaintext{
				Plaintext: block}}
		if err = stream.Send(dataReq); err != nil {
			return
		}

		dataResp, recvErr := stream.Recv()
		if recvErr != nil && recvErr != io.EOF {
			err = recvErr
			return
		}

		ciphertext = append(ciphertext, dataResp.GetCiphertext()...)
	}

	err = stream.CloseSend()
	return
}

type encryptDocumentStream struct {
	stream *proto.StrongDocService_EncryptDocumentStreamClient
	buffer *bytes.Buffer
	docId string
}

//func EncryptDocumentStream(token string, docName string) (docID string, reader io.Reader, writer io.Writer, err error) {
//	authConn, err := client.ConnectToServerWithAuth(token)
//	if err != nil {
//		log.Fatalf("Can not obtain auth connection %s", err)
//		return
//	}
//	defer authConn.Close()
//	authClient := proto.NewStrongDocServiceClient(authConn)
//
//	stream, err := authClient.EncryptDocumentStream(context.Background())
//	if err != nil {
//		return
//	}
//
//	streamObj := encryptDocumentStream{
//		stream: &stream,
//		buffer: new(bytes.Buffer),
//		docId:  "",
//	}
//
//
//	return
//}

// DecryptDocument encrypts a document with strongdoc. It requires original ciphertext, since the document is not stored
func DecryptDocument(token, docID string, ciphertext []byte) (plaintext []byte, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	stream, err := authClient.DecryptDocumentStream(context.Background())
	if err != nil {
		return
	}

	docIDReq := &proto.DecryptDocStreamReq{
		IdOrData: &proto.DecryptDocStreamReq_DocID{
			DocID: docID}}
	if err = stream.Send(docIDReq); err != nil {
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		return
	}

	if docID != resp.GetDocID() {
		err = fmt.Errorf("The requested document ID %v does not match the returned ID %v",
			docID, resp.GetDocID())
		return
	}

	var blockSize int = 10000
	for i := 0; i < len(ciphertext); i += blockSize {
		var block []byte
		if i+blockSize < len(ciphertext) {
			block = ciphertext[i : i+blockSize]
		} else {
			block = ciphertext[i:]
		}

		dataReq := &proto.DecryptDocStreamReq{
			IdOrData: &proto.DecryptDocStreamReq_Ciphertext{
				Ciphertext: block}}
		if err = stream.Send(dataReq); err != nil {
			return
		}

		resp, rcverr := stream.Recv()
		plaintext = append(plaintext, resp.GetPlaintext()...)
		err = rcverr
	}

	if err == io.EOF {
		err = nil
	}

	err = stream.CloseSend()
	return
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