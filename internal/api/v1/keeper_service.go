// Package v1 содержит gRPC-обработчики (driving-adapters) сервиса Keeper.
// Хэндлеры занимаются маппингом proto <-> доменная модель и делегируют
// бизнес-логику в use-case (internal/service/keeper).
//
//nolint:wrapcheck // reason: this package intentionally returns raw errors and errors are logged
package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/talx-hub/gophkeeper/internal/adapters/grpc/metadata"
	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/server/keeper"
	commonpb "github.com/talx-hub/gophkeeper/proto/v1/common"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
	metadatapb "github.com/talx-hub/gophkeeper/proto/v1/metadata"
)

const MsgAgentWrong = "agent error"
const MsgConversionFailed = "failed to convert ctx.Value(ContextKeyUserID) to (model.UserID)"
const MsgUserNotAuthenticated = "user not authenticated"
const MsgSyncFailed = "sync failed"
const KeyLoggerActualType = "actual_type"
const KeyLoggerUserID = "user_id"

// KeeperGRPCService реализует keeperpb.KeeperServer.
// Хэндлеры валидируют вход, извлекают userID из контекста,
// конвертируют типы и вызывают use-case. Ошибки отображаются в gRPC-коды.
type KeeperGRPCService struct {
	keeperpb.UnimplementedKeeperServer
	keeperUseCase KeeperUseCase
	log           *slog.Logger
}

// NewKeeperGRPCService создаёт экземпляр gRPC-сервиса Keeper.
// Параметры:
//   - log — логгер для диагностики;
//   - keeperUseCase — интерфейс прикладной логики (service/keeper).
func NewKeeperGRPCService(
	log *slog.Logger,
	keeperUseCase KeeperUseCase,
) *KeeperGRPCService {
	return &KeeperGRPCService{
		keeperUseCase: keeperUseCase,
		log:           log,
	}
}

// KeeperUseCase описывает операции прикладного уровня,
// которые вызываются из gRPC-слоя. Интерфейс позволяет
// прозрачно подключать разные стораджи (Postgres/S3)
// без изменения транспортного слоя.
type KeeperUseCase interface {
	// AddSealed добавляет объект (sealed bytes) с метаданными.
	AddSealed(ctx context.Context, userID model.UserID, meta *model.Metadata, sealed []byte) (model.DataID, error)

	// Delete удаляет объект и его метаданные.
	Delete(ctx context.Context, id model.DataID) error

	// GetSealed получает объект по id и отправляет его наружу через callback (возможен поток чанков).
	GetSealed(ctx context.Context, id model.DataID, callback keeper.StreamCallback) error

	// List возвращает список метаданных для секретов пользователя.
	List(ctx context.Context, userID model.UserID) ([]model.MetaLoc, error)

	// Sync синхронизирует данные пользователя (метаданные и/или payload) через callback.
	Sync(ctx context.Context, userID model.UserID, mode keeper.SyncMode, callback keeper.StreamCallback) error
}

// Add принимает клиентский стрим AddRequest и добавляет объекты.
func (s *KeeperGRPCService) Add(
	stream grpc.ClientStreamingServer[keeperpb.AddRequest, keeperpb.AddResponse],
) error {
	ctx := stream.Context()
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, MsgConversionFailed,
			KeyLoggerActualType, actualType)
		return status.Error(codes.Unauthenticated, MsgUserNotAuthenticated)
	}

	var requests []*keeperpb.AddRequest
loop:
	for {
		req, err := stream.Recv()
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			break loop
		case errors.Is(err, context.Canceled):
			s.log.InfoContext(ctx, "streaming was interrupted by agent",
				KeyLoggerUserID, userID, model.KeyLoggerError, err)
			return status.Error(codes.Canceled, err.Error())
		default:
			s.log.ErrorContext(ctx, "recv failed", KeyLoggerUserID, userID, model.KeyLoggerError, err)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}
		requests = append(requests, req)
	}

	for _, req := range requests {
		metaDTO := req.GetMetadata()
		if metaDTO == nil {
			s.log.ErrorContext(ctx, "bad metadata: metadata is empty", KeyLoggerUserID, userID)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}
		payload := req.GetPayload()
		if payload == nil {
			s.log.ErrorContext(ctx, "payload is nil", KeyLoggerUserID, userID)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}

		meta, err := metadata.FromProtoMetadata(metaDTO)
		if err != nil {
			s.log.ErrorContext(ctx,
				"bad metadata",
				KeyLoggerUserID, userID,
				model.KeyLoggerError, err,
			)
			return status.Errorf(codes.InvalidArgument, MsgAgentWrong)
		}
		sealedData := payload.GetSealedData()
		if sealedData == nil {
			s.log.ErrorContext(ctx, "sealedData and binaryChunk are empty",
				KeyLoggerUserID, userID,
			)
			return status.Error(codes.InvalidArgument, "data should be filled")
		}
		if _, err := s.keeperUseCase.AddSealed(ctx, userID, meta, sealedData); err != nil {
			s.log.ErrorContext(ctx, "failed to AddSealed",
				KeyLoggerUserID, userID,
				model.KeyLoggerError, err)
			return status.Error(codes.Internal, "add failed")
		}
	}

	return stream.SendAndClose(&keeperpb.AddResponse{})
}

// Delete удаляет объект и его метаданные по идентификатору.
// Требует валидного userID в контексте. При ошибке репозитория
// возвращает Internal.
func (s *KeeperGRPCService) Delete(ctx context.Context, req *keeperpb.DeleteRequest,
) (*keeperpb.DeleteResponse, error) {
	if req == nil {
		s.log.ErrorContext(ctx, "delete request is <nil>")
		return nil, status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, MsgConversionFailed,
			KeyLoggerActualType, actualType)
		return nil, status.Error(codes.Unauthenticated, MsgUserNotAuthenticated)
	}

	metaDTO := req.GetMetadata()
	if metaDTO == nil {
		s.log.ErrorContext(ctx, "bad metadata: metadata is empty")
		return nil, status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	err := s.keeperUseCase.Delete(ctx, model.DataID(metaDTO.GetId()))
	if err != nil {
		s.log.ErrorContext(ctx,
			"delete failed",
			KeyLoggerUserID, userID,
			"data_id", metaDTO.GetId(),
			model.KeyLoggerError, err)
		return nil, status.Error(codes.Internal, "delete failed")
	}

	return &keeperpb.DeleteResponse{}, nil
}

// Get отдаёт объект по идентификатору через серверный стрим.
// Use-case может вызывать callback несколько раз,
// хэндлер отправляет каждую порцию как отдельное gRPC-сообщение.
func (s *KeeperGRPCService) Get(
	req *keeperpb.GetRequest,
	stream grpc.ServerStreamingServer[keeperpb.GetResponse],
) error {
	ctx := stream.Context()
	if req == nil {
		s.log.ErrorContext(ctx, "get request is <nil>")
		return status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, MsgConversionFailed,
			KeyLoggerActualType, actualType)
		return status.Error(codes.Unauthenticated, MsgUserNotAuthenticated)
	}

	metaDTO := req.GetMetadata()
	if metaDTO == nil {
		s.log.ErrorContext(ctx, "bad metadata: metadata is empty")
		return status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	// запускаем use-case -- он питюкает в stream через коллбек
	err := s.keeperUseCase.GetSealed(ctx, model.DataID(metaDTO.GetId()),
		func(m *model.Metadata, sealed []byte) error {
			//nolint:wrapcheck // reason: error from callback
			return stream.Send(
				&keeperpb.GetResponse{
					Metadata: metadata.ToProtoMetadata(m),
					Payload:  &commonpb.Payload{SealedData: sealed},
				})
		})
	if err != nil {
		s.log.ErrorContext(ctx,
			MsgSyncFailed,
			KeyLoggerUserID, userID,
			"data_id", metaDTO.GetId(),
			model.KeyLoggerError, err)
		return status.Error(codes.Internal, MsgSyncFailed)
	}
	return nil
}

// List возвращает список метаданных (без payload) для текущего пользователя.
func (s *KeeperGRPCService) List(ctx context.Context, _ *keeperpb.ListRequest,
) (*keeperpb.ListResponse, error) {
	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok {
		s.log.ErrorContext(ctx,
			"failed to convert userID extracted from ctx to model.UserID",
			"real_type", reflect.TypeOf(userID).String(),
		)
		return nil, status.Error(codes.Unauthenticated, MsgAgentWrong)
	}

	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	metaLocs, err := s.keeperUseCase.List(ctxTO, userID)
	if err != nil {
		s.log.ErrorContext(ctxTO, "failed to list metadata from Repository",
			KeyLoggerUserID, userID,
			model.KeyLoggerError, err,
		)
		return nil, status.Error(codes.Internal, "server internal error")
	}

	listDTO := make([]*metadatapb.Metadata, len(metaLocs))
	for i := range metaLocs {
		listDTO[i] = metadata.ToProtoMetadata(&metaLocs[i].Meta)
	}

	return &keeperpb.ListResponse{
		Metadata: listDTO,
	}, nil
}

// Sync обрабатывает серверный стрим синхронизации.
// В зависимости от режима (SHORT/FULL) use-case формирует поток ответов,
// а хэндлер отправляет их в gRPC-стрим.
func (s *KeeperGRPCService) Sync(
	req *keeperpb.SyncRequest, stream grpc.ServerStreamingServer[keeperpb.SyncResponse],
) error {
	ctx := stream.Context()
	if req == nil {
		s.log.ErrorContext(ctx, "sync request is <nil>")
		return status.Error(codes.InvalidArgument, MsgAgentWrong)
	}

	userID, ok := ctx.Value(model.ContextKeyUserID).(model.UserID)
	if !ok || userID == "" {
		actualType := fmt.Sprintf("%T", ctx.Value(model.ContextKeyUserID))
		s.log.ErrorContext(ctx, MsgConversionFailed,
			KeyLoggerActualType, actualType)
		return status.Error(codes.Unauthenticated, MsgUserNotAuthenticated)
	}

	var mode keeper.SyncMode
	switch req.GetSyncMode() {
	case keeperpb.SyncRequest_SYNC_MODE_SHORT:
		mode = keeper.SyncModeShort
	case keeperpb.SyncRequest_SYNC_MODE_FULL:
		mode = keeper.SyncModeFull
	default:
		msg := "unknown sync mode"
		s.log.ErrorContext(ctx, msg)
		return status.Error(codes.InvalidArgument, msg)
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
			MsgSyncFailed,
			KeyLoggerUserID, userID,
			model.KeyLoggerError, err)
		return status.Error(codes.Internal, MsgSyncFailed)
	}
	return nil
}
