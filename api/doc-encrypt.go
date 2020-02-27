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

// EncryptDocument encrypts a document with Strongdoc
// and returns the encrypted ciphertext without storing it on
// any storage. It accepts a the plaintext and the document name.
// The returned docId uniquely identifies the document.
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

// EncryptDocumentStream encrypts a document with Strongdoc
// and makes the encrypted ciphertext available via an io.Reader interface.
// It accepts an io.Reader (which should
// contain the plaintext) and the document name.
//
// It then returns an io.Reader object that contains the ciphertext of
// the requested document, and a docId that uniquely
// identifies the document.
func EncryptDocumentStream(token string, docName string, plainStream io.Reader) (downloadStream io.Reader, docId string, err error) {
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
	}
	if err != nil && err != io.EOF {
		return
	}
	err = nil

	// close stream here
	err = stream.CloseSend()
	if err != nil {
		return
	}

	downloadStream = &encryptStream{
		stream: &stream,
		readBuffer: new(bytes.Buffer),
		docId:  docId,
	}
	return
}

// DecryptDocument decrypts a document with Strongdoc
// and returns the plaintext. It accepts a the cipherText and its docId.
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

// DecryptDocumentStream decrypts any document previously decrypted with Strongdoc
// and makes the plaintext available via an io.Reader interface.
// It accepts an io.Reader, which should
// contain the ciphertext, and you must also pass in its docId.
//
// It then returns an io.Reader object that contains the plaintext of
// the requested document.
func DecryptDocumentStream(token string, docId string, cipherStream io.Reader) (downloadStream io.Reader, err error) {
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
	for err != io.EOF {
		block := make([]byte, blockSize)
		numRead, readErr := cipherStream.Read(block)
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
	}
	if err != nil && err != io.EOF {
		return
	}
	err = nil

	// close stream here
	err = stream.CloseSend()
	if err != nil {
		return
	}

	downloadStream = &decryptStream{
		stream: &stream,
		readBuffer: new(bytes.Buffer),
		docId:  docId,
	}
	return
}

type encryptStream struct {
	stream *proto.StrongDocService_EncryptDocumentStreamClient
	readBuffer *bytes.Buffer
	docId string
}

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
	for i := 0; i < len(plaintext); i += blockSize {
		var block []byte
		if i+blockSize < len(plaintext) {
			block = plaintext[i : i+blockSize]
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

func (stream *encryptStream) DocId() string {
	return stream.docId
}

type decryptStream struct {
	stream *proto.StrongDocService_DecryptDocumentStreamClient
	readBuffer *bytes.Buffer
	docId string
}

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
		nPostRcvBytes, bufReadErr = stream.readBuffer.Read(p[nPreRcvBytes:]) // read remaining part of readBuffer.
		if bufReadErr != nil && bufReadErr != io.EOF {
			return 0, fmt.Errorf("readBuffer read err: [%v]", err)
		}
		return nPreRcvBytes + nPostRcvBytes, err
	}
}

func (stream *decryptStream) Write(cipherText []byte) (n int, err error) {
	svc := *stream.stream
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

func (stream *decryptStream) Close() (err error) {
	svc := *stream.stream
	err = svc.CloseSend()
	return
}

func (stream *decryptStream) DocId() string {
	return stream.docId
}
