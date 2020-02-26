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

	var blockSize = 10000
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

type s3Stream struct {
	svc *proto.StrongDocService_DownloadDocumentStreamClient
	buffer *bytes.Buffer
	docID string
}

// TODO: handle empty p
// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
func (stream *s3Stream) Read(p []byte) (n int, err error) {
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

// DownloadDocumentStream generates an s3Stream object, which implements
// the io.Reader interface. It contains a client which maintains a
// connection to Strongdoc-provided storage. Returns 0, EOF iff
// the stream has been exhausted.
func DownloadDocumentStream(token string, docID string) (s3stream io.Reader, err error) {
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
	s3stream = &s3Stream{
		svc:    &stream,
		buffer: new(bytes.Buffer),
		docID:  docID,
	}

	return s3stream, nil
}

// EncryptDocument encrypts a document with Strongdoc
// EncryptDocument encrypts a document with Strongdoc
// and returns the encrypted ciphertext. No data is ever
// written to storage on Strongdoc servers.
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

type encryptStream struct {
	stream *proto.StrongDocService_EncryptDocumentStreamClient
	readBuffer *bytes.Buffer
	writeBuffer *bytes.Buffer
	docId string
}

// EncryptDocumentStream returns an io.Reader.
// Call Read() on it to get the encrypted data. No data is ever
// written to storage on Strongdoc servers.
func EncryptDocumentStream(token string, docName string, plainStream io.Reader) (ec io.ReadWriter, docId string, n int, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
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

	res, err := stream.Recv()
	if err != nil {
		return
	}

	docId = res.GetDocID()

	blockSize := 10000
	for err != io.EOF {
		block := make([]byte, blockSize)
		numRead, readErr := plainStream.Read(block)
		block = block[:numRead]
		dataReq := &proto.EncryptDocStreamReq{
			NameOrData: &proto.EncryptDocStreamReq_Plaintext{
				Plaintext: block,
			},
		}
		if err = stream.Send(dataReq); err != nil {
			err = fmt.Errorf("send() err: [%v]", err)
			return
		}
		err = readErr
		n += numRead
	}
	if err != nil && err != io.EOF {
		return
	}
	err = nil

	ec = &encryptStream{
		stream: &stream,
		readBuffer: new(bytes.Buffer),
		docId:  docId,
	}
	return
}

func (stream *encryptStream) DocId() string {
	return stream.docId
}
// BUG: stream crashes when 0 bytes are
// available in the stream.
func (stream *encryptStream) Read(p []byte) (n int, err error) {
	docID := stream.docId
	nPreRcvBytes, bufReadErr := stream.readBuffer.Read(p) // drain readBuffer first
	if bufReadErr != nil && bufReadErr != io.EOF {
		return 0, fmt.Errorf("readBuffer read err: [%v]", err)
	}
	nPostRcvBytes := 0
	if nPreRcvBytes == len(p) { // p runs out
		return nPreRcvBytes, nil
	} else { // if readBuffer has been completely drained
		svc := *stream.stream
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
		cipherText := res.GetCiphertext()
		stream.readBuffer.Write(cipherText)
		nPostRcvBytes, err = stream.readBuffer.Read(p[nPreRcvBytes:]) // read remaining part of readBuffer.
		if err != nil {
			return 0, fmt.Errorf("readBuffer read err: [%v]", err)
		}
		return nPreRcvBytes + nPostRcvBytes, err
	}
}

func (stream *encryptStream) Write(plaintext []byte) (n int, err error) {
	svc := *stream.stream
	var blockSize int = 10000
	for i := 0; i < len(plaintext); i += blockSize {
		var block []byte
		if i+blockSize < len(plaintext) {
			block = plaintext[i : i+blockSize]
		} else {
			block = plaintext[i:]
		}
		n += len(block)
		dataReq := &proto.EncryptDocStreamReq{
			NameOrData: &proto.EncryptDocStreamReq_Plaintext{
				Plaintext: block,
			},
		}
		if err = svc.Send(dataReq); err != nil {
			return 0, fmt.Errorf("encryptStream.Write err: [%v]", err)
		}
	}
	return n,nil
}

type decryptStream struct {
	stream *proto.StrongDocService_DecryptDocumentStreamClient
	readBuffer *bytes.Buffer
	writeBuffer *bytes.Buffer
	docId string
}

// DecryptDocumentStream returns an io.Reader.
// Call Read() on it to get the decrypted data. No data is ever
// written to storage on Strongdoc servers.
func DecryptDocumentStream(token string, docId string, plainStream io.Reader) (ec io.Reader, n int, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	authClient := proto.NewStrongDocServiceClient(authConn)

	stream, err := authClient.DecryptDocumentStream(context.Background())
	if err != nil {
		return
	}

	docIdReq := &proto.DecryptDocStreamReq{
		IdOrData: &proto.DecryptDocStreamReq_DocID{
			DocID: docId,
		},
	}
	if err = stream.Send(docIdReq); err != nil {
		return
	}
	res, err := stream.Recv()
	if err != nil {
		return
	} else if res.DocID != docId {
		err = fmt.Errorf("incorrect docId Recv'd")
		return
	}

	// send contents of stream
	blockSize := 10000
	for err != io.EOF {
		block := make([]byte, blockSize)
		numRead, readErr := plainStream.Read(block)
		block = block[:numRead]
		dataReq := &proto.DecryptDocStreamReq{
			IdOrData: &proto.DecryptDocStreamReq_Ciphertext{
				Ciphertext: block,
			},
		}
		if err = stream.Send(dataReq); err != nil {
			err = fmt.Errorf("send() err: [%v]", err)
			return
		}
		err = readErr
		n += numRead
	}
	if err != nil && err != io.EOF {
		return
	}
	err = nil

	ec = &decryptStream{
		stream: &stream,
		readBuffer: new(bytes.Buffer),
		docId:  docId,
	}
	return
}

func (stream *decryptStream) DocId() string {
	return stream.docId
}
// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
//
// BUG(derpyplops): stream crashes when 0 bytes are
// available in the stream.
func (stream *decryptStream) Read(p []byte) (n int, err error) {
	docID := stream.docId
	nPreRcvBytes, bufReadErr := stream.readBuffer.Read(p) // drain readBuffer first
	if bufReadErr != nil && bufReadErr != io.EOF {
		return 0, fmt.Errorf("readBuffer read err: [%v]", err)
	}
	nPostRcvBytes := 0
	if nPreRcvBytes == len(p) { // p runs out
		return nPreRcvBytes, nil
	} else { // if readBuffer has been completely drained
		svc := *stream.stream
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
		cipherText := res.GetPlaintext()
		stream.readBuffer.Write(cipherText)
		nPostRcvBytes, err = stream.readBuffer.Read(p[nPreRcvBytes:]) // read remaining part of readBuffer.
		if err != nil {
			return 0, fmt.Errorf("readBuffer read err: [%v]", err)
		}
		return nPreRcvBytes + nPostRcvBytes, err
	}
}

// Write writes a stream of bytes that comprise the ciphertext to be decoded.
func (stream *decryptStream) Write(cipherText []byte) (n int, err error) {
	svc := *stream.stream
	var blockSize int = 10000
	for i := 0; i < len(cipherText); i += blockSize {
		var block []byte
		if i+blockSize < len(cipherText) {
			block = cipherText[i : i+blockSize]
		} else {
			block = cipherText[i:]
		}
		n += len(block)
		dataReq := &proto.DecryptDocStreamReq{
			IdOrData: &proto.DecryptDocStreamReq_Ciphertext{
				Ciphertext: block,
			},
		}
		if err = svc.Send(dataReq); err != nil {
			return 0, fmt.Errorf("deryptStream.Write err: [%v]", err)
		}
	}
	return n,nil
}


// DecryptDocument encrypts a document with Strongdoc.
// It requires original ciphertext, since the document is not stored
func DecryptDocument(token, docID string, cipherText []byte) (plaintext []byte, err error) {
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
	for i := 0; i < len(cipherText); i += blockSize {
		var block []byte
		if i+blockSize < len(cipherText) {
			block = cipherText[i : i+blockSize]
		} else {
			block = cipherText[i:]
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

// UnshareDocument rescinds permission earlier granted for other users
// to access their documents.
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