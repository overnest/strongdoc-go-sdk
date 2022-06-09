// Code generated by protoc-gen-go. DO NOT EDIT.
// source: store.proto

package proto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type StoreInitMeta_StoreContent int32

const (
	StoreInitMeta_GENERIC        StoreInitMeta_StoreContent = 0
	StoreInitMeta_DOCUMENT       StoreInitMeta_StoreContent = 1
	StoreInitMeta_DOCUMENT_INDEX StoreInitMeta_StoreContent = 2
	StoreInitMeta_SEARCH_INDEX   StoreInitMeta_StoreContent = 3
)

var StoreInitMeta_StoreContent_name = map[int32]string{
	0: "GENERIC",
	1: "DOCUMENT",
	2: "DOCUMENT_INDEX",
	3: "SEARCH_INDEX",
}

var StoreInitMeta_StoreContent_value = map[string]int32{
	"GENERIC":        0,
	"DOCUMENT":       1,
	"DOCUMENT_INDEX": 2,
	"SEARCH_INDEX":   3,
}

func (x StoreInitMeta_StoreContent) String() string {
	return proto.EnumName(StoreInitMeta_StoreContent_name, int32(x))
}

func (StoreInitMeta_StoreContent) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{0, 0}
}

type StoreReadReq_StoreReadType int32

const (
	StoreReadReq_READ_PREMETA StoreReadReq_StoreReadType = 0
	StoreReadReq_READ         StoreReadReq_StoreReadType = 1
	StoreReadReq_READ_SEEK    StoreReadReq_StoreReadType = 2
	StoreReadReq_READ_AT      StoreReadReq_StoreReadType = 3
)

var StoreReadReq_StoreReadType_name = map[int32]string{
	0: "READ_PREMETA",
	1: "READ",
	2: "READ_SEEK",
	3: "READ_AT",
}

var StoreReadReq_StoreReadType_value = map[string]int32{
	"READ_PREMETA": 0,
	"READ":         1,
	"READ_SEEK":    2,
	"READ_AT":      3,
}

func (x StoreReadReq_StoreReadType) String() string {
	return proto.EnumName(StoreReadReq_StoreReadType_name, int32(x))
}

func (StoreReadReq_StoreReadType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{4, 0}
}

type StoreWriteReq_StoreWriteType int32

const (
	StoreWriteReq_WRITE_PREMETA StoreWriteReq_StoreWriteType = 0
	StoreWriteReq_WRITE         StoreWriteReq_StoreWriteType = 1
	StoreWriteReq_WRITE_END     StoreWriteReq_StoreWriteType = 2
)

var StoreWriteReq_StoreWriteType_name = map[int32]string{
	0: "WRITE_PREMETA",
	1: "WRITE",
	2: "WRITE_END",
}

var StoreWriteReq_StoreWriteType_value = map[string]int32{
	"WRITE_PREMETA": 0,
	"WRITE":         1,
	"WRITE_END":     2,
}

func (x StoreWriteReq_StoreWriteType) String() string {
	return proto.EnumName(StoreWriteReq_StoreWriteType_name, int32(x))
}

func (StoreWriteReq_StoreWriteType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{7, 0}
}

type StoreInitMeta struct {
	// // In the future we might allow the client to choose the backend storage location
	// enum StoreLocation {
	//   // FILE = 0;
	//   // S3 = 1;
	// }
	Content StoreInitMeta_StoreContent `protobuf:"varint,1,opt,name=content,proto3,enum=proto.StoreInitMeta_StoreContent" json:"content,omitempty"`
	// StoreLocation location = 2;
	//
	// Types that are valid to be assigned to ContentMeta:
	//	*StoreInitMeta_GenericMeta
	//	*StoreInitMeta_JsonMeta
	ContentMeta          isStoreInitMeta_ContentMeta `protobuf_oneof:"contentMeta"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *StoreInitMeta) Reset()         { *m = StoreInitMeta{} }
func (m *StoreInitMeta) String() string { return proto.CompactTextString(m) }
func (*StoreInitMeta) ProtoMessage()    {}
func (*StoreInitMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{0}
}

func (m *StoreInitMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreInitMeta.Unmarshal(m, b)
}
func (m *StoreInitMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreInitMeta.Marshal(b, m, deterministic)
}
func (m *StoreInitMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreInitMeta.Merge(m, src)
}
func (m *StoreInitMeta) XXX_Size() int {
	return xxx_messageInfo_StoreInitMeta.Size(m)
}
func (m *StoreInitMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreInitMeta.DiscardUnknown(m)
}

var xxx_messageInfo_StoreInitMeta proto.InternalMessageInfo

func (m *StoreInitMeta) GetContent() StoreInitMeta_StoreContent {
	if m != nil {
		return m.Content
	}
	return StoreInitMeta_GENERIC
}

type isStoreInitMeta_ContentMeta interface {
	isStoreInitMeta_ContentMeta()
}

type StoreInitMeta_GenericMeta struct {
	GenericMeta *StoreGenericMeta `protobuf:"bytes,100,opt,name=genericMeta,proto3,oneof"`
}

type StoreInitMeta_JsonMeta struct {
	JsonMeta *StoreJsonMeta `protobuf:"bytes,101,opt,name=jsonMeta,proto3,oneof"`
}

func (*StoreInitMeta_GenericMeta) isStoreInitMeta_ContentMeta() {}

func (*StoreInitMeta_JsonMeta) isStoreInitMeta_ContentMeta() {}

func (m *StoreInitMeta) GetContentMeta() isStoreInitMeta_ContentMeta {
	if m != nil {
		return m.ContentMeta
	}
	return nil
}

func (m *StoreInitMeta) GetGenericMeta() *StoreGenericMeta {
	if x, ok := m.GetContentMeta().(*StoreInitMeta_GenericMeta); ok {
		return x.GenericMeta
	}
	return nil
}

func (m *StoreInitMeta) GetJsonMeta() *StoreJsonMeta {
	if x, ok := m.GetContentMeta().(*StoreInitMeta_JsonMeta); ok {
		return x.JsonMeta
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreInitMeta) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreInitMeta_GenericMeta)(nil),
		(*StoreInitMeta_JsonMeta)(nil),
	}
}

type StoreGenericMeta struct {
	Filename             string   `protobuf:"bytes,1,opt,name=filename,proto3" json:"filename,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreGenericMeta) Reset()         { *m = StoreGenericMeta{} }
func (m *StoreGenericMeta) String() string { return proto.CompactTextString(m) }
func (*StoreGenericMeta) ProtoMessage()    {}
func (*StoreGenericMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{1}
}

func (m *StoreGenericMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreGenericMeta.Unmarshal(m, b)
}
func (m *StoreGenericMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreGenericMeta.Marshal(b, m, deterministic)
}
func (m *StoreGenericMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreGenericMeta.Merge(m, src)
}
func (m *StoreGenericMeta) XXX_Size() int {
	return xxx_messageInfo_StoreGenericMeta.Size(m)
}
func (m *StoreGenericMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreGenericMeta.DiscardUnknown(m)
}

var xxx_messageInfo_StoreGenericMeta proto.InternalMessageInfo

func (m *StoreGenericMeta) GetFilename() string {
	if m != nil {
		return m.Filename
	}
	return ""
}

type StoreJsonMeta struct {
	Json                 string   `protobuf:"bytes,1,opt,name=json,proto3" json:"json,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreJsonMeta) Reset()         { *m = StoreJsonMeta{} }
func (m *StoreJsonMeta) String() string { return proto.CompactTextString(m) }
func (*StoreJsonMeta) ProtoMessage()    {}
func (*StoreJsonMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{2}
}

func (m *StoreJsonMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreJsonMeta.Unmarshal(m, b)
}
func (m *StoreJsonMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreJsonMeta.Marshal(b, m, deterministic)
}
func (m *StoreJsonMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreJsonMeta.Merge(m, src)
}
func (m *StoreJsonMeta) XXX_Size() int {
	return xxx_messageInfo_StoreJsonMeta.Size(m)
}
func (m *StoreJsonMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreJsonMeta.DiscardUnknown(m)
}

var xxx_messageInfo_StoreJsonMeta proto.InternalMessageInfo

func (m *StoreJsonMeta) GetJson() string {
	if m != nil {
		return m.Json
	}
	return ""
}

// StoreReadMeta
type StoreReadMeta struct {
	BufferSize           int64    `protobuf:"varint,1,opt,name=bufferSize,proto3" json:"bufferSize,omitempty"`
	Offset               int64    `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Whence               int32    `protobuf:"varint,3,opt,name=whence,proto3" json:"whence,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreReadMeta) Reset()         { *m = StoreReadMeta{} }
func (m *StoreReadMeta) String() string { return proto.CompactTextString(m) }
func (*StoreReadMeta) ProtoMessage()    {}
func (*StoreReadMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{3}
}

func (m *StoreReadMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreReadMeta.Unmarshal(m, b)
}
func (m *StoreReadMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreReadMeta.Marshal(b, m, deterministic)
}
func (m *StoreReadMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreReadMeta.Merge(m, src)
}
func (m *StoreReadMeta) XXX_Size() int {
	return xxx_messageInfo_StoreReadMeta.Size(m)
}
func (m *StoreReadMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreReadMeta.DiscardUnknown(m)
}

var xxx_messageInfo_StoreReadMeta proto.InternalMessageInfo

func (m *StoreReadMeta) GetBufferSize() int64 {
	if m != nil {
		return m.BufferSize
	}
	return 0
}

func (m *StoreReadMeta) GetOffset() int64 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *StoreReadMeta) GetWhence() int32 {
	if m != nil {
		return m.Whence
	}
	return 0
}

// StoreReadReq
type StoreReadReq struct {
	ReadType StoreReadReq_StoreReadType `protobuf:"varint,1,opt,name=readType,proto3,enum=proto.StoreReadReq_StoreReadType" json:"readType,omitempty"`
	// Types that are valid to be assigned to MetaData:
	//	*StoreReadReq_InitMeta
	//	*StoreReadReq_ReadMeta
	MetaData             isStoreReadReq_MetaData `protobuf_oneof:"metaData"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *StoreReadReq) Reset()         { *m = StoreReadReq{} }
func (m *StoreReadReq) String() string { return proto.CompactTextString(m) }
func (*StoreReadReq) ProtoMessage()    {}
func (*StoreReadReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{4}
}

func (m *StoreReadReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreReadReq.Unmarshal(m, b)
}
func (m *StoreReadReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreReadReq.Marshal(b, m, deterministic)
}
func (m *StoreReadReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreReadReq.Merge(m, src)
}
func (m *StoreReadReq) XXX_Size() int {
	return xxx_messageInfo_StoreReadReq.Size(m)
}
func (m *StoreReadReq) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreReadReq.DiscardUnknown(m)
}

var xxx_messageInfo_StoreReadReq proto.InternalMessageInfo

func (m *StoreReadReq) GetReadType() StoreReadReq_StoreReadType {
	if m != nil {
		return m.ReadType
	}
	return StoreReadReq_READ_PREMETA
}

type isStoreReadReq_MetaData interface {
	isStoreReadReq_MetaData()
}

type StoreReadReq_InitMeta struct {
	InitMeta *StoreInitMeta `protobuf:"bytes,2,opt,name=initMeta,proto3,oneof"`
}

type StoreReadReq_ReadMeta struct {
	ReadMeta *StoreReadMeta `protobuf:"bytes,3,opt,name=readMeta,proto3,oneof"`
}

func (*StoreReadReq_InitMeta) isStoreReadReq_MetaData() {}

func (*StoreReadReq_ReadMeta) isStoreReadReq_MetaData() {}

func (m *StoreReadReq) GetMetaData() isStoreReadReq_MetaData {
	if m != nil {
		return m.MetaData
	}
	return nil
}

func (m *StoreReadReq) GetInitMeta() *StoreInitMeta {
	if x, ok := m.GetMetaData().(*StoreReadReq_InitMeta); ok {
		return x.InitMeta
	}
	return nil
}

func (m *StoreReadReq) GetReadMeta() *StoreReadMeta {
	if x, ok := m.GetMetaData().(*StoreReadReq_ReadMeta); ok {
		return x.ReadMeta
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreReadReq) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreReadReq_InitMeta)(nil),
		(*StoreReadReq_ReadMeta)(nil),
	}
}

type StoreReadResp struct {
	Data                 []byte   `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Eof                  bool     `protobuf:"varint,2,opt,name=eof,proto3" json:"eof,omitempty"`
	Offset               int64    `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreReadResp) Reset()         { *m = StoreReadResp{} }
func (m *StoreReadResp) String() string { return proto.CompactTextString(m) }
func (*StoreReadResp) ProtoMessage()    {}
func (*StoreReadResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{5}
}

func (m *StoreReadResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreReadResp.Unmarshal(m, b)
}
func (m *StoreReadResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreReadResp.Marshal(b, m, deterministic)
}
func (m *StoreReadResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreReadResp.Merge(m, src)
}
func (m *StoreReadResp) XXX_Size() int {
	return xxx_messageInfo_StoreReadResp.Size(m)
}
func (m *StoreReadResp) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreReadResp.DiscardUnknown(m)
}

var xxx_messageInfo_StoreReadResp proto.InternalMessageInfo

func (m *StoreReadResp) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *StoreReadResp) GetEof() bool {
	if m != nil {
		return m.Eof
	}
	return false
}

func (m *StoreReadResp) GetOffset() int64 {
	if m != nil {
		return m.Offset
	}
	return 0
}

type StoreWriteMeta struct {
	Data                 []byte   `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreWriteMeta) Reset()         { *m = StoreWriteMeta{} }
func (m *StoreWriteMeta) String() string { return proto.CompactTextString(m) }
func (*StoreWriteMeta) ProtoMessage()    {}
func (*StoreWriteMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{6}
}

func (m *StoreWriteMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreWriteMeta.Unmarshal(m, b)
}
func (m *StoreWriteMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreWriteMeta.Marshal(b, m, deterministic)
}
func (m *StoreWriteMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreWriteMeta.Merge(m, src)
}
func (m *StoreWriteMeta) XXX_Size() int {
	return xxx_messageInfo_StoreWriteMeta.Size(m)
}
func (m *StoreWriteMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreWriteMeta.DiscardUnknown(m)
}

var xxx_messageInfo_StoreWriteMeta proto.InternalMessageInfo

func (m *StoreWriteMeta) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type StoreWriteReq struct {
	WriteType StoreWriteReq_StoreWriteType `protobuf:"varint,1,opt,name=writeType,proto3,enum=proto.StoreWriteReq_StoreWriteType" json:"writeType,omitempty"`
	// Types that are valid to be assigned to MetaData:
	//	*StoreWriteReq_InitMeta
	//	*StoreWriteReq_WriteMeta
	MetaData             isStoreWriteReq_MetaData `protobuf_oneof:"metaData"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *StoreWriteReq) Reset()         { *m = StoreWriteReq{} }
func (m *StoreWriteReq) String() string { return proto.CompactTextString(m) }
func (*StoreWriteReq) ProtoMessage()    {}
func (*StoreWriteReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{7}
}

func (m *StoreWriteReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreWriteReq.Unmarshal(m, b)
}
func (m *StoreWriteReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreWriteReq.Marshal(b, m, deterministic)
}
func (m *StoreWriteReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreWriteReq.Merge(m, src)
}
func (m *StoreWriteReq) XXX_Size() int {
	return xxx_messageInfo_StoreWriteReq.Size(m)
}
func (m *StoreWriteReq) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreWriteReq.DiscardUnknown(m)
}

var xxx_messageInfo_StoreWriteReq proto.InternalMessageInfo

func (m *StoreWriteReq) GetWriteType() StoreWriteReq_StoreWriteType {
	if m != nil {
		return m.WriteType
	}
	return StoreWriteReq_WRITE_PREMETA
}

type isStoreWriteReq_MetaData interface {
	isStoreWriteReq_MetaData()
}

type StoreWriteReq_InitMeta struct {
	InitMeta *StoreInitMeta `protobuf:"bytes,2,opt,name=initMeta,proto3,oneof"`
}

type StoreWriteReq_WriteMeta struct {
	WriteMeta *StoreWriteMeta `protobuf:"bytes,3,opt,name=writeMeta,proto3,oneof"`
}

func (*StoreWriteReq_InitMeta) isStoreWriteReq_MetaData() {}

func (*StoreWriteReq_WriteMeta) isStoreWriteReq_MetaData() {}

func (m *StoreWriteReq) GetMetaData() isStoreWriteReq_MetaData {
	if m != nil {
		return m.MetaData
	}
	return nil
}

func (m *StoreWriteReq) GetInitMeta() *StoreInitMeta {
	if x, ok := m.GetMetaData().(*StoreWriteReq_InitMeta); ok {
		return x.InitMeta
	}
	return nil
}

func (m *StoreWriteReq) GetWriteMeta() *StoreWriteMeta {
	if x, ok := m.GetMetaData().(*StoreWriteReq_WriteMeta); ok {
		return x.WriteMeta
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreWriteReq) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreWriteReq_InitMeta)(nil),
		(*StoreWriteReq_WriteMeta)(nil),
	}
}

type StoreWriteResp struct {
	BytesWrite           int64    `protobuf:"varint,1,opt,name=bytesWrite,proto3" json:"bytesWrite,omitempty"`
	UpdateID             string   `protobuf:"bytes,2,opt,name=updateID,proto3" json:"updateID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreWriteResp) Reset()         { *m = StoreWriteResp{} }
func (m *StoreWriteResp) String() string { return proto.CompactTextString(m) }
func (*StoreWriteResp) ProtoMessage()    {}
func (*StoreWriteResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{8}
}

func (m *StoreWriteResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreWriteResp.Unmarshal(m, b)
}
func (m *StoreWriteResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreWriteResp.Marshal(b, m, deterministic)
}
func (m *StoreWriteResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreWriteResp.Merge(m, src)
}
func (m *StoreWriteResp) XXX_Size() int {
	return xxx_messageInfo_StoreWriteResp.Size(m)
}
func (m *StoreWriteResp) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreWriteResp.DiscardUnknown(m)
}

var xxx_messageInfo_StoreWriteResp proto.InternalMessageInfo

func (m *StoreWriteResp) GetBytesWrite() int64 {
	if m != nil {
		return m.BytesWrite
	}
	return 0
}

func (m *StoreWriteResp) GetUpdateID() string {
	if m != nil {
		return m.UpdateID
	}
	return ""
}

type DocMeta struct {
	DocID                string   `protobuf:"bytes,1,opt,name=docID,proto3" json:"docID,omitempty"`
	DocVer               uint64   `protobuf:"varint,2,opt,name=docVer,proto3" json:"docVer,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DocMeta) Reset()         { *m = DocMeta{} }
func (m *DocMeta) String() string { return proto.CompactTextString(m) }
func (*DocMeta) ProtoMessage()    {}
func (*DocMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{9}
}

func (m *DocMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DocMeta.Unmarshal(m, b)
}
func (m *DocMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DocMeta.Marshal(b, m, deterministic)
}
func (m *DocMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DocMeta.Merge(m, src)
}
func (m *DocMeta) XXX_Size() int {
	return xxx_messageInfo_DocMeta.Size(m)
}
func (m *DocMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_DocMeta.DiscardUnknown(m)
}

var xxx_messageInfo_DocMeta proto.InternalMessageInfo

func (m *DocMeta) GetDocID() string {
	if m != nil {
		return m.DocID
	}
	return ""
}

func (m *DocMeta) GetDocVer() uint64 {
	if m != nil {
		return m.DocVer
	}
	return 0
}

func init() {
	proto.RegisterEnum("proto.StoreInitMeta_StoreContent", StoreInitMeta_StoreContent_name, StoreInitMeta_StoreContent_value)
	proto.RegisterEnum("proto.StoreReadReq_StoreReadType", StoreReadReq_StoreReadType_name, StoreReadReq_StoreReadType_value)
	proto.RegisterEnum("proto.StoreWriteReq_StoreWriteType", StoreWriteReq_StoreWriteType_name, StoreWriteReq_StoreWriteType_value)
	proto.RegisterType((*StoreInitMeta)(nil), "proto.StoreInitMeta")
	proto.RegisterType((*StoreGenericMeta)(nil), "proto.StoreGenericMeta")
	proto.RegisterType((*StoreJsonMeta)(nil), "proto.StoreJsonMeta")
	proto.RegisterType((*StoreReadMeta)(nil), "proto.StoreReadMeta")
	proto.RegisterType((*StoreReadReq)(nil), "proto.StoreReadReq")
	proto.RegisterType((*StoreReadResp)(nil), "proto.StoreReadResp")
	proto.RegisterType((*StoreWriteMeta)(nil), "proto.StoreWriteMeta")
	proto.RegisterType((*StoreWriteReq)(nil), "proto.StoreWriteReq")
	proto.RegisterType((*StoreWriteResp)(nil), "proto.StoreWriteResp")
	proto.RegisterType((*DocMeta)(nil), "proto.DocMeta")
}

func init() { proto.RegisterFile("store.proto", fileDescriptor_98bbca36ef968dfc) }

var fileDescriptor_98bbca36ef968dfc = []byte{
	// 622 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x53, 0x4f, 0x6f, 0xd3, 0x3e,
	0x18, 0x6e, 0xda, 0x75, 0x4b, 0xdf, 0xb6, 0x93, 0x7f, 0xd6, 0x7e, 0x50, 0x71, 0x40, 0xc3, 0xe3,
	0xb0, 0x53, 0x0f, 0x43, 0x88, 0xc3, 0xb4, 0x43, 0xdb, 0x58, 0x5b, 0x81, 0x76, 0xc8, 0x2d, 0x8c,
	0xdb, 0x94, 0x25, 0xee, 0x08, 0xac, 0x71, 0x49, 0x3c, 0x4d, 0xe3, 0x13, 0x4c, 0x7c, 0x07, 0xbe,
	0x2b, 0xf2, 0x1b, 0x27, 0xf5, 0x0a, 0x27, 0x4e, 0xf1, 0xf3, 0xfa, 0x79, 0xde, 0x7f, 0x4f, 0x0c,
	0xed, 0x5c, 0xab, 0x4c, 0xf6, 0x57, 0x99, 0xd2, 0x8a, 0x36, 0xf1, 0xc3, 0x7e, 0xd5, 0xa1, 0x3b,
	0x33, 0xe1, 0x71, 0x9a, 0xe8, 0x89, 0xd4, 0x21, 0x3d, 0x86, 0x9d, 0x48, 0xa5, 0x5a, 0xa6, 0xba,
	0xe7, 0xed, 0x7b, 0x87, 0xbb, 0x47, 0x2f, 0x0a, 0x45, 0xff, 0x11, 0xad, 0x40, 0xa3, 0x82, 0x28,
	0x4a, 0x05, 0x3d, 0x86, 0xf6, 0xb5, 0x4c, 0x65, 0x96, 0x44, 0x86, 0xd4, 0x8b, 0xf7, 0xbd, 0xc3,
	0xf6, 0xd1, 0x53, 0x37, 0xc1, 0xe9, 0xfa, 0xfa, 0xac, 0x26, 0x5c, 0x36, 0x3d, 0x02, 0xff, 0x6b,
	0xae, 0x52, 0x54, 0x4a, 0x54, 0xee, 0xb9, 0xca, 0xb7, 0xf6, 0xee, 0xac, 0x26, 0x2a, 0x1e, 0x3b,
	0x87, 0x8e, 0xdb, 0x09, 0x6d, 0xc3, 0xce, 0x29, 0x9f, 0x72, 0x31, 0x1e, 0x91, 0x1a, 0xed, 0x80,
	0x1f, 0x9c, 0x8f, 0x3e, 0x4e, 0xf8, 0x74, 0x4e, 0x3c, 0x4a, 0x61, 0xb7, 0x44, 0x97, 0xe3, 0x69,
	0xc0, 0x3f, 0x93, 0x3a, 0x25, 0xd0, 0x99, 0xf1, 0x81, 0x18, 0x9d, 0xd9, 0x48, 0x63, 0xd8, 0x85,
	0xb6, 0x1d, 0x06, 0xf3, 0xf7, 0x81, 0x6c, 0xb6, 0x4d, 0x9f, 0x81, 0xbf, 0x48, 0x6e, 0x64, 0x1a,
	0x2e, 0x25, 0xae, 0xa8, 0x25, 0x2a, 0xcc, 0x0e, 0xec, 0x3a, 0xcb, 0x66, 0x29, 0x85, 0x2d, 0xd3,
	0xac, 0x25, 0xe2, 0x99, 0x5d, 0x5a, 0x92, 0x90, 0x61, 0x8c, 0xa4, 0xe7, 0x00, 0x57, 0xb7, 0x8b,
	0x85, 0xcc, 0x66, 0xc9, 0x8f, 0x22, 0x67, 0x43, 0x38, 0x11, 0xfa, 0x04, 0xb6, 0xd5, 0x62, 0x91,
	0x4b, 0xdd, 0xab, 0xe3, 0x9d, 0x45, 0x26, 0x7e, 0xf7, 0x45, 0xa6, 0x91, 0xec, 0x35, 0xf6, 0xbd,
	0xc3, 0xa6, 0xb0, 0x88, 0xfd, 0xac, 0xdb, 0xb5, 0x98, 0x0a, 0x42, 0x7e, 0xa7, 0x27, 0xe0, 0x67,
	0x32, 0x8c, 0xe7, 0xf7, 0x2b, 0xf9, 0x37, 0x57, 0x2d, 0x6d, 0x0d, 0x0c, 0x51, 0x54, 0x12, 0xe3,
	0x4c, 0x62, 0x8d, 0xc7, 0x0e, 0x36, 0x9c, 0x29, 0x7f, 0x0a, 0xe3, 0x4c, 0xc9, 0x33, 0x9a, 0xcc,
	0xce, 0x87, 0xdd, 0x6d, 0x68, 0xca, 0xd9, 0x8d, 0xa6, 0xe4, 0xb1, 0x53, 0x67, 0x31, 0x58, 0x98,
	0x40, 0x47, 0xf0, 0x41, 0x70, 0xf9, 0x41, 0xf0, 0x09, 0x9f, 0x0f, 0x48, 0x8d, 0xfa, 0xb0, 0x65,
	0x22, 0xc4, 0xa3, 0x5d, 0x68, 0xe1, 0xdd, 0x8c, 0xf3, 0x77, 0xa4, 0x6e, 0x9c, 0x47, 0x38, 0x98,
	0x93, 0xc6, 0x10, 0xc0, 0x5f, 0x4a, 0x1d, 0x06, 0xa1, 0x0e, 0xd9, 0xc4, 0x49, 0x2a, 0x64, 0xbe,
	0x32, 0x96, 0xc4, 0xa1, 0x0e, 0x71, 0x11, 0x1d, 0x81, 0x67, 0x4a, 0xa0, 0x21, 0xd5, 0x02, 0x87,
	0xf3, 0x85, 0x39, 0x3a, 0x3b, 0x6f, 0xb8, 0x3b, 0x67, 0x2f, 0x61, 0x17, 0xd3, 0x5d, 0x64, 0x89,
	0x96, 0xa5, 0xc5, 0x9b, 0xf9, 0xd8, 0x43, 0xf9, 0xae, 0x90, 0x66, 0x2c, 0x18, 0x40, 0xeb, 0xce,
	0x9c, 0x1d, 0x0f, 0x0e, 0xdc, 0x85, 0x94, 0x44, 0x07, 0xa1, 0x0b, 0x6b, 0xd5, 0x3f, 0xd9, 0xf0,
	0xda, 0x96, 0x75, 0x7c, 0xf8, 0xff, 0x8f, 0xb2, 0x56, 0xb5, 0x66, 0xb2, 0x13, 0x77, 0x4a, 0x2c,
	0xfe, 0x1f, 0x74, 0x2f, 0xc4, 0x78, 0xce, 0x1d, 0x2f, 0x5a, 0xd0, 0xc4, 0x50, 0x61, 0x46, 0x71,
	0xcb, 0xa7, 0x01, 0xa9, 0x3f, 0xda, 0xff, 0x7b, 0x37, 0x15, 0x1a, 0x60, 0x7e, 0xf7, 0x7b, 0x2d,
	0x73, 0x8c, 0x54, 0xbf, 0x7b, 0x15, 0x31, 0x0f, 0xec, 0x76, 0x15, 0x87, 0x5a, 0x8e, 0x03, 0x9c,
	0xb3, 0x25, 0x2a, 0xcc, 0xde, 0xc0, 0x4e, 0xa0, 0x8a, 0x77, 0xb8, 0x07, 0xcd, 0x58, 0x45, 0xe3,
	0xc0, 0xbe, 0xad, 0x02, 0x18, 0xdf, 0x62, 0x15, 0x7d, 0x92, 0x19, 0x4a, 0xb7, 0x84, 0x45, 0xc3,
	0x3e, 0xb0, 0x48, 0x2d, 0xfb, 0xb9, 0xce, 0x54, 0x7a, 0x9d, 0x87, 0x37, 0xda, 0x1e, 0x63, 0x15,
	0xf5, 0xf3, 0xf8, 0x5b, 0xb1, 0x93, 0x61, 0x13, 0x5b, 0x7d, 0xf0, 0xbc, 0xab, 0x6d, 0x0c, 0xbc,
	0xfa, 0x1d, 0x00, 0x00, 0xff, 0xff, 0x13, 0x95, 0xef, 0x7d, 0x36, 0x05, 0x00, 0x00,
}