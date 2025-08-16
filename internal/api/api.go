package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	v1 "github.com/talx-hub/gophkeeper/internal/api/v1"
	"github.com/talx-hub/gophkeeper/internal/model/common"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
	healthpb "github.com/talx-hub/gophkeeper/proto/v1/health"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type Storage interface {
	Add(context.Context, common.Metadata, []byte) error
	Get(context.Context, common.Metadata) ([]byte, error)
	List(context.Context) ([]common.Metadata, error)
	Delete(context.Context, common.Metadata) error
}

type Server struct {
	grpcServer *grpc.Server
	log        *slog.Logger
	storage    Storage
	// cfg *server.Builder
	// storage Storage
	address string
}

func NewServer(address string, storage Storage) *Server {
	return &Server{
		address:    address,
		grpcServer: grpc.NewServer(grpc.ChainUnaryInterceptor()),
		log:        slog.Default(),
		storage:    storage,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf(
			"failed to start listening on address %s: %w", s.address, err)
	}

	authpb.RegisterAuthServiceServer(s.grpcServer, &v1.AuthService{})
	healthpb.RegisterHealthServiceServer(s.grpcServer, &v1.HealthService{})
	keeperpb.RegisterKeeperServer(s.grpcServer, &v1.KeeperService{})

	//nolint:wrapcheck // error could be nil
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)

		s.grpcServer.GracefulStop()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context closed: %w", ctx.Err())
	case <-done:
		return nil
	}
}
