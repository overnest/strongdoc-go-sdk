package documentv1

import (
	"fmt"

	"github.com/overnest/strongdoc-go-sdk/api/apidef"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/document/common"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	"github.com/overnest/strongdoc-go-sdk/store"
)

//////////////////////////////////////////////////////////////////
//
//                      Document Writer V1
//
// The following graph is generated with
//   https://arthursonzogni.com/Diagon/#Flowchart
//
// Input data:
// if ("Do Not Have Encryptor Key")
//   "Get Encryptor Keys For DocKey";
// else
//   noop;
// "Generate New DocKey"
// "Encrypt New DocKey with Encryptor Keys"
// "Upload Encrypted DocKey and Get New DocKey ID"
// "Get New DocID From Server"
// "Encrypt And Stream Document Content To Server"
//
// Output graph:
//   __________________
//  ╱                  ╲
// ╱                    ╲___
// ╲ Have Encryptor Key ╱yes│
//  ╲__________________╱    │
//          │no             │
//  ┌───────▽───────┐       │
//  │Get Encryptor  │       │
//  │Keys For DocKey│       │
//  └───────┬───────┘       │
// 		    └──────┬────────┘
// 	  ┌────────────▽──────┐
// 	  │Generate New DocKey│
// 	  └─────────┬─────────┘
// 	  ┌─────────▽─────────┐
// 	  │Encrypt New DocKey │
// 	  │with Encryptor Keys│
// 	  └─────────┬─────────┘
// 	┌───────────▽───────────┐
// 	│Upload Encrypted DocKey│
// 	│and Get New DocKey ID  │
// 	└───────────┬───────────┘
// 		 ┌──────▽──────┐
// 		 │Get New DocID│
// 		 │From Server  │
// 		 └──────┬──────┘
// ┌────────────▽─────────────┐
// │Encrypt And Stream        │
// │Document Content To Server│
// └──────────────────────────┘
//
//////////////////////////////////////////////////////////////////

type docWriterV1 struct {
	docName   string
	docID     string
	docVer    string
	docKeyID  string
	docKey    *apidef.Key
	encryptor crypto.StreamCrypto
	encbuf    []byte
	output    store.StoreWriter
}

func CreateDocumentV1(sdc client.StrongDocClient, docName string, docKey *apidef.Key) (common.DocWriter, error) {
	return createDocumentV1(sdc, docName, "", docKey)
}

func UpdateDocumentV1(sdc client.StrongDocClient, docID string, docKey *apidef.Key) (common.DocWriter, error) {
	return createDocumentV1(sdc, "", docID, docKey)
}

func createDocumentV1(sdc client.StrongDocClient, docName, docID string, docKey *apidef.Key) (docV1 *docWriterV1, err error) {
	storeWriter, err := store.CreateStore(sdc, &proto.StoreInit{
		Content: proto.StoreInit_DOCUMENT,
		Init: &proto.StoreInit_Doc{
			Doc: &proto.DocStoreInit{
				MetaVer: common.DOC_META_CUR,
				Doc: &proto.DocStoreInit_V1{
					V1: &proto.DocStoreInitV1{
						DocID:  docID,
						DocVer: "",
						DocKey: docKey.PKey,
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	defer func() {
		if docV1 == nil && storeWriter != nil {
			storeWriter.Close()
		}
	}()

	respInit := storeWriter.GetInit().GetDoc()
	switch respInit.GetMetaVer() {
	case common.DOC_META_V1:
		respInitV1 := respInit.GetV1()
		if respInitV1 == nil {
			return nil, fmt.Errorf("create document response is missing")
		}

		key, err := apidef.ParseProtoKey(respInitV1.GetDocKey())
		if err != nil {
			return nil, err
		}

		if key == nil {
			return nil, fmt.Errorf("document key missing from create document response")
		}

		nonce, err := crypto.CreateNonce(key.SKey)
		if err != nil {
			return nil, err
		}

		encryptor, err := crypto.CreateStreamCrypto(key.SKey, nonce, storeWriter, 0)
		if err != nil {
			return nil, err
		}

		plainHdr := &DocPlainHdrBodyV1{
			DocFormatVer: common.DocFormatVer{DocFormatVer: common.DOC_FORMAT_CUR},
			KeyID:        key.PKey.KeyID,
			KeyType:      key.SKey.Type.Name,
			Nonce:        nonce,
			DocID:        respInitV1.GetDocID(),
			DocVer:       respInitV1.GetDocVer(),
		}

		_, err = plainHdr.Write(storeWriter)
		if err != nil {
			return nil, err
		}

		docV1 = &docWriterV1{
			docName:   docName,
			docID:     respInitV1.GetDocID(),
			docVer:    respInitV1.GetDocVer(),
			docKeyID:  respInitV1.GetDocKey().GetKeyID(),
			docKey:    key,
			encryptor: encryptor,
			encbuf:    make([]byte, 1024),
			output:    storeWriter,
		}

		cipherHdr := &DocCipherHdrBodyV1{
			DocFormatVer: common.DocFormatVer{DocFormatVer: common.DOC_FORMAT_CUR},
			DocName:      docName,
		}

		_, err = cipherHdr.Write(docV1)
		if err != nil {
			docV1 = nil
			return nil, err
		}

		return docV1, nil
	default:
		return nil, fmt.Errorf("unsupported document meta version %v", respInit.GetMetaVer())
	}
}

func (dw *docWriterV1) DocID() string {
	return dw.docID
}

func (dw *docWriterV1) DocVer() string {
	return dw.docVer
}

func (dw *docWriterV1) DocName() string {
	return dw.docName
}

func (dw *docWriterV1) DocKeyID() string {
	return dw.docKeyID
}

func (dw *docWriterV1) DocKeyType() string {
	return dw.docKey.SKey.Type.Name
}

func (dw *docWriterV1) DocKey() *apidef.Key {
	return dw.docKey
}

func (dw *docWriterV1) Write(p []byte) (n int, err error) {
	if len(p) <= 0 {
		return 0, nil
	}

	n, err = dw.encryptor.Write(p)
	if err != nil {
		return 0, err
	}

	return n, err
}

func (dw *docWriterV1) Close() error {
	dw.encryptor.End()
	return dw.output.Close()
}

//////////////////////////////////////////////////////////////////
//
//                      Document Reader V1
//
//////////////////////////////////////////////////////////////////

type docReaderV1 struct {
	docName   string
	docID     string
	docVer    string
	docKeyID  string
	docKey    *apidef.Key
	decryptor crypto.StreamCrypto
	reader    store.StoreReader
}

func OpenDocumentV1(sdc client.StrongDocClient, docID string, docKey *apidef.Key) (common.DocReader, error) {
	return openDocumentV1(sdc, docID, "1", docKey)
}

func openDocumentV1(sdc client.StrongDocClient, docID, docVer string, docKey *apidef.Key) (docV1 *docReaderV1, err error) {
	storeReader, err := store.OpenStore(sdc, &proto.StoreInit{
		Content: proto.StoreInit_DOCUMENT,
		Init: &proto.StoreInit_Doc{
			Doc: &proto.DocStoreInit{
				MetaVer: common.DOC_META_CUR,
				Doc: &proto.DocStoreInit_V1{
					V1: &proto.DocStoreInitV1{
						DocID:  docID,
						DocVer: docVer,
						DocKey: nil,
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	defer func() {
		if docV1 == nil && storeReader != nil {
			storeReader.Close()
		}
	}()

	respInit := storeReader.GetInit().GetDoc()
	switch respInit.GetMetaVer() {
	case common.DOC_META_V1:
		respInitV1 := respInit.GetV1()
		if respInitV1 == nil {
			return nil, fmt.Errorf("open document response is missing")
		}

		plainHdr := &DocPlainHdrBodyV1{}
		err = plainHdr.Read(storeReader)
		if err != nil {
			return nil, err
		}

		decryptor, err := crypto.OpenStreamCrypto(docKey.SKey, plainHdr.Nonce, storeReader, 0)
		if err != nil {
			return nil, err
		}

		docV1 = &docReaderV1{
			docName:   "",
			docID:     respInitV1.GetDocID(),
			docVer:    respInitV1.GetDocVer(),
			docKeyID:  respInitV1.GetDocKey().GetKeyID(),
			docKey:    docKey,
			decryptor: decryptor,
			reader:    storeReader,
		}

		cipherHdr := &DocCipherHdrBodyV1{}
		err = cipherHdr.Read(docV1)
		if err != nil {
			docV1 = nil
			return nil, err
		}

		docV1.docName = cipherHdr.DocName
		return docV1, nil
	default:
		return nil, fmt.Errorf("unsupported document type version %v", respInit.GetMetaVer())
	}
}

func (dr *docReaderV1) DocID() string {
	return dr.docID
}

func (dr *docReaderV1) DocVer() string {
	return dr.docVer
}

func (dr *docReaderV1) DocName() string {
	return dr.docName
}

func (dr *docReaderV1) DocKeyID() string {
	return dr.docKeyID
}

func (dr *docReaderV1) DocKeyType() string {
	return dr.docKey.SKey.Type.Name
}

func (dr *docReaderV1) DocKey() *apidef.Key {
	return dr.docKey
}

func (dr *docReaderV1) Read(p []byte) (n int, err error) {
	return dr.decryptor.Read(p)
}

func (dr *docReaderV1) Close() error {
	dr.decryptor.End()
	return dr.reader.Close()
}
