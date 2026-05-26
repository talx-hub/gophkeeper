package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/repo/db/internal/sqlc"
)

const msgWrongUserID = "wrong userID format: %w"

type UserRepository struct {
	DB
}

//goland:noinspection GoUnusedExportedFunction
func NewUserRepository(pool *pgxpool.Pool, log *slog.Logger) *UserRepository {
	return &UserRepository{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) (model.UserID, error) {
	createLogic := func() (model.UserID, error) {
		queries := sqlc.New(r.pool)

		userID, err := queries.Create(ctx,
			sqlc.CreateParams{
				LoginHash:   u.LoginHash,
				PasswordPhc: u.PasswordPHC,
			})
		if err != nil {
			return "", fmt.Errorf("queries.Create failed: %w", err)
		}
		return model.UserID(userID.String()), nil
	}

	userID, err := WithRetry[model.UserID](createLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", err
	}
	return userID, nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, loginHash []byte) (model.UserID, model.User, error) {
	type userProxy struct {
		id   model.UserID
		user model.User
	}

	findLogic := func() (userProxy, error) {
		queries := sqlc.New(r.pool)

		data, err := queries.FindByLogin(ctx, loginHash)
		if err != nil {
			return userProxy{}, fmt.Errorf("queries.FindByLogin failed: %w", err)
		}

		return userProxy{
				id:   model.UserID(data.ID.String()),
				user: model.User{PasswordPHC: data.PasswordPhc},
			},
			nil
	}

	proxy, err := WithRetry[userProxy](findLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", model.User{}, err
	}
	return proxy.id, proxy.user, nil
}

func (r *UserRepository) Delete(ctx context.Context, uuid model.UserID) error {
	deleteLogic := func() (struct{}, error) {
		queries := sqlc.New(r.pool)

		pgID, err := ToPgUUID(string(uuid))
		if err != nil {
			return struct{}{}, fmt.Errorf("wrong userID format: %w", err)
		}

		err = queries.DeleteUser(ctx, pgID)
		if err != nil {
			return struct{}{}, fmt.Errorf("queries.Delete failed: %w", err)
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](deleteLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return err
	}
	return nil
}
