package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/proto/v1/auth"
)

type AuthService struct {
	auth.UnimplementedAuthServiceServer
}

func (s *AuthService) Register(context.Context, *auth.RegisterRequest,
) (*auth.RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (s *AuthService) Login(context.Context, *auth.LoginRequest,
) (*auth.LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (s *AuthService) Logout(context.Context, *auth.LogoutRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
