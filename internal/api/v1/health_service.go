package v1

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/internal/model"
	healthpb "github.com/talx-hub/gophkeeper/proto/v1/health"
)

type HealthRepository interface {
	Ping(context.Context) error
}

type HealthService struct {
	healthpb.UnimplementedHealthServiceServer
	log  *slog.Logger
	repo HealthRepository
}

func NewHealthService(log *slog.Logger, repo HealthRepository) *HealthService {
	return &HealthService{
		log:  log,
		repo: repo,
	}
}

func (h *HealthService) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()

	err := h.repo.Ping(ctxTO)
	if err != nil {
		h.log.ErrorContext(ctx, "repository ping failed", "err", err)
		return nil, status.Errorf(codes.Internal, "repository disconnected")
	}

	return &emptypb.Empty{}, nil
}
