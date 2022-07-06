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

type StoreWhence int32

const (
	StoreWhence_SEEK_START   StoreWhence = 0
	StoreWhence_SEEK_CURRENT StoreWhence = 1
	StoreWhence_SEEK_END     StoreWhence = 2
)

var StoreWhence_name = map[int32]string{
	0: "SEEK_START",
	1: "SEEK_CURRENT",
	2: "SEEK_END",
}

var StoreWhence_value = map[string]int32{
	"SEEK_START":   0,
	"SEEK_CURRENT": 1,
	"SEEK_END":     2,
}

func (x StoreWhence) String() string {
	return proto.EnumName(StoreWhence_name, int32(x))
}

func (StoreWhence) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{0}
}

type StoreInit_StoreContent int32

const (
	StoreInit_GENERIC        StoreInit_StoreContent = 0
	StoreInit_DOCUMENT       StoreInit_StoreContent = 1
	StoreInit_DOCUMENT_INDEX StoreInit_StoreContent = 2
	StoreInit_SEARCH_INDEX   StoreInit_StoreContent = 3
)

var StoreInit_StoreContent_name = map[int32]string{
	0: "GENERIC",
	1: "DOCUMENT",
	2: "DOCUMENT_INDEX",
	3: "SEARCH_INDEX",
}

var StoreInit_StoreContent_value = map[string]int32{
	"GENERIC":        0,
	"DOCUMENT":       1,
	"DOCUMENT_INDEX": 2,
	"SEARCH_INDEX":   3,
}

func (x StoreInit_StoreContent) String() string {
	return proto.EnumName(StoreInit_StoreContent_name, int32(x))
}

func (StoreInit_StoreContent) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{0, 0}
}

type StoreReadReq_StoreReadOp int32

const (
	StoreReadReq_READ_INIT StoreReadReq_StoreReadOp = 0
	StoreReadReq_READ_DATA StoreReadReq_StoreReadOp = 1
	StoreReadReq_READ_SEEK StoreReadReq_StoreReadOp = 2
	StoreReadReq_READ_AT   StoreReadReq_StoreReadOp = 3
	StoreReadReq_READ_END  StoreReadReq_StoreReadOp = 4
)

var StoreReadReq_StoreReadOp_name = map[int32]string{
	0: "READ_INIT",
	1: "READ_DATA",
	2: "READ_SEEK",
	3: "READ_AT",
	4: "READ_END",
}

var StoreReadReq_StoreReadOp_value = map[string]int32{
	"READ_INIT": 0,
	"READ_DATA": 1,
	"READ_SEEK": 2,
	"READ_AT":   3,
	"READ_END":  4,
}

func (x StoreReadReq_StoreReadOp) String() string {
	return proto.EnumName(StoreReadReq_StoreReadOp_name, int32(x))
}

func (StoreReadReq_StoreReadOp) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{4, 0}
}

type StoreWriteReq_StoreWriteOp int32

const (
	StoreWriteReq_WRITE_INIT StoreWriteReq_StoreWriteOp = 0
	StoreWriteReq_WRITE_DATA StoreWriteReq_StoreWriteOp = 1
	StoreWriteReq_WRITE_SEEK StoreWriteReq_StoreWriteOp = 2
	StoreWriteReq_WRITE_AT   StoreWriteReq_StoreWriteOp = 3
	StoreWriteReq_WRITE_END  StoreWriteReq_StoreWriteOp = 4
)

var StoreWriteReq_StoreWriteOp_name = map[int32]string{
	0: "WRITE_INIT",
	1: "WRITE_DATA",
	2: "WRITE_SEEK",
	3: "WRITE_AT",
	4: "WRITE_END",
}

var StoreWriteReq_StoreWriteOp_value = map[string]int32{
	"WRITE_INIT": 0,
	"WRITE_DATA": 1,
	"WRITE_SEEK": 2,
	"WRITE_AT":   3,
	"WRITE_END":  4,
}

func (x StoreWriteReq_StoreWriteOp) String() string {
	return proto.EnumName(StoreWriteReq_StoreWriteOp_name, int32(x))
}

func (StoreWriteReq_StoreWriteOp) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{7, 0}
}

type StoreInit struct {
	Content StoreInit_StoreContent `protobuf:"varint,1,opt,name=content,proto3,enum=proto.StoreInit_StoreContent" json:"content,omitempty"`
	// Types that are valid to be assigned to Init:
	//	*StoreInit_Generic
	//	*StoreInit_Doc
	//	*StoreInit_Json
	Init                 isStoreInit_Init `protobuf_oneof:"init"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *StoreInit) Reset()         { *m = StoreInit{} }
func (m *StoreInit) String() string { return proto.CompactTextString(m) }
func (*StoreInit) ProtoMessage()    {}
func (*StoreInit) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{0}
}

func (m *StoreInit) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreInit.Unmarshal(m, b)
}
func (m *StoreInit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreInit.Marshal(b, m, deterministic)
}
func (m *StoreInit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreInit.Merge(m, src)
}
func (m *StoreInit) XXX_Size() int {
	return xxx_messageInfo_StoreInit.Size(m)
}
func (m *StoreInit) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreInit.DiscardUnknown(m)
}

var xxx_messageInfo_StoreInit proto.InternalMessageInfo

func (m *StoreInit) GetContent() StoreInit_StoreContent {
	if m != nil {
		return m.Content
	}
	return StoreInit_GENERIC
}

type isStoreInit_Init interface {
	isStoreInit_Init()
}

type StoreInit_Generic struct {
	Generic *GenericStoreInit `protobuf:"bytes,100,opt,name=generic,proto3,oneof"`
}

type StoreInit_Doc struct {
	Doc *DocStoreInit `protobuf:"bytes,200,opt,name=doc,proto3,oneof"`
}

type StoreInit_Json struct {
	Json *StoreJson `protobuf:"bytes,1000,opt,name=json,proto3,oneof"`
}

func (*StoreInit_Generic) isStoreInit_Init() {}

func (*StoreInit_Doc) isStoreInit_Init() {}

func (*StoreInit_Json) isStoreInit_Init() {}

func (m *StoreInit) GetInit() isStoreInit_Init {
	if m != nil {
		return m.Init
	}
	return nil
}

func (m *StoreInit) GetGeneric() *GenericStoreInit {
	if x, ok := m.GetInit().(*StoreInit_Generic); ok {
		return x.Generic
	}
	return nil
}

func (m *StoreInit) GetDoc() *DocStoreInit {
	if x, ok := m.GetInit().(*StoreInit_Doc); ok {
		return x.Doc
	}
	return nil
}

func (m *StoreInit) GetJson() *StoreJson {
	if x, ok := m.GetInit().(*StoreInit_Json); ok {
		return x.Json
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreInit) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreInit_Generic)(nil),
		(*StoreInit_Doc)(nil),
		(*StoreInit_Json)(nil),
	}
}

type GenericStoreInit struct {
	Filename             string   `protobuf:"bytes,1,opt,name=filename,proto3" json:"filename,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GenericStoreInit) Reset()         { *m = GenericStoreInit{} }
func (m *GenericStoreInit) String() string { return proto.CompactTextString(m) }
func (*GenericStoreInit) ProtoMessage()    {}
func (*GenericStoreInit) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{1}
}

func (m *GenericStoreInit) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GenericStoreInit.Unmarshal(m, b)
}
func (m *GenericStoreInit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GenericStoreInit.Marshal(b, m, deterministic)
}
func (m *GenericStoreInit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenericStoreInit.Merge(m, src)
}
func (m *GenericStoreInit) XXX_Size() int {
	return xxx_messageInfo_GenericStoreInit.Size(m)
}
func (m *GenericStoreInit) XXX_DiscardUnknown() {
	xxx_messageInfo_GenericStoreInit.DiscardUnknown(m)
}

var xxx_messageInfo_GenericStoreInit proto.InternalMessageInfo

func (m *GenericStoreInit) GetFilename() string {
	if m != nil {
		return m.Filename
	}
	return ""
}

type StoreJson struct {
	Version              int64    `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Json                 string   `protobuf:"bytes,2,opt,name=json,proto3" json:"json,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StoreJson) Reset()         { *m = StoreJson{} }
func (m *StoreJson) String() string { return proto.CompactTextString(m) }
func (*StoreJson) ProtoMessage()    {}
func (*StoreJson) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{2}
}

func (m *StoreJson) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreJson.Unmarshal(m, b)
}
func (m *StoreJson) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreJson.Marshal(b, m, deterministic)
}
func (m *StoreJson) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreJson.Merge(m, src)
}
func (m *StoreJson) XXX_Size() int {
	return xxx_messageInfo_StoreJson.Size(m)
}
func (m *StoreJson) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreJson.DiscardUnknown(m)
}

var xxx_messageInfo_StoreJson proto.InternalMessageInfo

func (m *StoreJson) GetVersion() int64 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *StoreJson) GetJson() string {
	if m != nil {
		return m.Json
	}
	return ""
}

// StoreReadMeta
type StoreRead struct {
	BufferSize           int64       `protobuf:"varint,1,opt,name=bufferSize,proto3" json:"bufferSize,omitempty"`
	Offset               int64       `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Whence               StoreWhence `protobuf:"varint,3,opt,name=whence,proto3,enum=proto.StoreWhence" json:"whence,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *StoreRead) Reset()         { *m = StoreRead{} }
func (m *StoreRead) String() string { return proto.CompactTextString(m) }
func (*StoreRead) ProtoMessage()    {}
func (*StoreRead) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{3}
}

func (m *StoreRead) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreRead.Unmarshal(m, b)
}
func (m *StoreRead) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreRead.Marshal(b, m, deterministic)
}
func (m *StoreRead) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreRead.Merge(m, src)
}
func (m *StoreRead) XXX_Size() int {
	return xxx_messageInfo_StoreRead.Size(m)
}
func (m *StoreRead) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreRead.DiscardUnknown(m)
}

var xxx_messageInfo_StoreRead proto.InternalMessageInfo

func (m *StoreRead) GetBufferSize() int64 {
	if m != nil {
		return m.BufferSize
	}
	return 0
}

func (m *StoreRead) GetOffset() int64 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *StoreRead) GetWhence() StoreWhence {
	if m != nil {
		return m.Whence
	}
	return StoreWhence_SEEK_START
}

// StoreReadReq
type StoreReadReq struct {
	ReadOp StoreReadReq_StoreReadOp `protobuf:"varint,1,opt,name=readOp,proto3,enum=proto.StoreReadReq_StoreReadOp" json:"readOp,omitempty"`
	// Types that are valid to be assigned to Req:
	//	*StoreReadReq_Init
	//	*StoreReadReq_Read
	Req                  isStoreReadReq_Req `protobuf_oneof:"req"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
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

func (m *StoreReadReq) GetReadOp() StoreReadReq_StoreReadOp {
	if m != nil {
		return m.ReadOp
	}
	return StoreReadReq_READ_INIT
}

type isStoreReadReq_Req interface {
	isStoreReadReq_Req()
}

type StoreReadReq_Init struct {
	Init *StoreInit `protobuf:"bytes,10,opt,name=init,proto3,oneof"`
}

type StoreReadReq_Read struct {
	Read *StoreRead `protobuf:"bytes,20,opt,name=read,proto3,oneof"`
}

func (*StoreReadReq_Init) isStoreReadReq_Req() {}

func (*StoreReadReq_Read) isStoreReadReq_Req() {}

func (m *StoreReadReq) GetReq() isStoreReadReq_Req {
	if m != nil {
		return m.Req
	}
	return nil
}

func (m *StoreReadReq) GetInit() *StoreInit {
	if x, ok := m.GetReq().(*StoreReadReq_Init); ok {
		return x.Init
	}
	return nil
}

func (m *StoreReadReq) GetRead() *StoreRead {
	if x, ok := m.GetReq().(*StoreReadReq_Read); ok {
		return x.Read
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreReadReq) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreReadReq_Init)(nil),
		(*StoreReadReq_Read)(nil),
	}
}

type StoreReadResp struct {
	Data   []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Eof    bool   `protobuf:"varint,2,opt,name=eof,proto3" json:"eof,omitempty"`
	Offset int64  `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	Size   int64  `protobuf:"varint,4,opt,name=size,proto3" json:"size,omitempty"`
	// Types that are valid to be assigned to Resp:
	//	*StoreReadResp_Init
	//	*StoreReadResp_Json
	Resp                 isStoreReadResp_Resp `protobuf_oneof:"resp"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
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

func (m *StoreReadResp) GetSize() int64 {
	if m != nil {
		return m.Size
	}
	return 0
}

type isStoreReadResp_Resp interface {
	isStoreReadResp_Resp()
}

type StoreReadResp_Init struct {
	Init *StoreInit `protobuf:"bytes,100,opt,name=init,proto3,oneof"`
}

type StoreReadResp_Json struct {
	Json *StoreJson `protobuf:"bytes,200,opt,name=json,proto3,oneof"`
}

func (*StoreReadResp_Init) isStoreReadResp_Resp() {}

func (*StoreReadResp_Json) isStoreReadResp_Resp() {}

func (m *StoreReadResp) GetResp() isStoreReadResp_Resp {
	if m != nil {
		return m.Resp
	}
	return nil
}

func (m *StoreReadResp) GetInit() *StoreInit {
	if x, ok := m.GetResp().(*StoreReadResp_Init); ok {
		return x.Init
	}
	return nil
}

func (m *StoreReadResp) GetJson() *StoreJson {
	if x, ok := m.GetResp().(*StoreReadResp_Json); ok {
		return x.Json
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreReadResp) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreReadResp_Init)(nil),
		(*StoreReadResp_Json)(nil),
	}
}

type StoreWrite struct {
	Data                 []byte      `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Offset               int64       `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Whence               StoreWhence `protobuf:"varint,3,opt,name=whence,proto3,enum=proto.StoreWhence" json:"whence,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *StoreWrite) Reset()         { *m = StoreWrite{} }
func (m *StoreWrite) String() string { return proto.CompactTextString(m) }
func (*StoreWrite) ProtoMessage()    {}
func (*StoreWrite) Descriptor() ([]byte, []int) {
	return fileDescriptor_98bbca36ef968dfc, []int{6}
}

func (m *StoreWrite) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StoreWrite.Unmarshal(m, b)
}
func (m *StoreWrite) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StoreWrite.Marshal(b, m, deterministic)
}
func (m *StoreWrite) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StoreWrite.Merge(m, src)
}
func (m *StoreWrite) XXX_Size() int {
	return xxx_messageInfo_StoreWrite.Size(m)
}
func (m *StoreWrite) XXX_DiscardUnknown() {
	xxx_messageInfo_StoreWrite.DiscardUnknown(m)
}

var xxx_messageInfo_StoreWrite proto.InternalMessageInfo

func (m *StoreWrite) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *StoreWrite) GetOffset() int64 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *StoreWrite) GetWhence() StoreWhence {
	if m != nil {
		return m.Whence
	}
	return StoreWhence_SEEK_START
}

type StoreWriteReq struct {
	WriteOp StoreWriteReq_StoreWriteOp `protobuf:"varint,1,opt,name=writeOp,proto3,enum=proto.StoreWriteReq_StoreWriteOp" json:"writeOp,omitempty"`
	// Types that are valid to be assigned to Req:
	//	*StoreWriteReq_Init
	//	*StoreWriteReq_Write
	Req                  isStoreWriteReq_Req `protobuf_oneof:"req"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
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

func (m *StoreWriteReq) GetWriteOp() StoreWriteReq_StoreWriteOp {
	if m != nil {
		return m.WriteOp
	}
	return StoreWriteReq_WRITE_INIT
}

type isStoreWriteReq_Req interface {
	isStoreWriteReq_Req()
}

type StoreWriteReq_Init struct {
	Init *StoreInit `protobuf:"bytes,100,opt,name=init,proto3,oneof"`
}

type StoreWriteReq_Write struct {
	Write *StoreWrite `protobuf:"bytes,200,opt,name=write,proto3,oneof"`
}

func (*StoreWriteReq_Init) isStoreWriteReq_Req() {}

func (*StoreWriteReq_Write) isStoreWriteReq_Req() {}

func (m *StoreWriteReq) GetReq() isStoreWriteReq_Req {
	if m != nil {
		return m.Req
	}
	return nil
}

func (m *StoreWriteReq) GetInit() *StoreInit {
	if x, ok := m.GetReq().(*StoreWriteReq_Init); ok {
		return x.Init
	}
	return nil
}

func (m *StoreWriteReq) GetWrite() *StoreWrite {
	if x, ok := m.GetReq().(*StoreWriteReq_Write); ok {
		return x.Write
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreWriteReq) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreWriteReq_Init)(nil),
		(*StoreWriteReq_Write)(nil),
	}
}

type StoreWriteResp struct {
	WrittenBytes int64 `protobuf:"varint,1,opt,name=writtenBytes,proto3" json:"writtenBytes,omitempty"`
	SeekOffset   int64 `protobuf:"varint,2,opt,name=seekOffset,proto3" json:"seekOffset,omitempty"`
	// Types that are valid to be assigned to Resp:
	//	*StoreWriteResp_Init
	//	*StoreWriteResp_Json
	Resp                 isStoreWriteResp_Resp `protobuf_oneof:"resp"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
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

func (m *StoreWriteResp) GetWrittenBytes() int64 {
	if m != nil {
		return m.WrittenBytes
	}
	return 0
}

func (m *StoreWriteResp) GetSeekOffset() int64 {
	if m != nil {
		return m.SeekOffset
	}
	return 0
}

type isStoreWriteResp_Resp interface {
	isStoreWriteResp_Resp()
}

type StoreWriteResp_Init struct {
	Init *StoreInit `protobuf:"bytes,100,opt,name=init,proto3,oneof"`
}

type StoreWriteResp_Json struct {
	Json *StoreJson `protobuf:"bytes,200,opt,name=json,proto3,oneof"`
}

func (*StoreWriteResp_Init) isStoreWriteResp_Resp() {}

func (*StoreWriteResp_Json) isStoreWriteResp_Resp() {}

func (m *StoreWriteResp) GetResp() isStoreWriteResp_Resp {
	if m != nil {
		return m.Resp
	}
	return nil
}

func (m *StoreWriteResp) GetInit() *StoreInit {
	if x, ok := m.GetResp().(*StoreWriteResp_Init); ok {
		return x.Init
	}
	return nil
}

func (m *StoreWriteResp) GetJson() *StoreJson {
	if x, ok := m.GetResp().(*StoreWriteResp_Json); ok {
		return x.Json
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StoreWriteResp) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StoreWriteResp_Init)(nil),
		(*StoreWriteResp_Json)(nil),
	}
}

func init() {
	proto.RegisterEnum("proto.StoreWhence", StoreWhence_name, StoreWhence_value)
	proto.RegisterEnum("proto.StoreInit_StoreContent", StoreInit_StoreContent_name, StoreInit_StoreContent_value)
	proto.RegisterEnum("proto.StoreReadReq_StoreReadOp", StoreReadReq_StoreReadOp_name, StoreReadReq_StoreReadOp_value)
	proto.RegisterEnum("proto.StoreWriteReq_StoreWriteOp", StoreWriteReq_StoreWriteOp_name, StoreWriteReq_StoreWriteOp_value)
	proto.RegisterType((*StoreInit)(nil), "proto.StoreInit")
	proto.RegisterType((*GenericStoreInit)(nil), "proto.GenericStoreInit")
	proto.RegisterType((*StoreJson)(nil), "proto.StoreJson")
	proto.RegisterType((*StoreRead)(nil), "proto.StoreRead")
	proto.RegisterType((*StoreReadReq)(nil), "proto.StoreReadReq")
	proto.RegisterType((*StoreReadResp)(nil), "proto.StoreReadResp")
	proto.RegisterType((*StoreWrite)(nil), "proto.StoreWrite")
	proto.RegisterType((*StoreWriteReq)(nil), "proto.StoreWriteReq")
	proto.RegisterType((*StoreWriteResp)(nil), "proto.StoreWriteResp")
}

func init() { proto.RegisterFile("store.proto", fileDescriptor_98bbca36ef968dfc) }

var fileDescriptor_98bbca36ef968dfc = []byte{
	// 714 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x54, 0xcd, 0x6e, 0xd3, 0x40,
	0x10, 0x8e, 0xe3, 0xfc, 0x4e, 0xdc, 0xc8, 0x2c, 0x08, 0xac, 0x4a, 0x40, 0xf1, 0x01, 0xaa, 0x1e,
	0x72, 0x68, 0x0f, 0x15, 0x42, 0x1c, 0xf2, 0x63, 0x35, 0xa1, 0x22, 0x91, 0x36, 0xa9, 0x8a, 0xc4,
	0x21, 0x4a, 0xed, 0x4d, 0x31, 0x6d, 0x77, 0x5d, 0xef, 0x96, 0x0a, 0xce, 0x1c, 0x78, 0x11, 0x8e,
	0xbc, 0x00, 0x4f, 0xc0, 0xa3, 0xf0, 0x02, 0xdc, 0xd1, 0xfe, 0xd8, 0x38, 0x04, 0x04, 0x07, 0x38,
	0x65, 0xbe, 0xd9, 0x6f, 0x32, 0x3b, 0xdf, 0x7c, 0x6b, 0x68, 0x71, 0xc1, 0x52, 0xd2, 0x49, 0x52,
	0x26, 0x18, 0xaa, 0xaa, 0x9f, 0xcd, 0x76, 0xc4, 0xc2, 0xab, 0x0b, 0x42, 0x85, 0x4e, 0xfb, 0x1f,
	0xcb, 0xd0, 0x9c, 0x4a, 0xda, 0x88, 0xc6, 0x02, 0xed, 0x43, 0x3d, 0x64, 0x54, 0x10, 0x2a, 0x3c,
	0x6b, 0xcb, 0xda, 0x6e, 0xef, 0xde, 0xd5, 0xb4, 0x4e, 0x4e, 0xd1, 0x51, 0x5f, 0x93, 0x70, 0xc6,
	0x46, 0x7b, 0x50, 0x3f, 0x25, 0x94, 0xa4, 0x71, 0xe8, 0x45, 0x5b, 0xd6, 0x76, 0x6b, 0xf7, 0x8e,
	0x29, 0x3c, 0xd0, 0xd9, 0xbc, 0x7e, 0x58, 0xc2, 0x19, 0x13, 0x6d, 0x83, 0x1d, 0xb1, 0xd0, 0xfb,
	0x62, 0xa9, 0x8a, 0x9b, 0xa6, 0x62, 0xc0, 0x56, 0xd8, 0x92, 0x82, 0x1e, 0x41, 0xe5, 0x35, 0x67,
	0xd4, 0xfb, 0x5a, 0x57, 0x54, 0xb7, 0x78, 0xab, 0x67, 0x9c, 0xd1, 0x61, 0x09, 0x2b, 0x82, 0x3f,
	0x01, 0xa7, 0x78, 0x41, 0xd4, 0x82, 0xfa, 0x41, 0x30, 0x0e, 0xf0, 0xa8, 0xef, 0x96, 0x90, 0x03,
	0x8d, 0xc1, 0xa4, 0x7f, 0xf4, 0x3c, 0x18, 0xcf, 0x5c, 0x0b, 0x21, 0x68, 0x67, 0x68, 0x3e, 0x1a,
	0x0f, 0x82, 0x17, 0x6e, 0x19, 0xb9, 0xe0, 0x4c, 0x83, 0x2e, 0xee, 0x0f, 0x4d, 0xc6, 0xee, 0xd5,
	0xa0, 0x12, 0xd3, 0x58, 0xf8, 0x1d, 0x70, 0x7f, 0x1e, 0x05, 0x6d, 0x42, 0x63, 0x19, 0x9f, 0x13,
	0xba, 0xb8, 0x20, 0x4a, 0xae, 0x26, 0xce, 0xb1, 0xff, 0xd8, 0xc8, 0x2a, 0x6f, 0x87, 0x3c, 0xa8,
	0xbf, 0x21, 0x29, 0x8f, 0x19, 0x55, 0x3c, 0x1b, 0x67, 0x10, 0x21, 0x33, 0x58, 0x59, 0x95, 0xeb,
	0x19, 0x98, 0x29, 0xc5, 0x64, 0x11, 0xa1, 0x7b, 0x00, 0x27, 0x57, 0xcb, 0x25, 0x49, 0xa7, 0xf1,
	0x3b, 0x62, 0xaa, 0x0b, 0x19, 0x74, 0x1b, 0x6a, 0x6c, 0xb9, 0xe4, 0x44, 0xa8, 0xbf, 0xb0, 0xb1,
	0x41, 0x68, 0x07, 0x6a, 0xd7, 0xaf, 0x08, 0x0d, 0x89, 0x67, 0xab, 0x45, 0xa2, 0xa2, 0x64, 0xc7,
	0xea, 0x04, 0x1b, 0x86, 0xff, 0xcd, 0x32, 0xaa, 0xc9, 0x8e, 0x98, 0x5c, 0xa2, 0x7d, 0xa8, 0xa5,
	0x64, 0x11, 0x4d, 0x12, 0xe3, 0x82, 0xfb, 0xc5, 0x62, 0x43, 0xfa, 0x01, 0x26, 0x09, 0x36, 0x74,
	0xf4, 0x50, 0xab, 0xe5, 0xc1, 0xfa, 0x9a, 0xcc, 0x3a, 0xd5, 0xb9, 0xe4, 0xc9, 0x0a, 0xef, 0xd6,
	0x3a, 0x4f, 0xfe, 0xa3, 0xe4, 0xc9, 0x73, 0xff, 0x08, 0x5a, 0x85, 0x36, 0x68, 0x03, 0x9a, 0x38,
	0xe8, 0x0e, 0xe6, 0xa3, 0xf1, 0x68, 0xe6, 0x96, 0x72, 0x38, 0xe8, 0xce, 0xba, 0xae, 0x95, 0xc3,
	0x69, 0x10, 0x1c, 0xba, 0x65, 0xb9, 0x7a, 0x05, 0xbb, 0x33, 0xd7, 0x96, 0xab, 0x57, 0x20, 0x18,
	0x0f, 0xdc, 0x4a, 0xaf, 0x0a, 0x76, 0x4a, 0x2e, 0xfd, 0xcf, 0x16, 0x6c, 0x14, 0x46, 0xe2, 0x89,
	0x5c, 0x47, 0xb4, 0x10, 0x0b, 0x35, 0xb6, 0x83, 0x55, 0x8c, 0x5c, 0xb0, 0x09, 0x5b, 0x2a, 0x79,
	0x1b, 0x58, 0x86, 0x05, 0xcd, 0xed, 0x15, 0xcd, 0x11, 0x54, 0xb8, 0xdc, 0x52, 0x45, 0x65, 0x55,
	0x9c, 0x2b, 0x12, 0xfd, 0x41, 0x91, 0xcc, 0xe1, 0xe6, 0x31, 0xfc, 0xd6, 0xe1, 0xd2, 0x90, 0x29,
	0xe1, 0x89, 0x1f, 0x01, 0xe8, 0x5d, 0xa6, 0xb1, 0x20, 0xbf, 0xbc, 0xf8, 0xbf, 0xb0, 0xc6, 0xfb,
	0xb2, 0x91, 0x48, 0xb5, 0x91, 0xde, 0x78, 0x02, 0xf5, 0x6b, 0x19, 0xe7, 0xe6, 0x78, 0xb0, 0x52,
	0x6e, 0x68, 0x05, 0x34, 0x49, 0x70, 0x56, 0xf1, 0xd7, 0x6a, 0xec, 0x40, 0x55, 0x95, 0x64, 0x72,
	0xdc, 0x58, 0xeb, 0x31, 0x2c, 0x61, 0x4d, 0xf1, 0x5f, 0x1a, 0xf3, 0x9a, 0x66, 0xa8, 0x0d, 0x70,
	0x8c, 0x47, 0xb3, 0x20, 0x73, 0x49, 0x8e, 0x8d, 0x4d, 0x72, 0x6c, 0x7c, 0xe2, 0x40, 0x43, 0x63,
	0x65, 0x94, 0x0d, 0x68, 0x6a, 0xb4, 0xe2, 0x94, 0x4f, 0x16, 0xb4, 0x8b, 0xf3, 0xf1, 0x04, 0xf9,
	0xe0, 0xc8, 0xfe, 0x82, 0xd0, 0xde, 0x5b, 0x41, 0xb8, 0x79, 0x9a, 0x2b, 0x39, 0xf9, 0x78, 0x39,
	0x21, 0x67, 0x93, 0xe2, 0x16, 0x0a, 0x99, 0xff, 0x66, 0x8e, 0x9d, 0xa7, 0xe6, 0xdd, 0xe8, 0x6d,
	0xca, 0x91, 0xe5, 0xb0, 0xf3, 0xe9, 0xac, 0x8b, 0xa5, 0x24, 0xea, 0x33, 0x17, 0x1c, 0xce, 0xfb,
	0x47, 0x18, 0xeb, 0x8f, 0xa1, 0x03, 0x0d, 0x95, 0x91, 0x53, 0x97, 0x7b, 0x1d, 0xf0, 0x43, 0x76,
	0xd1, 0xe1, 0x22, 0x65, 0xf4, 0x94, 0x2f, 0xce, 0x85, 0x09, 0x23, 0x16, 0x76, 0x78, 0x74, 0xa6,
	0xdb, 0xf7, 0xaa, 0xaa, 0xc5, 0x07, 0xcb, 0x3a, 0xa9, 0xa9, 0xc4, 0xde, 0xf7, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x97, 0x64, 0x41, 0x18, 0x71, 0x06, 0x00, 0x00,
}
