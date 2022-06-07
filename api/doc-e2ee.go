package api

// import (
// 	"encoding/base64"
// 	// "bytes"
// 	"context"
// 	"fmt"
// 	"io"

// 	"github.com/overnest/strongdoc-go-sdk/client"
// 	"github.com/overnest/strongdoc-go-sdk/proto"
// 	ssc "github.com/overnest/strongsalt-crypto-go"
// 	// "github.com/overnest/strongdoc-go-sdk/utils"
// )

// /*type EncryptedKey struct {
// 	encKey           string
// 	encryptorID      string
// 	encryptorVersion int32
// 	keyID            string
// 	keyVersion       int32
// 	ownerID          string
// }*/

// func CreateDocument(sdc client.StrongDocClient, docName string) (docID string, err error) {
// 	docKey, err := ssc.GenerateKey(ssc.Type_XChaCha20HMAC)
// 	if err != nil {
// 		return "", fmt.Errorf("UploadDocument cannot generate document key")
// 	}
// 	docKeySerialized, err := docKey.Serialize()
// 	if err != nil {
// 		err = fmt.Errorf("UploadDocument cannot serialize document key")
// 		return
// 	}

// 	stream, err := sdc.GetGrpcClient().CreateDocStream(context.Background())
// 	defer stream.CloseSend()

// 	if err != nil {
// 		err = fmt.Errorf("UploadDocument err: [%v]", err)
// 		return
// 	}

// 	preReq := &proto.CreateDocReq{
// 		CreateDocReqStage: &proto.CreateDocReq_PreMetaData{
// 			PreMetaData: &proto.CreateDocReq_PreMetaDataType{
// 				DocName: docName,
// 			},
// 		},
// 	}

// 	err = stream.Send(preReq)
// 	if err != nil {
// 		err = fmt.Errorf("UploadDocument stage PreMetaData err: [%v]", err)
// 		return
// 	}

// 	resp, err := stream.Recv()

// 	for resp.GetDocID() == "" {
// 		protoPubKeys := resp.GetEncryptors().GetPubKeys()
// 		protoEncDocKeys := make([]*proto.EncryptedKey, len(protoPubKeys))
// 		for i, protoPubKey := range protoPubKeys {
// 			keyBytes, err := base64.URLEncoding.DecodeString(protoPubKey.GetKey())
// 			if err != nil {
// 				return "", fmt.Errorf("UploadDocument error base64 decoding public key string: [%v]", err)
// 			}

// 			key, err := ssc.DeserializeKey(keyBytes)
// 			if err != nil {
// 				return "", fmt.Errorf("UploadDocument error deserializing public key bytes: [%v]", err)
// 			}

// 			encDocKey, err := key.Encrypt(docKeySerialized)
// 			if err != nil {
// 				return "", fmt.Errorf("UploadDocument error encrypting document key: [%v]", err)
// 			}
// 			encDocKeyBase64 := base64.URLEncoding.EncodeToString(encDocKey)

// 			encKey := &proto.EncryptedKey{
// 				EncryptorID:      protoPubKey.GetKeyID(),
// 				OwnerID:          protoPubKey.GetOwnerID(),
// 				OwnerType:        protoPubKey.GetOwnerType(),
// 				EncryptorVersion: protoPubKey.GetVersion(),
// 				EncKey:           encDocKeyBase64,
// 			}

// 			protoEncDocKeys[i] = encKey
// 		}

// 		req := &proto.CreateDocReq{
// 			CreateDocReqStage: &proto.CreateDocReq_EncDocKeys{
// 				EncDocKeys: &proto.CreateDocReq_EncKeyList{
// 					EncDocKeys: protoEncDocKeys,
// 				},
// 			},
// 		}
// 		err = stream.Send(req)
// 		if err != nil {
// 			return "", fmt.Errorf("UploadDocument error sending encrypted document keys: [%v]", err)
// 		}

// 		resp, err = stream.Recv()
// 		if err != nil {
// 			return "", fmt.Errorf("UploadDocument error receiving response after sending document keys: [%v]", err)
// 		}
// 	}

// 	docID = resp.GetDocID()

// 	if err = stream.CloseSend(); err != nil {
// 		return "", err
// 	}

// 	_, err = stream.Recv()
// 	if err == io.EOF {
// 		err = nil
// 	}
// 	if err != nil {
// 		return "", err
// 	}

// 	return
// }

// func UpdateDocument(sdc client.StrongDocClient, docID string, skipIndexing bool, plainStream io.Reader) (docVersion uint64, err error) {
// 	stream, err := sdc.GetGrpcClient().UpdateDocStream(context.Background())
// 	defer stream.CloseSend()

// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument err: [%v]", err)
// 		return
// 	}

// 	preReq := &proto.UpdateDocReq{
// 		UpdateDocReqStage: &proto.UpdateDocReq_PreMetaData{
// 			PreMetaData: &proto.UpdateDocReq_PreMetaDataType{
// 				DocID:        docID,
// 				SkipIndexing: skipIndexing,
// 			},
// 		},
// 	}

// 	err = stream.Send(preReq)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument stage PreMetaData err: [%v]", err)
// 		return
// 	}

// 	resp, err := stream.Recv()
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error receiving PreMetaData: [%v]", err)
// 		return
// 	}
// 	docAccess := resp.GetDocumentAccessMetadata()
// 	if docAccess == nil {
// 		err = fmt.Errorf("Document Access Metadata is nil")
// 		return
// 	}
// 	if !docAccess.GetIsAccessible() {
// 		err = fmt.Errorf("UpdateDocument: User %v does not have acccess to docID: %v", sdc.GetUserID(), docID)
// 		return
// 	}

// 	keyBytes, _, _, err := decryptKeyChain(sdc, docAccess.GetDocKeyChain())
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error decrypting document key: [%v]", err)
// 		return
// 	}

// 	docKey, err := ssc.DeserializeKey(keyBytes)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument: error deserializing document key: [%v]", err)
// 		return
// 	}

// 	encryptor, err := docKey.EncryptStream()
// 	if err != nil {
// 		return
// 	}

// 	nonce := encryptor.GetNonce()
// 	n, err := docKey.MACWrite(nonce)
// 	if err != nil {
// 		return
// 	}
// 	if n != len(nonce) {
// 		err = fmt.Errorf("UpdateDocument wrong number of bytes written to MAC. Expected: %v. Actual: %v.", len(nonce), n)
// 	}

// 	dataReq := &proto.UpdateDocReq{
// 		UpdateDocReqStage: &proto.UpdateDocReq_CipherText{
// 			CipherText: nonce,
// 		},
// 	}

// 	err = stream.Send(dataReq)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error sending ciphertext: [%v]", err)
// 		return
// 	}

// 	block := make([]byte, blockSize)
// 	for {
// 		n, inerr := plainStream.Read(block)
// 		if inerr == nil || inerr == io.EOF {
// 			ciphertext := make([]byte, n)
// 			if n > 0 {
// 				m, xerr := encryptor.Write(block[:n])
// 				if xerr != nil {
// 					err = xerr
// 					return
// 				}
// 				if m != n {
// 					err = fmt.Errorf("UpdateDocument: Wrote wrong number of bytes to encryptor. %v instead of %v", m, n)
// 					return
// 				}
// 				m, xerr = encryptor.Read(ciphertext)
// 				if xerr != nil {
// 					err = xerr
// 					return
// 				}
// 				ciphertext = ciphertext[:m]
// 			} else {
// 				data, xerr := encryptor.ReadLast()
// 				if xerr != nil {
// 					err = xerr
// 					return
// 				}
// 				ciphertext = data
// 			}
// 			if len(ciphertext) > 0 {
// 				n, err = docKey.MACWrite(ciphertext)
// 				if err != nil {
// 					return 0, fmt.Errorf("UpdateDocument error writing ciphertext to MAC: [%v]", err)
// 				}
// 				if n != len(ciphertext) {
// 					return 0, fmt.Errorf("UpdateDocument: Wrong number of bytes written to MAC. Expected: %v. Actual: %v.", len(ciphertext), n)
// 				}

// 				dataReq := &proto.UpdateDocReq{
// 					UpdateDocReqStage: &proto.UpdateDocReq_CipherText{
// 						CipherText: ciphertext,
// 					},
// 				}

// 				err = stream.Send(dataReq)
// 				if err != nil {
// 					err = fmt.Errorf("UpdateDocument error sending ciphertext: [%v]", err)
// 					return
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

// 	docMAC, err := docKey.MACSum(nil)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error reading final MAC: [%v]", err)
// 		return
// 	}

// 	req := &proto.UpdateDocReq{
// 		UpdateDocReqStage: &proto.UpdateDocReq_PostMetaData{
// 			PostMetaData: &proto.UpdateDocReq_PostMetaDataType{
// 				MacOfCipherText: base64.URLEncoding.EncodeToString(docMAC),
// 			},
// 		},
// 	}

// 	err = stream.Send(req)
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error sending PostMetaData: [%v]", err)
// 		return
// 	}

// 	resp, err = stream.Recv()
// 	if err != nil {
// 		err = fmt.Errorf("UpdateDocument error receiving DocID: [%v]", err)
// 	}
// 	docVersion = resp.GetVersion()

// 	if err = stream.CloseSend(); err != nil {
// 		return 0, err
// 	}

// 	_, err = stream.Recv()
// 	if err == io.EOF {
// 		err = nil
// 	}
// 	if err != nil {
// 		return 0, err
// 	}

// 	return
// }

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely identifies the document.
/*func E2EEUploadDocument(sdc client.StrongDocClient, docName string, plainStream io.Reader) (docID string, err error) {
	docKey, err := ssc.GenerateKey(ssc.Type_XChaCha20HMAC)
	if err != nil {
		return "", fmt.Errorf("UploadDocument cannot generate document key")
	}
	docKeySerialized, err := docKey.Serialize()
	if err != nil {
		err = fmt.Errorf("UploadDocument cannot serialize document key")
		return
	}

	stream, err := sdc.GetGrpcClient().E2EEUploadDocumentStream(context.Background())
	defer stream.CloseSend()

	if err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	preReq := &proto.E2EEUploadDocStreamReq{
		UploadReqStageData: &proto.E2EEUploadDocStreamReq_PreMetaData{
			PreMetaData: &proto.E2EEUploadDocStreamReq_PreMetaDataType{
				DocName: docName,
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		err = fmt.Errorf("UploadDocument stage PreMetaData err: [%v]", err)
		return
	}

	resp, err := stream.Recv()

	for !resp.GetReadyForData() {
		protoPubKeys := resp.GetEncryptors().GetPubKeys()
		protoEncDocKeys := make([]*proto.EncryptedKey, len(protoPubKeys))
		for i, protoPubKey := range protoPubKeys {
			keyBytes, err := base64.URLEncoding.DecodeString(protoPubKey.GetKey())
			if err != nil {
				return "", fmt.Errorf("UploadDocument error base64 decoding public key string: [%v]", err)
			}

			key, err := ssc.DeserializeKey(keyBytes)
			if err != nil {
				return "", fmt.Errorf("UploadDocument error deserializing public key bytes: [%v]", err)
			}

			encDocKey, err := key.Encrypt(docKeySerialized)
			if err != nil {
				return "", fmt.Errorf("UploadDocument error encrypting document key: [%v]", err)
			}
			encDocKeyBase64 := base64.URLEncoding.EncodeToString(encDocKey)

			encKey := &proto.EncryptedKey{
				EncryptorID:      protoPubKey.GetKeyID(),
				OwnerID:          protoPubKey.GetOwnerID(),
				OwnerType:        protoPubKey.GetOwnerType(),
				EncryptorVersion: protoPubKey.GetVersion(),
				EncKey:           encDocKeyBase64,
			}

			protoEncDocKeys[i] = encKey
		}

		req := &proto.E2EEUploadDocStreamReq{
			UploadReqStageData: &proto.E2EEUploadDocStreamReq_EncDocKeys{
				EncDocKeys: &proto.E2EEUploadDocStreamReq_EncKeyList{
					EncDocKeys: protoEncDocKeys,
				},
			},
		}
		err = stream.Send(req)
		if err != nil {
			return "", fmt.Errorf("UploadDocument error sending encrypted document keys: [%v]", err)
		}

		resp, err = stream.Recv()
		if err != nil {
			return "", fmt.Errorf("UploadDocument error receiving response after sending document keys: [%v]", err)
		}
	}

	encryptor, err := docKey.EncryptStream()
	if err != nil {
		return
	}

	nonce := encryptor.GetNonce()
	n, err := docKey.MACWrite(nonce)
	if err != nil {
		return
	}
	if n != len(nonce) {
		err = fmt.Errorf("UploadDocument wrong number of bytes written to MAC. Expected: %v. Actual: %v.", len(nonce), n)
	}

	dataReq := &proto.E2EEUploadDocStreamReq{
		UploadReqStageData: &proto.E2EEUploadDocStreamReq_CipherText{
			CipherText: nonce,
		},
	}

	err = stream.Send(dataReq)
	if err != nil {
		err = fmt.Errorf("UploadDocument error sending ciphertext: [%v]", err)
		return
	}

	block := make([]byte, blockSize)
	for {
		n, inerr := plainStream.Read(block)
		if inerr == nil || inerr == io.EOF {
			ciphertext := make([]byte, n)
			if n > 0 {
				m, xerr := encryptor.Write(block[:n])
				if xerr != nil {
					err = xerr
					return
				}
				if m != n {
					err = fmt.Errorf("Wrote wrong number of bytes to encryptor. %v instead of %v", m, n)
					return
				}
				m, xerr = encryptor.Read(ciphertext)
				if xerr != nil {
					err = xerr
					return
				}
				ciphertext = ciphertext[:m]
			} else {
				data, xerr := encryptor.ReadLast()
				if xerr != nil {
					err = xerr
					return
				}
				ciphertext = data
			}
			if len(ciphertext) > 0 {
				n, err = docKey.MACWrite(ciphertext)
				if err != nil {
					return "", fmt.Errorf("UploadDocument error writing ciphertext to MAC: [%v]", err)
				}
				if n != len(ciphertext) {
					return "", fmt.Errorf("UploadDocument: Wrong number of bytes written to MAC. Expected: %v. Actual: %v.", len(ciphertext), n)
				}

				dataReq := &proto.E2EEUploadDocStreamReq{
					UploadReqStageData: &proto.E2EEUploadDocStreamReq_CipherText{
						CipherText: ciphertext,
					},
				}

				err = stream.Send(dataReq)
				if err != nil {
					err = fmt.Errorf("UploadDocument error sending ciphertext: [%v]", err)
					return
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

	docMAC, err := docKey.MACSum(nil)
	if err != nil {
		err = fmt.Errorf("UploadDocument error reading final MAC: [%v]", err)
		return
	}

	req := &proto.E2EEUploadDocStreamReq{
		UploadReqStageData: &proto.E2EEUploadDocStreamReq_PostMetaData{
			PostMetaData: &proto.E2EEUploadDocStreamReq_PostMetaDataType{
				MacOfCipherText: base64.URLEncoding.EncodeToString(docMAC),
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		err = fmt.Errorf("UploadDocument error sending PostMetaData: [%v]", err)
		return
	}

	resp, err = stream.Recv()
	if err != nil {
		err = fmt.Errorf("UploadDocument error receiving DocID: [%v]", err)
	}
	docID = resp.GetDocID()

	if err = stream.CloseSend(); err != nil {
		return "", err
	}

	_, err = stream.Recv()
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return "", err
	}

	return
}*/

//func e2eeDownloadDocumentStream(sdc client.StrongDocClient, docID string, prepareResp *proto.E2EEPrepareDownloadDocResp) (io.Reader, error)
