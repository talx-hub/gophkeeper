package v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gophkeeper/internal/api/v1/mocks"
	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/server/keeper"
)

type repoMockBuilder struct {
	repo *mocks.MockUserRepository
}

func newRepoMock(t *testing.T) *repoMockBuilder {
	t.Helper()

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
		RunAndReturn(func(_ context.Context, loginHash []byte) (model.UserID, model.User, error) {
			switch string(loginHash) {
			case string(fixtureLoginDBFail):
				return "", model.User{}, errors.New("db error")
			case string(fixtureLoginNewUser), string(fixtureLoginNotFound),
				string(fixtureLoginCreateFailed):
				return "", model.User{}, model.ErrNotFound
			case string(fixtureLoginAlreadyExists):
				return "", model.User{}, nil
			case string(fixtureLoginSessionFail):
				return keySessionFail, model.User{
					LoginHash:   loginHash,
					PasswordPHC: fixturePHC,
				}, nil
			case string(fixtureLoginSessionFailRegister):
				return keySessionFail, model.User{
					LoginHash:   loginHash,
					PasswordPHC: fixturePHC,
				}, model.ErrNotFound
			}

			return keyDummyUserID,
				model.User{
					LoginHash:   loginHash,
					PasswordPHC: fixturePHC,
				}, nil
		})
	return b
}

func (b *repoMockBuilder) WithCreate() *repoMockBuilder {
	b.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, u *model.User) (model.UserID, error) {
			if bytes.Equal(u.LoginHash, fixtureLoginCreateFailed) {
				return "", errors.New("user create failed")
			}
			if bytes.Equal(u.LoginHash, fixtureLoginSessionFailRegister) {
				return keySessionFail, nil
			}

			return model.UserID(u.LoginHash), nil
		})

	return b
}

func (b *repoMockBuilder) WithDelete() *repoMockBuilder {
	b.repo.EXPECT().
		Delete(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, uuid model.UserID) error {
			if uuid == keyDBFail {
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
	service *mocks.MockSessionManager
}

func newSessionMock(t *testing.T) *sessionMockBuilder {
	t.Helper()

	s := mocks.NewMockSessionManager(t)
	t.Cleanup(func() {
		s.AssertExpectations(t)
	})
	return &sessionMockBuilder{service: s}
}

func (s *sessionMockBuilder) Build() *mocks.MockSessionManager {
	return s.service
}

func (s *sessionMockBuilder) WithCreateSession() *sessionMockBuilder {
	s.service.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, userID model.UserID) (string, []byte, error) {
			if userID == keyDummyUserID {
				return "valid-jwt", []byte("valid-refresh"), nil
			}
			if userID == keySessionFail {
				return "", nil, errors.New("create session error")
			}
			return "valid-jwt", []byte("valid-refresh"), nil
		})

	return s
}

func (s *sessionMockBuilder) WithRevokeSession() *sessionMockBuilder {
	s.service.EXPECT().
		RevokeSession(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, refreshToken []byte) error {
			if bytes.Equal(refreshToken, []byte("revoke-fail")) {
				return errors.New("revoke failed")
			}
			return nil
		})

	return s
}

type useCaseMockBuilder struct {
	usecase *mocks.MockKeeperUseCase
}

func newUseCaseMockBuilder(t *testing.T) *useCaseMockBuilder {
	t.Helper()

	useCase := mocks.NewMockKeeperUseCase(t)
	t.Cleanup(func() {
		useCase.AssertExpectations(t)
	})

	return &useCaseMockBuilder{
		usecase: useCase,
	}
}

func (b *useCaseMockBuilder) Build() *mocks.MockKeeperUseCase {
	return b.usecase
}

func (b *useCaseMockBuilder) WithAddSealed() *useCaseMockBuilder {
	b.usecase.EXPECT().
		AddSealed(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(
			func(
				ctx context.Context,
				userID model.UserID,
				meta *model.Metadata,
				sealed []byte,
			) (model.DataID, error) {
				if userID == "error" {
					return "", errors.New(msgExpectedError)
				}
				return dummyID, nil
			})

	return b
}

func (b *useCaseMockBuilder) WithDelete() *useCaseMockBuilder {
	b.usecase.EXPECT().
		Delete(mock.Anything, mock.Anything).
		RunAndReturn(
			func(
				ctx context.Context,
				id model.DataID,
			) error {
				if id == "brake-delete" {
					return errors.New(msgExpectedError)
				}
				return nil
			})

	return b
}

func (b *useCaseMockBuilder) WithGetSealed() *useCaseMockBuilder {
	dummyTime, _ := time.Parse(time.RFC3339, time.RFC3339)

	b.usecase.EXPECT().
		GetSealed(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(
			func(
				ctx context.Context,
				id model.DataID,
				callback keeper.StreamCallback,
			) error {
				if id == "brake-get" {
					return errors.New(msgExpectedError)
				}

				if err := callback(
					&model.Metadata{
						CreatedAt:       dummyTime,
						ChunkDescriptor: nil,
						Name:            "dummy name",
						Description:     "dummy description",
						ID:              dummyID,
						DataType:        model.DataTypeUnspecified,
					},
					[]byte("dummy secret bytes"),
				); err != nil {
					return fmt.Errorf("callback failed: %w", err)
				}
				return nil
			})

	return b
}

func (b *useCaseMockBuilder) WithList() *useCaseMockBuilder {
	dummyTime, _ := time.Parse(time.RFC3339, time.RFC3339)

	b.usecase.EXPECT().
		List(mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			userID model.UserID,
		) ([]model.MetaLoc, error) {
			switch userID {
			case "error":
				return nil, errors.New(msgExpectedError)
			case "no-data":
				return []model.MetaLoc{}, nil
			case "single-data":
				return []model.MetaLoc{
					{
						Locator: "pg:/single-data/data1",
						Meta: model.Metadata{
							Name:        "data1",
							ID:          dummyID,
							DataType:    model.DataTypeUnspecified,
							Description: "",
							CreatedAt:   dummyTime,
						},
					},
				}, nil
			case "multiple-data":
				return []model.MetaLoc{
					{
						Locator: "pg://multiple-data/data1",
						Meta: model.Metadata{
							Name:        "data1",
							ID:          dummyID,
							DataType:    model.DataTypeUnspecified,
							Description: "",
							CreatedAt:   dummyTime,
						},
					},
					{
						Locator: "s3://multiple-data/data2",
						Meta: model.Metadata{
							Name:        "data2",
							ID:          dummyID,
							DataType:    model.DataTypeUnspecified,
							Description: "",
							CreatedAt:   dummyTime,
						},
					},
				}, nil
			default:
				return nil, fmt.Errorf("no such user: %s", userID)
			}
		})

	return b
}

func (b *useCaseMockBuilder) WithSync() *useCaseMockBuilder {
	dummyTime, _ := time.Parse(time.RFC3339, time.RFC3339)

	b.usecase.EXPECT().
		Sync(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			userID model.UserID,
			mode keeper.SyncMode,
			callback keeper.StreamCallback,
		) error {
			if userID == "error" {
				return errors.New(msgExpectedError)
			}

			metadata := &model.Metadata{
				CreatedAt:       dummyTime,
				ChunkDescriptor: nil,
				Name:            "dummy name",
				Description:     "dummy description",
				ID:              dummyID,
				DataType:        model.DataTypeUnspecified,
			}

			var err error
			switch mode {
			case keeper.SyncModeShort:
				err = callback(metadata, nil)
			case keeper.SyncModeFull:
				err = callback(metadata, []byte("dummy secret bytes"))
			default:
				return fmt.Errorf("unknown  sync mode: %d", mode)
			}

			if err != nil {
				return fmt.Errorf("callback failed: %w", err)
			}
			return nil
		})

	return b
}
