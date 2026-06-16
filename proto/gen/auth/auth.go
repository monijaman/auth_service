// Package authpb contains the gRPC service types for the auth service.
//
// NOTE: This file is hand-written and uses a JSON codec instead of protobuf
// binary encoding. Run `make proto` to replace it with proper protoc-generated
// code once the protoc toolchain is installed.
package authpb

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

func init() {
	// Override the default proto codec with JSON so no protoc toolchain is required.
	// Replace with proper generated code after running `make proto`.
	encoding.RegisterCodec(jsonCodec{})
}

// ─── Codec ───────────────────────────────────────────────────────────────────

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)        { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v any) error   { return json.Unmarshal(data, v) }
func (jsonCodec) Name() string                         { return "proto" }

// ─── Messages ────────────────────────────────────────────────────────────────

type TokenRequest struct {
	Token string `json:"token"`
}

type TokenResponse struct {
	Valid        bool     `json:"valid"`
	UserID       string   `json:"user_id"`
	Email        string   `json:"email"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
}

type UserRequest struct {
	UserID string `json:"user_id"`
}

type UserResponse struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	Roles         []string `json:"roles"`
}

type PermissionRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

type PermissionResponse struct {
	HasPermission bool `json:"has_permission"`
}

// ─── Server interface ────────────────────────────────────────────────────────

type AuthServiceServer interface {
	ValidateToken(context.Context, *TokenRequest) (*TokenResponse, error)
	GetUser(context.Context, *UserRequest) (*UserResponse, error)
	HasPermission(context.Context, *PermissionRequest) (*PermissionResponse, error)
}

// ─── Client interface ────────────────────────────────────────────────────────

type AuthServiceClient interface {
	ValidateToken(ctx context.Context, in *TokenRequest, opts ...grpc.CallOption) (*TokenResponse, error)
	GetUser(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*UserResponse, error)
	HasPermission(ctx context.Context, in *PermissionRequest, opts ...grpc.CallOption) (*PermissionResponse, error)
}

type authServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthServiceClient(cc grpc.ClientConnInterface) AuthServiceClient {
	return &authServiceClient{cc}
}

func (c *authServiceClient) ValidateToken(ctx context.Context, in *TokenRequest, opts ...grpc.CallOption) (*TokenResponse, error) {
	out := &TokenResponse{}
	err := c.cc.Invoke(ctx, "/auth.AuthService/ValidateToken", in, out, opts...)
	return out, err
}

func (c *authServiceClient) GetUser(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*UserResponse, error) {
	out := &UserResponse{}
	err := c.cc.Invoke(ctx, "/auth.AuthService/GetUser", in, out, opts...)
	return out, err
}

func (c *authServiceClient) HasPermission(ctx context.Context, in *PermissionRequest, opts ...grpc.CallOption) (*PermissionResponse, error) {
	out := &PermissionResponse{}
	err := c.cc.Invoke(ctx, "/auth.AuthService/HasPermission", in, out, opts...)
	return out, err
}

// ─── Registration ────────────────────────────────────────────────────────────

func RegisterAuthServiceServer(s *grpc.Server, srv AuthServiceServer) {
	s.RegisterService(&AuthService_ServiceDesc, srv)
}

var AuthService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.AuthService",
	HandlerType: (*AuthServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ValidateToken", Handler: _AuthService_ValidateToken_Handler},
		{MethodName: "GetUser", Handler: _AuthService_GetUser_Handler},
		{MethodName: "HasPermission", Handler: _AuthService_HasPermission_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/auth.proto",
}

func _AuthService_ValidateToken_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := &TokenRequest{}
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServiceServer).ValidateToken(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.AuthService/ValidateToken"},
		func(ctx context.Context, req any) (any, error) {
			return srv.(AuthServiceServer).ValidateToken(ctx, req.(*TokenRequest))
		})
}

func _AuthService_GetUser_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := &UserRequest{}
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServiceServer).GetUser(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.AuthService/GetUser"},
		func(ctx context.Context, req any) (any, error) {
			return srv.(AuthServiceServer).GetUser(ctx, req.(*UserRequest))
		})
}

func _AuthService_HasPermission_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := &PermissionRequest{}
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServiceServer).HasPermission(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.AuthService/HasPermission"},
		func(ctx context.Context, req any) (any, error) {
			return srv.(AuthServiceServer).HasPermission(ctx, req.(*PermissionRequest))
		})
}
