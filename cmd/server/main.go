package main

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/talx-hub/gophkeeper/internal/api"
	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/server/dbmanager"
	"github.com/talx-hub/gophkeeper/pkg/config"
	sqlassets "github.com/talx-hub/gophkeeper/sql"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	log := slog.Default()
	cfg := config.NewBuilder(log).
		FromEnv().
		FromFlags().
		GetConfig()

	ctx := context.Background()

	log.InfoContext(ctx, "connecting to DB", "dsn", cfg.DatabaseURI)
	dbManager, err := dbConnect(ctx, cfg.DatabaseURI, log)
	if err != nil {
		msg := "failed to connect to DB"
		log.ErrorContext(ctx, msg, model.KeyLoggerError, err)
		return fmt.Errorf("%s %w", msg, err)
	}
	defer dbManager.Close()

	g := new(errgroup.Group)
	g.Go(func() error {
		err := s.Start()
		if err != nil {
			log.ErrorContext(context.Background(),
				"failed to start server",
				"err", err)
			return fmt.Errorf("failed to start server: %w", err)
	s := api.NewServer(cfg, dbManager, log)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}
	return nil

	// log.InfoContext(context.Background(), "stopping gRPC server gracefully...")
	// log.InfoContext(context.Background(), "successful server graceful shutdown")
}

func dbConnect(ctx context.Context, dsn string, log *slog.Logger,
) (*dbmanager.DBManager, error) {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	dbManager, err := dbmanager.FluentNew(dsn, sqlassets.Migrations, log).
		Connect(ctxTO).ApplyMigrations().Ping(ctxTO).Result()
	if err != nil {
		return nil, fmt.Errorf("DBManager init: %w", err)
	}

	return dbManager, nil
}
