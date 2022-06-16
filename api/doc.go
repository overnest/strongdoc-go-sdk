/*
Package api exposes the functions that comprise the Strongsalt Go API.
It consists of high-level wrapper functions around GRPC function calls.
*/
package api

import (
	"encoding/json"
	"fmt"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/store"
)

const (
	DOC_V1      = int64(1)
	DOC_VER_CUR = DOC_V1
)

type DocV1 struct {
	DocID  string
	DocVer string
}

type DocWriter interface {
	GetDocID() string
	GetDocVer() string
	store.StoreWriter
}

type DocReader interface {
	GetDocID() string
	GetDocVer() string
	store.StoreReader
}

///////////////////////////////////////////////////////////////////////////////////////
//
//                                    Doc Writer
//
///////////////////////////////////////////////////////////////////////////////////////

type docWriter struct {
	docID       string
	docVer      string
	storeWriter store.StoreWriter
}

func CreateDocWriter(sdc client.StrongDocClient, docID, docVer string) (DocWriter, error) {
	docv1 := &DocV1{
		DocID:  docID,
		DocVer: docVer,
	}

	docv1Json, err := json.Marshal(docv1)
	if err != nil {
		return nil, err
	}

	storeInit := &store.StoreInit{
		StoreContent: proto.StoreInit_DOCUMENT,
		Json: &store.StoreJson{
			Version: DOC_VER_CUR,
			String:  string(docv1Json),
		},
	}

	storeWriter, err := store.CreateStore(sdc, storeInit)
	if err != nil {
		return nil, err
	}

	docWriter := &docWriter{
		docID:       docID,
		docVer:      docVer,
		storeWriter: storeWriter,
	}

	return docWriter, nil
}

func (dw *docWriter) GetDocID() string {
	return dw.docID
}

func (dw *docWriter) GetDocVer() string {
	return dw.docVer
}

func (dw *docWriter) Write(p []byte) (int, error) {
	if dw.storeWriter == nil {
		return 0, fmt.Errorf("document not open for writing")
	}

	return dw.storeWriter.Write(p)
}

func (dw *docWriter) WriteAt(p []byte, off int64) (int, error) {
	if dw.storeWriter == nil {
		return 0, fmt.Errorf("document not open for writing")
	}

	return dw.storeWriter.WriteAt(p, off)
}

func (dw *docWriter) Seek(offset int64, whence int) (int64, error) {
	if dw.storeWriter == nil {
		return 0, fmt.Errorf("document not open for writing")
	}

	return dw.storeWriter.Seek(offset, whence)
}

func (dw *docWriter) Close() error {
	if dw.storeWriter == nil {
		return fmt.Errorf("document not open for writing")
	}

	return dw.storeWriter.Close()
}

///////////////////////////////////////////////////////////////////////////////////////
//
//                                    Doc Reader
//
///////////////////////////////////////////////////////////////////////////////////////

type docReader struct {
	docID       string
	docVer      string
	storeReader store.StoreReader
}

func CreateDocReader(sdc client.StrongDocClient, docID, docVer string) (DocReader, error) {
	docv1 := &DocV1{
		DocID:  docID,
		DocVer: docVer,
	}

	docv1Json, err := json.Marshal(docv1)
	if err != nil {
		return nil, err
	}

	storeInit := &store.StoreInit{
		StoreContent: proto.StoreInit_DOCUMENT,
		Json: &store.StoreJson{
			Version: DOC_VER_CUR,
			String:  string(docv1Json),
		},
	}

	storeReader, err := store.OpenStore(sdc, storeInit)
	if err != nil {
		return nil, err
	}

	docReader := &docReader{
		docID:       docID,
		docVer:      docVer,
		storeReader: storeReader,
	}

	return docReader, nil
}

func (dr *docReader) GetDocID() string {
	return dr.docID
}

func (dr *docReader) GetDocVer() string {
	return dr.docVer
}

func (dr *docReader) Read(p []byte) (int, error) {
	if dr.storeReader == nil {
		return 0, fmt.Errorf("document not open for reading")
	}

	return dr.storeReader.Read(p)
}

func (dr *docReader) ReadAt(p []byte, off int64) (int, error) {
	if dr.storeReader == nil {
		return 0, fmt.Errorf("document not open for reading")
	}

	return dr.storeReader.ReadAt(p, off)
}

func (dr *docReader) Seek(offset int64, whence int) (int64, error) {
	if dr.storeReader == nil {
		return 0, fmt.Errorf("document not open for reading")
	}

	return dr.storeReader.Seek(offset, whence)
}

func (dr *docReader) GetSize() (uint64, error) {
	return dr.storeReader.GetSize()
}

func (dr *docReader) Close() error {
	if dr.storeReader == nil {
		return fmt.Errorf("document not open for reading")
	}

	return dr.storeReader.Close()
}
