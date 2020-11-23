package api

import (
	// "bytes"
	"context"
	"fmt"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	// ssc "github.com/overnest/strongsalt-crypto-go"
	// "github.com/overnest/strongdoc-go-sdk/utils"
)

type EncryptedKey struct{
	encKey string
   	encryptorID string
    encryptorVersion int32
   	keyID string
    keyVersion int32
   	ownerID string
}

// UploadDocument uploads a document to Strongdoc-provided storage.
// It then returns a docId that uniquely identifies the document.
func E2EEUploadDocument(docName string, plainStream io.Reader) (docID string, err error) {
	// sdc, err := client.GetStrongDocClient()
	// if err != nil {
	// 	return
	// }
	// sdm, err := client.GetStrongDocManager()
	// if err != nil {
	// 	return
	// }
	// passwordKey := sdm.GetPasswordKey()

	// stream, err := sdc.E2EEUploadDocumentStream(context.Background())
	// if err != nil {
	// 	err = fmt.Errorf("UploadDocument err: [%v]", err)
	// 	return
	// }

	// preReq := &proto.E2EEUploadDocStreamReq{
	// 	StagesOfUploading: &proto.E2EEUploadDocStreamReq_PreMetaData_ {
	// 		PreMetaData: &proto.E2EEUploadDocStreamReq_PreMetaData{
	// 			DocName: docName,
	// 			// IsEncryptedByClient: true,
	// 		},
	// 	},
	// }

	// err = stream.Send(preReq)
	// if err != nil {
	// 	err = fmt.Errorf("UploadDocument stage PreMetaData err: [%v]", err)
	// 	return
	// }

	// userPrivKeysReq := &proto.GetUserPrivateKeysReq{}
	// userPrivKeysResp, err := sdc.GetUserPrivateKeys(context.Background(), userPrivKeysReq)
	// if err != nil {
	// 	err = fmt.Errorf("UploadDocument cannot get user private keys")
	// 	return
	// }
	// userPrivKeys := userPrivKeysResp.GetEncryptedKeys()
	// docKey, err := ssc.GenerateKey(ssc.Type_Secretbox)
	// if err != nil {
	// 	return "", fmt.Errorf("UploadDocument cannot generate document key")
	// }
	// docKeySerialized, err :=  docKey.Serialize();
	// if err != nil {
	// 	err = fmt.Errorf("UploadDocument cannot serialize document key")
	// 	return
	// }

	// encKeyListReq := &proto.E2EEUploadDocStreamReq{
	// 	StagesOfUploading: &proto.E2EEUploadDocStreamReq_EncKeyList {
	// 		EncDocKeys: &proto.E2EEUploadDocStreamReq_EncKeyList{
	// 			EncDocKeys: make(*proto.EncryptedKey, 3),
	// 		},
	// 	},
	// }

	// for _, userPrivKey := range userPrivKeys{
		// TODO: Check user password key version and id
		// decryptedUserPrivKey, err := passwordKey.Decrypt([]byte(userPrivKey.GetEncKey()))
		// if err != nil {
		// 	return "", fmt.Errorf("UploadDocument cannot decrypt user private key")
		// }
		// userPrivKey, err := ssc.DeserializeKey(decryptedUserPrivKey)
		// if err != nil {
		// 	return "", fmt.Errorf("UploadDocument cannot deserialize user private key")
		// }
		// encryptedDocKey, err := userPrivKey.Encrypt(docKeySerialized)
		// if err != nil {
		// 	return "", fmt.Errorf("UploadDocument cannot encrypt document key")
		// }
	// }


	// block := make([]byte, blockSize)
	// for {
	// 	n, inerr := plainStream.Read(block)
	// 	if inerr == nil || inerr == io.EOF {
	// 		if n > 0 {
	// 			dataReq := &proto.E2EEUploadDocStreamReq{
	// 				StagesOfUploading: &proto.E2EEUploadDocStreamReq_CipherText {
	// 					//TODO: need encryption
	// 					CipherText: block[:n]}}
	// 			if err = stream.Send(dataReq); err != nil {
	// 				return "", err
	// 			}
	// 		}
	// 		if inerr == io.EOF {
	// 			break
	// 		}
	// 	} else {
	// 		err = inerr
	// 		return
	// 	}
	// }

	// if err = stream.CloseSend(); err != nil {
	// 	err = fmt.Errorf("UploadDocument err: [%v]", err)
	// 	return
	// }

	// resp, err := stream.CloseAndRecv()
	// if err != nil {
	// 	err = fmt.Errorf("UploadDocument err: [%v]", err)
	// 	return
	// }

	// if resp == nil {
	// 	err = fmt.Errorf("UploadDocument Resp is nil: [%v]", err)
	// 	return
	// }

	// docID = resp.GetDocID()
	return
}