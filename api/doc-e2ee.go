package api

import (
	"encoding/base64"
	// "bytes"
	"context"
	"fmt"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	ssc "github.com/overnest/strongsalt-crypto-go"
	// "github.com/overnest/strongdoc-go-sdk/utils"
)

type EncryptedKey struct {
	encKey           string
	encryptorID      string
	encryptorVersion int32
	keyID            string
	keyVersion       int32
	ownerID          string
}

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely identifies the document.
/*func E2EEUploadDocument_Old(docName string, plainStream io.Reader) (docID string, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	passwordKey := sdc.GetPasswordKey()

	stream, err := sdc.GetGrpcClient().E2EEUploadDocumentStream(context.Background())
	if err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	preReq := &proto.E2EEUploadDocStreamReq{
		StagesOfUploading: &proto.E2EEUploadDocStreamReq_PreMetaData_{
			PreMetaData: &proto.E2EEUploadDocStreamReq_PreMetaData{
				DocName: docName,
				// IsEncryptedByClient: true,
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		err = fmt.Errorf("UploadDocument stage PreMetaData err: [%v]", err)
		return
	}

	userPrivKeysReq := &proto.GetUserPrivateKeysReq{}
	userPrivKeysResp, err := sdc.GetGrpcClient().GetUserPrivateKeys(context.Background(), userPrivKeysReq)
	if err != nil {
		err = fmt.Errorf("UploadDocument cannot get user private keys")
		return
	}
	userPrivKeys := userPrivKeysResp.GetEncryptedKeys()
	docKey, err := ssc.GenerateKey(ssc.Type_Secretbox)
	if err != nil {
		return "", fmt.Errorf("UploadDocument cannot generate document key")
	}
	docKeySerialized, err := docKey.Serialize()
	if err != nil {
		err = fmt.Errorf("UploadDocument cannot serialize document key")
		return
	}

	encKeyListReq := &proto.E2EEUploadDocStreamReq{
		StagesOfUploading: &proto.E2EEUploadDocStreamReq_EncDocKeys{
			EncDocKeys: &proto.E2EEUploadDocStreamReq_EncKeyList{
				EncDocKeys: make([]*proto.EncryptedKey, 3),
			},
		},
	}

	for _, userPrivKey := range userPrivKeys {
		// TODO: Check user password key version and id
		decryptedUserPrivKey, err := passwordKey.Decrypt([]byte(userPrivKey.GetEncKey()))
		if err != nil {
			return "", fmt.Errorf("UploadDocument cannot decrypt user private key")
		}
		userPrivKey, err := ssc.DeserializeKey(decryptedUserPrivKey)
		if err != nil {
			return "", fmt.Errorf("UploadDocument cannot deserialize user private key")
		}
		encryptedDocKey, err := userPrivKey.Encrypt(docKeySerialized)
		if err != nil {
			return "", fmt.Errorf("UploadDocument cannot encrypt document key")
		}
	}

	block := make([]byte, blockSize)
	for {
		n, inerr := plainStream.Read(block)
		if inerr == nil || inerr == io.EOF {
			if n > 0 {
				dataReq := &proto.E2EEUploadDocStreamReq{
					StagesOfUploading: &proto.E2EEUploadDocStreamReq_CipherText{
						//TODO: need encryption
						CipherText: block[:n]}}
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
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		err = fmt.Errorf("UploadDocument err: [%v]", err)
		return
	}

	if resp == nil {
		err = fmt.Errorf("UploadDocument Resp is nil: [%v]", err)
		return
	}

	docID = resp.GetDocID()
	return
}*/

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely identifies the document.
func E2EEUploadDocument(sdc client.StrongDocClient, docName string, plainStream io.Reader) (docID string, err error) {
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
		StageData: &proto.E2EEUploadDocStreamReq_PreMetaData{
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
			StageData: &proto.E2EEUploadDocStreamReq_EncDocKeys{
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

	req := &proto.E2EEUploadDocStreamReq{
		StageData: &proto.E2EEUploadDocStreamReq_CipherText{
			CipherText: []byte("TODO: actual data"),
		},
	}

	err = stream.Send(req)
	if err != nil {
		err = fmt.Errorf("UploadDocument error sending ciphertext: [%v]", err)
		return
	}

	_, err = docKey.MACWrite([]byte("TODO: actual data"))
	if err != nil {
		return "", fmt.Errorf("UploadDocument error writing ciphertext to MAC: [%v]", err)
	}

	docMAC, err := docKey.MACSum(nil)
	if err != nil {
		err = fmt.Errorf("UploadDocument error reading final MAC: [%v]", err)
		return
	}

	req = &proto.E2EEUploadDocStreamReq{
		StageData: &proto.E2EEUploadDocStreamReq_PostMetaData{
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

	stream.CloseSend()
	return
}
