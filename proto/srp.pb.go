// Code generated by protoc-gen-go. DO NOT EDIT.
// source: srp.proto

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

type RegSrpAuth struct {
	// Which version of the srp authenticator the user will use
	SrpVersion int32 `protobuf:"varint,1,opt,name=srpVersion,proto3" json:"srpVersion,omitempty"`
	// If using the SRP authentication type: the user's srp verifier string
	SrpVerifier          string   `protobuf:"bytes,2,opt,name=srpVerifier,proto3" json:"srpVerifier,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RegSrpAuth) Reset()         { *m = RegSrpAuth{} }
func (m *RegSrpAuth) String() string { return proto.CompactTextString(m) }
func (*RegSrpAuth) ProtoMessage()    {}
func (*RegSrpAuth) Descriptor() ([]byte, []int) {
	return fileDescriptor_226cb96591f587e6, []int{0}
}

func (m *RegSrpAuth) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RegSrpAuth.Unmarshal(m, b)
}
func (m *RegSrpAuth) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RegSrpAuth.Marshal(b, m, deterministic)
}
func (m *RegSrpAuth) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegSrpAuth.Merge(m, src)
}
func (m *RegSrpAuth) XXX_Size() int {
	return xxx_messageInfo_RegSrpAuth.Size(m)
}
func (m *RegSrpAuth) XXX_DiscardUnknown() {
	xxx_messageInfo_RegSrpAuth.DiscardUnknown(m)
}

var xxx_messageInfo_RegSrpAuth proto.InternalMessageInfo

func (m *RegSrpAuth) GetSrpVersion() int32 {
	if m != nil {
		return m.SrpVersion
	}
	return 0
}

func (m *RegSrpAuth) GetSrpVerifier() string {
	if m != nil {
		return m.SrpVerifier
	}
	return ""
}

type SrpAuth struct {
	// Which version of the srp authenticator the user will use
	SrpVersion           int32    `protobuf:"varint,1,opt,name=srpVersion,proto3" json:"srpVersion,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SrpAuth) Reset()         { *m = SrpAuth{} }
func (m *SrpAuth) String() string { return proto.CompactTextString(m) }
func (*SrpAuth) ProtoMessage()    {}
func (*SrpAuth) Descriptor() ([]byte, []int) {
	return fileDescriptor_226cb96591f587e6, []int{1}
}

func (m *SrpAuth) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SrpAuth.Unmarshal(m, b)
}
func (m *SrpAuth) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SrpAuth.Marshal(b, m, deterministic)
}
func (m *SrpAuth) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SrpAuth.Merge(m, src)
}
func (m *SrpAuth) XXX_Size() int {
	return xxx_messageInfo_SrpAuth.Size(m)
}
func (m *SrpAuth) XXX_DiscardUnknown() {
	xxx_messageInfo_SrpAuth.DiscardUnknown(m)
}

var xxx_messageInfo_SrpAuth proto.InternalMessageInfo

func (m *SrpAuth) GetSrpVersion() int32 {
	if m != nil {
		return m.SrpVersion
	}
	return 0
}

// SrpCredential is used for both client and server credentials
type SrpCredential struct {
	SrpCredential        string   `protobuf:"bytes,1,opt,name=srpCredential,proto3" json:"srpCredential,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SrpCredential) Reset()         { *m = SrpCredential{} }
func (m *SrpCredential) String() string { return proto.CompactTextString(m) }
func (*SrpCredential) ProtoMessage()    {}
func (*SrpCredential) Descriptor() ([]byte, []int) {
	return fileDescriptor_226cb96591f587e6, []int{2}
}

func (m *SrpCredential) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SrpCredential.Unmarshal(m, b)
}
func (m *SrpCredential) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SrpCredential.Marshal(b, m, deterministic)
}
func (m *SrpCredential) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SrpCredential.Merge(m, src)
}
func (m *SrpCredential) XXX_Size() int {
	return xxx_messageInfo_SrpCredential.Size(m)
}
func (m *SrpCredential) XXX_DiscardUnknown() {
	xxx_messageInfo_SrpCredential.DiscardUnknown(m)
}

var xxx_messageInfo_SrpCredential proto.InternalMessageInfo

func (m *SrpCredential) GetSrpCredential() string {
	if m != nil {
		return m.SrpCredential
	}
	return ""
}

// SrpAuthProof is used for client generated auth data and server generated proof
type SrpAuthProof struct {
	SrpAuthProof         string   `protobuf:"bytes,1,opt,name=srpAuthProof,proto3" json:"srpAuthProof,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SrpAuthProof) Reset()         { *m = SrpAuthProof{} }
func (m *SrpAuthProof) String() string { return proto.CompactTextString(m) }
func (*SrpAuthProof) ProtoMessage()    {}
func (*SrpAuthProof) Descriptor() ([]byte, []int) {
	return fileDescriptor_226cb96591f587e6, []int{3}
}

func (m *SrpAuthProof) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SrpAuthProof.Unmarshal(m, b)
}
func (m *SrpAuthProof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SrpAuthProof.Marshal(b, m, deterministic)
}
func (m *SrpAuthProof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SrpAuthProof.Merge(m, src)
}
func (m *SrpAuthProof) XXX_Size() int {
	return xxx_messageInfo_SrpAuthProof.Size(m)
}
func (m *SrpAuthProof) XXX_DiscardUnknown() {
	xxx_messageInfo_SrpAuthProof.DiscardUnknown(m)
}

var xxx_messageInfo_SrpAuthProof proto.InternalMessageInfo

func (m *SrpAuthProof) GetSrpAuthProof() string {
	if m != nil {
		return m.SrpAuthProof
	}
	return ""
}

func init() {
	proto.RegisterType((*RegSrpAuth)(nil), "proto.RegSrpAuth")
	proto.RegisterType((*SrpAuth)(nil), "proto.SrpAuth")
	proto.RegisterType((*SrpCredential)(nil), "proto.SrpCredential")
	proto.RegisterType((*SrpAuthProof)(nil), "proto.SrpAuthProof")
}

func init() { proto.RegisterFile("srp.proto", fileDescriptor_226cb96591f587e6) }

var fileDescriptor_226cb96591f587e6 = []byte{
	// 196 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2c, 0x2e, 0x2a, 0xd0,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x4a, 0x7e, 0x5c, 0x5c, 0x41, 0xa9, 0xe9,
	0xc1, 0x45, 0x05, 0x8e, 0xa5, 0x25, 0x19, 0x42, 0x72, 0x5c, 0x5c, 0xc5, 0x45, 0x05, 0x61, 0xa9,
	0x45, 0xc5, 0x99, 0xf9, 0x79, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xac, 0x41, 0x48, 0x22, 0x42, 0x0a,
	0x5c, 0xdc, 0x10, 0x5e, 0x66, 0x5a, 0x66, 0x6a, 0x91, 0x04, 0x93, 0x02, 0xa3, 0x06, 0x67, 0x10,
	0xb2, 0x90, 0x92, 0x26, 0x17, 0x3b, 0x91, 0x86, 0x29, 0x99, 0x72, 0xf1, 0x06, 0x17, 0x15, 0x38,
	0x17, 0xa5, 0xa6, 0xa4, 0xe6, 0x95, 0x64, 0x26, 0xe6, 0x08, 0xa9, 0x70, 0xf1, 0x16, 0x23, 0x0b,
	0x80, 0xf5, 0x70, 0x06, 0xa1, 0x0a, 0x2a, 0x19, 0x71, 0xf1, 0x40, 0x6d, 0x08, 0x28, 0xca, 0xcf,
	0x4f, 0x13, 0x52, 0xe2, 0xe2, 0x29, 0x46, 0xe2, 0x43, 0x35, 0xa1, 0x88, 0x39, 0xe9, 0x70, 0x29,
	0x25, 0xe7, 0xe7, 0xea, 0x15, 0x97, 0x14, 0xe5, 0xe7, 0xa5, 0x17, 0x27, 0xe6, 0x94, 0x40, 0x99,
	0x29, 0xf9, 0xc9, 0x7a, 0xc5, 0x29, 0xd9, 0x90, 0x20, 0x71, 0x62, 0x0e, 0x2e, 0x2a, 0xe8, 0x60,
	0x64, 0x4c, 0x62, 0x03, 0x73, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x1a, 0x8e, 0x0e, 0x37,
	0x2e, 0x01, 0x00, 0x00,
}
