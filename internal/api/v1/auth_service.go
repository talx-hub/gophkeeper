package v1

import (
	"context"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/internal/model"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
)

type SessionService interface {
	CreateSession(ctx context.Context, userID model.UserID,
	) (accessToken string, refreshToken string, err error)

	RefreshSession(ctx context.Context, userID model.UserID, refreshToken string,
	) (newAccessToken string, newRefreshToken string, err error)

	ValidateAccessToken(ctx context.Context, token string) (userID model.UserID, err error)
	RevokeSession(ctx context.Context, refreshToken string) error
}

type UserRepository interface {
	Create(ctx context.Context, u *model.User) (model.UserID, error)
	FindByLogin(ctx context.Context, login string) (model.User, error)
	FindByID(ctx context.Context, uuid model.UserID) (model.User, error)
	Delete(ctx context.Context, uuid model.UserID) error
}

type AuthService struct {
	authpb.UnimplementedAuthServiceServer
	log            *slog.Logger
	repo           UserRepository
	sessionService SessionService
}

func NewAuthService(
	log *slog.Logger,
	repo UserRepository,
	session SessionService,
) *AuthService {
	return &AuthService{
		log:            log,
		repo:           repo,
		sessionService: session,
	}
}

func (s *AuthService) Login(ctx context.Context, r *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {
	userData := r.GetAuthData()
	if userData == nil {
		s.log.ErrorContext(ctx, "request validation failed: the registration data is nil")
		return nil, status.Errorf(
			codes.InvalidArgument,
			"request validation failed: check if both Login and Password are filled")
	} else {
		if userData.Login == nil || userData.Password == nil {
			s.log.ErrorContext(ctx,
				"request validation failed: invalid login or password")
			return nil, status.Errorf(
				codes.InvalidArgument,
				"request validation failed: check if both Login and Password are filled")
		}
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	user, err := s.repo.FindByLogin(ctx1, userData.GetLogin())
	if err != nil {
		s.log.ErrorContext(ctx, "failed to find user by login",
			"login", userData.GetLogin(),
			"err", err)
		return nil, status.Errorf(codes.Unauthenticated, "failed to find user")
	}
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(userData.GetPassword())); err != nil {
		s.log.ErrorContext(ctx, "password format is wrong", "err", err)
		return nil, status.Error(codes.Unauthenticated, "password format is wrong")
	}

	access, refresh, err := s.sessionService.CreateSession(ctx, user.UUID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create session", "err", err)
		return nil, status.Error(codes.Internal, "failed to create session")
	}

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
		s.log.ErrorContext(ctx, "request validation failed: the refresh uuid is nil")
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed")
	}
	if err := s.sessionService.RevokeSession(ctx, refreshToken.GetRefreshToken()); err != nil {
		s.log.ErrorContext(ctx, "logout failed", "err", err)
		return nil, status.Errorf(codes.Internal, "logout failed")
	}

	return &emptypb.Empty{}, nil
}

func (s *AuthService) Register(ctx context.Context, r *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	userData := r.GetAuthData()
	if userData == nil {
		s.log.ErrorContext(ctx, "request validation failed: the registration data is nil")
		return nil, status.Errorf(
			codes.InvalidArgument,
			"request validation failed: check if both Login and Password are filled")
	} else {
		if userData.Login == nil || userData.Password == nil {
			s.log.ErrorContext(ctx,
				"request validation failed: invalid login or password")
			return nil, status.Errorf(
				codes.InvalidArgument,
				"request validation failed: check if both Login and Password are filled")
		}
	}

	const msgRegistrationFailed = "registration failed, try again"
	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	if _, err := s.repo.FindByLogin(ctx1, userData.GetLogin()); err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "the username is already taken")
	} else if !errors.Is(err, model.ErrNotFound) {
		s.log.ErrorContext(ctx, "failed to check login existence", "err", err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel2()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userData.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to hash password", "err", err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}
	userID, err := s.repo.Create(ctx2, &model.User{
		Login:        userData.GetLogin(),
		PasswordHash: passwordHash,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create user", "err", err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}

	access, refresh, err := s.sessionService.CreateSession(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create session", "err", err)
		return nil, status.Errorf(codes.Internal, msgRegistrationFailed)
	}
	resp := &authpb.RegisterResponse{
		Credentials: &authpb.Credentials{
			AccessToken:  &authpb.AccessToken{AccessToken: &access},
			RefreshToken: &authpb.RefreshToken{RefreshToken: &refresh},
		},
	}

	return resp, nil
}
