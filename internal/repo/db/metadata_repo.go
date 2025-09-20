package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/repo/db/internal/sqlc"
)

type MetadataRepository struct {
	DB
}

//goland:noinspection GoUnusedExportedFunction
func NewMetadataRepository(pool *pgxpool.Pool, log *slog.Logger,
) *MetadataRepository {
	return &MetadataRepository{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func (r MetadataRepository) Put(
	ctx context.Context, meta *model.Metadata, loc model.ObjectLocator,
) (model.DataID, error) {
	putLogic := func() (model.DataID, error) {
		queries := sqlc.New(r.pool)
		userUUID, err := ToPgUUID(string(meta.UserID))
		if err != nil {
			return "", fmt.Errorf(msgWrongUserID, err)
		}

		dataID, err := queries.PutMetadata(ctx,
			sqlc.PutMetadataParams{
				UserID:         userUUID,
				ObjectName:     meta.Name,
				DataTypeName:   string(meta.DataType),
				Description:    pgtype.Text{String: meta.Description},
				StorageLocator: string(loc),
			},
		)
		if err != nil {
			return "", fmt.Errorf("queries.PutMetadata failed: %w", err)
		}
		return model.DataID(dataID.String()), nil
	}

	dataID, err := WithRetry[model.DataID](putLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", err
	}
	return dataID, nil
}

func (r MetadataRepository) Get(ctx context.Context, id model.DataID,
) (model.Metadata, model.ObjectLocator, error) {
	getLogic := func() (sqlc.GetMetadataRow, error) {
		queries := sqlc.New(r.pool)
		dataUUID, err := ToPgUUID(string(id))
		if err != nil {
			return sqlc.GetMetadataRow{}, fmt.Errorf("wrong dataID format: %w", err)
		}

		row, err := queries.GetMetadata(ctx, dataUUID)
		if err != nil {
			return sqlc.GetMetadataRow{},
				fmt.Errorf("queries.GetMetadata failed: %w", err)
		}

		return row, nil
	}

	row, err := WithRetry[sqlc.GetMetadataRow](getLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return model.Metadata{}, "", err
	}

	return model.Metadata{
			CreatedAt:   row.CreatedAt.Time,
			Description: row.Description.String,
			Name:        row.ObjectName,
			DataType:    model.DataType(row.DataTypeName),
			ID:          model.DataID(row.ID.String()),
		},
		model.ObjectLocator(row.StorageLocator),
		nil
}

func (r MetadataRepository) ListByUser(ctx context.Context, userID model.UserID,
) ([]model.MetaLoc, error) {
	listLogic := func() ([]sqlc.ListByUserRow, error) {
		queries := sqlc.New(r.pool)
		userUUID, err := ToPgUUID(string(userID))
		if err != nil {
			return nil, fmt.Errorf(msgWrongUserID, err)
		}

		rows, err := queries.ListByUser(ctx, userUUID)
		if err != nil {
			return nil, fmt.Errorf("queries.ListByUser failed: %w", err)
		}

		return rows, nil
	}

	rows, err := WithRetry[[]sqlc.ListByUserRow](listLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return nil, err
	}

	res := make([]model.MetaLoc, len(rows))
	for i := range rows {
		row := &rows[i]
		res[i] = model.MetaLoc{
			Locator: model.ObjectLocator(row.StorageLocator),
			Meta: model.Metadata{
				CreatedAt:   row.CreatedAt.Time,
				Description: row.Description.String,
				Name:        row.ObjectName,
				DataType:    model.DataType(row.DataTypeName),
				ID:          model.DataID(row.ID.String()),
			},
		}
	}
	return res, nil
}

func (r MetadataRepository) Delete(ctx context.Context, id model.DataID,
) (model.ObjectLocator, error) {
	deleteLogic := func() (model.ObjectLocator, error) {
		queries := sqlc.New(r.pool)
		dataUUID, err := ToPgUUID(string(id))
		if err != nil {
			return "", fmt.Errorf("wrong dataID format: %w", err)
		}

		loc, err := queries.DeleteMetadata(ctx, dataUUID)
		if err != nil {
			return "", fmt.Errorf("queries.Delete failed: %w", err)
		}
		return model.ObjectLocator(loc), nil
	}

	loc, err := WithRetry[model.ObjectLocator](deleteLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", err
	}
	return loc, nil
}
