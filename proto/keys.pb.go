// Code generated by protoc-gen-go. DO NOT EDIT.
// source: keys.proto

package proto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
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

type KeyType int32

const (
	KeyType_KDF        KeyType = 0
	KeyType_HMAC       KeyType = 1
	KeyType_SYMMETRIC  KeyType = 2
	KeyType_ASYMMETRIC KeyType = 3
)

var KeyType_name = map[int32]string{
	0: "KDF",
	1: "HMAC",
	2: "SYMMETRIC",
	3: "ASYMMETRIC",
}

var KeyType_value = map[string]int32{
	"KDF":        0,
	"HMAC":       1,
	"SYMMETRIC":  2,
	"ASYMMETRIC": 3,
}

func (x KeyType) String() string {
	return proto.EnumName(KeyType_name, int32(x))
}

func (KeyType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_9084e97af2346a26, []int{0}
}

type Key struct {
	// This is the unique identifier of the key.
	KeyID   string  `protobuf:"bytes,1,opt,name=keyID,proto3" json:"keyID,omitempty"`
	KeyType KeyType `protobuf:"varint,2,opt,name=keyType,proto3,enum=proto.KeyType" json:"keyType,omitempty"`
	// This is the public data for the key. For KDF, this is the KDF metadata.
	// For asymmetric keys, this would be the serialized public key
	PublicData []byte `protobuf:"bytes,3,opt,name=publicData,proto3" json:"publicData,omitempty"`
	// NOTE: OwnerID is not required. In general:
	// 1. For user keys like KDF for asymmetric Key, the OwnerID is probably the userID
	// 2. For org asymmetric key, the OwnerID is probably the orgID
	// 3. For doc symmetric key, the OwnerID is probably the docID
	OwnerID string `protobuf:"bytes,4,opt,name=ownerID,proto3" json:"ownerID,omitempty"`
	// The encrypted version of this key
	EncryptedKeys        []*EncryptedKey      `protobuf:"bytes,5,rep,name=encryptedKeys,proto3" json:"encryptedKeys,omitempty"`
	CreatedAt            *timestamp.Timestamp `protobuf:"bytes,6,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Key) Reset()         { *m = Key{} }
func (m *Key) String() string { return proto.CompactTextString(m) }
func (*Key) ProtoMessage()    {}
func (*Key) Descriptor() ([]byte, []int) {
	return fileDescriptor_9084e97af2346a26, []int{0}
}

func (m *Key) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Key.Unmarshal(m, b)
}
func (m *Key) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Key.Marshal(b, m, deterministic)
}
func (m *Key) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Key.Merge(m, src)
}
func (m *Key) XXX_Size() int {
	return xxx_messageInfo_Key.Size(m)
}
func (m *Key) XXX_DiscardUnknown() {
	xxx_messageInfo_Key.DiscardUnknown(m)
}

var xxx_messageInfo_Key proto.InternalMessageInfo

func (m *Key) GetKeyID() string {
	if m != nil {
		return m.KeyID
	}
	return ""
}

func (m *Key) GetKeyType() KeyType {
	if m != nil {
		return m.KeyType
	}
	return KeyType_KDF
}

func (m *Key) GetPublicData() []byte {
	if m != nil {
		return m.PublicData
	}
	return nil
}

func (m *Key) GetOwnerID() string {
	if m != nil {
		return m.OwnerID
	}
	return ""
}

func (m *Key) GetEncryptedKeys() []*EncryptedKey {
	if m != nil {
		return m.EncryptedKeys
	}
	return nil
}

func (m *Key) GetCreatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAt
	}
	return nil
}

type EncryptedKey struct {
	// Specifies which key is being encrypted
	KeyID string `protobuf:"bytes,1,opt,name=keyID,proto3" json:"keyID,omitempty"`
	// Specifies the encrypting key. If it's nil, then the key is not encrypted
	Encryptor *Encryptor `protobuf:"bytes,2,opt,name=encryptor,proto3" json:"encryptor,omitempty"`
	// The encrypted key
	Ciphertext           []byte               `protobuf:"bytes,3,opt,name=ciphertext,proto3" json:"ciphertext,omitempty"`
	CreatedAt            *timestamp.Timestamp `protobuf:"bytes,4,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
	DeletedAt            *timestamp.Timestamp `protobuf:"bytes,5,opt,name=deletedAt,proto3" json:"deletedAt,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *EncryptedKey) Reset()         { *m = EncryptedKey{} }
func (m *EncryptedKey) String() string { return proto.CompactTextString(m) }
func (*EncryptedKey) ProtoMessage()    {}
func (*EncryptedKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_9084e97af2346a26, []int{1}
}

func (m *EncryptedKey) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EncryptedKey.Unmarshal(m, b)
}
func (m *EncryptedKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EncryptedKey.Marshal(b, m, deterministic)
}
func (m *EncryptedKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EncryptedKey.Merge(m, src)
}
func (m *EncryptedKey) XXX_Size() int {
	return xxx_messageInfo_EncryptedKey.Size(m)
}
func (m *EncryptedKey) XXX_DiscardUnknown() {
	xxx_messageInfo_EncryptedKey.DiscardUnknown(m)
}

var xxx_messageInfo_EncryptedKey proto.InternalMessageInfo

func (m *EncryptedKey) GetKeyID() string {
	if m != nil {
		return m.KeyID
	}
	return ""
}

func (m *EncryptedKey) GetEncryptor() *Encryptor {
	if m != nil {
		return m.Encryptor
	}
	return nil
}

func (m *EncryptedKey) GetCiphertext() []byte {
	if m != nil {
		return m.Ciphertext
	}
	return nil
}

func (m *EncryptedKey) GetCreatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAt
	}
	return nil
}

func (m *EncryptedKey) GetDeletedAt() *timestamp.Timestamp {
	if m != nil {
		return m.DeletedAt
	}
	return nil
}

type Encryptor struct {
	// Location:
	//    mongo - The encryptor is another key in the mongo database. The LocID is the KeyID of the
	//            encryption key
	Location   string `protobuf:"bytes,1,opt,name=location,proto3" json:"location,omitempty"`
	LocationID string `protobuf:"bytes,2,opt,name=locationID,proto3" json:"locationID,omitempty"`
	// This specifies who owns the encryptor key. This is an optimization to help speed up query.
	// See Key.OwnerID
	OwnerID              string   `protobuf:"bytes,3,opt,name=ownerID,proto3" json:"ownerID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Encryptor) Reset()         { *m = Encryptor{} }
func (m *Encryptor) String() string { return proto.CompactTextString(m) }
func (*Encryptor) ProtoMessage()    {}
func (*Encryptor) Descriptor() ([]byte, []int) {
	return fileDescriptor_9084e97af2346a26, []int{2}
}

func (m *Encryptor) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Encryptor.Unmarshal(m, b)
}
func (m *Encryptor) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Encryptor.Marshal(b, m, deterministic)
}
func (m *Encryptor) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Encryptor.Merge(m, src)
}
func (m *Encryptor) XXX_Size() int {
	return xxx_messageInfo_Encryptor.Size(m)
}
func (m *Encryptor) XXX_DiscardUnknown() {
	xxx_messageInfo_Encryptor.DiscardUnknown(m)
}

var xxx_messageInfo_Encryptor proto.InternalMessageInfo

func (m *Encryptor) GetLocation() string {
	if m != nil {
		return m.Location
	}
	return ""
}

func (m *Encryptor) GetLocationID() string {
	if m != nil {
		return m.LocationID
	}
	return ""
}

func (m *Encryptor) GetOwnerID() string {
	if m != nil {
		return m.OwnerID
	}
	return ""
}

type KeyChain struct {
	KeyChain             []*Key   `protobuf:"bytes,1,rep,name=keyChain,proto3" json:"keyChain,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *KeyChain) Reset()         { *m = KeyChain{} }
func (m *KeyChain) String() string { return proto.CompactTextString(m) }
func (*KeyChain) ProtoMessage()    {}
func (*KeyChain) Descriptor() ([]byte, []int) {
	return fileDescriptor_9084e97af2346a26, []int{3}
}

func (m *KeyChain) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeyChain.Unmarshal(m, b)
}
func (m *KeyChain) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeyChain.Marshal(b, m, deterministic)
}
func (m *KeyChain) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeyChain.Merge(m, src)
}
func (m *KeyChain) XXX_Size() int {
	return xxx_messageInfo_KeyChain.Size(m)
}
func (m *KeyChain) XXX_DiscardUnknown() {
	xxx_messageInfo_KeyChain.DiscardUnknown(m)
}

var xxx_messageInfo_KeyChain proto.InternalMessageInfo

func (m *KeyChain) GetKeyChain() []*Key {
	if m != nil {
		return m.KeyChain
	}
	return nil
}

func init() {
	proto.RegisterEnum("proto.KeyType", KeyType_name, KeyType_value)
	proto.RegisterType((*Key)(nil), "proto.Key")
	proto.RegisterType((*EncryptedKey)(nil), "proto.EncryptedKey")
	proto.RegisterType((*Encryptor)(nil), "proto.Encryptor")
	proto.RegisterType((*KeyChain)(nil), "proto.KeyChain")
}

func init() { proto.RegisterFile("keys.proto", fileDescriptor_9084e97af2346a26) }

var fileDescriptor_9084e97af2346a26 = []byte{
	// 873 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xec, 0x56, 0xcf, 0x6f, 0x23, 0x35,
	0x14, 0x26, 0x9b, 0x49, 0xdb, 0xb8, 0xbb, 0x55, 0x35, 0x70, 0x88, 0x7a, 0x00, 0xcb, 0x07, 0x14,
	0x55, 0xe9, 0xb4, 0xcd, 0x02, 0x82, 0x70, 0xd9, 0x6c, 0x53, 0xb4, 0x51, 0xb6, 0xbb, 0xd5, 0x34,
	0x6c, 0x85, 0x28, 0x07, 0x67, 0xfc, 0x32, 0xb1, 0x32, 0x63, 0x8f, 0x6c, 0x4f, 0xcb, 0x50, 0xf5,
	0xce, 0xb9, 0xff, 0x1d, 0x17, 0x24, 0x84, 0xc4, 0x85, 0x03, 0x42, 0x02, 0x89, 0x23, 0xf2, 0xfc,
	0xc8, 0x8f, 0x03, 0x42, 0x7b, 0xd8, 0x5b, 0x4f, 0x7e, 0xef, 0x7d, 0xef, 0x9b, 0xe7, 0xe7, 0x79,
	0x9f, 0x6c, 0x84, 0xe6, 0x90, 0x69, 0x2f, 0x51, 0xd2, 0x48, 0xb7, 0x91, 0x2f, 0x7b, 0x9d, 0x7c,
	0x09, 0x0e, 0x42, 0x10, 0x07, 0xfa, 0x86, 0x86, 0x21, 0xa8, 0x43, 0x99, 0x18, 0x2e, 0x85, 0x3e,
	0xa4, 0x42, 0x48, 0x43, 0x73, 0xbb, 0x20, 0xed, 0x7d, 0x14, 0x4a, 0x19, 0x46, 0x70, 0x98, 0x7b,
	0x93, 0x74, 0x7a, 0x68, 0x78, 0x0c, 0xda, 0xd0, 0x38, 0x29, 0x12, 0xc8, 0x6f, 0x1b, 0xa8, 0x3e,
	0x82, 0xcc, 0xfd, 0x00, 0x35, 0xe6, 0x90, 0x0d, 0x07, 0xad, 0x1a, 0xae, 0xb5, 0x9b, 0x7e, 0xe1,
	0xb8, 0x6d, 0xb4, 0x39, 0x87, 0x6c, 0x9c, 0x25, 0xd0, 0x7a, 0x84, 0x6b, 0xed, 0x9d, 0xee, 0x4e,
	0x41, 0xf3, 0x46, 0x45, 0xd4, 0xaf, 0x60, 0xf7, 0x43, 0x84, 0x92, 0x74, 0x12, 0xf1, 0x60, 0x40,
	0x0d, 0x6d, 0xd5, 0x71, 0xad, 0xfd, 0xd8, 0x5f, 0x89, 0xb8, 0x2d, 0xb4, 0x29, 0x6f, 0x04, 0xa8,
	0xe1, 0xa0, 0xe5, 0xe4, 0x15, 0x2a, 0xd7, 0xfd, 0x02, 0x3d, 0x01, 0x11, 0xa8, 0x2c, 0x31, 0xc0,
	0x46, 0x90, 0xe9, 0x56, 0x03, 0xd7, 0xdb, 0xdb, 0xdd, 0xf7, 0xcb, 0x4a, 0xa7, 0x2b, 0x98, 0xbf,
	0x9e, 0xe9, 0x7e, 0x8e, 0x9a, 0x81, 0x02, 0x6a, 0x80, 0xf5, 0x4d, 0x6b, 0x03, 0xd7, 0xda, 0xdb,
	0xdd, 0x3d, 0xaf, 0xe8, 0xd8, 0xab, 0x3a, 0xf6, 0xc6, 0x55, 0xc7, 0xfe, 0x32, 0xb9, 0xf7, 0x87,
	0x73, 0xdf, 0xff, 0xdd, 0x41, 0x8d, 0x7d, 0xdb, 0x7c, 0xf7, 0x17, 0xc7, 0xfd, 0xd9, 0xb9, 0x25,
	0x52, 0x85, 0xaf, 0x68, 0x0c, 0xa4, 0x87, 0xc9, 0xe5, 0x8c, 0x1b, 0x78, 0x21, 0x53, 0x0d, 0xa4,
	0x83, 0x2d, 0x70, 0x1a, 0x53, 0x1e, 0x59, 0x04, 0xac, 0xf1, 0x2c, 0x90, 0x71, 0x42, 0x45, 0xe6,
	0x05, 0x32, 0x2e, 0x13, 0xfa, 0x8c, 0x29, 0x8b, 0x1f, 0x7f, 0x76, 0x74, 0x84, 0xcf, 0x41, 0x08,
	0x9d, 0x45, 0xd7, 0x54, 0x70, 0x8a, 0xfb, 0xd7, 0x80, 0x5f, 0x5d, 0x76, 0xf0, 0x25, 0xd5, 0x33,
	0x2e, 0x42, 0x23, 0x45, 0x07, 0x0f, 0x4e, 0x70, 0xf7, 0xe8, 0xd3, 0xa3, 0x23, 0x4b, 0x4e, 0x35,
	0xa8, 0xaa, 0xee, 0x40, 0x0a, 0x1a, 0x31, 0x3c, 0x56, 0x69, 0x9c, 0x58, 0x2c, 0xa1, 0x5a, 0xdf,
	0x48, 0xc5, 0x2c, 0x96, 0xce, 0x15, 0xe5, 0x82, 0xeb, 0x38, 0x9b, 0x2a, 0x0e, 0x82, 0x59, 0x9c,
	0xb2, 0x98, 0x8b, 0xc5, 0xde, 0x58, 0xce, 0xf6, 0x8c, 0x65, 0x3f, 0xbb, 0xb1, 0x2d, 0xcc, 0x6c,
	0x0b, 0x5e, 0x28, 0xaf, 0x6d, 0xb2, 0x9e, 0x51, 0x45, 0x27, 0x11, 0xbc, 0x56, 0xa1, 0x26, 0x3d,
	0xfc, 0x2d, 0x09, 0x84, 0xb0, 0xf1, 0xe9, 0x84, 0xdb, 0x85, 0x41, 0x2c, 0x03, 0x45, 0x8d, 0x26,
	0xdf, 0x75, 0x30, 0x89, 0xd3, 0xc8, 0xf0, 0x97, 0x70, 0x0d, 0xd1, 0xc5, 0x8c, 0x2a, 0xbb, 0x39,
	0xa3, 0x52, 0xb0, 0x5f, 0x91, 0xa9, 0x0a, 0xf2, 0xcd, 0x92, 0x85, 0x67, 0x7f, 0xb1, 0x8d, 0xdc,
	0x5e, 0x11, 0x05, 0x21, 0xd7, 0x46, 0xe5, 0x53, 0x38, 0x96, 0x73, 0x10, 0x57, 0xa4, 0x87, 0xaf,
	0x08, 0x9d, 0x04, 0xec, 0xb8, 0xfb, 0xf4, 0x93, 0x2b, 0x72, 0x67, 0x69, 0x73, 0x36, 0x3d, 0x03,
	0x43, 0x59, 0xce, 0x23, 0x1a, 0x14, 0xa7, 0x11, 0xff, 0x01, 0x18, 0x9e, 0xb3, 0x69, 0x75, 0x26,
	0xe7, 0xe9, 0x64, 0x04, 0x59, 0xde, 0xf9, 0xd2, 0xeb, 0x60, 0x02, 0x22, 0xf8, 0xda, 0x06, 0x14,
	0x2f, 0xe1, 0xf5, 0x40, 0xf1, 0x3b, 0x2a, 0xf2, 0x8a, 0x5d, 0x50, 0x5f, 0xab, 0xb0, 0x62, 0xae,
	0xbb, 0xd5, 0x69, 0xbe, 0x94, 0x21, 0x17, 0x65, 0x43, 0xb7, 0x24, 0xb2, 0x9e, 0x9d, 0x70, 0x5b,
	0xe8, 0xc2, 0x3f, 0xb7, 0x69, 0x79, 0xec, 0x0d, 0x28, 0xcd, 0xa5, 0x20, 0x3d, 0x7c, 0x6c, 0x8f,
	0x41, 0x25, 0x6f, 0x40, 0xf1, 0x29, 0x87, 0x7c, 0x08, 0x04, 0x00, 0x03, 0x36, 0x9c, 0x5a, 0xa2,
	0x25, 0xdd, 0xdd, 0x91, 0x5f, 0x37, 0xd0, 0xe3, 0xd5, 0x59, 0xfe, 0x0f, 0xc5, 0x79, 0xa8, 0x59,
	0xce, 0xb8, 0x54, 0xb9, 0xe6, 0xb6, 0xbb, 0xbb, 0xeb, 0x4a, 0x90, 0xca, 0x5f, 0xa6, 0x58, 0xdd,
	0x05, 0x3c, 0x99, 0x81, 0x32, 0xf0, 0xbd, 0xa9, 0x74, 0xb7, 0x8c, 0xac, 0x4b, 0xc4, 0x79, 0x0b,
	0x89, 0x58, 0x26, 0x83, 0x08, 0x0a, 0x66, 0xe3, 0xff, 0x99, 0x8b, 0xe4, 0xde, 0x3f, 0xce, 0x7d,
	0xff, 0x2f, 0x07, 0xed, 0xec, 0xaf, 0x35, 0xfc, 0xa0, 0xb2, 0x07, 0x95, 0xbd, 0x73, 0x95, 0xfd,
	0xd4, 0x40, 0xcd, 0x85, 0x4e, 0xdc, 0x3d, 0xb4, 0x15, 0xc9, 0x20, 0x3f, 0xa4, 0x52, 0x65, 0x0b,
	0xdf, 0x0a, 0xa7, 0xb2, 0x87, 0x83, 0x5c, 0x69, 0x4d, 0x7f, 0x25, 0xb2, 0x7a, 0x61, 0xd5, 0xd7,
	0x2e, 0xac, 0xde, 0xdf, 0xce, 0x7d, 0xff, 0x4f, 0x07, 0x6d, 0xef, 0x2f, 0x2b, 0x3d, 0xcc, 0xf6,
	0xc3, 0x6c, 0xbf, 0xf3, 0xd9, 0xee, 0xa2, 0xad, 0x11, 0x64, 0x27, 0x33, 0xca, 0x85, 0xfb, 0x31,
	0xda, 0x9a, 0x97, 0x76, 0xab, 0x96, 0xbf, 0x97, 0xd0, 0xf2, 0x65, 0xe6, 0x2f, 0xb0, 0xfd, 0x2f,
	0xd1, 0x66, 0xf9, 0x54, 0x73, 0x37, 0x51, 0x7d, 0x34, 0xf8, 0x6a, 0xf7, 0x3d, 0x77, 0x0b, 0x39,
	0x2f, 0xce, 0xfa, 0x27, 0xbb, 0x35, 0xf7, 0x09, 0x6a, 0x5e, 0x7c, 0x73, 0x76, 0x76, 0x3a, 0xf6,
	0x87, 0x27, 0xbb, 0x8f, 0xdc, 0x1d, 0x84, 0xfa, 0x4b, 0xbf, 0xfe, 0xfc, 0x00, 0x91, 0x40, 0xc6,
	0x9e, 0x36, 0x4a, 0x8a, 0x50, 0xd3, 0xc8, 0x94, 0x26, 0x93, 0x81, 0xa7, 0xd9, 0xbc, 0x28, 0xf8,
	0xdc, 0xb1, 0x4f, 0xb1, 0x1f, 0x6b, 0xb5, 0xc9, 0x46, 0xee, 0x3f, 0xfd, 0x37, 0x00, 0x00, 0xff,
	0xff, 0xc6, 0x8f, 0xfb, 0x44, 0xb5, 0x0a, 0x00, 0x00,
}
