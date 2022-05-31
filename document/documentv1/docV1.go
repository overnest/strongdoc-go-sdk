package documentv1

import (
	"io"

	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

//////////////////////////////////////////////////////////////////
//
//                         Document V1
//
//////////////////////////////////////////////////////////////////
type DocumentV1 struct {
	DocName       string
	DocPath       string
	DocID         string
	DocVer        uint64
	DocKeyID      string
	DocKey        *sscrypto.StrongSaltKey
	EncryptStream io.Reader
	DecryptStream io.Reader
}

//////////////////////////////////////////////////////////////////
//
//                      Create Document V1
//
// The following graph is generated with https://arthursonzogni.com/Diagon/#Flowchart
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

// func CreateDocumentV1(docPath, docName, docID, docVer, docKeyID string, docKey *sscrypto.StrongSaltKey) (*DocumentV1, error) {

// }
