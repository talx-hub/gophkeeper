package v1

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/talx-hub/gophkeeper/internal/model"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
)

func strPtr(s string) *string { return &s }

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name     string
		req      *authpb.LoginRequest
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:     "invalid: nil authData",
			req:      &authpb.LoginRequest{AuthData: nil},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid: nil login",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: nil, Password: strPtr("p")}},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid: nil password",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: strPtr("dummy-user-id"), Password: nil}},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "repo error on FindByLogin → Unauthenticated",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: strPtr("db-fail"), Password: strPtr("p")}},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "wrong password → Unauthenticated",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: strPtr("dummy-user-id"), Password: strPtr("wrong")}},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "create session failed → Internal",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: strPtr("session-fail"), Password: strPtr("very-long-dummy-bytes")}},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name:     "success",
			req:      &authpb.LoginRequest{AuthData: &authpb.AuthData{Login: strPtr("dummy-user-id"), Password: strPtr("very-long-dummy-bytes")}},
			wantCode: codes.OK,
		},
	}

	mockRepo := newRepoMock(t).WithFindByLogin().Build()
	mockService := newSessionMock(t).WithCreateSession().Build()
	s := &AuthService{
		repo:           mockRepo,
		sessionService: mockService,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxTO, cancel := context.WithTimeout(context.Background(), model.RepoOperationTO)
			_, err := s.Login(ctxTO, tt.req)
			cancel()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name    string
		req     *authpb.LogoutRequest
		wantErr bool
	}{
		{
			name:    "nil refresh token",
			req:     &authpb.LogoutRequest{RefreshToken: nil},
			wantErr: true,
		},
		{
			name:    "empty refresh token string",
			req:     &authpb.LogoutRequest{RefreshToken: &authpb.RefreshToken{RefreshToken: strPtr("")}},
			wantErr: false, // или true, если мок настроен падать на пустой токен
		},
		{
			name:    "valid refresh token",
			req:     &authpb.LogoutRequest{RefreshToken: &authpb.RefreshToken{RefreshToken: strPtr("refresh-ok")}},
			wantErr: false,
		},
		{
			name:    "session service returns error",
			req:     &authpb.LogoutRequest{RefreshToken: &authpb.RefreshToken{RefreshToken: strPtr("revoke-fail")}},
			wantErr: true,
		},
	}

	mockRepo := newRepoMock(t).Build()
	mockService := newSessionMock(t).WithRevokeSession().Build()
	s := &AuthService{
		repo:           mockRepo,
		sessionService: mockService,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxTO, cancel := context.WithTimeout(context.Background(), model.RepoOperationTO)
			_, err := s.Logout(ctxTO, tt.req)
			cancel()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthService_Register(t *testing.T) {
	const passMultiplier = 4
	tests := []struct {
		name     string
		req      *authpb.RegisterRequest
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name:     "nil auth data",
			req:      &authpb.RegisterRequest{AuthData: nil},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "nil login",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: nil, Password: strPtr("pass")},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "nil password",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("user"), Password: nil},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "login already exists",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("already-exists"), Password: strPtr("pass")},
			},
			wantErr:  true,
			wantCode: codes.AlreadyExists,
		},
		{
			name: "repo findByLogin error not ErrNotFound",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("db-fail"), Password: strPtr("pass")},
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "bcrypt generation error",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{
					Login:    strPtr("new-user"),
					Password: strPtr(strings.Repeat("more-than-72-byte-long-pass", passMultiplier)),
				},
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "create user error",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("create-fail"), Password: strPtr("pass")},
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "generate tokens error",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("session-fail-register"), Password: strPtr("pass")},
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req: &authpb.RegisterRequest{
				AuthData: &authpb.AuthData{Login: strPtr("new-user"), Password: strPtr("pass")},
			},
			wantErr:  false,
			wantCode: codes.OK,
		},
	}

	s := &AuthService{
		repo:           newRepoMock(t).WithCreate().WithFindByLogin().Build(),
		sessionService: newSessionMock(t).WithCreateSession().Build(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxTO, cancel := context.WithTimeout(context.Background(), model.RepoOperationTO)
			_, err := s.Register(ctxTO, tt.req)
			cancel()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNewAuthService(t *testing.T) {
	type args struct {
		repo    UserRepository
		session SessionService
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "both non-nil",
			args: args{
				repo:    newRepoMock(t).Build(),
				session: newSessionMock(t).Build(),
			},
		},
		{
			name: "nil repo, non-nil session",
			args: args{
				repo:    nil,
				session: newSessionMock(t).Build(),
			},
		},
		{
			name: "non-nil repo, nil session",
			args: args{
				repo:    newRepoMock(t).Build(),
				session: nil,
			},
		},
		{
			name: "both nil",
			args: args{
				repo:    nil,
				session: nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(tt.args.repo, tt.args.session)
			if svc == nil {
				t.Fatalf("NewAuthService() returned nil")
			}
			if !reflect.DeepEqual(svc.repo, tt.args.repo) {
				t.Errorf("repo mismatch: got %v, want %v", svc.repo, tt.args.repo)
			}
			if !reflect.DeepEqual(svc.sessionService, tt.args.session) {
				t.Errorf("sessionService mismatch: got %v, want %v", svc.sessionService, tt.args.session)
			}
		})
	}
}
