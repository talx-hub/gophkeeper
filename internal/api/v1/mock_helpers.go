package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gophkeeper/internal/api/v1/mocks"
	"github.com/talx-hub/gophkeeper/internal/model"
)

type repoMockBuilder struct {
	repo *mocks.MockUserRepository
}

func newRepoMock(t *testing.T) *repoMockBuilder {
	r := mocks.NewMockUserRepository(t)
	t.Cleanup(func() {
		r.AssertExpectations(t)
	})
	return &repoMockBuilder{repo: r}
}

func (b *repoMockBuilder) Build() *mocks.MockUserRepository {
	return b.repo
}

func (b *repoMockBuilder) WithFindByLogin() *repoMockBuilder {
	b.repo.EXPECT().
		FindByLogin(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, login string) (model.User, error) {
			switch login {
			case "db-fail":
				return model.User{}, errors.New("db error")
			case "new-user", "not-found", "create-fail", "session-fail-register":
				return model.User{}, model.ErrNotFound
			case "already-exists":
				return model.User{}, nil
			}

			const hashForDummyPassword = "$2a$10$1rrMx6jcq7KoehP5mXc4HOBn2BgKi.6O1Blgc1uBTNHwBYvhTP2VC"
			return model.User{
				Login:        login,
				PasswordHash: []byte(hashForDummyPassword),
				UUID:         model.UserID(login),
			}, nil
		})
	return b
}

func (b *repoMockBuilder) WithCreate() *repoMockBuilder {
	b.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, u *model.User) (model.UserID, error) {
			if u.Login == "create-fail" {
				return "", errors.New("user create failed")
			}
			return model.UserID(u.Login), nil
		})

	return b
}

func (b *repoMockBuilder) WithFindByID() *repoMockBuilder {
	b.repo.EXPECT().
		FindByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, uuid model.UserID) (model.User, error) {
			if uuid == "db-fail" {
				return model.User{}, errors.New("db error")
			}
			if uuid == "not-found" {
				return model.User{}, model.ErrNotFound
			}

			return model.User{
				Login:        "dummy-login",
				PasswordHash: []byte("dummy bytes"),
				UUID:         uuid,
			}, nil
		})

	return b
}

func (b *repoMockBuilder) WithDelete() *repoMockBuilder {
	b.repo.EXPECT().
		Delete(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, uuid model.UserID) error {
			if uuid == "db-fail" {
				return errors.New("db error")
			}
			if uuid == "not-found" {
				return model.ErrNotFound
			}
			return nil
		})

	return b
}

type sessionMockBuilder struct {
	service *mocks.MockSessionService
}

func newSessionMock(t *testing.T) *sessionMockBuilder {
	s := mocks.NewMockSessionService(t)
	t.Cleanup(func() {
		s.AssertExpectations(t)
	})
	return &sessionMockBuilder{service: s}
}

func (s *sessionMockBuilder) Build() *mocks.MockSessionService {
	return s.service
}

func (s *sessionMockBuilder) WithCreateSession() *sessionMockBuilder {
	s.service.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, userID model.UserID) (string, string, error) {
			if userID == "dummy-user-id" {
				return "valid-jwt", "valid-refresh", nil
			}
			if userID == "session-fail" || userID == "session-fail-register" {
				return "", "", errors.New("create session error")
			}
			return "valid-jwt", "valid-refresh", nil
		})

	return s
}

func (s *sessionMockBuilder) WithRevokeSession() *sessionMockBuilder {
	s.service.EXPECT().
		RevokeSession(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, refreshToken string) error {
			if refreshToken == "revoke-fail" {
				return errors.New("revoke failed")
			}
			return nil
		})

	return s
}
