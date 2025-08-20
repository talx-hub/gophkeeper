package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/talx-hub/gophkeeper/internal/model"
)

type DataRepo interface {
	AddSealed(ctx context.Context, userID model.UserID, meta *model.Metadata, sealed []byte) (model.DataID, error)
	GetSealed(ctx context.Context, userID model.UserID, id model.DataID) (*model.Metadata, []byte, error)
	ListMeta(ctx context.Context, userID model.UserID) ([]model.Metadata, error)
	Delete(ctx context.Context, userID model.UserID, id model.DataID) error
}

type SyncMode int

const (
	SyncModeShort SyncMode = iota + 1 // Только данные из БД
	SyncModeFull                      // Данные из БД + S3
)

type GRPCStreamSenderCB func(meta *model.Metadata, sealed []byte) error

type Service struct {
	repo DataRepo
}

func NewService(r DataRepo) *Service {
	return &Service{
		repo: r,
	}
}

func (s *Service) AddSealed(ctx context.Context,
	userID model.UserID,
	meta *model.Metadata,
	sealed []byte,
) (model.DataID, error) {
	if userID == "" {
		return 0, errors.New("empty userID")
	}
	if meta == nil {
		return 0, errors.New("nil metadata")
	}
	if len(sealed) == 0 {
		return 0, errors.New("empty sealed_data")
	}
	switch meta.DataType {
	case model.DataTypeAuthenticationCredentials, model.DataTypeCard, model.DataTypeBinary:
	default:
		return 0, fmt.Errorf("unsupported data type: %v", meta.DataType)
	}
	meta.UserID = userID
	return s.repo.AddSealed(ctx, userID, meta, sealed)
}

func (s *Service) GetSealed(ctx context.Context,
	userID model.UserID,
	id model.DataID,
	cb GRPCStreamSenderCB,
) error {
	if userID == "" {
		return errors.New("empty userID")
	}
	if id == 0 {
		return errors.New("empty id")
	}
	meta, sealed, err := s.repo.GetSealed(ctx, userID, id)
	if err != nil {
		return err
	}
	return cb(meta, sealed)
}

func (s *Service) List(ctx context.Context, userID model.UserID,
) ([]model.Metadata, error) {
	if userID == "" {
		return nil, errors.New("empty userID")
	}
	return s.repo.ListMeta(ctx, userID)
}

func (s *Service) Delete(ctx context.Context,
	userID model.UserID,
	id model.DataID,
) error {
	if userID == "" {
		return errors.New("empty userID")
	}
	if id == 0 {
		return errors.New("empty id")
	}
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) Sync(ctx context.Context,
	userID model.UserID,
	mode SyncMode,
	cb GRPCStreamSenderCB,
) error {
	if userID == "" {
		return errors.New("empty userID")
	}

	metas, err := s.repo.ListMeta(ctx, userID)
	if err != nil {
		return fmt.Errorf("repo.ListMetadata method failed: %w", err)
	}

	for i := range metas {
		select {
		case <-ctx.Done():
			return fmt.Errorf("metadata run through failed: %w", ctx.Err())
		default:
		}
		m := metas[i]

		switch mode {
		case SyncModeShort:
			if err := cb(&m, nil); err != nil {
				return fmt.Errorf(
					"metadata grpc stream write callback failed: %w", err)
			}
		case SyncModeFull:
			mm, sealed, err := s.repo.GetSealed(ctx, userID, m.ID)
			if err != nil {
				return fmt.Errorf("repo.GetSealed method failed: %w", err)
			}
			if err := cb(mm, sealed); err != nil {
				return fmt.Errorf(
					"metadata and data grpc stream write callback failed: %w", err)
			}
		default:
			return errors.New("unknown sync mode")
		}
	}

	return nil
}
