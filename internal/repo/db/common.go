package db

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func ToPgUUID(id string) (pgtype.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("uuid.Parse(): %w", err)
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func WithRetry[T any](dbQuery func() (T, error), counter int) (T, error) {
	res, err := dbQuery()
	if err == nil {
		return res, nil
	}

	var dummy T
	const maxAttemptCount = 3
	if counter >= maxAttemptCount {
		return dummy, fmt.Errorf("reatempt failed: %w", err)
	}
	if isRetryableError(err) {
		time.Sleep((time.Duration(counter*2 + 1)) * time.Second) // count: 0 1 2 -> seconds: 1 3 5
		return WithRetry[T](dbQuery, counter+1)
	}
	return dummy, fmt.Errorf("on attempt #%d error occured: %w", counter, err)
}

func isRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.CannotConnectNow,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			pgerrcode.TransactionResolutionUnknown:
			return true
		}
	}

	return false
}
