package v1

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	service := NewKeeperGRPCService(
		slog.Default(),
		newUseCaseMockBuilder(t).WithAddSealed().Build(),
	)

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
			name:         "nil request",
			request:      nil,
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
			wantResponse: nil,
		},
		{
			name: "user id empty",
			request: &keeperpb.DeleteRequest{
				Metadata: &metadatapb.Metadata{
					DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
					Name:     ptr("error"),
				},
			},
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:   "user id conversion failed",
			userID: "wrong type",
			request: &keeperpb.DeleteRequest{
				Metadata: &metadatapb.Metadata{
					DataType: ptr(metadatapb.Metadata_DATA_TYPE_AUTH),
					Name:     ptr("error"),
				},
			},
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

	service := NewKeeperGRPCService(
		slog.Default(),
		newUseCaseMockBuilder(t).WithDelete().Build(),
	)
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
	dummyID := int64(42)
	dummyTime, _ := time.Parse(time.RFC3339, time.RFC3339)
	correctResponse := &keeperpb.GetResponse{
		Metadata: &metadatapb.Metadata{
			DataType:      ptr(metadatapb.Metadata_DATA_TYPE_UNSPECIFIED),
			Id:            ptr(dummyID),
			Name:          ptr("dummy name"),
			Description:   ptr("dummy description"),
			CreatedAt:     timestamppb.New(dummyTime),
			ChunkMetadata: nil,
		},
		Payload: &commonpb.Payload{
			SealedData: []byte("dummy secret bytes"),
		},
	}

	tests := []struct {
		request       *keeperpb.GetRequest
		userID        interface{}
		name          string
		wantErr       bool
		wantCode      codes.Code
		wantResponses []*keeperpb.GetResponse
	}{
		{
			name:          "nil request",
			request:       nil,
			wantErr:       true,
			wantCode:      codes.InvalidArgument,
			wantResponses: nil,
		},
		{
			name: "user id empty",
			request: &keeperpb.GetRequest{
				Metadata: &metadatapb.Metadata{
					Id: ptr(dummyID),
				}},
			wantErr:       true,
			wantCode:      codes.Unauthenticated,
			wantResponses: nil,
		},
		{
			name:   "user id conversion failed",
			userID: "wrong type",
			request: &keeperpb.GetRequest{
				Metadata: &metadatapb.Metadata{
					Id: ptr(dummyID),
				}},
			wantErr:       true,
			wantCode:      codes.Unauthenticated,
			wantResponses: nil,
		},
		{
			name:   "fail to get metadata",
			userID: model.UserID("userID"),
			request: &keeperpb.GetRequest{
				Metadata: nil,
			},
			wantErr:       true,
			wantCode:      codes.InvalidArgument,
			wantResponses: nil,
		},
		{
			name:   "UseCase.Get failed",
			userID: model.UserID("error"),
			request: &keeperpb.GetRequest{
				Metadata: &metadatapb.Metadata{
					Id: ptr(dummyID),
				},
			},
			wantErr:       true,
			wantCode:      codes.Internal,
			wantResponses: nil,
		},
		{
			name:   "ok",
			userID: model.UserID("userOK"),
			request: &keeperpb.GetRequest{
				Metadata: &metadatapb.Metadata{
					Id: ptr(dummyID),
				},
			},
			wantErr:  false,
			wantCode: codes.OK,
			wantResponses: []*keeperpb.GetResponse{
				correctResponse,
			},
		},
	}

	service := NewKeeperGRPCService(
		slog.Default(),
		newUseCaseMockBuilder(t).WithGetSealed().Build(),
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(
				context.Background(), model.ContextKeyUserID, tt.userID)
			stream := newFakeGetStream(ctx)
			err := service.Get(tt.request, stream)
			if tt.wantErr {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantResponses, stream.responses)
		})
	}
}

func TestKeeperGRPCService_List(t *testing.T) {
	dummyTime, _ := time.Parse(time.RFC3339, time.RFC3339)
	tests := []struct {
		request      *keeperpb.ListRequest
		userID       interface{}
		name         string
		wantErr      bool
		wantCode     codes.Code
		wantResponse *keeperpb.ListResponse
	}{
		{
			name:         "user id empty",
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:         "user id conversion failed",
			userID:       "wrong type",
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
			wantResponse: nil,
		},
		{
			name:     "UseCase.List failed",
			userID:   model.UserID("error"),
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name:     "UseCase.List failed: unknown user",
			userID:   model.UserID("UNKNOWN"),
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name:     "ok #1: return no objects",
			userID:   model.UserID("no-data"),
			wantErr:  false,
			wantCode: codes.OK,
			wantResponse: &keeperpb.ListResponse{
				Metadata: []*metadatapb.Metadata{},
			},
		},
		{
			name:     "ok #2: return single object",
			userID:   model.UserID("single-data"),
			wantErr:  false,
			wantCode: codes.OK,
			wantResponse: &keeperpb.ListResponse{
				Metadata: []*metadatapb.Metadata{
					{
						DataType:    ptr(metadatapb.Metadata_DATA_TYPE_UNSPECIFIED),
						Id:          ptr(int64(dummyID)),
						Name:        ptr("data1"),
						Description: ptr(""),
						CreatedAt:   timestamppb.New(dummyTime),
					},
				}},
		},
		{
			name:     "ok #3: return multiple objects",
			userID:   model.UserID("multiple-data"),
			wantErr:  false,
			wantCode: codes.OK,
			wantResponse: &keeperpb.ListResponse{
				Metadata: []*metadatapb.Metadata{
					{
						DataType:    ptr(metadatapb.Metadata_DATA_TYPE_UNSPECIFIED),
						Id:          ptr(int64(dummyID)),
						Name:        ptr("data1"),
						Description: ptr(""),
						CreatedAt:   timestamppb.New(dummyTime),
					},
					{
						DataType:    ptr(metadatapb.Metadata_DATA_TYPE_UNSPECIFIED),
						Id:          ptr(int64(dummyID)),
						Name:        ptr("data2"),
						Description: ptr(""),
						CreatedAt:   timestamppb.New(dummyTime),
					},
				},
			},
		},
	}

	service := NewKeeperGRPCService(
		slog.Default(),
		newUseCaseMockBuilder(t).WithList().Build(),
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(),
				model.ContextKeyUserID, tt.userID)
			response, err := service.List(ctx, tt.request)
			if tt.wantErr {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.True(t, proto.Equal(tt.wantResponse, response))
		})
	}
}

func TestKeeperGRPCService_Sync(t *testing.T) {

}
