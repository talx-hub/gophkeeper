package keeper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/keeper/mocks"
)

func TestService_AddSealed(t *testing.T) {
	tests := []struct {
		objectRepoMock   *mocks.MockObjectRepo
		metadataRepoMock *mocks.MockMetadataRepo
		meta             *model.Metadata
		sealed           []byte
		name             string
		userID           model.UserID
		wantErr          bool
		wantDataID       model.DataID
	}{
		{
			name:             "empty userID",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "",
			wantErr:          true,
			wantDataID:       0,
		},
		{
			name:             "nil metadata",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "user1",
			wantErr:          true,
			wantDataID:       0,
		},
		{
			name:             "nil sealed data",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "user1",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     nil,
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "empty sealed data",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "user1",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     []byte{},
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "unsupported data type",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "user1",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeUnspecified,
			},
			sealed:     []byte("some secret bytes"),
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "objectRepo.Put() failed",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithPut().Build(),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "put-error-user",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     []byte("some secret bytes"),
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "metadataRepo.Create() failed",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithPut().WithDelete().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithCreate().Build(),
			userID:           "create-error-user",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     []byte("some secret bytes"),
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "metadataRepo.Create() failed && objectRepo.Delete() fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithPut().WithDelete().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithCreate().Build(),
			userID:           "create-and-delete-error-user",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     []byte("some secret bytes"),
			wantErr:    true,
			wantDataID: 0,
		},
		{
			name:             "ok",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithPut().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithCreate().Build(),
			userID:           "user1",
			meta: &model.Metadata{
				CreatedAt:     time.Now().UTC(),
				ChunkMetadata: nil,
				Name:          "data1",
				Description:   "some data",
				DataType:      model.DataTypeBinary,
			},
			sealed:     []byte("some secret bytes"),
			wantErr:    false,
			wantDataID: dummyDataID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.objectRepoMock, tt.metadataRepoMock)

			dataID, err := service.AddSealed(context.Background(),
				tt.userID, tt.meta, tt.sealed)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantDataID, dataID)
		})
	}
}
