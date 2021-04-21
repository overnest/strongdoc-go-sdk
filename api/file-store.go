package api

import (
	"context"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"google.golang.org/grpc"
	"io"
	"os"
)

// ============================================== File ==============================================
type FileReader interface {
	Read(buf []byte) (bytesRead int, err error)
	ReadAt(buf []byte, offset int64) (bytesRead int, err error)
	ReadFile(bufferSize int) ([]byte, error)
	Seek(offset int64, whence int) (int64, error)
	Close() (err error)
}

type fileReader struct {
	stream proto.StrongDocService_FileReadClient
}

func initFileReadStream(sdc client.StrongDocClient) (proto.StrongDocService_FileReadClient, error) {
	maxSizeOption := grpc.MaxCallRecvMsgSize(utils.MaxRecvMsgSize)
	return sdc.GetGrpcClient().FileRead(context.Background(), maxSizeOption)
}

func NewFileReader(sdc client.StrongDocClient, filename string) (reader FileReader, err error) {
	stream, err := initFileReadStream(sdc)
	if err != nil {
		return nil, err
	}
	preReq := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_PREMETA,
		MetaData: &proto.ReadFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_FILE,
				Filename: filename,
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return nil, err
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, err
	}
	return &fileReader{
		stream: stream,
	}, nil
}

/**
bytesRead > 0, err = nil, read some data from server
bytesRead = 0, err = EOF, no data read, reach end of file
bytesRead = 0, err != nil && err != EOF, other err
*/
func (reader *fileReader) Read(buf []byte) (bytesRead int, err error) {
	bytesRead = 0
	stream := reader.stream
	if stream == nil {
		return 0, fmt.Errorf("stream already closed")
	}
	n := len(buf)
	req := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ,
		MetaData: &proto.ReadFileReq_ReadMeta{
			ReadMeta: &proto.ReadMeta{
				BufferSize: (int64)(n),
			},
		},
	}
	err = stream.Send(req)
	if err != nil {
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		return
	}

	if resp == nil {
		err = fmt.Errorf("unexpected response")
		return
	}

	if resp.Data == nil || len(resp.Data) == 0 {
		err = io.EOF
		return
	}

	bytesRead = len(resp.Data)
	copy(buf, resp.Data)
	return
}

/**
bytesRead > 0, err = nil, read some data from server
bytesRead = 0, err = EOF, no data read, reach end of file
bytesRead = 0, err != nil && err != EOF, other err
*/
func (reader *fileReader) ReadAt(buf []byte, offset int64) (bytesRead int, err error) {
	bytesRead = 0
	stream := reader.stream
	if stream == nil {
		return 0, fmt.Errorf("stream already closed")
	}

	n := len(buf)

	req := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_AT,
		MetaData: &proto.ReadFileReq_ReadMeta{
			ReadMeta: &proto.ReadMeta{
				BufferSize: (int64)(n),
				Offset:     offset},
		},
	}

	err = stream.Send(req)
	if err != nil {
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		return
	}

	if resp == nil {
		err = fmt.Errorf("unexpected response")
		return
	}

	if resp.Data == nil || len(resp.Data) == 0 {
		err = io.EOF
		return
	}

	bytesRead = len(resp.Data)
	copy(buf, resp.Data)
	return
}

func (reader *fileReader) ReadFile(bufferSize int) ([]byte, error) {
	buffer := make([]byte, bufferSize)
	var data []byte

	for true {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		data = append(data, buffer[:n]...)
	}
	return data, nil
}

/**
whence:
0 from beginning
1 from current position
2 from end
*/
func (reader *fileReader) Seek(offset int64, whence int) (int64, error) {
	stream := reader.stream
	if stream == nil {
		return 0, fmt.Errorf("stream already closed")
	}

	req := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_SEEK,
		MetaData: &proto.ReadFileReq_ReadMeta{
			ReadMeta: &proto.ReadMeta{
				Offset: offset,
				Whence: int32(whence),
			},
		},
	}

	err := stream.Send(req)
	if err != nil {
		return 0, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, fmt.Errorf("unexpected response")
	}
	return resp.GetOffset(), nil
}

func (reader *fileReader) Close() (err error) {
	if reader.stream == nil {
		return fmt.Errorf("stream already closed")
	}
	err = reader.stream.CloseSend()
	if err != nil {
		return err
	}
	reader.stream = nil
	return nil
}

func getFileSize(sdc client.StrongDocClient, fileMeta *proto.FileMeta) (uint64, error) {
	req := &proto.GetFileSizeReq{FileMeta: fileMeta}
	res, err := sdc.GetGrpcClient().GetFileSize(context.Background(), req)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, fmt.Errorf("unexpected response")
	}
	return res.Size, nil
}

type FileWriter interface {
	Write(data []byte) (bytesWrite int, err error)
	WriteFile(data []byte, bufferSize int) error
	Close() (err error)
}

type fileWriter struct {
	stream proto.StrongDocService_FileWriteClient
}

func NewFileWriter(sdc client.StrongDocClient, filename string) (writer FileWriter, err error) {
	stream, err := sdc.GetGrpcClient().FileWrite(context.Background())
	if err != nil {
		return nil, err
	}
	preReq := &proto.WriteFileReq{
		WriteType: proto.WriteType_WRITE_PREMETA,
		MetaData: &proto.WriteFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_FILE,
				Filename: filename,
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return nil, err
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, err
	}
	return &fileWriter{
		stream: stream,
	}, nil
}

func (writer *fileWriter) Write(data []byte) (bytesWrite int, err error) {
	if writer.stream == nil {
		return 0, fmt.Errorf("stream already closed")
	}

	//fmt.Println("DocIndexWriter write ", data)
	req := &proto.WriteFileReq{
		WriteType: proto.WriteType_WRITE,
		MetaData: &proto.WriteFileReq_WriteMeta{
			WriteMeta: &proto.WriteMeta{
				Data: data,
			},
		},
	}

	err = writer.stream.Send(req)
	if err != nil {
		return 0, err
	}
	resp, err := writer.stream.Recv()
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, fmt.Errorf("unexpected response")
	}

	return int(resp.BytesWrite), nil
}

func (writer *fileWriter) WriteFile(data []byte, bufferSize int) error {
	n := len(data)
	totalBytesWrite := 0
	for totalBytesWrite < n {
		to := totalBytesWrite + bufferSize
		if to < n {
			to = n
		}
		writeData := data[totalBytesWrite:to]
		bytesWrite, err := writer.Write(writeData)
		if err != nil {
			return err
		}
		totalBytesWrite += bytesWrite
	}
	return nil
}

func (writer *fileWriter) Close() (err error) {
	if writer.stream == nil {
		return fmt.Errorf("stream already closed")
	}
	// send write_end to server, make sure file exists
	req := &proto.WriteFileReq{
		WriteType: proto.WriteType_WRITE_END,
	}
	err = writer.stream.Send(req)
	if err != nil {
		return err
	}
	_, err = writer.stream.Recv()
	if err != nil {
		return err
	}
	// close send
	err = writer.stream.CloseSend()
	if err != nil {
		return err
	}
	writer.stream = nil
	return nil
}

func RemoveFile(sdc client.StrongDocClient, filename string) error {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_FILE,
		Filename: filename,
	}
	return doRemoveFile(sdc, fileMeta)
}

func doRemoveFile(sdc client.StrongDocClient, fileMeta *proto.FileMeta) error {
	req := &proto.RemoveFileReq{
		FileMeta: fileMeta,
	}
	res, err := sdc.GetGrpcClient().RemoveFile(context.Background(), req)
	if err != nil {
		return err
	}
	if res.GetNotExistErr() {
		return os.ErrNotExist
	}
	return nil
}

func RemoveDir(sdc client.StrongDocClient, filename string) error {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_FILE,
		Filename: filename,
	}
	return doRemoveDir(sdc, fileMeta)
}

func doRemoveDir(sdc client.StrongDocClient, fileMeta *proto.FileMeta) error {
	req := &proto.RemoveDirReq{
		FileMeta: fileMeta,
	}
	res, err := sdc.GetGrpcClient().RemoveDir(context.Background(), req)
	if err != nil {
		return err
	}
	if res.GetNotExistErr() {
		return os.ErrNotExist
	}
	return nil
}

// ============================================== DocIndex ==============================================
func NewDocOffsetIdxReader(sdc client.StrongDocClient, docID string, ver uint64) (reader FileReader, err error) {
	return newDocIndexReader(sdc, docID, ver, proto.DocIndexType_DOC_OFFSET_INDEX)
}

func NewDocTermIdxReader(sdc client.StrongDocClient, docID string, ver uint64) (reader FileReader, err error) {
	return newDocIndexReader(sdc, docID, ver, proto.DocIndexType_DOC_TERM_INDEX)
}

func newDocIndexReader(sdc client.StrongDocClient, docID string, ver uint64, indexType proto.DocIndexType) (reader FileReader, err error) {
	stream, err := initFileReadStream(sdc)
	if err != nil {
		return nil, err
	}
	preReq := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_PREMETA,
		MetaData: &proto.ReadFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_DOC_INDEX,
				DocIndexMeta: &proto.DocIndexMeta{
					DocID:     docID,
					Version:   ver,
					IndexType: indexType,
				},
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return nil, err
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, err
	}
	return &fileReader{
		stream: stream,
	}, nil
}

func GetDocTermIndexSize(sdc client.StrongDocClient, docID string, docVer uint64) (uint64, error) {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_DOC_INDEX,
		DocIndexMeta: &proto.DocIndexMeta{
			DocID:     docID,
			Version:   docVer,
			IndexType: proto.DocIndexType_DOC_TERM_INDEX,
		},
	}
	return getFileSize(sdc, fileMeta)
}

func NewDocOffsetIdxWriter(sdc client.StrongDocClient, docID string, ver uint64) (writer FileWriter, err error) {
	return newDocIndexWriter(sdc, docID, ver, proto.DocIndexType_DOC_OFFSET_INDEX)
}

func NewDocTermIdxWriter(sdc client.StrongDocClient, docID string, ver uint64) (writer FileWriter, err error) {
	return newDocIndexWriter(sdc, docID, ver, proto.DocIndexType_DOC_TERM_INDEX)
}

func newDocIndexWriter(sdc client.StrongDocClient, docID string, ver uint64, indexType proto.DocIndexType) (writer FileWriter, err error) {
	stream, err := sdc.GetGrpcClient().FileWrite(context.Background())
	if err != nil {
		return nil, err
	}
	preReq := &proto.WriteFileReq{
		WriteType: proto.WriteType_WRITE_PREMETA,
		MetaData: &proto.WriteFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_DOC_INDEX,
				DocIndexMeta: &proto.DocIndexMeta{
					DocID:     docID,
					Version:   ver,
					IndexType: indexType,
				},
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return nil, err
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, err
	}
	return &fileWriter{
		stream: stream,
	}, nil
}

func RemoveDocOffsetIndex(sdc client.StrongDocClient, docID string, docVer uint64) error {
	return removeDocIndex(sdc, docID, docVer, proto.DocIndexType_DOC_OFFSET_INDEX)
}

func RemoveDocTermIndex(sdc client.StrongDocClient, docID string, docVer uint64) error {
	return removeDocIndex(sdc, docID, docVer, proto.DocIndexType_DOC_TERM_INDEX)
}

func removeDocIndex(sdc client.StrongDocClient, docID string, docVer uint64, indexType proto.DocIndexType) error {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_DOC_INDEX,
		DocIndexMeta: &proto.DocIndexMeta{
			DocID:     docID,
			Version:   docVer,
			IndexType: indexType,
		},
	}

	return doRemoveFile(sdc, fileMeta)
}

// remove all indexes(offset + term) of doc (docID + docVer)
func RemoveDocIndexes(sdc client.StrongDocClient, docID string, docVer uint64) error {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_DOC_INDEX,
		DocIndexMeta: &proto.DocIndexMeta{
			DocID:   docID,
			Version: docVer,
		},
	}
	return doRemoveDir(sdc, fileMeta)
}

// remove all indexes(offset + term) of doc (docID)
func RemoveDocIndexesAllVersions(sdc client.StrongDocClient, docID string) error {
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_DOC_INDEX,
		DocIndexMeta: &proto.DocIndexMeta{
			DocID: docID,
		},
	}
	return doRemoveDir(sdc, fileMeta)
}

// ============================================== SearchIndex  ==============================================

func NewSearchTermIdxWriter(sdc client.StrongDocClient, ownerType utils.OwnerType, term string) (writer FileWriter, updateID string, err error) {
	return newSearchIndexWriter(sdc, term, "", ownerType, proto.SearchIndexType_SEARCH_TERM_INDEX)
}

func NewSearchSortedDocIdxWriter(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string) (writer FileWriter, err error) {
	writer, _, err = newSearchIndexWriter(sdc, term, updateID, ownerType, proto.SearchIndexType_SEARCH_SORTED_DOC_INDEX)
	return
}

func newSearchIndexWriter(sdc client.StrongDocClient, term, inputUpdateID string, ownerType utils.OwnerType, indexType proto.SearchIndexType) (writer FileWriter, updateID string, err error) {
	stream, err := sdc.GetGrpcClient().FileWrite(context.Background())
	if err != nil {
		return
	}
	protoOwnerType, err := utils.TranslateOwnerType(ownerType)
	if err != nil {
		return
	}
	preReq := &proto.WriteFileReq{
		WriteType: proto.WriteType_WRITE_PREMETA,
		MetaData: &proto.WriteFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_SEARCH_INDEX,
				SearchIndexMeta: &proto.SearchIndexMeta{
					Term:       term,
					UpdateID:   inputUpdateID,
					AccessType: protoOwnerType,
					IndexType:  indexType,
				},
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return
	}

	var res *proto.WriteFileResp
	res, err = stream.Recv()
	if err != nil {
		return
	}
	updateID = res.GetUpdateID()

	return &fileWriter{
		stream: stream,
	}, updateID, nil
}

func NewSearchTermIdxReader(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string) (reader FileReader, err error) {
	return newSearchIndexReader(sdc, term, updateID, ownerType, proto.SearchIndexType_SEARCH_TERM_INDEX)
}

func NewSearchSortedDocIdxReader(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string) (reader FileReader, err error) {
	return newSearchIndexReader(sdc, term, updateID, ownerType, proto.SearchIndexType_SEARCH_SORTED_DOC_INDEX)
}

func newSearchIndexReader(sdc client.StrongDocClient, term, updateID string, ownerType utils.OwnerType, indexType proto.SearchIndexType) (reader FileReader, err error) {
	protoOwnerType, err := utils.TranslateOwnerType(ownerType)
	if err != nil {
		return nil, err
	}
	stream, err := initFileReadStream(sdc)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	preReq := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_PREMETA,
		MetaData: &proto.ReadFileReq_FileMeta{
			FileMeta: &proto.FileMeta{
				FileType: proto.FileType_SEARCH_INDEX,
				SearchIndexMeta: &proto.SearchIndexMeta{
					Term:       term,
					AccessType: protoOwnerType,
					IndexType:  indexType,
					UpdateID:   updateID,
				},
			},
		},
	}

	err = stream.Send(preReq)
	if err != nil {
		return nil, err
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, err
	}
	return &fileReader{
		stream: stream,
	}, nil
}

func GetSearchSortedDocIdxSize(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string) (uint64, error) {
	return getSearchIndexSize(sdc, ownerType, term, updateID, proto.SearchIndexType_SEARCH_SORTED_DOC_INDEX)
}

func GetSearchTermIndexSize(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string) (uint64, error) {
	return getSearchIndexSize(sdc, ownerType, term, updateID, proto.SearchIndexType_SEARCH_TERM_INDEX)
}

func getSearchIndexSize(sdc client.StrongDocClient, ownerType utils.OwnerType, term, updateID string, indexType proto.SearchIndexType) (uint64, error) {
	protoOwnerType, err := utils.TranslateOwnerType(ownerType)
	if err != nil {
		return 0, err
	}
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_SEARCH_INDEX,
		SearchIndexMeta: &proto.SearchIndexMeta{
			Term:       term,
			AccessType: protoOwnerType,
			IndexType:  indexType,
			UpdateID:   updateID,
		},
	}
	return getFileSize(sdc, fileMeta)
}

func GetUpdateIDs(sdc client.StrongDocClient, ownerType utils.OwnerType, term string) ([]string, error) {
	protoOwnerType, err := utils.TranslateOwnerType(ownerType)
	if err != nil {
		return nil, err
	}
	req := &proto.GetUpdateIDsReq{
		SearchIdxMeta: &proto.SearchIndexMeta{
			Term:       term,
			AccessType: protoOwnerType,
		},
	}
	res, err := sdc.GetGrpcClient().GetUpdateIDs(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return res.UpdateIDs, nil
}

// remove owner's all search indexes (term index + sorted doc index with all available updateIDs)
func RemoveSearchIndexes(sdc client.StrongDocClient, ownerType utils.OwnerType) error {
	protoOwnerType, err := utils.TranslateOwnerType(ownerType)
	if err != nil {
		return err
	}
	fileMeta := &proto.FileMeta{
		FileType: proto.FileType_SEARCH_INDEX,
		SearchIndexMeta: &proto.SearchIndexMeta{
			AccessType: protoOwnerType,
		},
	}
	return doRemoveDir(sdc, fileMeta)
}
