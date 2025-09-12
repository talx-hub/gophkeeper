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

func TestService_GetSealed(t *testing.T) {
	tests := []struct {
		objectRepoMock   *mocks.MockObjectRepo
		metadataRepoMock *mocks.MockMetadataRepo
		name             string
		userID           model.UserID
		id               model.DataID
		cbFake           StreamCallback
		wantError        bool
	}{
		{
			name:             "userID is empty",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "",
			wantError:        true,
		},
		{
			name:             "dataID is empty",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "user-ok",
			id:               0,
			wantError:        true,
		},
		{
			name:             "metadataRepo.Get() fail",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "metadata-repo-error-user",
			id:               dummyDataID,
			wantError:        true,
		},
		{
			name:             "objectRepo.Get() fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "brake-object-repo-get",
			id:               dummyDataID,
			wantError:        true,
		},
		{
			name:             "io.ReadAll() from objectRepo fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "brake-read",
			id:               dummyDataID,
			wantError:        true,
		},
		{
			name:             "readCloser.Close() from objectRepo fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "brake-close",
			id:               dummyDataID,
			wantError:        true,
		},
		{
			name:             "callback fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "user-ok",
			id:               dummyDataID,
			cbFake:           fakeErrorCallback,
			wantError:        true,
		},
		{
			name:             "ok",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithGet().Build(),
			userID:           "user-ok",
			id:               dummyDataID,
			cbFake:           fakeOKCallback,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.objectRepoMock, tt.metadataRepoMock)
			err := service.GetSealed(context.Background(), tt.userID, tt.id, tt.cbFake)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	tests := []struct {
		metadataRepoMock *mocks.MockMetadataRepo
		name             string
		userID           model.UserID
		wantError        bool
	}{
		{
			name:             "empty userID",
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "",
			wantError:        true,
		},
		{
			name:             "metadataRepo.ListByUser() fail",
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "brake-repo-user",
			wantError:        true,
		},
		{
			name:             "ok",
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "ok-user",
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil, tt.metadataRepoMock)
			list, err := service.List(context.Background(), tt.userID)
			if tt.wantError {
				require.Error(t, err)
				assert.Empty(t, list)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, list)
		})
	}
}

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name             string
		objectRepoMock   *mocks.MockObjectRepo
		metadataRepoMock *mocks.MockMetadataRepo
		userID           model.UserID
		dataID           model.DataID
		wantError        bool
	}{
		{
			name:             "userID empty",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "",
			dataID:           dummyDataID,
			wantError:        true,
		},
		{
			name:             "dataID empty",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "ok-user",
			dataID:           0,
			wantError:        true,
		},
		{
			name:             "metadataRepo.Delete() fail",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithDelete().Build(),
			userID:           "break-metadataRepo",
			dataID:           dummyDataID,
			wantError:        true,
		},
		{
			name:             "objectRepo.Delete() fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithDelete().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithDelete().Build(),
			userID:           "break-objectRepo",
			dataID:           dummyDataID,
			wantError:        true,
		},
		{
			name:             "ok",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithDelete().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithDelete().Build(),
			userID:           "ok-user",
			dataID:           dummyDataID,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.objectRepoMock, tt.metadataRepoMock)
			err := service.Delete(context.Background(), tt.userID, tt.dataID)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Sync(t *testing.T) {
	tests := []struct {
		name             string
		objectRepoMock   *mocks.MockObjectRepo
		metadataRepoMock *mocks.MockMetadataRepo
		userID           model.UserID
		mode             SyncMode
		cbFake           StreamCallback
		wantError        bool
	}{
		{
			name:             "userID empty",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: mocks.NewMockMetadataRepo(t),
			userID:           "",
			wantError:        true,
		},
		{
			name:             "metadataRepo.ListByUser() fail",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "brake-repo-user",
			wantError:        true,
		},
		{
			name:             "objectRepo.Get() fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "brake-object-repo-get",
			mode:             SyncModeShort,
			wantError:        true,
		},
		{
			name:             "unknown sync mode",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "ok-user",
			mode:             SyncMode(0),
			cbFake:           fakeOKCallback,
			wantError:        true,
		},
		{
			name:             "callback fail",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "ok-user",
			mode:             SyncModeShort,
			cbFake:           fakeErrorCallback,
			wantError:        true,
		},
		{
			name:             "full sync mode",
			objectRepoMock:   mocks.NewMockObjectRepo(t),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "ok-user",
			mode:             SyncModeFull,
			cbFake:           fakeOKCallback,
			wantError:        true,
		},
		{
			name:             "ok",
			objectRepoMock:   newObjectRepoMockBuilder(t).WithGet().Build(),
			metadataRepoMock: newMetadataRepoMockBuilder(t).WithListByUser().Build(),
			userID:           "ok-user",
			mode:             SyncModeShort,
			cbFake:           fakeOKCallback,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		objectRepo := tt.objectRepoMock
		metadataRepo := tt.metadataRepoMock
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(objectRepo, metadataRepo)
			err := service.Sync(context.Background(), tt.userID, tt.mode, tt.cbFake)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
