package v1

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/pkg/session"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
)

type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByLogin(ctx context.Context, loginHash string) (model.User, error)
	FindByID(ctx context.Context, uuid string) (model.User, error)
	Delete(ctx context.Context, uuid string) error
}

type AuthService struct {
	authpb.UnimplementedAuthServiceServer
	repo           UserRepository
	sessionManager *session.Manager
}

func NewAuthService(repo UserRepository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

func (s *AuthService) Register(ctx context.Context, r *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	userData := r.GetAuthData()
	if userData == nil || (userData.Login == nil || userData.Password == nil) {
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed: the registration data is nil")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	if _, err := s.repo.FindByLogin(ctx1, userData.GetLogin()); err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "the username is already taken")
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check login existance: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, model.RepoOperationTO)
	cancel2()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userData.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}
	if err := s.repo.Create(ctx2, &model.User{
		Login:        userData.GetLogin(),
		PasswordHash: passwordHash,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	access, refresh, err := s.sessionManager.GenerateTokens(ctx)
	resp := &authpb.RegisterResponse{
		Credentials: &authpb.Credentials{
			AccessToken:  &authpb.AccessToken{AccessToken: &access},
			RefreshToken: &authpb.RefreshToken{RefreshToken: &refresh},
		},
	}

	return resp, nil
}

func (s *AuthService) Login(ctx context.Context, r *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {
	userData := r.GetAuthData()
	if userData == nil || (userData.Login == nil || userData.Password == nil) {
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed: the registration data is nil")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	user, err := s.repo.FindByLogin(ctx1, userData.GetLogin())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "login or password is wrong")
	}
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(userData.GetPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "login or password is wrong")
	}

	access, refresh, err := s.sessionManager.GenerateTokens(ctx)
	resp := &authpb.LoginResponse{
		Credentials: &authpb.Credentials{
			AccessToken:  &authpb.AccessToken{AccessToken: &access},
			RefreshToken: &authpb.RefreshToken{RefreshToken: &refresh},
		},
	}

	return resp, nil
}

func (s *AuthService) Logout(ctx context.Context, r *authpb.LogoutRequest,
) (*emptypb.Empty, error) {
	refreshToken := r.GetRefreshToken()
	if refreshToken == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed: the refresh uuid is nil")
	}
	if err := s.sessionManager.Logout(ctx, refreshToken.GetRefreshToken()); err != nil {
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}
