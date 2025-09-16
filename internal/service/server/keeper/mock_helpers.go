package keeper

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/server/keeper/mocks"
)

const dummyDataID = 42
const dummyError = "error"

type objectRepoMockBuilder struct {
	objectRepo *mocks.MockObjectRepo
}

func newObjectRepoMockBuilder(t *testing.T) *objectRepoMockBuilder {
	t.Helper()

	mockObjectRepo := mocks.NewMockObjectRepo(t)
	t.Cleanup(func() {
		mockObjectRepo.AssertExpectations(t)
	})
	return &objectRepoMockBuilder{objectRepo: mockObjectRepo}
}

func (b *objectRepoMockBuilder) Build() *mocks.MockObjectRepo {
	return b.objectRepo
}

func (b *objectRepoMockBuilder) WithPut() *objectRepoMockBuilder {
	b.objectRepo.EXPECT().
		Put(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			meta *model.Metadata,
			r io.Reader,
			size uint64,
			sha256 []byte,
		) (model.ObjectLocator, error) {
			switch meta.UserID {
			case "put-error-user":
				return "", errors.New("error1")
			case "create-and-delete-error-user":
				return "locator://break/object/repo", nil
			default:
				return "dummy/locator", nil
			}
		})

	return b
}

func (b *objectRepoMockBuilder) WithDelete() *objectRepoMockBuilder {
	b.objectRepo.EXPECT().
		Delete(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context,
			loc model.ObjectLocator,
		) error {
			switch loc {
			case "locator://break/object/repo":
				return errors.New("error2")
			default:
				return nil
			}
		})
	return b
}

func (b *objectRepoMockBuilder) WithGet() *objectRepoMockBuilder {
	b.objectRepo.EXPECT().
		Get(mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			loc model.ObjectLocator,
		) (io.ReadCloser, error) {
			switch loc {
			case "brake://read/closer/read":
				return &fakeReadErrorReadCloser{}, nil
			case "brake://read/closer/close":
				return &fakeCloseErrorReadCloser{}, nil
			case "locator://error":
				return nil, errors.New(dummyError)
			default:
				return &fakeOKReadCloser{}, nil
			}
		})
	return b
}

type metadataRepoMockBuilder struct {
	metadataRepo *mocks.MockMetadataRepo
}

func newMetadataRepoMockBuilder(t *testing.T) *metadataRepoMockBuilder {
	t.Helper()

	mockMetadataRepo := mocks.NewMockMetadataRepo(t)
	t.Cleanup(func() {
		mockMetadataRepo.AssertExpectations(t)
	})
	return &metadataRepoMockBuilder{metadataRepo: mockMetadataRepo}
}

func (b *metadataRepoMockBuilder) Build() *mocks.MockMetadataRepo {
	return b.metadataRepo
}

func (b *metadataRepoMockBuilder) WithCreate() *metadataRepoMockBuilder {
	b.metadataRepo.EXPECT().
		Create(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			meta *model.Metadata,
			loc model.ObjectLocator,
		) (model.DataID, error) {
			switch meta.UserID {
			case "create-error-user", "create-and-delete-error-user":
				return 0, errors.New("error3")
			default:
				return dummyDataID, nil
			}
		})
	return b
}

func (b *metadataRepoMockBuilder) WithDelete() *metadataRepoMockBuilder {
	b.metadataRepo.EXPECT().
		Delete(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context,
			userID model.UserID,
			id model.DataID,
		) (model.ObjectLocator, error) {
			switch userID {
			case "break-metadataRepo":
				return "", errors.New(dummyError)
			case "break-objectRepo":
				return "locator://break/object/repo", nil
			default:
				return "locator://dummy", nil
			}
		})
	return b
}

func (b *metadataRepoMockBuilder) WithGet() *metadataRepoMockBuilder {
	b.metadataRepo.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context,
			userID model.UserID,
			id model.DataID,
		) (model.Metadata, model.ObjectLocator, error) {
			switch userID {
			case "metadata-repo-error-user":
				return model.Metadata{}, "", errors.New(dummyError)
			case "brake-object-repo-get":
				return model.Metadata{UserID: userID, ID: id},
					"locator://error", nil
			case "brake-read":
				return model.Metadata{UserID: userID, ID: id},
					"brake://read/closer/read", nil
			case "brake-close":
				return model.Metadata{UserID: userID, ID: id},
					"brake://read/closer/close", nil
			default:
				return model.Metadata{UserID: userID, ID: id},
					"locator://ok", nil
			}
		})
	return b
}

func (b *metadataRepoMockBuilder) WithListByUser() *metadataRepoMockBuilder {
	b.metadataRepo.EXPECT().
		ListByUser(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context,
			userID model.UserID,
		) ([]model.MetaLoc, error) {
			switch userID {
			case "ok-user":
				return []model.MetaLoc{{}, {}}, nil
			case "brake-object-repo-get":
				return []model.MetaLoc{{Locator: "locator://error"}}, nil
			default:
				return nil, errors.New(dummyError)
			}
		})
	return b
}

type fakeOKReadCloser struct {
}

func (rc *fakeOKReadCloser) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (rc *fakeOKReadCloser) Close() error {
	return nil
}

type fakeReadErrorReadCloser struct {
}

func (rc *fakeReadErrorReadCloser) Read(_ []byte) (int, error) {
	return 0, errors.New("dummy read error")
}

func (rc *fakeReadErrorReadCloser) Close() error {
	return nil
}

type fakeCloseErrorReadCloser struct {
}

func (rc *fakeCloseErrorReadCloser) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (rc *fakeCloseErrorReadCloser) Close() error {
	return errors.New("dummy close error")
}

var fakeOKCallback StreamCallback = func(_ *model.Metadata, _ []byte) error {
	return nil
}

var fakeErrorCallback StreamCallback = func(_ *model.Metadata, _ []byte) error {
	return errors.New("callback error")
}
