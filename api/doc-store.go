package api

// import (
// 	"bytes"
// 	"context"
// 	"encoding/base64"
// 	"fmt"
// 	"io"

// 	"github.com/golang/protobuf/ptypes/timestamp"

// 	"github.com/overnest/strongdoc-go-sdk/client"
// 	"github.com/overnest/strongdoc-go-sdk/proto"
// 	"github.com/overnest/strongdoc-go-sdk/utils"
// 	ssc "github.com/overnest/strongsalt-crypto-go"
// )

// func decryptKeyChain(sdc client.StrongDocClient, encKeyChain []*proto.EncryptedKey) (keyBytes []byte, keyID string, keyVersion int32, err error) {
// 	if len(encKeyChain) == 0 {
// 		err = fmt.Errorf("Received no document key")
// 		return
// 	}
// 	protoPriKey := encKeyChain[0]
// 	if protoPriKey.EncryptorID != sdc.GetUserKeyID() {
// 		err = fmt.Errorf("Error decrypting user keys: User information out of date. User must log out and log back in again to continue.")
// 		return
// 	}

// 	encKeyBytes, err := base64.URLEncoding.DecodeString(protoPriKey.GetEncKey())
// 	if err != nil {
// 		return
// 	}
// 	keyBytes, err = sdc.UserDecrypt(encKeyBytes)
// 	if err != nil {
// 		return
// 	}
// 	for _, protoEncKey := range encKeyChain[1:] {
// 		key, err := ssc.DeserializeKey(keyBytes)
// 		if err != nil {
// 			return nil, "", 0, err
// 		}
// 		encKeyBytes, err := base64.URLEncoding.DecodeString(protoEncKey.GetEncKey())
// 		if err != nil {
// 			return nil, "", 0, err
// 		}
// 		keyBytes, err = key.Decrypt(encKeyBytes)
// 		if err != nil {
// 			return nil, "", 0, err
// 		}
// 	}
// 	protoEncDocKey := encKeyChain[len(encKeyChain)-1]
// 	keyID = protoEncDocKey.GetKeyID()
// 	keyVersion = protoEncDocKey.GetKeyVersion()

// 	return
// }

// // UploadDocument uploads a document to Strongdoc-provided storage.
// // It then returns a docId that uniquely identifies the document.
// func UploadDocument(sdc client.StrongDocClient, docName string, plaintext []byte) (docID string, err error) {
// 	stream, err := sdc.GetGrpcClient().UploadDocumentStream(context.Background())
// 	if err != nil {
// 		err = fmt.Errorf("UploadDocument err: [%v]", err)
// 		return
// 	}

// 	docNameReq := &proto.UploadDocStreamReq{
// 		NameOrData: &proto.UploadDocStreamReq_DocName{
// 			DocName: docName},
// 	}
// 	if err = stream.Send(docNameReq); err != nil {
// 		err = fmt.Errorf("UploadDocument err: [%v]", err)
// 		return
// 	}

// 	for i := 0; i < len(plaintext); i += blockSize {
// 		var block []byte
// 		if i+blockSize < len(plaintext) {
// 			block = plaintext[i : i+blockSize]
// 		} else {
// 			block = plaintext[i:]
// 		}

// 		dataReq := &proto.UploadDocStreamReq{
// 			NameOrData: &proto.UploadDocStreamReq_Plaintext{
// 				Plaintext: block}}
// 		if err = stream.Send(dataReq); err != nil {
// 			err = fmt.Errorf("UploadDocument err: [%v]", err)
// 			return
// 		}
// 	}

// 	if err = stream.CloseSend(); err != nil {
// 		err = fmt.Errorf("UploadDocument err: [%v]", err)
// 		return
// 	}

// 	res, err := stream.CloseAndRecv()
// 	if err != nil {
// 		err = fmt.Errorf("UploadDocument err: [%v]", err)
// 		return
// 	}
// 	if res == nil {
// 		err = fmt.Errorf("UploadDocument Resp is nil: [%v]", err)
// 		return
// 	}

// 	docID = res.GetDocID()
// 	return
// }

// // UploadDocumentStream encrypts a document with Strongdoc and stores it in Strongdoc-provided storage.
// // It accepts an io.Reader (which should contain the plaintext) and the document name.
// //
// // It then returns a docID that uniquely identifies the document.
// func UploadDocumentStream(sdc client.StrongDocClient, docName string, plainStream io.Reader) (docID string, err error) {
// 	stream, err := sdc.GetGrpcClient().UploadDocumentStream(context.Background())
// 	if err != nil {
// 		return
// 	}

// 	docNameReq := &proto.UploadDocStreamReq{
// 		NameOrData: &proto.UploadDocStreamReq_DocName{
// 			DocName: docName},
// 	}
// 	if err = stream.Send(docNameReq); err != nil {
// 		return
// 	}

// 	block := make([]byte, blockSize)
// 	for {
// 		n, inerr := plainStream.Read(block)
// 		if inerr == nil || inerr == io.EOF {
// 			if n > 0 {
// 				dataReq := &proto.UploadDocStreamReq{
// 					NameOrData: &proto.UploadDocStreamReq_Plaintext{
// 						Plaintext: block[:n]}}
// 				if err = stream.Send(dataReq); err != nil {
// 					return "", err
// 				}
// 			}
// 			if inerr == io.EOF {
// 				break
// 			}
// 		} else {
// 			err = inerr
// 			return
// 		}
// 	}

// 	if err = stream.CloseSend(); err != nil {
// 		return "", err
// 	}

// 	resp, err := stream.CloseAndRecv()
// 	if err != nil {
// 		return
// 	}

// 	docID = resp.GetDocID()
// 	return
// }

// // DownloadDocument downloads a document stored in Strongdoc-provided
// // storage. You must provide it with its docID.
// func DownloadDocument(sdc client.StrongDocClient, docID string) (plaintext []byte, err error) {
// 	req := &proto.DownloadDocStreamReq{DocID: docID}
// 	stream, err := sdc.GetGrpcClient().DownloadDocumentStream(context.Background(), req)
// 	if err != nil {
// 		return
// 	}

// 	for err == nil {
// 		res, rcverr := stream.Recv()
// 		if res != nil && docID != res.GetDocID() {
// 			err = fmt.Errorf("Reqed document %v, received document %v instead",
// 				docID, res.GetDocID())
// 			return
// 		}
// 		plaintext = append(plaintext, res.GetPlaintext()...)
// 		err = rcverr
// 	}

// 	if err == io.EOF {
// 		err = nil
// 	}

// 	return
// }

// // DownloadDocumentStream decrypts any document previously stored on
// // Strongdoc-provided storage, and makes the plaintext available via an io.Reader interface.
// // You must also pass in its docId.
// //
// // It then returns an io.Reader object that contains the plaintext of
// // the Reqed document.
// func DownloadDocumentStream(sdc client.StrongDocClient, docID string) (plainStream io.Reader, err error) {
// 	prepareReq := &proto.E2EEPrepareDownloadDocReq{DocID: docID}
// 	resp, err := sdc.GetGrpcClient().E2EEPrepareDownloadDocument(context.Background(), prepareReq)
// 	if err != nil {
// 		return
// 	}

// 	if !resp.GetDocumentAccessMetadata().GetIsAccessible() {
// 		return nil, fmt.Errorf("Cannot access document %v", docID)
// 	}

// 	if resp.GetDocumentAccessMetadata().GetIsClientSide() {
// 		docKeyBytes, _, _, err := decryptKeyChain(sdc, resp.GetDocumentAccessMetadata().GetDocKeyChain())
// 		if err != nil {
// 			return nil, err
// 		}
// 		docKey, err := ssc.DeserializeKey(docKeyBytes)
// 		if err != nil {
// 			return nil, err
// 		}
// 		decryptor, err := docKey.DecryptStream(0)
// 		if err != nil {
// 			return nil, err
// 		}

// 		mac, err := base64.URLEncoding.DecodeString(resp.GetDocumentAccessMetadata().GetMac())
// 		if err != nil {
// 			return nil, err
// 		}

// 		req := &proto.E2EEDownloadDocStreamReq{DocID: docID}
// 		stream, err := sdc.GetGrpcClient().E2EEDownloadDocumentStream(context.Background(), req)
// 		if err != nil {
// 			return nil, err
// 		}

// 		return &e2eeDownloadStream{
// 			grpcStream: &stream,
// 			grpcEOF:    false,
// 			decryptor:  decryptor,
// 			macKey:     docKey,
// 			mac:        mac,
// 			docID:      docID,
// 		}, nil
// 	}
// 	req := &proto.DownloadDocStreamReq{DocID: docID}
// 	stream, err := sdc.GetGrpcClient().DownloadDocumentStream(context.Background(), req)
// 	if err != nil {
// 		return
// 	}
// 	plainStream = &downloadStream{
// 		grpcStream: &stream,
// 		grpcEOF:    false,
// 		buffer:     new(bytes.Buffer),
// 		docID:      docID,
// 	}
// 	return
// }

// type downloadStream struct {
// 	grpcStream *proto.StrongDocService_DownloadDocumentStreamClient
// 	grpcEOF    bool
// 	buffer     *bytes.Buffer
// 	docID      string
// }

// // Read reads the length of the stream. It assumes that
// // the stream from server still has data or that the internal
// // buffer still has some data in it.
// func (stream *downloadStream) Read(p []byte) (n int, err error) {
// 	if stream.buffer.Len() > 0 {
// 		return stream.buffer.Read(p)
// 	}

// 	if stream.grpcEOF {
// 		return 0, io.EOF
// 	}

// 	resp, err := (*stream.grpcStream).Recv()
// 	if resp != nil && stream.docID != resp.GetDocID() {
// 		return 0, fmt.Errorf("Reqed document %v, received document %v instead",
// 			stream.docID, resp.GetDocID())
// 	}

// 	if err != nil {
// 		if err != io.EOF {
// 			return 0, err
// 		}
// 		stream.grpcEOF = true
// 	}

// 	if resp != nil {
// 		plaintext := resp.GetPlaintext()
// 		copied := copy(p, plaintext)
// 		if copied < len(plaintext) {
// 			stream.buffer.Write(plaintext[copied:])
// 		}
// 		return copied, nil
// 	}

// 	return 0, nil
// }

// type e2eeDownloadStream struct {
// 	grpcStream *proto.StrongDocService_E2EEDownloadDocumentStreamClient
// 	grpcEOF    bool
// 	decryptor  *ssc.Decryptor
// 	macKey     *ssc.StrongSaltKey
// 	mac        []byte
// 	docID      string
// }

// func (stream *e2eeDownloadStream) Read(p []byte) (n int, err error) {
// 	n, err = stream.decryptor.Read(p)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if n > 0 {
// 		return n, nil
// 	}

// 	if stream.grpcEOF {
// 		return 0, io.EOF
// 	}

// 	for n == 0 {
// 		resp, err := (*stream.grpcStream).Recv()
// 		if err != nil {
// 			if err != io.EOF {
// 				return 0, err
// 			}
// 			stream.grpcEOF = true
// 			err = stream.decryptor.CloseWrite()
// 			if err != nil {
// 				return 0, err
// 			}
// 			ok, err := stream.macKey.MACVerify(stream.mac)
// 			if err != nil {
// 				return 0, fmt.Errorf("Document integrity check failed with error: %v", err)
// 			}
// 			if !ok {
// 				return 0, fmt.Errorf("Document integrity check failed.")
// 			}
// 		}
// 		if resp != nil {
// 			ciphertext := resp.GetCipherText()
// 			x, err := stream.decryptor.Write(ciphertext)
// 			if err != nil {
// 				return 0, err
// 			}
// 			if x < len(ciphertext) {
// 				return 0, fmt.Errorf("Incorrect number of bytes written to decryptor. Exected: %v. Actual: %v.", len(ciphertext), x)
// 			}
// 			x, err = stream.macKey.MACWrite(ciphertext)
// 			if err != nil {
// 				return 0, err
// 			}
// 			if x < len(ciphertext) {
// 				return 0, fmt.Errorf("Incorrect number of bytes written to MAC Key. Exected: %v. Actual: %v.", len(ciphertext), x)
// 			}
// 		}

// 		n, err = stream.decryptor.Read(p)
// 	}

// 	return
// }

// // RemoveDocument deletes a document from Strongdoc-provided
// // storage. If you are a regular user, you may only remove
// // a document that belongs to you. If you are an administrator,
// // you can remove all the documents of the organization
// // for which you are an administrator.
// func RemoveDocument(sdc client.StrongDocClient, docID string) error {
// 	req := &proto.RemoveDocumentReq{DocID: docID}
// 	_, err := sdc.GetGrpcClient().RemoveDocument(context.Background(), req)
// 	return err
// }

// func BulkShareDocWithUsers(sdc client.StrongDocClient, docID string, receiverIDs []string) (
// 	isAccessible, isSharable bool, sharedReceivers, alreadyAccessibleUsers, unsharableReceivers map[string]bool, err error) {
// 	return bulkShareDoc(sdc, docID, proto.AccessType_USER, receiverIDs)
// }

// func BulkShareDocWithOrgs(sdc client.StrongDocClient, docID string, receiverIDs []string) (
// 	isAccessible, isSharable bool, sharedReceivers, alreadyAccessibleOrgs, unsharableReceivers map[string]bool, err error) {
// 	return bulkShareDoc(sdc, docID, proto.AccessType_ORG, receiverIDs)
// }

// /**
// bulkShareDoc shares the document with other receivers.
// return
// isAccessible: whether logged-in user can access the document
// isSharable: whether logged-in is allowed to share the document
// sharedReceivers: receiverIDs who has been successfully shared with
// alreadyAccessibleReceivers: receiverIDs who already have access to the document, no need to share
// unsharableReceivers: receiverIDs which are not allowed to be given the document
// */
// func bulkShareDoc(sdc client.StrongDocClient, docID string, receiverType proto.AccessType, receiverIDs []string) (
// 	isAccessible bool, isSharable bool, sharedReceivers, alreadyAccessibleReceivers, unsharableReceivers map[string]bool, err error) {
// 	sharedReceivers = make(map[string]bool)
// 	alreadyAccessibleReceivers = make(map[string]bool)
// 	unsharableReceivers = make(map[string]bool)
// 	prepareShareReq := &proto.PrepareShareDocumentReq{
// 		DocID:        docID,
// 		ReceiverType: receiverType,
// 		ReceiverIDs:  receiverIDs,
// 	}
// 	prepareShareRes, err := sdc.GetGrpcClient().PrepareShareDocument(context.Background(), prepareShareReq)
// 	if err != nil {
// 		return
// 	}
// 	accessMeta := prepareShareRes.GetAccessMetaData()

// 	isAccessible = accessMeta.GetIsAccessible()
// 	isSharable = accessMeta.GetIsSharable()

// 	// user has no access to the doc or cannot share the document
// 	if !isAccessible || !isSharable {
// 		return
// 	}
// 	// receivers who already have access, no need to share
// 	for _, receiverID := range prepareShareRes.GetReceiversWithDoc() {
// 		alreadyAccessibleReceivers[receiverID] = true
// 	}
// 	// receivers who cannot be shared with this doc
// 	for _, receiverID := range prepareShareRes.GetUnsharableReceivers() {
// 		unsharableReceivers[receiverID] = true
// 	}

// 	// the document is encrypted on client side
// 	if accessMeta.GetIsClientSide() {
// 		var docKeyBytes []byte
// 		var keyID string
// 		var keyVersion int32
// 		docKeyBytes, keyID, keyVersion, err = decryptKeyChain(sdc, accessMeta.GetDocKeyChain())
// 		if err != nil {
// 			return
// 		}

// 		for _, encryptor := range prepareShareRes.GetEncryptors() {

// 			for true { // todo
// 				var keyBytes []byte
// 				keyBytes, err = base64.URLEncoding.DecodeString(encryptor.Key)
// 				if err != nil {
// 					return
// 				}
// 				var pubKey *ssc.StrongSaltKey
// 				pubKey, err = ssc.DeserializeKey(keyBytes)
// 				if err != nil {
// 					return
// 				}
// 				var encDocKeyBytes []byte
// 				encDocKeyBytes, err = pubKey.Encrypt(docKeyBytes)
// 				shareDocReq := &proto.ShareDocumentReq{
// 					DocID:        docID,
// 					ReceiverType: receiverType,
// 					ReceiverID:   encryptor.OwnerID, // receiverID

// 					EncDocKey: &proto.EncryptedKey{
// 						EncryptorID:      encryptor.KeyID,                                   // shareUser or shareOrg public key ID
// 						EncKey:           base64.URLEncoding.EncodeToString(encDocKeyBytes), // encrypted data
// 						EncryptorVersion: encryptor.Version,                                 // encryptor (public key) version
// 						KeyID:            keyID,
// 						KeyVersion:       keyVersion,
// 					},
// 				}
// 				var shareDocRes *proto.ShareDocumentResp
// 				shareDocRes, err = sdc.GetGrpcClient().ShareDocument(context.Background(), shareDocReq)
// 				if err != nil {
// 					return
// 				}
// 				if shareDocRes.GetPubKey() != nil {
// 					encryptor = shareDocRes.GetPubKey()
// 				} else {
// 					if shareDocRes.GetReceiverAlreadyAccessible() {
// 						alreadyAccessibleReceivers[encryptor.OwnerID] = true
// 					} else if shareDocRes.GetReceiverUnsharable() {
// 						unsharableReceivers[encryptor.OwnerID] = true
// 					} else if shareDocRes.GetSuccess() {
// 						sharedReceivers[encryptor.OwnerID] = true
// 					}
// 					break
// 				}
// 			}

// 		}

// 	} else {
// 		// the document is encrypted on server side
// 		for _, receiverID := range receiverIDs {
// 			shareDocReq := &proto.ShareDocumentReq{
// 				DocID:        docID,
// 				ReceiverType: receiverType,
// 				ReceiverID:   receiverID,
// 			}
// 			var shareDocRes *proto.ShareDocumentResp
// 			shareDocRes, err = sdc.GetGrpcClient().ShareDocument(context.Background(), shareDocReq)
// 			if err != nil {
// 				return
// 			}
// 			if shareDocRes.GetReceiverAlreadyAccessible() {
// 				alreadyAccessibleReceivers[receiverID] = true
// 			} else if shareDocRes.GetReceiverUnsharable() {
// 				unsharableReceivers[receiverID] = true
// 			} else if shareDocRes.GetSuccess() {
// 				sharedReceivers[receiverID] = true
// 			}

// 		}

// 	}
// 	return
// }

// // UnshareWithUser rescinds permission granted earlier, removing other user's access to the document.
// func UnshareWithUser(sdc client.StrongDocClient, docID, userIDOrEmail string) (succ bool, alreadyNoAccess, allowed bool, err error) {
// 	return unshareDocument(sdc, docID, proto.AccessType_USER, userIDOrEmail)
// }

// // UnshareWithOrg rescinds permission granted earlier, removing other org's access to the document.
// func UnshareWithOrg(sdc client.StrongDocClient, docID, orgID string) (succ bool, alreadyNoAccess, allowed bool, err error) {
// 	return unshareDocument(sdc, docID, proto.AccessType_ORG, orgID)
// }

// func unshareDocument(sdc client.StrongDocClient, docID string, receiverType proto.AccessType, receiverID string) (succ, alreadyNoAccess, allowed bool, err error) {
// 	req := &proto.UnshareDocumentReq{
// 		DocID:        docID,
// 		ReceiverType: receiverType,
// 		ReceiverID:   receiverID,
// 	}
// 	res, err := sdc.GetGrpcClient().UnshareDocument(context.Background(), req)
// 	if err != nil {
// 		return
// 	}
// 	succ = res.GetSuccess()
// 	alreadyNoAccess = res.GetAlreadyUnshared()
// 	allowed = res.GetAllowed()
// 	return
// }

// // Document contains the document information
// type Document struct {
// 	// The document ID.
// 	DocID string
// 	// The document name.
// 	DocName string
// 	// The document size.
// 	Size uint64
// }

// // ListDocuments returns a slice of Document objects, representing
// // the documents accessible by the user.
// func ListDocuments(sdc client.StrongDocClient) (docs []Document, err error) {
// 	req := &proto.ListDocumentsReq{}
// 	res, err := sdc.GetGrpcClient().ListDocuments(context.Background(), req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	documents, err := utils.ConvertStruct(res.GetDocuments(), []Document{})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return *documents.(*[]Document), nil
// }

// // DocActionHistory contains the document information
// type DocActionHistory struct {
// 	// The document ID.
// 	DocID string

// 	UserID string
// 	// The document name.
// 	DocName string

// 	ActionTime timestamp.Timestamp

// 	ActionType string

// 	OtherUserID string
// }

// // ListDocActionHistory returns a slice of Document objects, representing
// // the documents accessible by the user.
// func ListDocActionHistory(sdc client.StrongDocClient) ([]DocActionHistory, int32, int32, error) {
// 	req := &proto.ListDocActionHistoryReq{}
// 	res, err := sdc.GetGrpcClient().ListDocActionHistory(context.Background(), req)
// 	if err != nil {
// 		return nil, 0, 0, err
// 	}

// 	docActionHistoryListReq := res.GetDocActionHistoryList()
// 	temp, err := utils.ConvertStruct(docActionHistoryListReq, []DocActionHistory{})
// 	resultTotalCount := res.GetResultTotalCount()
// 	offset := res.GetOffset()
// 	if err != nil {
// 		return nil, 0, 0, err
// 	}

// 	// return *docActionHistoryList.(*[]DocActionHistory), nil
// 	return *temp.(*[]DocActionHistory), resultTotalCount, offset, nil
// }
