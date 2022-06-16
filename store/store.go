package store

import (
	"context"
	"fmt"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

type StoreReader interface {
	Read(p []byte) (n int, err error)              // io.Reader
	ReadAt(p []byte, off int64) (n int, err error) // io.ReaderAt
	Seek(offset int64, whence int) (int64, error)  // io.Seeker
	GetSize() (size uint64, err error)
	Close() (err error)
}

type StoreWriter interface {
	Write(p []byte) (n int, err error)              // io.Writer
	WriteAt(p []byte, off int64) (n int, err error) // io.WriterAt
	Seek(offset int64, whence int) (int64, error)   // io.Seeker
	Close() (err error)
}

type StoreInit struct {
	StoreContent proto.StoreInit_StoreContent
	Json         *StoreJson
	GenericFile  string
}

type StoreJson struct {
	Version int64
	String  string
}

func (sj *StoreJson) ToProto() *proto.StoreJson {
	return &proto.StoreJson{
		Version: sj.Version,
		Json:    sj.String,
	}
}

func StoreJsonFromProto(sj *proto.StoreJson) *StoreJson {
	if sj == nil {
		return nil
	}
	return &StoreJson{
		Version: sj.GetVersion(),
		String:  sj.GetJson(),
	}
}

///////////////////////////////////////////////////////////////////////////////////////
//
//                                    Store Writer
//
///////////////////////////////////////////////////////////////////////////////////////

type storeWriter struct {
	storeInit *StoreInit
	respJson  *StoreJson
	stream    proto.StrongDocService_StoreWriteClient
}

func CreateStore(sdc client.StrongDocClient, storeInit *StoreInit) (StoreWriter, error) {

	stream, err := sdc.GetGrpcClient().StoreWrite(context.Background())
	if err != nil {
		return nil, err
	}

	switch storeInit.StoreContent {
	case proto.StoreInit_GENERIC:
		err = stream.Send(&proto.StoreWriteReq{
			WriteOp: proto.StoreWriteReq_WRITE_INIT,
			Data: &proto.StoreWriteReq_Init{
				Init: &proto.StoreInit{
					Content: storeInit.StoreContent,
					Init: &proto.StoreInit_Generic{
						Generic: &proto.StoreInitGeneric{
							Filename: storeInit.GenericFile,
						},
					},
				},
			},
		})
	default:
		err = stream.Send(&proto.StoreWriteReq{
			WriteOp: proto.StoreWriteReq_WRITE_INIT,
			Data: &proto.StoreWriteReq_Init{
				Init: &proto.StoreInit{
					Content: storeInit.StoreContent,
					Init: &proto.StoreInit_Json{
						Json: storeInit.Json.ToProto(),
					},
				},
			},
		})
	}

	if err != nil {
		stream.CloseSend()
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		stream.CloseSend()
		return nil, err
	}

	respJson := StoreJsonFromProto(resp.GetJson())

	return &storeWriter{
		storeInit: storeInit,
		respJson:  respJson,
		stream:    stream,
	}, nil
}

func (sw *storeWriter) Write(p []byte) (n int, err error) {
	if sw.stream == nil {
		return 0, fmt.Errorf("store writer not open for writing")
	}

	err = sw.stream.Send(&proto.StoreWriteReq{
		WriteOp: proto.StoreWriteReq_WRITE_DATA,
		Data: &proto.StoreWriteReq_Write{
			Write: &proto.StoreWrite{
				Data: p,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	resp, err := sw.stream.Recv()
	if err != nil {
		return 0, err
	}

	n = int(resp.GetWrittenBytes())
	return
}

func (sw *storeWriter) WriteAt(p []byte, off int64) (n int, err error) {
	if sw.stream == nil {
		return 0, fmt.Errorf("store writer not open for writing")
	}

	err = sw.stream.Send(&proto.StoreWriteReq{
		WriteOp: proto.StoreWriteReq_WRITE_AT,
		Data: &proto.StoreWriteReq_Write{
			Write: &proto.StoreWrite{
				Data:   p,
				Offset: off,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	resp, err := sw.stream.Recv()
	if err != nil {
		return 0, err
	}

	n = int(resp.GetWrittenBytes())
	return
}

func (sw *storeWriter) Seek(off int64, whence int) (n int64, err error) {
	if sw.stream == nil {
		return 0, fmt.Errorf("store writer not open for writing")
	}

	var pwhence proto.StoreWhence
	switch whence {
	case io.SeekStart, io.SeekCurrent, io.SeekEnd:
		pwhence = proto.StoreWhence(whence)
	default:
		return 0, fmt.Errorf("invalid whence value %v", whence)
	}

	err = sw.stream.Send(&proto.StoreWriteReq{
		WriteOp: proto.StoreWriteReq_WRITE_SEEK,
		Data: &proto.StoreWriteReq_Write{
			Write: &proto.StoreWrite{
				Whence: pwhence,
				Offset: off,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	resp, err := sw.stream.Recv()
	if err != nil {
		return 0, err
	}

	n = resp.GetSeekOffset()
	return
}

func (sw *storeWriter) Close() (err error) {
	if sw.stream == nil {
		return fmt.Errorf("store writer not open for writing")
	}

	defer sw.stream.CloseSend()

	err = sw.stream.Send(&proto.StoreWriteReq{
		WriteOp: proto.StoreWriteReq_WRITE_END,
		Data:    nil,
	})
	if err != nil {
		return err
	}

	_, err = sw.stream.Recv()
	if err != nil {
		return err
	}

	return
}

///////////////////////////////////////////////////////////////////////////////////////
//
//                                    Store Reader
//
///////////////////////////////////////////////////////////////////////////////////////

type storeReader struct {
	storeInit *StoreInit
	respJson  *StoreJson
	stream    proto.StrongDocService_StoreReadClient
	size      int64
}

func OpenStore(sdc client.StrongDocClient, storeInit *StoreInit) (StoreReader, error) {
	stream, err := sdc.GetGrpcClient().StoreRead(context.Background())
	if err != nil {
		return nil, err
	}

	switch storeInit.StoreContent {
	case proto.StoreInit_GENERIC:
		err = stream.Send(&proto.StoreReadReq{
			ReadOp: proto.StoreReadReq_READ_INIT,
			Data: &proto.StoreReadReq_Init{
				Init: &proto.StoreInit{
					Content: storeInit.StoreContent,
					Init: &proto.StoreInit_Generic{
						Generic: &proto.StoreInitGeneric{
							Filename: storeInit.GenericFile,
						},
					},
				},
			},
		})
	default:
		err = stream.Send(&proto.StoreReadReq{
			ReadOp: proto.StoreReadReq_READ_INIT,
			Data: &proto.StoreReadReq_Init{
				Init: &proto.StoreInit{
					Content: storeInit.StoreContent,
					Init: &proto.StoreInit_Json{
						Json: storeInit.Json.ToProto(),
					},
				},
			},
		})
	}

	if err != nil {
		stream.CloseSend()
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		stream.CloseSend()
		return nil, err
	}

	respJson := StoreJsonFromProto(resp.GetJson())

	return &storeReader{
		storeInit: storeInit,
		respJson:  respJson,
		stream:    stream,
		size:      resp.GetSize(),
	}, nil
}

func (sr *storeReader) Read(p []byte) (int, error) {
	req := &proto.StoreReadReq{
		ReadOp: proto.StoreReadReq_READ_DATA,
		Data: &proto.StoreReadReq_Read{
			Read: &proto.StoreRead{
				BufferSize: int64(len(p)),
				Offset:     0,
				Whence:     proto.StoreWhence_SEEK_CURRENT,
			},
		},
	}

	return sr.read(p, req)
}

func (sr *storeReader) ReadAt(p []byte, off int64) (int, error) {
	req := &proto.StoreReadReq{
		ReadOp: proto.StoreReadReq_READ_AT,
		Data: &proto.StoreReadReq_Read{
			Read: &proto.StoreRead{
				BufferSize: int64(len(p)),
				Offset:     off,
				Whence:     proto.StoreWhence_SEEK_START,
			},
		},
	}

	return sr.read(p, req)
}

func (sr *storeReader) read(p []byte, req *proto.StoreReadReq) (n int, err error) {
	if sr.stream == nil {
		return 0, fmt.Errorf("store reader not open for reading")
	}

	err = sr.stream.Send(req)
	if err != nil {
		return 0, err
	}

	resp, err := sr.stream.Recv()
	if err != nil {
		return 0, err
	}

	if resp.GetEof() {
		err = io.EOF
	}

	n = len(resp.GetData())
	if n > len(p) {
		return 0, fmt.Errorf("Received bytes(%v) too large for buffer size(%v)",
			n, len(p))
	}

	copy(p, resp.GetData())
	return
}

func (sr *storeReader) Seek(off int64, whence int) (n int64, err error) {
	if sr.stream == nil {
		return 0, fmt.Errorf("store reader not open for reading")
	}

	var pwhence proto.StoreWhence
	switch whence {
	case io.SeekStart, io.SeekCurrent, io.SeekEnd:
		pwhence = proto.StoreWhence(whence)
	default:
		return 0, fmt.Errorf("invalid whence value %v", whence)
	}

	err = sr.stream.Send(&proto.StoreReadReq{
		ReadOp: proto.StoreReadReq_READ_SEEK,
		Data: &proto.StoreReadReq_Read{
			Read: &proto.StoreRead{
				BufferSize: 0,
				Offset:     off,
				Whence:     pwhence,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	resp, err := sr.stream.Recv()
	if err != nil {
		return 0, err
	}

	n = resp.GetOffset()
	return
}

func (sr *storeReader) GetSize() (size uint64, err error) {
	return uint64(sr.size), nil
}

func (sr *storeReader) Close() (err error) {
	if sr.stream == nil {
		return fmt.Errorf("store reader not open for reading")
	}

	defer sr.stream.CloseSend()

	err = sr.stream.Send(&proto.StoreReadReq{
		ReadOp: proto.StoreReadReq_READ_END,
		Data:   nil,
	})
	if err != nil {
		return err
	}

	_, err = sr.stream.Recv()
	if err != nil {
		return err
	}

	return
}
