// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.1
// source: github.com/containerd/containerd/api/services/content/v1/content.proto

package content

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ContentClient is the client API for Content service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ContentClient interface {
	// Info returns information about a committed object.
	//
	// This call can be used for getting the size of content and checking for
	// existence.
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error)
	// Update updates content metadata.
	//
	// This call can be used to manage the mutable content labels. The
	// immutable metadata such as digest, size, and committed at cannot
	// be updated.
	Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateResponse, error)
	// List streams the entire set of content as Info objects and closes the
	// stream.
	//
	// Typically, this will yield a large response, chunked into messages.
	// Clients should make provisions to ensure they can handle the entire data
	// set.
	List(ctx context.Context, in *ListContentRequest, opts ...grpc.CallOption) (Content_ListClient, error)
	// Delete will delete the referenced object.
	Delete(ctx context.Context, in *DeleteContentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// Read allows one to read an object based on the offset into the content.
	//
	// The requested data may be returned in one or more messages.
	Read(ctx context.Context, in *ReadContentRequest, opts ...grpc.CallOption) (Content_ReadClient, error)
	// Status returns the status for a single reference.
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	// ListStatuses returns the status of ongoing object ingestions, started via
	// Write.
	//
	// Only those matching the regular expression will be provided in the
	// response. If the provided regular expression is empty, all ingestions
	// will be provided.
	ListStatuses(ctx context.Context, in *ListStatusesRequest, opts ...grpc.CallOption) (*ListStatusesResponse, error)
	// Write begins or resumes writes to a resource identified by a unique ref.
	// Only one active stream may exist at a time for each ref.
	//
	// Once a write stream has started, it may only write to a single ref, thus
	// once a stream is started, the ref may be omitted on subsequent writes.
	//
	// For any write transaction represented by a ref, only a single write may
	// be made to a given offset. If overlapping writes occur, it is an error.
	// Writes should be sequential and implementations may throw an error if
	// this is required.
	//
	// If expected_digest is set and already part of the content store, the
	// write will fail.
	//
	// When completed, the commit flag should be set to true. If expected size
	// or digest is set, the content will be validated against those values.
	Write(ctx context.Context, opts ...grpc.CallOption) (Content_WriteClient, error)
	// Abort cancels the ongoing write named in the request. Any resources
	// associated with the write will be collected.
	Abort(ctx context.Context, in *AbortRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type contentClient struct {
	cc grpc.ClientConnInterface
}

func NewContentClient(cc grpc.ClientConnInterface) ContentClient {
	return &contentClient{cc}
}

func (c *contentClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/Info", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *contentClient) Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateResponse, error) {
	out := new(UpdateResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/Update", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *contentClient) List(ctx context.Context, in *ListContentRequest, opts ...grpc.CallOption) (Content_ListClient, error) {
	stream, err := c.cc.NewStream(ctx, &Content_ServiceDesc.Streams[0], "/containerd.services.content.v1.Content/List", opts...)
	if err != nil {
		return nil, err
	}
	x := &contentListClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Content_ListClient interface {
	Recv() (*ListContentResponse, error)
	grpc.ClientStream
}

type contentListClient struct {
	grpc.ClientStream
}

func (x *contentListClient) Recv() (*ListContentResponse, error) {
	m := new(ListContentResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *contentClient) Delete(ctx context.Context, in *DeleteContentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/Delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *contentClient) Read(ctx context.Context, in *ReadContentRequest, opts ...grpc.CallOption) (Content_ReadClient, error) {
	stream, err := c.cc.NewStream(ctx, &Content_ServiceDesc.Streams[1], "/containerd.services.content.v1.Content/Read", opts...)
	if err != nil {
		return nil, err
	}
	x := &contentReadClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Content_ReadClient interface {
	Recv() (*ReadContentResponse, error)
	grpc.ClientStream
}

type contentReadClient struct {
	grpc.ClientStream
}

func (x *contentReadClient) Recv() (*ReadContentResponse, error) {
	m := new(ReadContentResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *contentClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *contentClient) ListStatuses(ctx context.Context, in *ListStatusesRequest, opts ...grpc.CallOption) (*ListStatusesResponse, error) {
	out := new(ListStatusesResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/ListStatuses", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *contentClient) Write(ctx context.Context, opts ...grpc.CallOption) (Content_WriteClient, error) {
	stream, err := c.cc.NewStream(ctx, &Content_ServiceDesc.Streams[2], "/containerd.services.content.v1.Content/Write", opts...)
	if err != nil {
		return nil, err
	}
	x := &contentWriteClient{stream}
	return x, nil
}

type Content_WriteClient interface {
	Send(*WriteContentRequest) error
	Recv() (*WriteContentResponse, error)
	grpc.ClientStream
}

type contentWriteClient struct {
	grpc.ClientStream
}

func (x *contentWriteClient) Send(m *WriteContentRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *contentWriteClient) Recv() (*WriteContentResponse, error) {
	m := new(WriteContentResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *contentClient) Abort(ctx context.Context, in *AbortRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/containerd.services.content.v1.Content/Abort", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ContentServer is the server API for Content service.
// All implementations must embed UnimplementedContentServer
// for forward compatibility
type ContentServer interface {
	// Info returns information about a committed object.
	//
	// This call can be used for getting the size of content and checking for
	// existence.
	Info(context.Context, *InfoRequest) (*InfoResponse, error)
	// Update updates content metadata.
	//
	// This call can be used to manage the mutable content labels. The
	// immutable metadata such as digest, size, and committed at cannot
	// be updated.
	Update(context.Context, *UpdateRequest) (*UpdateResponse, error)
	// List streams the entire set of content as Info objects and closes the
	// stream.
	//
	// Typically, this will yield a large response, chunked into messages.
	// Clients should make provisions to ensure they can handle the entire data
	// set.
	List(*ListContentRequest, Content_ListServer) error
	// Delete will delete the referenced object.
	Delete(context.Context, *DeleteContentRequest) (*emptypb.Empty, error)
	// Read allows one to read an object based on the offset into the content.
	//
	// The requested data may be returned in one or more messages.
	Read(*ReadContentRequest, Content_ReadServer) error
	// Status returns the status for a single reference.
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
	// ListStatuses returns the status of ongoing object ingestions, started via
	// Write.
	//
	// Only those matching the regular expression will be provided in the
	// response. If the provided regular expression is empty, all ingestions
	// will be provided.
	ListStatuses(context.Context, *ListStatusesRequest) (*ListStatusesResponse, error)
	// Write begins or resumes writes to a resource identified by a unique ref.
	// Only one active stream may exist at a time for each ref.
	//
	// Once a write stream has started, it may only write to a single ref, thus
	// once a stream is started, the ref may be omitted on subsequent writes.
	//
	// For any write transaction represented by a ref, only a single write may
	// be made to a given offset. If overlapping writes occur, it is an error.
	// Writes should be sequential and implementations may throw an error if
	// this is required.
	//
	// If expected_digest is set and already part of the content store, the
	// write will fail.
	//
	// When completed, the commit flag should be set to true. If expected size
	// or digest is set, the content will be validated against those values.
	Write(Content_WriteServer) error
	// Abort cancels the ongoing write named in the request. Any resources
	// associated with the write will be collected.
	Abort(context.Context, *AbortRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedContentServer()
}

// UnimplementedContentServer must be embedded to have forward compatible implementations.
type UnimplementedContentServer struct {
}

func (UnimplementedContentServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedContentServer) Update(context.Context, *UpdateRequest) (*UpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedContentServer) List(*ListContentRequest, Content_ListServer) error {
	return status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedContentServer) Delete(context.Context, *DeleteContentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedContentServer) Read(*ReadContentRequest, Content_ReadServer) error {
	return status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedContentServer) Status(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedContentServer) ListStatuses(context.Context, *ListStatusesRequest) (*ListStatusesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStatuses not implemented")
}
func (UnimplementedContentServer) Write(Content_WriteServer) error {
	return status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedContentServer) Abort(context.Context, *AbortRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Abort not implemented")
}
func (UnimplementedContentServer) mustEmbedUnimplementedContentServer() {}

// UnsafeContentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ContentServer will
// result in compilation errors.
type UnsafeContentServer interface {
	mustEmbedUnimplementedContentServer()
}

func RegisterContentServer(s grpc.ServiceRegistrar, srv ContentServer) {
	s.RegisterService(&Content_ServiceDesc, srv)
}

func _Content_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/Info",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Content_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/Update",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).Update(ctx, req.(*UpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Content_List_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ListContentRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ContentServer).List(m, &contentListServer{stream})
}

type Content_ListServer interface {
	Send(*ListContentResponse) error
	grpc.ServerStream
}

type contentListServer struct {
	grpc.ServerStream
}

func (x *contentListServer) Send(m *ListContentResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Content_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteContentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).Delete(ctx, req.(*DeleteContentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Content_Read_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ReadContentRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ContentServer).Read(m, &contentReadServer{stream})
}

type Content_ReadServer interface {
	Send(*ReadContentResponse) error
	grpc.ServerStream
}

type contentReadServer struct {
	grpc.ServerStream
}

func (x *contentReadServer) Send(m *ReadContentResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Content_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Content_ListStatuses_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListStatusesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).ListStatuses(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/ListStatuses",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).ListStatuses(ctx, req.(*ListStatusesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Content_Write_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ContentServer).Write(&contentWriteServer{stream})
}

type Content_WriteServer interface {
	Send(*WriteContentResponse) error
	Recv() (*WriteContentRequest, error)
	grpc.ServerStream
}

type contentWriteServer struct {
	grpc.ServerStream
}

func (x *contentWriteServer) Send(m *WriteContentResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *contentWriteServer) Recv() (*WriteContentRequest, error) {
	m := new(WriteContentRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Content_Abort_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AbortRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContentServer).Abort(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.content.v1.Content/Abort",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContentServer).Abort(ctx, req.(*AbortRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Content_ServiceDesc is the grpc.ServiceDesc for Content service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Content_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "containerd.services.content.v1.Content",
	HandlerType: (*ContentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Info",
			Handler:    _Content_Info_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Content_Update_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Content_Delete_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _Content_Status_Handler,
		},
		{
			MethodName: "ListStatuses",
			Handler:    _Content_ListStatuses_Handler,
		},
		{
			MethodName: "Abort",
			Handler:    _Content_Abort_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "List",
			Handler:       _Content_List_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Read",
			Handler:       _Content_Read_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Write",
			Handler:       _Content_Write_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "github.com/containerd/containerd/api/services/content/v1/content.proto",
}
