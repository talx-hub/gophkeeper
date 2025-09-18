//nolint:wrapcheck // reason: this package intentionally returns raw errors and errors are logged
package v1

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/pkg/hash"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
)

const MsgRequestValidationFailed = "request validation failed: check if both Login and Password are filled"

type SessionService interface {
	CreateSession(ctx context.Context, userID model.UserID,
	) (accessToken string, refreshToken []byte, err error)

	RefreshSession(ctx context.Context, userID model.UserID, refreshToken []byte,
	) (newAccessToken string, newRefreshToken []byte, err error)

	ValidateAccessToken(ctx context.Context, token string) (userID model.UserID, err error)
	RevokeSession(ctx context.Context, refreshToken []byte) error
}

type UserRepository interface {
	Create(ctx context.Context, u *model.User) (model.UserID, error)
	FindByLogin(ctx context.Context, loginHash []byte) (model.UserID, model.User, error)
	FindByID(ctx context.Context, uuid model.UserID) (model.User, error)
	Delete(ctx context.Context, uuid model.UserID) error
}

type AuthService struct {
	authpb.UnimplementedAuthServiceServer
	log            *slog.Logger
	repo           UserRepository
	sessionService SessionService
	secret         []byte
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
	if r == nil {
		s.log.ErrorContext(ctx, "login request is <nil>")
		return nil, status.Errorf(codes.InvalidArgument, MsgAgentWrong)
	}

	credentials := r.GetAuthData()
	if credentials == nil {
		s.log.ErrorContext(ctx, "request validation failed: the registration data is nil")
		return nil, status.Errorf(codes.InvalidArgument, MsgRequestValidationFailed)
	} else if credentials.Login == nil || credentials.Password == nil {
		s.log.ErrorContext(ctx,
			"request validation failed: invalid login or password")
		return nil, status.Errorf(codes.InvalidArgument, MsgRequestValidationFailed)
	}

	loginHash := hash.GenerateHMAC(credentials.GetLogin(), s.secret)
	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	userID, userData, err := s.repo.FindByLogin(ctx1, loginHash)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to find user by login",
			"login", credentials.GetLogin(),
			model.KeyLoggerError, err)
		return nil, status.Errorf(codes.Unauthenticated, "failed to find user")
	}
	if err := hash.CompareHashAndPassword(
		userData.PasswordPHC, credentials.GetPassword()); err != nil {
		s.log.ErrorContext(ctx, "password format is wrong", model.KeyLoggerError, err)
		return nil, status.Error(codes.Unauthenticated, "password format is wrong")
	}

	access, refresh, err := s.sessionService.CreateSession(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create session", model.KeyLoggerError, err)
		return nil, status.Error(codes.Internal, "failed to create session")
	}

	resp := &authpb.LoginResponse{
		Tokens: &authpb.Tokens{
			AccessToken:  &authpb.AccessToken{AccessToken: &access},
			RefreshToken: &authpb.RefreshToken{RefreshToken: refresh},
		},
	}

	return resp, nil
}

func (s *AuthService) Logout(ctx context.Context, r *authpb.LogoutRequest,
) (*emptypb.Empty, error) {
	if r == nil {
		s.log.ErrorContext(ctx, "logout request is <nil>")
		return nil, status.Errorf(codes.InvalidArgument, MsgAgentWrong)
	}

	refreshToken := r.GetRefreshToken()
	if refreshToken == nil {
		s.log.ErrorContext(ctx, "request validation failed: the refresh uuid is nil")
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed")
	}
	if err := s.sessionService.RevokeSession(ctx, refreshToken.GetRefreshToken()); err != nil {
		s.log.ErrorContext(ctx, "logout failed", model.KeyLoggerError, err)
		return nil, status.Errorf(codes.Internal, "logout failed")
	}

	return &emptypb.Empty{}, nil
}

func (s *AuthService) Register(ctx context.Context, r *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	if r == nil {
		s.log.ErrorContext(ctx, "register request is <nil>")
		return nil, status.Errorf(codes.InvalidArgument, MsgAgentWrong)
	}

	credentials := r.GetAuthData()
	if credentials == nil {
		s.log.ErrorContext(ctx, "request validation failed: the registration data is nil")
		return nil, status.Errorf(codes.InvalidArgument, MsgRequestValidationFailed)
	} else if credentials.Login == nil || credentials.Password == nil {
		s.log.ErrorContext(ctx,
			"request validation failed: invalid login or password")
		return nil, status.Errorf(codes.InvalidArgument, MsgRequestValidationFailed)
	}

	loginHash := hash.GenerateHMAC(credentials.GetLogin(), s.secret)
	const msgRegistrationFailed = "registration failed, try again"
	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	if _, _, err := s.repo.FindByLogin(ctx1, loginHash); err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "the username is already taken")
	} else if !errors.Is(err, model.ErrNotFound) {
		s.log.ErrorContext(ctx, "failed to check login existence", model.KeyLoggerError, err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}

	passwordHash, err := hash.GenerateFromPassword(credentials.GetPassword())
	if err != nil {
		s.log.ErrorContext(ctx, "failed to hash password", model.KeyLoggerError, err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel2()
	userID, err := s.repo.Create(ctx2,
		&model.User{
			LoginHash:   loginHash,
			PasswordPHC: passwordHash,
		})
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create user", model.KeyLoggerError, err)
		return nil, status.Error(codes.Internal, msgRegistrationFailed)
	}

	access, refresh, err := s.sessionService.CreateSession(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create session", model.KeyLoggerError, err)
		return nil, status.Errorf(codes.Internal, msgRegistrationFailed)
	}

	resp := &authpb.RegisterResponse{
		Tokens: &authpb.Tokens{
			AccessToken:  &authpb.AccessToken{AccessToken: &access},
			RefreshToken: &authpb.RefreshToken{RefreshToken: refresh},
		},
	}
	return resp, nil
}
