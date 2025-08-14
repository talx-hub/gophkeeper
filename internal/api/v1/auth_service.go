package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
)

type AuthService struct {
	authpb.UnimplementedAuthServiceServer
}

func (s *AuthService) Register(context.Context, *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (s *AuthService) Login(context.Context, *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (s *AuthService) Logout(context.Context, *authpb.LogoutRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
