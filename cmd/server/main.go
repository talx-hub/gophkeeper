package main

import (
	"fmt"

	"github.com/talx-hub/gophkeeper/internal/api"
)

func main() {
	if err := run("localhost:9999"); err != nil {
		fmt.Println(err)
	}
}

func run(address string) error {
	s := api.NewServer(address, nil)
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}
