package keeper

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/keeper/mocks"
)

const dummyDataID = 42

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
		) (model.ObjectLocator, model.ObjectInfo, error) {
			switch meta.UserID {
			case "put-error-user":
				return "", model.ObjectInfo{}, errors.New("error1")
			default:
				return "dummy/locator", model.ObjectInfo{}, nil
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
			case "error/locator":
				return errors.New("error2")
			default:
				return nil
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
		Create(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			meta *model.Metadata,
			info model.ObjectInfo,
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
