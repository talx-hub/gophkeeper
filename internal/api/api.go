package api

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/talx-hub/gophkeeper/internal/api/v1"
	"github.com/talx-hub/gophkeeper/proto/v1/auth"
	"github.com/talx-hub/gophkeeper/proto/v1/health"
	"github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type Server struct {
	grpcServer *grpc.Server
	// cfg *server.Builder
	// storage Storage
	// log *slog.Logger
	address string
}

func NewServer(address string) *Server {
	return &Server{
		address: address,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		errMsg := "failed to start listening " + s.address
		// TODO: s.log.Fatal().Err(err).Msg(errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor())
	auth.RegisterAuthServiceServer(s.grpcServer, &v1.AuthService{})
	health.RegisterHealthServiceServer(s.grpcServer, &v1.HealthService{})
	keeper.RegisterKeeperServer(s.grpcServer, &v1.KeeperService{})

	errCh := make(chan error)
	defer close(errCh)

	go func() {
		errCh <- s.grpcServer.Serve(lis)
	}()

	return <-errCh
}

func (s *Server) Stop(ctx context.Context) error {
	done := make(chan struct{})
	defer close(done)

	go func() {
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
