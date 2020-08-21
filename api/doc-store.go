package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	ssc "github.com/overnest/strongsalt-crypto-go"

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

// downloadDocumentStream decrypts any document previously stored on
// Strongdoc-provided storage, and makes the plaintext available via an io.Reader interface.
// You must also pass in its docId.
//
// It then returns an io.Reader object that contains the plaintext of
// the Reqed document.
func downloadDocumentStream(docID string) (plainStream io.ReadCloser, err error) {
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

func (stream *downloadStream) Close() error {
	return nil
}

func UploadDocumentStreamE2EE(docName string, plainStream io.Reader) (docID string, err error) {
	manager, err := client.GetStrongDocManager()
	if err != nil {
		return
	}

	sdc := manager.GetClient()

	req := &proto.GetOwnKeysReq{}
	resp, err := sdc.GetOwnKeys(context.Background(), req)

	// Get keys from response
	pubKeys := resp.GetUserPubKeys()

	userPub, orgPubs, err := deserializeUserPubKeys(pubKeys)
	if err != nil {
		return
	}

	// Generate doc key and MAC key
	docKey, err := ssc.GenerateKey(ssc.Type_XChaCha20)
	if err != nil {
		return
	}
	serialDocKey, err := docKey.Serialize()
	if err != nil {
		return
	}

	MACKey, err := ssc.GenerateKey(ssc.Type_HMACSha512)
	if err != nil {
		return
	}
	serialMACKey, err := MACKey.Serialize()
	if err != nil {
		return
	}

	// Encrypt keys
	userEncDocKey, err := userPub.key.Encrypt(serialDocKey)
	if err != nil {
		return
	}
	userEncMACKey, err := userPub.key.Encrypt(serialMACKey)
	if err != nil {
		return
	}

	protoUserEncDocKey := &proto.EncryptedKey{
		EncKey:      base64.URLEncoding.EncodeToString(userEncDocKey),
		EncryptorID: userPub.keyID,
		OwnerID:     userPub.ownerID,
	}
	protoUserEncMACKey := &proto.EncryptedKey{
		EncKey:      base64.URLEncoding.EncodeToString(userEncMACKey),
		EncryptorID: userPub.keyID,
		OwnerID:     userPub.ownerID,
	}

	protoOrgEncDocKeys := make([]*proto.EncryptedKey, 0)
	protoOrgEncMACKeys := make([]*proto.EncryptedKey, 0)

	for _, orgPub := range orgPubs {
		orgEncDocKey, err := orgPub.key.Encrypt(serialDocKey)
		if err != nil {
			return "", err
		}

		orgEncMACKey, err := orgPub.key.Encrypt(serialMACKey)
		if err != nil {
			return "", err
		}

		protoOrgEncDocKeys = append(protoOrgEncDocKeys, &proto.EncryptedKey{
			EncKey:      base64.URLEncoding.EncodeToString(orgEncDocKey),
			EncryptorID: orgPub.keyID,
			OwnerID:     orgPub.ownerID,
		})
		protoOrgEncDocKeys = append(protoOrgEncMACKeys, &proto.EncryptedKey{
			EncKey:      base64.URLEncoding.EncodeToString(orgEncMACKey),
			EncryptorID: orgPub.keyID,
			OwnerID:     orgPub.ownerID,
		})
	}

	// Begin streaming
	stream, err := sdc.UploadDocumentStream(context.Background())
	if err != nil {
		return
	}

	preMetadata := &proto.UploadDocPreMetadata{
		DocName:       docName,
		ClientSide:    true,
		UserEncDocKey: protoUserEncDocKey,
		UserEncMACKey: protoUserEncMACKey,
		OrgEncDocKeys: protoOrgEncDocKeys,
		OrgEncMACKeys: protoOrgEncMACKeys,
	}
	preMetadataReq := &proto.UploadDocStreamReq{
		NameOrData: &proto.UploadDocStreamReq_PreMetadata{
			PreMetadata: preMetadata},
	}
	if err = stream.Send(preMetadataReq); err != nil {
		return
	}

	encryptor, err := docKey.EncryptStream()
	if err != nil {
		return
	}

	block := make([]byte, blockSize)
	for {
		n, inerr := plainStream.Read(block)
		if inerr == nil || inerr == io.EOF {
			if n > 0 {
				_, _ = encryptor.Write(block)

				numCiph, _ := encryptor.Read(block)

				ciphertextBlock := block[:numCiph]

				numMac, err := MACKey.MACWrite(ciphertextBlock)
				if numMac != len(ciphertextBlock) {
					return "", err
				}

				dataReq := &proto.UploadDocStreamReq{
					NameOrData: &proto.UploadDocStreamReq_Data{
						Data: ciphertextBlock}}
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

	mac, err := MACKey.MACSum(nil)
	if err != nil {
		return
	}

	postMetadata := &proto.UploadDocPostMetadata{
		Mac: base64.URLEncoding.EncodeToString(mac),
	}
	postMetadataReq := &proto.UploadDocStreamReq{
		NameOrData: &proto.UploadDocStreamReq_PostMetadata{
			PostMetadata: postMetadata},
	}
	if err = stream.Send(postMetadataReq); err != nil {
		return
	}

	if err = stream.CloseSend(); err != nil {
		return "", err
	}

	streamResp, err := stream.CloseAndRecv()
	if err != nil {
		return
	}

	docID = streamResp.GetDocID()
	return
}

func DownloadDocumentStream(docID string) (plainStream io.ReadCloser, err error) {
	manager, err := client.GetStrongDocManager()
	if err != nil {
		return
	}

	sdc := manager.GetClient()

	prepareReq := &proto.PrepareDownloadDocReq{
		DocID: docID,
	}

	prepareResp, err := sdc.PrepareDownloadDoc(context.Background(), prepareReq)
	if err != nil {
		return
	}

	docAccessMeta := prepareResp.GetDocumentAccessMetadata()

	if docAccessMeta.GetIsEncryptedByClientSide() {
		return downloadDocumentStreamE2EE(docID, prepareResp)
	} else {
		return downloadDocumentStream(docID)
	}
}

func serialKeysFromDocAccessMeta(manager client.StrongDocManager, docAccessMeta *proto.DocumentAccessMetadata) (serialDocKey, serialMACKey []byte, err error) {
	if docAccessMeta.GetUserAsymKeyEncryptorId() != manager.GetPasswordKeyID() {
		return nil, nil, fmt.Errorf("User must login again.")
	}

	var docKeyEncryptor *ssc.StrongSaltKey
	if docAccessMeta.GetIsKeyOrgs() {
		encSerialAsymKey, err := base64.URLEncoding.DecodeString(docAccessMeta.GetEncUserAsymKey())
		if err != nil {
			return nil, nil, err
		}
		serialAsymKey, err := manager.GetPasswordKey().Decrypt(encSerialAsymKey)
		if err != nil {
			return nil, nil, err
		}
		docKeyEncryptor, err = ssc.DeserializeKey(serialAsymKey)
		if err != nil {
			return nil, nil, err
		}
	} else {
		docKeyEncryptor = manager.GetPasswordKey()
	}

	encSerialAsymKey, err := base64.URLEncoding.DecodeString(docAccessMeta.GetDocKeyEncryptor())
	if err != nil {
		return nil, nil, err
	}
	serialAsymKey, err := docKeyEncryptor.Decrypt(encSerialAsymKey)
	if err != nil {
		return nil, nil, err
	}
	asymKey, err := ssc.DeserializeKey(serialAsymKey)
	if err != nil {
		return nil, nil, err
	}

	encSerialDocKey, err := base64.URLEncoding.DecodeString(docAccessMeta.GetEncDocKey())
	if err != nil {
		return nil, nil, err
	}
	serialDocKey, err = asymKey.Decrypt(encSerialDocKey)
	if err != nil {
		return nil, nil, err
	}

	encMACKey, err := base64.URLEncoding.DecodeString(docAccessMeta.GetEncMACKey())
	if err != nil {
		return nil, nil, err
	}
	serialMACKey, err = asymKey.Decrypt(encMACKey)
	if err != nil {
		return nil, nil, err
	}
	return
}

func downloadDocumentStreamE2EE(docID string, prepareResp *proto.PrepareDownloadDocResp) (plainStream io.ReadCloser, err error) {
	/*
		- Get password key, deserialize, generate key
		- Get private key, decrypt with password key, deserialize
		- Get doc key, decrypt with private key, deserialize
		- Get hmac key, decrypt with private key, deserialize
		- Get MAC
		- Get Decryptor using doc key
		- Get GRPC Stream, return ReadCloser which reads from stream:
			-writes to Decryptor
			-writes to HMAC
			-Reads from Decryptor (readlast() at EOF)
		- At Close: Verify MAC, Return Error if MAC fails
	*/
	manager, err := client.GetStrongDocManager()
	if err != nil {
		return
	}

	sdc := manager.GetClient()

	docAccessMeta := prepareResp.GetDocumentAccessMetadata()

	serialDocKey, serialMACKey, err := serialKeysFromDocAccessMeta(manager, docAccessMeta)
	if err != nil {
		return nil, err
	}

	docKey, err := ssc.DeserializeKey(serialDocKey)
	if err != nil {
		return
	}
	MACKey, err := ssc.DeserializeKey(serialMACKey)
	if err != nil {
		return
	}

	decryptor, err := docKey.DecryptStream(0)
	if err != nil {
		return
	}

	mac, err := base64.URLEncoding.DecodeString(docAccessMeta.GetMac())
	if err != nil {
		return
	}

	req := &proto.DownloadDocStreamReq{DocID: docID}
	stream, err := sdc.DownloadDocumentStream(context.Background(), req)
	if err != nil {
		return
	}
	plainStream = &downloadStreamE2EE{
		grpcStream: &stream,
		grpcEOF:    false,
		buffer:     new(bytes.Buffer),
		docID:      docID,
		decryptor:  decryptor,
		macKey:     MACKey,
		mac:        mac,
	}

	return
}

type downloadStreamE2EE struct {
	grpcStream *proto.StrongDocService_DownloadDocumentStreamClient
	grpcEOF    bool
	buffer     *bytes.Buffer
	docID      string
	decryptor  *ssc.Decryptor
	macKey     *ssc.StrongSaltKey
	mac        []byte
}

func (stream *downloadStreamE2EE) Read(p []byte) (n int, err error) {
	return
}

func (stream *downloadStreamE2EE) Close() error {
	return nil
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

type Key struct {
	key     *ssc.StrongSaltKey
	keyID   string
	ownerID string
}

func deserializeUserPubKeys(pubKeys *proto.UserPubKeys) (userKey *Key, orgKeys []*Key, err error) {
	protoUserPub := pubKeys.GetUserPubKey()
	serialUserPub, err := base64.URLEncoding.DecodeString(protoUserPub.GetPubKey())
	if err != nil {
		return
	}
	userPubKey, err := ssc.DeserializeKey(serialUserPub)
	if err != nil {
		return
	}

	userKey = &Key{
		key:     userPubKey,
		keyID:   protoUserPub.GetKeyID(),
		ownerID: protoUserPub.GetOwnerID(),
	}

	orgKeys = make([]*Key, 0)
	for _, orgPub := range pubKeys.GetOrgPubKeys() {
		serialOrgPub, err := base64.URLEncoding.DecodeString(orgPub.GetPubKey())
		if err != nil {
			return nil, nil, err
		}
		orgPubKey, err := ssc.DeserializeKey(serialOrgPub)
		if err != nil {
			return nil, nil, err
		}
		orgKeys = append(orgKeys, &Key{
			key:     orgPubKey,
			keyID:   orgPub.GetKeyID(),
			ownerID: orgPub.GetOwnerID(),
		})
	}
	return
}

// ShareDocument shares the document with other users.
// Note that the user that you are sharing with be be in an organization
// that has been declared available for sharing with the Add Sharable Organizations function.
func ShareDocument(docID, userID string) (success bool, err error) {
	manager, err := client.GetStrongDocManager()
	if err != nil {
		return
	}

	sdc := manager.GetClient()

	prepareReq := &proto.PrepareShareDocumentReq{}
	prepareResp, err := sdc.PrepareShareDocument(context.Background(), prepareReq)
	if err != nil {
		return
	}

	docAccessMeta := prepareResp.GetDocumentAccessMetadata()

	if docAccessMeta.GetIsEncryptedByClientSide() {
		serialDocKey, serialMACKey, err := serialKeysFromDocAccessMeta(manager, docAccessMeta)
		if err != nil {
			return false, err
		}

		userPubs := prepareResp.GetToUserPubKeys()

		toUserPub, toOrgPubs, err := deserializeUserPubKeys(userPubs)
		if err != nil {
			return false, err
		}

		userEncDocKey, err := toUserPub.key.Encrypt(serialDocKey)
		if err != nil {
			return false, err
		}
		userEncMACKey, err := toUserPub.key.Encrypt(serialMACKey)
		if err != nil {
			return false, err
		}

		protoUserEncDocKey := &proto.EncryptedKey{
			EncKey:      base64.URLEncoding.EncodeToString(userEncDocKey),
			EncryptorID: toUserPub.keyID,
			OwnerID:     toUserPub.ownerID,
		}

		protoUserEncMACKey := &proto.EncryptedKey{
			EncKey:      base64.URLEncoding.EncodeToString(userEncMACKey),
			EncryptorID: toUserPub.keyID,
			OwnerID:     toUserPub.ownerID,
		}

		orgEncDocKeys := make([]*proto.EncryptedKey, 0)
		orgEncMACKeys := make([]*proto.EncryptedKey, 0)
		for _, orgPub := range toOrgPubs {
			orgEncDocKey, err := orgPub.key.Encrypt(serialDocKey)
			if err != nil {
				return false, err
			}
			orgEncMACKey, err := orgPub.key.Encrypt(serialMACKey)
			if err != nil {
				return false, err
			}

			orgEncDocKeys = append(orgEncDocKeys, &proto.EncryptedKey{
				EncKey:      base64.URLEncoding.EncodeToString(orgEncDocKey),
				EncryptorID: orgPub.keyID,
				OwnerID:     orgPub.ownerID,
			})

			orgEncDocKeys = append(orgEncDocKeys, &proto.EncryptedKey{
				EncKey:      base64.URLEncoding.EncodeToString(orgEncMACKey),
				EncryptorID: orgPub.keyID,
				OwnerID:     orgPub.ownerID,
			})
		}

		req := &proto.ShareDocumentReq{
			DocID:         docID,
			UserID:        userID,
			UserEncDocKey: protoUserEncDocKey,
			UserEncMACKey: protoUserEncMACKey,
			OrgEncDocKeys: orgEncDocKeys,
			OrgEncMACKeys: orgEncMACKeys,
		}
		res, err := sdc.ShareDocument(context.Background(), req)
		if err != nil {
			fmt.Printf("Failed to share: %v", err)
			return false, err
		}
		success = res.Success
	} else {
		req := &proto.ShareDocumentReq{
			DocID:  docID,
			UserID: userID,
		}
		res, err := sdc.ShareDocument(context.Background(), req)
		if err != nil {
			fmt.Printf("Failed to share: %v", err)
			return false, err
		}
		success = res.Success
	}
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
