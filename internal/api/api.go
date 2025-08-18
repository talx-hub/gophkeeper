package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	v1 "github.com/talx-hub/gophkeeper/internal/api/v1"
	"github.com/talx-hub/gophkeeper/pkg/session"
	"github.com/talx-hub/gophkeeper/pkg/tokens"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
	healthpb "github.com/talx-hub/gophkeeper/proto/v1/health"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type Server struct {
	grpcServer *grpc.Server
	log        *slog.Logger
	// storageManager StorageManager
	// cfg *server.Builder
	address string // TODO: fill address from cfg
}

func NewServer(address string, log *slog.Logger) *Server {
	return &Server{
		address:    address,
		grpcServer: grpc.NewServer(grpc.ChainUnaryInterceptor()),
		log:        log,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf(
			"failed to start listening on address %s: %w", s.address, err)
	}

	// TODO: fill repo
	// TODO: fill secret from cfg
	authpb.RegisterAuthServiceServer(s.grpcServer,
		v1.NewAuthService(
			s.log,
			nil,
			session.NewManager(nil, tokens.NewGenerator([]byte("TODO: secret"))),
		))

	healthpb.RegisterHealthServiceServer(s.grpcServer,
		v1.NewHealthService(
			s.log,
			nil,
		))

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
