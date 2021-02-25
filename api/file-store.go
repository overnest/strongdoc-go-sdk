package api

import (
	"context"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"google.golang.org/grpc"
	"io"
)

// ===================================== FileReader =====================================
type FileReader interface {
	Read(buf []byte) (bytesRead int, err error)
	ReadAt(buf []byte, offset int64) (bytesRead int, err error)
	ReadFile(bufferSize int) ([]byte, error)
	Seek(offset int64, whence int) (int64, error)
	Close() (err error)
	GetFileSize() (uint64, error)
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

func NewDocIndexReader(sdc client.StrongDocClient, docID string, ver uint64, indexType utils.DocIndexType) (reader FileReader, err error) {
	stream, err := initFileReadStream(sdc)
	if err != nil {
		return nil, err
	}
	docIndexType, err := utils.TranslateDocIndexType(indexType)
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
					IndexType: docIndexType,
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

func (reader *fileReader) GetFileSize() (uint64, error) {
	stream := reader.stream
	if stream == nil {
		return 0, fmt.Errorf("stream already closed")
	}

	req := &proto.ReadFileReq{
		ReadType: proto.ReadType_READ_SIZE,
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
	return resp.GetSize(), nil
}

// ===================================== FileWriter =====================================
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

func NewDocIndexWriter(sdc client.StrongDocClient, docID string, ver uint64, indexType utils.DocIndexType) (writer FileWriter, err error) {
	stream, err := sdc.GetGrpcClient().FileWrite(context.Background())
	if err != nil {
		return nil, err
	}
	docIndexType, err := utils.TranslateDocIndexType(indexType)
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
					IndexType: docIndexType,
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
	err = writer.stream.CloseSend()
	if err != nil {
		return err
	}
	writer.stream = nil
	return nil
}
