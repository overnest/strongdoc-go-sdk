package api

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

// EncryptDocument encrypts a document with Strongdoc
// and returns the encrypted ciphertext without storing it on
// any storage. It accepts a the plaintext and the document name.
// The returned docId uniquely identifies the document.
func EncryptDocument(sdc client.StrongDocClient, docName string, plaintext []byte) (docID string, ciphertext []byte, err error) {
	stream, err := sdc.GetGrpcClient().EncryptDocumentStream(context.Background())
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
// the Reqed document, and a docID that uniquely
// identifies the document.
func EncryptDocumentStream(sdc client.StrongDocClient, docName string, plainStream io.Reader) (cipherStream io.Reader, docID string, err error) {
	stream, err := sdc.GetGrpcClient().EncryptDocumentStream(context.Background())
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

	docID = res.GetDocID()

	cipherStream = &encryptStream{
		grpcStream:  &stream,
		plainStream: plainStream,
		grpcEOF:     false,
		plainEOF:    false,
		buffer:      new(bytes.Buffer),
		docID:       docID,
	}
	return
}

// DecryptDocument decrypts a document with Strongdoc
// and returns the plaintext. It accepts a the cipherText and its docId.
func DecryptDocument(sdc client.StrongDocClient, docID string, cipherText []byte) (plaintext []byte, err error) {
	stream, err := sdc.GetGrpcClient().DecryptDocumentStream(context.Background())
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
		err = fmt.Errorf("The Reqed document ID %v does not match the returned ID %v",
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
// contain the ciphertext, and you must also pass in its docID.
//
// It then returns an io.Reader object that contains the plaintext of
// the Reqed document.
func DecryptDocumentStream(sdc client.StrongDocClient, docID string, cipherStream io.Reader) (plainStream io.Reader, err error) {
	stream, err := sdc.GetGrpcClient().DecryptDocumentStream(context.Background())
	if err != nil {
		return
	}

	docIDReq := &proto.DecryptDocStreamReq{
		IdOrData: &proto.DecryptDocStreamReq_DocID{
			DocID: docID,
		},
	}
	if err = stream.Send(docIDReq); err != nil {
		return
	}
	res, err := stream.Recv()
	if err != nil {
		return
	} else if res.DocID != docID {
		err = fmt.Errorf("incorrect docId Recv'd")
		return
	}

	plainStream = &decryptStream{
		grpcStream:   &stream,
		cipherStream: cipherStream,
		grpcEOF:      false,
		cipherEOF:    false,
		buffer:       new(bytes.Buffer),
		docID:        docID,
	}
	return
}

type encryptStream struct {
	grpcStream  *proto.StrongDocService_EncryptDocumentStreamClient
	plainStream io.Reader
	grpcEOF     bool
	plainEOF    bool
	buffer      *bytes.Buffer
	docID       string
}

func (stream *encryptStream) Read(p []byte) (n int, err error) {
	if stream.buffer.Len() > 0 {
		return stream.buffer.Read(p)
	}

	if stream.plainEOF && stream.grpcEOF {
		return 0, io.EOF
	}

	if !stream.plainEOF {
		plaintext := make([]byte, blockSize)
		n, err = stream.plainStream.Read(plaintext)
		if err != nil {
			if err != io.EOF {
				(*stream.grpcStream).CloseSend()
				return
			}
			stream.plainEOF = true
		}

		plaintext = plaintext[:n]
		dataReq := &proto.EncryptDocStreamReq{
			NameOrData: &proto.EncryptDocStreamReq_Plaintext{
				Plaintext: plaintext,
			},
		}
		if err = (*stream.grpcStream).Send(dataReq); err != nil {
			err = fmt.Errorf("send() err: [%v]", err)
			(*stream.grpcStream).CloseSend()
			return
		}

		if stream.plainEOF {
			err = (*stream.grpcStream).CloseSend()
		}
	}

	resp, err := (*stream.grpcStream).Recv()
	if resp != nil && stream.docID != resp.GetDocID() {
		(*stream.grpcStream).CloseSend()
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
		ciphertext := resp.GetCiphertext()
		n = copy(p, ciphertext)
		if n < len(ciphertext) {
			stream.buffer.Write(ciphertext[n:])
		}
		return n, nil
	}

	return 0, nil
}

type decryptStream struct {
	grpcStream   *proto.StrongDocService_DecryptDocumentStreamClient
	cipherStream io.Reader
	grpcEOF      bool
	cipherEOF    bool
	buffer       *bytes.Buffer
	docID        string
}

// Read reads the length of the stream. It assumes that
// the stream from server still has data or that the internal
// buffer still has some data in it.
func (stream *decryptStream) Read(p []byte) (n int, err error) {
	if stream.buffer.Len() > 0 {
		return stream.buffer.Read(p)
	}

	if stream.cipherEOF && stream.grpcEOF {
		return 0, io.EOF
	}

	if !stream.cipherEOF {
		ciphertext := make([]byte, blockSize)
		n, err = stream.cipherStream.Read(ciphertext)
		if err != nil {
			if err != io.EOF {
				(*stream.grpcStream).CloseSend()
				return
			}
			stream.cipherEOF = true
		}

		ciphertext = ciphertext[:n]
		dataReq := &proto.DecryptDocStreamReq{
			IdOrData: &proto.DecryptDocStreamReq_Ciphertext{
				Ciphertext: ciphertext,
			},
		}
		if err = (*stream.grpcStream).Send(dataReq); err != nil {
			err = fmt.Errorf("send() err: [%v]", err)
			(*stream.grpcStream).CloseSend()
			return
		}

		if stream.cipherEOF {
			err = (*stream.grpcStream).CloseSend()
		}
	}

	resp, err := (*stream.grpcStream).Recv()
	if resp != nil && stream.docID != resp.GetDocID() {
		(*stream.grpcStream).CloseSend()
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
		n = copy(p, plaintext)
		if n < len(plaintext) {
			stream.buffer.Write(plaintext[n:])
		}
		return n, nil
	}

	return 0, nil
}
