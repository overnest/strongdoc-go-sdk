// Code generated by protoc-gen-go. DO NOT EDIT.
// source: strongdoc.proto

package proto

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

func init() { proto.RegisterFile("strongdoc.proto", fileDescriptor_d003557e9d9c9339) }

var fileDescriptor_d003557e9d9c9339 = []byte{
	// 498 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x52, 0xcb, 0x6e, 0x13, 0x31,
	0x14, 0x65, 0x06, 0x15, 0x21, 0x8b, 0xa6, 0xd1, 0x14, 0x51, 0x11, 0x21, 0x30, 0x23, 0xa1, 0xa2,
	0xd2, 0xcc, 0xb4, 0x61, 0x81, 0x94, 0x15, 0x09, 0x2f, 0x45, 0x80, 0x88, 0x12, 0xd8, 0xb0, 0x41,
	0x8e, 0xe7, 0xd6, 0x63, 0x3a, 0xf1, 0x75, 0x6d, 0x4f, 0x43, 0x58, 0x22, 0x24, 0x10, 0x4b, 0xf8,
	0x0c, 0xd6, 0x7c, 0x09, 0xbf, 0xc0, 0x07, 0xf0, 0x09, 0x68, 0x3c, 0x43, 0x79, 0x45, 0xac, 0xac,
	0x7b, 0xee, 0xb9, 0xe7, 0xdc, 0x87, 0xc9, 0x86, 0x75, 0x06, 0x95, 0xc8, 0x90, 0x27, 0xda, 0xa0,
	0xc3, 0x68, 0xcd, 0x3f, 0x9d, 0x4b, 0x02, 0x51, 0x14, 0x90, 0x32, 0x2d, 0x53, 0xa6, 0x14, 0x3a,
	0xe6, 0x24, 0x2a, 0x5b, 0x93, 0x3a, 0xbb, 0xfe, 0xe1, 0x5d, 0x01, 0xaa, 0x6b, 0x17, 0x4c, 0x08,
	0x30, 0x29, 0x6a, 0xcf, 0x58, 0xc1, 0x6e, 0x31, 0xce, 0xb1, 0x54, 0xae, 0x89, 0x7b, 0x9f, 0x43,
	0xd2, 0x9e, 0x7a, 0xdb, 0xbb, 0xc8, 0xa7, 0x60, 0x8e, 0x25, 0x87, 0xe8, 0x6d, 0x40, 0xce, 0x4f,
	0x40, 0x48, 0xeb, 0xc0, 0x3c, 0x31, 0x82, 0x29, 0xf9, 0xda, 0x8b, 0x44, 0x97, 0xeb, 0xaa, 0x64,
	0x55, 0x72, 0x02, 0x47, 0x9d, 0x2b, 0xff, 0xcd, 0x5b, 0x1d, 0xdf, 0x78, 0xf3, 0xf5, 0xdb, 0xa7,
	0xf0, 0x5a, 0x4c, 0xd3, 0xe3, 0xfd, 0xb4, 0x69, 0x25, 0x35, 0x2b, 0xd8, 0xfd, 0x60, 0x27, 0xba,
	0x47, 0xce, 0xfd, 0x14, 0x7a, 0x66, 0xc1, 0x44, 0x17, 0xfe, 0x52, 0xaf, 0xc0, 0xca, 0x75, 0x6b,
	0x25, 0x6e, 0x75, 0x7c, 0xea, 0x7a, 0xb0, 0x17, 0x44, 0x23, 0xb2, 0xf6, 0x08, 0x85, 0x54, 0xd1,
	0x46, 0xc3, 0xf3, 0x51, 0x55, 0xd8, 0xfe, 0x13, 0xb0, 0x3a, 0xbe, 0xe8, 0xfb, 0xdb, 0x8c, 0x5b,
	0xbe, 0xbf, 0xd2, 0xe5, 0x69, 0x51, 0xe5, 0xfa, 0xc1, 0x4e, 0x25, 0x35, 0xfc, 0x1e, 0x7e, 0x1c,
	0x7c, 0x09, 0xa3, 0x5b, 0x64, 0xfd, 0x64, 0x67, 0x74, 0x30, 0x1e, 0xc5, 0x57, 0x09, 0xa9, 0x81,
	0x29, 0x2b, 0x5c, 0x67, 0x53, 0xaa, 0x03, 0xbc, 0x5d, 0x1f, 0xd3, 0xb2, 0xc2, 0x25, 0x1c, 0xe7,
	0xbd, 0xd3, 0xfb, 0xc9, 0x5e, 0xaf, 0xcd, 0xb4, 0x2e, 0x24, 0xf7, 0x13, 0xa6, 0x2f, 0x2d, 0xaa,
	0xfe, 0x3f, 0xc8, 0xf3, 0x0f, 0x01, 0x79, 0x17, 0x10, 0x32, 0xd0, 0xf2, 0x21, 0x2c, 0x07, 0xa5,
	0xcb, 0xa3, 0xe5, 0xd9, 0x30, 0xca, 0x9e, 0xe6, 0x40, 0x17, 0x68, 0x32, 0xba, 0x3d, 0x04, 0x66,
	0xc0, 0x6c, 0x53, 0xa6, 0x32, 0xca, 0xa8, 0xd5, 0x8c, 0x03, 0x95, 0x96, 0x1a, 0x38, 0x2a, 0xa5,
	0x81, 0x8c, 0xce, 0xe0, 0x00, 0x0d, 0x50, 0x97, 0x03, 0x75, 0x78, 0x08, 0x2a, 0x21, 0xf7, 0xd1,
	0x50, 0x78, 0xc5, 0xe6, 0xba, 0x80, 0x5d, 0x52, 0x97, 0xd3, 0x6a, 0x38, 0x50, 0xae, 0xb1, 0x7e,
	0xe1, 0x99, 0x9d, 0xf5, 0xca, 0x0f, 0x4d, 0x73, 0x03, 0x1a, 0x9a, 0x11, 0xd9, 0x7a, 0x5c, 0x89,
	0xb1, 0x19, 0x96, 0x8e, 0x8a, 0xc9, 0xf8, 0x4e, 0xf7, 0x01, 0x73, 0xb0, 0x60, 0xcb, 0x28, 0xc9,
	0x9d, 0xd3, 0xb6, 0x9f, 0xa6, 0x42, 0xba, 0xbc, 0x9c, 0x55, 0x43, 0xa6, 0xc2, 0x68, 0xde, 0x05,
	0x8e, 0x76, 0x69, 0x1d, 0x34, 0xa1, 0xa8, 0xf9, 0x24, 0xe6, 0x38, 0x4f, 0x7e, 0x5b, 0xca, 0xaf,
	0xcf, 0x6e, 0xb3, 0xc3, 0xfa, 0x10, 0xc3, 0xd6, 0xc9, 0x5a, 0xc7, 0x55, 0xfc, 0x3e, 0x08, 0x66,
	0x67, 0x7c, 0xe6, 0xe6, 0x8f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe7, 0xa7, 0xb7, 0x19, 0x1d, 0x03,
	0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// StrongDocServiceClient is the client API for StrongDocService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type StrongDocServiceClient interface {
	// Registers a new organization
	//
	// The user who created the organization is automatically an administrator
	//
	// Does not require Login
	RegisterOrganization(ctx context.Context, in *RegisterOrganizationReq, opts ...grpc.CallOption) (*RegisterOrganizationResp, error)
	// Register new user
	//
	// Creates new user if it doesn't already exist. If the user already exist, and
	// error is thrown
	//
	// Does not require Login
	RegisterUser(ctx context.Context, opts ...grpc.CallOption) (StrongDocService_RegisterUserClient, error)
	// Obtain an authentication token to be used with other APIs
	//
	// An authentication token will be returned after user has been validated
	// The returned token will be used as a Bearer Token and need to be set in
	// the request header
	Login(ctx context.Context, opts ...grpc.CallOption) (StrongDocService_LoginClient, error)
}

type strongDocServiceClient struct {
	cc *grpc.ClientConn
}

func NewStrongDocServiceClient(cc *grpc.ClientConn) StrongDocServiceClient {
	return &strongDocServiceClient{cc}
}

func (c *strongDocServiceClient) RegisterOrganization(ctx context.Context, in *RegisterOrganizationReq, opts ...grpc.CallOption) (*RegisterOrganizationResp, error) {
	out := new(RegisterOrganizationResp)
	err := c.cc.Invoke(ctx, "/proto.StrongDocService/RegisterOrganization", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *strongDocServiceClient) RegisterUser(ctx context.Context, opts ...grpc.CallOption) (StrongDocService_RegisterUserClient, error) {
	stream, err := c.cc.NewStream(ctx, &_StrongDocService_serviceDesc.Streams[0], "/proto.StrongDocService/RegisterUser", opts...)
	if err != nil {
		return nil, err
	}
	x := &strongDocServiceRegisterUserClient{stream}
	return x, nil
}

type StrongDocService_RegisterUserClient interface {
	Send(*RegisterUserReq) error
	Recv() (*RegisterUserResp, error)
	grpc.ClientStream
}

type strongDocServiceRegisterUserClient struct {
	grpc.ClientStream
}

func (x *strongDocServiceRegisterUserClient) Send(m *RegisterUserReq) error {
	return x.ClientStream.SendMsg(m)
}

func (x *strongDocServiceRegisterUserClient) Recv() (*RegisterUserResp, error) {
	m := new(RegisterUserResp)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *strongDocServiceClient) Login(ctx context.Context, opts ...grpc.CallOption) (StrongDocService_LoginClient, error) {
	stream, err := c.cc.NewStream(ctx, &_StrongDocService_serviceDesc.Streams[1], "/proto.StrongDocService/Login", opts...)
	if err != nil {
		return nil, err
	}
	x := &strongDocServiceLoginClient{stream}
	return x, nil
}

type StrongDocService_LoginClient interface {
	Send(*LoginReq) error
	Recv() (*LoginResp, error)
	grpc.ClientStream
}

type strongDocServiceLoginClient struct {
	grpc.ClientStream
}

func (x *strongDocServiceLoginClient) Send(m *LoginReq) error {
	return x.ClientStream.SendMsg(m)
}

func (x *strongDocServiceLoginClient) Recv() (*LoginResp, error) {
	m := new(LoginResp)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// StrongDocServiceServer is the server API for StrongDocService service.
type StrongDocServiceServer interface {
	// Registers a new organization
	//
	// The user who created the organization is automatically an administrator
	//
	// Does not require Login
	RegisterOrganization(context.Context, *RegisterOrganizationReq) (*RegisterOrganizationResp, error)
	// Register new user
	//
	// Creates new user if it doesn't already exist. If the user already exist, and
	// error is thrown
	//
	// Does not require Login
	RegisterUser(StrongDocService_RegisterUserServer) error
	// Obtain an authentication token to be used with other APIs
	//
	// An authentication token will be returned after user has been validated
	// The returned token will be used as a Bearer Token and need to be set in
	// the request header
	Login(StrongDocService_LoginServer) error
}

// UnimplementedStrongDocServiceServer can be embedded to have forward compatible implementations.
type UnimplementedStrongDocServiceServer struct {
}

func (*UnimplementedStrongDocServiceServer) RegisterOrganization(ctx context.Context, req *RegisterOrganizationReq) (*RegisterOrganizationResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterOrganization not implemented")
}
func (*UnimplementedStrongDocServiceServer) RegisterUser(srv StrongDocService_RegisterUserServer) error {
	return status.Errorf(codes.Unimplemented, "method RegisterUser not implemented")
}
func (*UnimplementedStrongDocServiceServer) Login(srv StrongDocService_LoginServer) error {
	return status.Errorf(codes.Unimplemented, "method Login not implemented")
}

func RegisterStrongDocServiceServer(s *grpc.Server, srv StrongDocServiceServer) {
	s.RegisterService(&_StrongDocService_serviceDesc, srv)
}

func _StrongDocService_RegisterOrganization_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterOrganizationReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StrongDocServiceServer).RegisterOrganization(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.StrongDocService/RegisterOrganization",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StrongDocServiceServer).RegisterOrganization(ctx, req.(*RegisterOrganizationReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _StrongDocService_RegisterUser_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(StrongDocServiceServer).RegisterUser(&strongDocServiceRegisterUserServer{stream})
}

type StrongDocService_RegisterUserServer interface {
	Send(*RegisterUserResp) error
	Recv() (*RegisterUserReq, error)
	grpc.ServerStream
}

type strongDocServiceRegisterUserServer struct {
	grpc.ServerStream
}

func (x *strongDocServiceRegisterUserServer) Send(m *RegisterUserResp) error {
	return x.ServerStream.SendMsg(m)
}

func (x *strongDocServiceRegisterUserServer) Recv() (*RegisterUserReq, error) {
	m := new(RegisterUserReq)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _StrongDocService_Login_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(StrongDocServiceServer).Login(&strongDocServiceLoginServer{stream})
}

type StrongDocService_LoginServer interface {
	Send(*LoginResp) error
	Recv() (*LoginReq, error)
	grpc.ServerStream
}

type strongDocServiceLoginServer struct {
	grpc.ServerStream
}

func (x *strongDocServiceLoginServer) Send(m *LoginResp) error {
	return x.ServerStream.SendMsg(m)
}

func (x *strongDocServiceLoginServer) Recv() (*LoginReq, error) {
	m := new(LoginReq)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _StrongDocService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "proto.StrongDocService",
	HandlerType: (*StrongDocServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterOrganization",
			Handler:    _StrongDocService_RegisterOrganization_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RegisterUser",
			Handler:       _StrongDocService_RegisterUser_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "Login",
			Handler:       _StrongDocService_Login_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "strongdoc.proto",
}
