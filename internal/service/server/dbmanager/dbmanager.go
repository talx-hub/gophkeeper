package dbmanager

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBManager struct {
	log          *slog.Logger
	migrationsFS fs.FS
	pool         *pgxpool.Pool
	dsn          string
}

func New(dsn string, migrationsFS fs.FS, log *slog.Logger) *DBManager {
	return &DBManager{
		log:          log,
		migrationsFS: migrationsFS,
		dsn:          dsn,
	}
}

type upCloser interface {
	Up() error
	Close() (error, error)
}

var newMigrator = func(src source.Driver, dsn string) (upCloser, error) {
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return nil, errors.New("failed to get a new migrate instance")
	}
	return m, nil
}

func (m *DBManager) ApplyMigrations() (err error) {
	srcDrv, err := iofs.New(m.migrationsFS, ".")
	if err != nil {
		return errors.New("failed to return an iofs driver")
	}

	migrations, err := newMigrator(srcDrv, m.dsn)
	defer func() {
		sourceErr, dbErr := migrations.Close()
		if sourceErr != nil {
			err = errors.Join(err,
				errors.New("failed to close migrations source"))
		}
		if dbErr != nil {
			err = errors.Join(err,
				errors.New("failed to close DB connection used by migrate"))
		}
	}()

	if err = migrations.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errors.New("failed to apply migrations to the DB")
	}
	return nil
}

func (m *DBManager) Close() {
	if m.pool == nil {
		return
	}

	m.pool.Close()
}

func (m *DBManager) Connect(ctx context.Context) error {
	cfg, err := pgxpool.ParseConfig(m.dsn)
	if err != nil {
		return errors.New("failed to parse DSN")
	}
	cfg.MinConns = 1
	cfg.MaxConns = 10
	cfg.ConnConfig.Tracer = &queryTracer{m.log}

	m.pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return errors.New("failed to parse DSN")
	}

	return nil
}

func (m *DBManager) GetPool() (*pgxpool.Pool, error) {
	if m.pool == nil {
		return nil, errors.New("pool is nil")
	}
	return m.pool, nil
}

func (m *DBManager) Ping(ctx context.Context) error {
	if m.pool == nil {
		return errors.New("pool is nil")
	}

	if err := m.pool.Ping(ctx); err != nil {
		return errors.New("failed to ping the DB")
	}

	return nil
}

type Fluent struct {
	m   *DBManager
	err error
}

func FluentNew(dsn string, migrationsFS fs.FS, log *slog.Logger) *Fluent {
	return &Fluent{
		m: New(dsn, migrationsFS, log),
	}
}

func (f *Fluent) Connect(ctx context.Context) *Fluent {
	if f.err == nil {
		f.err = f.m.Connect(ctx)
	}
	return f
}

func (f *Fluent) ApplyMigrations() *Fluent {
	if f.err == nil {
		f.err = f.m.ApplyMigrations()
	}
	return f
}

func (f *Fluent) Ping(ctx context.Context) *Fluent {
	if f.err == nil {
		f.err = f.m.Ping(ctx)
		f.m.log.InfoContext(ctx, "connection to DB succeeded")
	}
	return f
}

func (f *Fluent) Result() (*DBManager, error) {
	return f.m, f.err
}
