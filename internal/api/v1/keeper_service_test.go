package v1

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/talx-hub/gophkeeper/internal/model"
	commonpb "github.com/talx-hub/gophkeeper/proto/v1/common"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
	metadatapb "github.com/talx-hub/gophkeeper/proto/v1/metadata"
)

func TestKeeperGRPCService_Add(t *testing.T) {
	tests := []struct {
		userID       interface{}
		wantResponse *keeperpb.AddResponse
		reqs         []*keeperpb.AddRequest
		name         string
		wantCode     codes.Code
		wantErr      bool
	}{
		{
			name:         "user id empty",
			reqs:         nil,
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:         "user id conversion failed",
			userID:       "wrong type",
			reqs:         nil,
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:         "instant EOF",
			userID:       model.UserID("userID"),
			reqs:         []*keeperpb.AddRequest{},
			wantErr:      false,
			wantCode:     codes.OK,
			wantResponse: &keeperpb.AddResponse{},
		},
		{
			name:         "unexpected Recv error",
			userID:       model.UserID("userID"),
			reqs:         nil,
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "fail to get metadata",
			userID: model.UserID("userID"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: nil,
					Payload: &commonpb.Payload{
						SealedData: []byte("secret bytes"),
					},
				},
			},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "fail to get payload",
			userID: model.UserID("userID"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: &metadatapb.Metadata{
						DataType: ptr(metadatapb.Metadata_DATA_TYPE_CARD),
						Name:     ptr("nil payload"),
					},
					Payload: nil,
				},
			},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "fail to parse metadata",
			userID: model.UserID("userID"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: &metadatapb.Metadata{
						DataType: nil,
					},
					Payload: &commonpb.Payload{
						SealedData: []byte("secret bytes"),
					},
				},
			},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "fail to convert payload to []byte",
			userID: model.UserID("userID"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: &metadatapb.Metadata{
						DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
						Name:     ptr("nil sealed data"),
					},
					Payload: &commonpb.Payload{
						SealedData: nil,
					},
				},
			},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "UseCase add failed",
			userID: model.UserID("error"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: &metadatapb.Metadata{
						DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
						Name:     ptr("name0"),
					},
					Payload: &commonpb.Payload{
						SealedData: []byte("secret bytes"),
					},
				},
			},
			wantErr:      true,
			wantCode:     codes.Internal,
			wantResponse: nil,
		},
		{
			name:   "ok",
			userID: model.UserID("userOK"),
			reqs: []*keeperpb.AddRequest{
				{
					Metadata: &metadatapb.Metadata{
						DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
						Name:     ptr("name0"),
					},
					Payload: &commonpb.Payload{
						SealedData: []byte("secret bytes"),
					},
				},
			},
			wantErr:      false,
			wantCode:     codes.OK,
			wantResponse: &keeperpb.AddResponse{},
		},
	}

	service := KeeperGRPCService{
		keeperUseCase: newUseCaseMockBuilder(t).WithAddSealed().Build(),
		log:           slog.Default(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(),
				model.ContextKeyUserID, tt.userID)

			stream := newFakeAddStream(ctx, tt.reqs...)
			err := service.Add(stream)
			if tt.wantErr {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantResponse, stream.resp)
		})
	}
}

func TestKeeperGRPCService_Delete(t *testing.T) {
	tests := []struct {
		userID       interface{}
		wantResponse *keeperpb.DeleteResponse
		request      *keeperpb.DeleteRequest
		name         string
		wantCode     codes.Code
		wantErr      bool
	}{
		{
			name:         "user id empty",
			request:      nil,
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:         "user id conversion failed",
			userID:       "wrong type",
			request:      nil,
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:   "fail to get metadata",
			userID: model.UserID("userID"),
			request: &keeperpb.DeleteRequest{
				Metadata: nil,
			},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name:   "UseCase delete failed",
			userID: model.UserID("error"),
			request: &keeperpb.DeleteRequest{
				Metadata: &metadatapb.Metadata{
					DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
					Name:     ptr("error"),
				},
			},
			wantErr:      true,
			wantCode:     codes.Internal,
			wantResponse: nil,
		},
		{
			name:   "ok",
			userID: model.UserID("userOK"),
			request: &keeperpb.DeleteRequest{
				Metadata: &metadatapb.Metadata{
					DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
					Name:     ptr("name0"),
				},
			},
			wantErr:      false,
			wantCode:     codes.OK,
			wantResponse: &keeperpb.DeleteResponse{},
		},
	}

	service := KeeperGRPCService{
		keeperUseCase: newUseCaseMockBuilder(t).WithDelete().Build(),
		log:           slog.Default(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(),
				model.ContextKeyUserID, tt.userID)

			response, err := service.Delete(ctx, tt.request)
			if tt.wantErr {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantResponse, response)
		})
	}
}

func TestKeeperGRPCService_Get(t *testing.T) {

}

func TestKeeperGRPCService_List(t *testing.T) {

}

func TestKeeperGRPCService_Sync(t *testing.T) {

}
