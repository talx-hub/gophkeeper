package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	s := api.NewServer(cfg, dbManager, log)
	go func() {
		log.InfoContext(ctx, "starting gRPC server", "address", cfg.RunAddr)
		if err := s.Start(); err != nil {
			log.ErrorContext(ctx, "server", "err", err)
		}
	}()

	idleShutdown := make(chan struct{})
	defer close(idleShutdown)

	go func() {
		sigCh := make(chan os.Signal, 1)
		defer close(sigCh)

		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		<-sigCh
		idleShutdown <- struct{}{}
	}()

	<-idleShutdown

	const stopTO = time.Second * 5
	ctxTO, cancel := context.WithTimeout(ctx, stopTO)
	defer cancel()

	log.InfoContext(ctx, "stopping gRPC server gracefully...")
	err = s.Stop(ctxTO)
	if err != nil {
		return fmt.Errorf("gracefull shutdown server: %w", err)
	}

	log.InfoContext(ctx, "successful server graceful shutdown")
	return nil
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
