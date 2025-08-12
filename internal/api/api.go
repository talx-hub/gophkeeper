package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	v1 "github.com/talx-hub/gophkeeper/internal/api/v1"
	"github.com/talx-hub/gophkeeper/proto/v1/auth"
	"github.com/talx-hub/gophkeeper/proto/v1/health"
	"github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type Server struct {
	grpcServer *grpc.Server
	m          sync.Mutex
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
	s.m.Lock()
	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor())
	s.m.Unlock()
	auth.RegisterAuthServiceServer(s.grpcServer, &v1.AuthService{})
	health.RegisterHealthServiceServer(s.grpcServer, &v1.HealthService{})
	keeper.RegisterKeeperServer(s.grpcServer, &v1.KeeperService{})

	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	s.m.Lock()
	if s.grpcServer == nil {
		s.m.Unlock()
		return errors.New("trying to close nil gRPC-server")
	}
	s.m.Unlock()

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
