package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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
	s := api.NewServer(address, log, nil)

	log.InfoContext(context.Background(),
		"starting gRPC server",
		"address", address,
	)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Start()
		if err != nil {
			log.ErrorContext(context.Background(),
				"failed to start server",
				"err", err)
		}
	}()
	wg.Wait()

	//log.InfoContext(context.Background(), "stopping gRPC server gracefully...")
	//log.InfoContext(context.Background(), "successful server graceful shutdown")

	return nil
}
