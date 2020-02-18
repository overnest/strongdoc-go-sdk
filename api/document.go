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

// UploadDocument uploads a document to be stored by strongdoc
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

	var blockSize int = 10000
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

// DownloadDocument downloads a document stored in strongdoc
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

//
//func DownloadDocumentStream(token string, docID string) (reader *bytes.Reader, err error) {
//	authConn, err := client.ConnectToServerWithAuth(token)
//	if err != nil {
//		log.Fatalf("Can not obtain auth connection %s", err)
//		return
//	}
//	defer authConn.Close()
//	authClient := proto.NewStrongDocServiceClient(authConn)
//
//	req := &proto.DownloadDocStreamReq{DocID: docID}
//	stream, err := authClient.DownloadDocumentStream(context.Background(), req)
//	if err != nil {
//		return
//	}
//
//	for err == nil {
//		resp, rcverr := stream.Recv()
//		if rcverr != nil {
//			return
//		}
//		if resp != nil && docID != resp.GetDocID() {
//			err = fmt.Errorf("Requested document %v, received document %v instead",
//				docID, resp.GetDocID())
//			return
//		}
//		_, writeErr := buffer.Write(resp.GetPlaintext())
//		err = writeErr
//	}
//
//	if err == io.EOF {
//		err = nil
//	}
//
//	return
//}

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

// RemoveDocument deletes the document from strongdoc storage
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
