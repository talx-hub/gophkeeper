package main

import (
	"fmt"

	"github.com/talx-hub/gophkeeper/internal/api"
)

func main() {
	s := api.NewServer("localhost:9999")
	if err := s.Start(); err != nil {
		fmt.Println(err)
	}
}
