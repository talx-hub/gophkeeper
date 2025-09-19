package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/repo/db/internal/sqlc"
)

type TokenRepository struct {
	DB
}

func NewTokenRepository(pool *pgxpool.Pool, log *slog.Logger) *TokenRepository {
	return &TokenRepository{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func (r *TokenRepository) Save(ctx context.Context,
	tokenHash []byte, userID model.UserID, expiresAt time.Time,
) error {
	saveLogic := func() (struct{}, error) {
		queries := sqlc.New(r.pool)

		userUUID, err := ToPgUUID(userID)
		if err != nil {
			return struct{}{}, fmt.Errorf("wrong userID format: %w", err)
		}

		err = queries.Save(ctx,
			sqlc.SaveParams{
				UserID:    userUUID,
				TokenHash: tokenHash,
				ExpiresAt: pgtype.Timestamptz{
					Time:  expiresAt,
					Valid: true,
				},
			})
		if err != nil {
			return struct{}{}, fmt.Errorf("queries.Save failed: %w", err)
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](saveLogic, 0)
	if err != nil {
		//nolint:wrapcheck // reason: err from wrapped func
		return err
	}
	return nil
}

func (r *TokenRepository) Validate(ctx context.Context, tokenHash []byte, userID model.UserID) error {
	validateLogic := func() (struct{}, error) {
		queries := sqlc.New(r.pool)

		userUUID, err := ToPgUUID(userID)
		if err != nil {
			return struct{}{}, fmt.Errorf("wrong userID format: %w", err)
		}

		ok, err := queries.Validate(ctx,
			sqlc.ValidateParams{
				TokenHash: tokenHash,
				UserID:    userUUID,
			})
		if err != nil {
			return struct{}{}, fmt.Errorf("queries.Validate failed: %w", err)
		}
		if !ok {
			return struct{}{}, errors.New("invalid token")
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](validateLogic, 0)
	if err != nil {
		//nolint:wrapcheck // reason: err from wrapped func
		return err
	}
	return nil
}

func (r *TokenRepository) Delete(ctx context.Context, tokenHash []byte) error {
	deleteLogic := func() (struct{}, error) {
		queries := sqlc.New(r.pool)

		err := queries.DeleteToken(ctx, tokenHash)
		if err != nil {
			return struct{}{}, fmt.Errorf("queries.Validate failed: %w", err)
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](deleteLogic, 0)
	if err != nil {
		//nolint:wrapcheck // reason: err from wrapped func
		return err
	}
	return nil
}
