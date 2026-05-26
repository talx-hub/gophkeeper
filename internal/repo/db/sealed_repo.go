package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/repo/db/internal/sqlc"
	"github.com/talx-hub/gophkeeper/pkg/hash"
)

type SealedRepo struct {
	DB
}

func NewObjectRepository(pool *pgxpool.Pool, log *slog.Logger) *SealedRepo {
	return &SealedRepo{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func putHelper(ctx context.Context,
	r io.Reader, queries *sqlc.Queries, idx, size int32, dataUUID pgtype.UUID, sha []byte,
) error {
	sealed := make([]byte, size)
	n, err := r.Read(sealed)
	if int32(len(sealed)) < size {
		return errors.New(
			"sealed size invariant violation: read from Reader return less than expected")
	}

	if errors.Is(err, io.EOF) {
		return io.EOF
	}

	sealed = sealed[:n]
	computed := hash.GenerateSHA256(sealed)
	if !bytes.Equal(computed, sha) {
		return errors.New("sha256 mismatch")
	}

	blobID, err := queries.InsertBlob(ctx,
		sqlc.InsertBlobParams{
			ObjectsID: dataUUID,
			Sealed:    sealed,
		})
	if err != nil {
		return fmt.Errorf("InsertBlob: %w", err)
	}

	err = queries.PutManifest(ctx,
		sqlc.PutManifestParams{
			ObjectsID:  dataUUID,
			BlobID:     blobID,
			ChunkIndex: idx,
			Length:     size,
		})
	if err != nil {
		return fmt.Errorf("PutManifest: %w", err)
	}
	return nil
}

func (s *SealedRepo) Put(ctx context.Context,
	meta *model.Metadata, r io.Reader, size int32, sha256 []byte,
) (model.ObjectLocator, error) {
	putLogic := func() (string, error) {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to start TX: %w", err)
		}
		defer func() {
			if rbErr := tx.Rollback(ctx); rbErr != nil &&
				!errors.Is(rbErr, pgx.ErrTxClosed) {
				s.log.ErrorContext(ctx,
					"rollback",
					model.KeyLoggerError,
					err)
			}
		}()

		queries := sqlc.New(tx)
		dataUUID, err := ToPgUUID(string(meta.ID))
		if err != nil {
			return "", fmt.Errorf("wrong dataID format: %w", err)
		}

		var idx int32
		for {
			err = putHelper(ctx, r, queries, idx, size, dataUUID, sha256)
			if errors.Is(err, io.EOF) {
				break
			}
			idx++
		}

		err = tx.Commit(ctx)
		if err != nil {
			return "", fmt.Errorf("commit: %w", err)
		}

		return dataUUID.String(), nil
	}

	uuid, err := WithRetry[string](putLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", err
	}

	return model.ObjectLocator("pg://" + uuid), nil
}

const getQuery = `SELECT
sb.sealed,
cm.length
FROM chunk_manifest AS cm
JOIN secret_blobs  AS sb
ON sb.id = cm.blob_id
WHERE cm.objects_id = (
SELECT id FROM objects
WHERE storage_locator = $1
)
ORDER BY cm.chunk_index;
`

func (s *SealedRepo) Get(ctx context.Context, loc model.ObjectLocator,
) (io.ReadCloser, error) {
	getLogic := func() (io.ReadCloser, error) {
		pr, pw := io.Pipe()

		go func() {
			defer func() {
				err := pw.Close()
				if err != nil {
					_ = pw.CloseWithError(
						fmt.Errorf("PipeWriter.Close: %w", err))
				}
			}()

			rows, err := s.pool.Query(ctx, getQuery, string(loc))
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("pgxpool.Query: %w", err))
			}
			defer rows.Close()

			for rows.Next() {
				var sealed []byte
				var length int32
				if err := rows.Scan(&sealed, &length); err != nil {
					_ = pw.CloseWithError(fmt.Errorf("rows.Scan: %w", err))
				}
				if len(sealed) != int(length) {
					_ = pw.CloseWithError(errors.New(
						"rows.Scan: data corrupted: length mismatch"))
				}
				if _, err := pw.Write(sealed); err != nil {
					_ = pw.CloseWithError(fmt.Errorf("PipeWriter: %w", err))
				}
			}
			if err := rows.Err(); err != nil {
				_ = pw.CloseWithError(fmt.Errorf("rows.Next(): %w", err))
				return
			}
		}()

		return pr, nil
	}

	readCloser, err := WithRetry[io.ReadCloser](getLogic, 0)
	if err != nil {
		//nolint // reason: err from wrapped func
		return nil, err
	}

	return readCloser, nil
}

func (s *SealedRepo) Delete(ctx context.Context, loc model.ObjectLocator) error {
	deleteLogic := func() (struct{}, error) {
		queries := sqlc.New(s.pool)

		err := queries.DeleteObject(ctx, string(loc))
		if err != nil {
			return struct{}{}, fmt.Errorf("queries.DeleteObject: %w", err)
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
