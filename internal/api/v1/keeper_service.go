package v1

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/talx-hub/gophkeeper/internal/adapters/grpc/metadata"
	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/keeper"
	commonpb "github.com/talx-hub/gophkeeper/proto/v1/common"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
	metadatapb "github.com/talx-hub/gophkeeper/proto/v1/metadata"
)

const MsgAgentWrong = "agent error"

type KeeperGRPCService struct {
	keeperpb.UnimplementedKeeperServer
	keeperUseCase KeeperUseCase
	log           *slog.Logger
}

type KeeperUseCase interface {
	AddSealed(ctx context.Context, userID model.UserID, meta *model.Metadata, sealed []byte) (model.DataID, error)
	GetSealed(ctx context.Context, userID model.UserID, id model.DataID, callback keeper.GRPCStreamSenderCB) error
	List(ctx context.Context, userID model.UserID) ([]model.Metadata, error)
	Delete(ctx context.Context, userID model.UserID, id model.DataID) error
	Sync(ctx context.Context, userID model.UserID, mode keeper.SyncMode, callback keeper.GRPCStreamSenderCB) error
}

func (s *KeeperGRPCService) Sync(
	req *keeperpb.SyncRequest, stream grpc.ServerStreamingServer[keeperpb.SyncResponse],
) error {
	ctx := stream.Context()
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, "failed to convert ctx.Value(ContextKeyUserID) to (model.UserID)",
			"actual_type", actualType)
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	mode := keeper.SyncModeShort
	if req.GetSyncMode() == keeperpb.SyncRequest_SYNC_MODE_FULL {
		mode = keeper.SyncModeFull
	}

	// запускаем use-case -- он питюкает в stream через коллбек
	err := s.keeperUseCase.Sync(ctx, userID, mode,
		func(m *model.Metadata, sealed []byte) error {
			return stream.Send(
				&keeperpb.SyncResponse{
					Metadata: metadata.ToProtoMetadata(m),
					Payload:  &commonpb.Payload{SealedData: sealed},
				})
		})
	if err != nil {
		s.log.ErrorContext(ctx,
			"sync failed",
			"userID", userID,
			"err", err)
		return status.Error(codes.Internal, "sync failed")
	}
	return nil
}

func (s *KeeperGRPCService) Add(
	stream grpc.ClientStreamingServer[keeperpb.AddRequest, keeperpb.AddResponse],
) error {
	ctx := stream.Context()
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, "failed to convert ctx.Value(ContextKeyUserID) to (model.UserID)",
			"actual_type", actualType)
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.log.ErrorContext(ctx, "recv failed", "userID", userID, "err", err)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}

		metaDTO := req.GetMetadata()
		if metaDTO == nil {
			s.log.ErrorContext(ctx, "bad metadata: metadata is empty", "userID", userID)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}
		payload := req.GetPayload()
		if payload == nil {
			s.log.ErrorContext(ctx, "payload is nil", "userID", userID)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}

		meta, err := metadata.FromProtoMetadata(metaDTO)
		if err != nil {
			s.log.ErrorContext(ctx,
				"bad metadata",
				"userID", userID,
				"err", err,
			)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}
		sealedData := payload.GetSealedData()
		if sealedData == nil {
			s.log.ErrorContext(ctx, "sealedData and binaryChunk are empty",
				"userID", userID,
			)
			return status.Error(codes.InvalidArgument, "data should be filled")
		}
		if _, err := s.keeperUseCase.AddSealed(ctx, userID, meta, sealedData); err != nil {
			s.log.ErrorContext(ctx, "failed to AddSealed",
				"userID", userID,
				"err", err)
			return status.Error(codes.Internal, "add failed")
		}
	}

	return stream.SendAndClose(&keeperpb.AddResponse{})
}

func (s *KeeperGRPCService) List(ctx context.Context, _ *keeperpb.ListRequest,
) (*keeperpb.ListResponse, error) {
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok {
		s.log.ErrorContext(ctx,
			"failed to convert userID extracted from ctx to model.UserID",
			"real_type", reflect.TypeOf(userID).String(),
		)
		return nil, status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	list, err := s.keeperUseCase.List(ctxTO, userID)
	if err != nil {
		s.log.ErrorContext(ctxTO, "failed to list metadata from Repository",
			"userID", userID,
			"err", err,
		)
		return nil, status.Error(codes.Internal, "server internal error")
	}

	listDTO := make([]*metadatapb.Metadata, len(list))
	for i, elem := range list {
		listDTO[i] = metadata.ToProtoMetadata(&elem)
	}

	return &keeperpb.ListResponse{
		Metadata: listDTO,
	}, nil
}

func (s *KeeperGRPCService) Get(
	req *keeperpb.GetRequest,
	stream grpc.ServerStreamingServer[keeperpb.GetResponse],
) error {
	ctx := stream.Context()
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, "failed to convert ctx.Value(ContextKeyUserID) to (model.UserID)",
			"actual_type", actualType)
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	metaDTO := req.GetMetadata()
	if metaDTO == nil {
		s.log.ErrorContext(ctx, "bad metadata: metadata is empty")
		return status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	// запускаем use-case -- он питюкает в stream через коллбек
	err := s.keeperUseCase.GetSealed(ctx, userID, model.DataID(metaDTO.GetId()),
		func(m *model.Metadata, sealed []byte) error {
			return stream.Send(
				&keeperpb.GetResponse{
					Metadata: metadata.ToProtoMetadata(m),
					Payload:  &commonpb.Payload{SealedData: sealed},
				})
		})
	if err != nil {
		s.log.ErrorContext(ctx,
			"sync failed",
			"userID", userID,
			"dataID", metaDTO.GetId(),
			"err", err)
		return status.Error(codes.Internal, "sync failed")
	}
	return nil
}

func (s *KeeperGRPCService) Delete(ctx context.Context, req *keeperpb.DeleteRequest,
) (*keeperpb.DeleteResponse, error) {
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, "failed to convert ctx.Value(ContextKeyUserID) to (model.UserID)",
			"actual_type", actualType)
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	metaDTO := req.GetMetadata()
	if metaDTO == nil {
		s.log.ErrorContext(ctx, "bad metadata: metadata is empty")
		return nil, status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	err := s.keeperUseCase.Delete(ctx, userID, model.DataID(metaDTO.GetId()))
	if err != nil {
		s.log.ErrorContext(ctx,
			"delete failed",
			"userID", userID,
			"dataID", metaDTO.GetId(),
			"err", err)
		return nil, status.Error(codes.Internal, "delete failed")
	}

	return &keeperpb.DeleteResponse{}, nil
}
