package main

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/talx-hub/gophkeeper/internal/api"
)

func main() {
	if err := run("localhost:9999"); err != nil {
		fmt.Println(err)
	}
}

func run(address string) error {
	log := slog.Default()
	// TODO: fill storage
	s := api.NewServer(address, log)

	log.InfoContext(context.Background(),
		"starting gRPC server",
		"address", address,
	)

	g := new(errgroup.Group)
	g.Go(func() error {
		err := s.Start()
		if err != nil {
			log.ErrorContext(context.Background(),
				"failed to start server",
				"err", err)
			return fmt.Errorf("failed to start server: %w", err)
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
