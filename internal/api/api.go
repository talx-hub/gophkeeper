package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

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
	storage    Storage
	// cfg *server.Builder
	// storage Storage
	// log *slog.Logger
	address string
	m       sync.Mutex
}

func NewServer(address string, storage Storage) *Server {
	return &Server{
		address: address,
		storage: storage,
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
	authpb.RegisterAuthServiceServer(s.grpcServer, &v1.AuthService{})
	healthpb.RegisterHealthServiceServer(s.grpcServer, &v1.HealthService{})
	keeperpb.RegisterKeeperServer(s.grpcServer, &v1.KeeperService{})

	//nolint:wrapcheck // error could be nil
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
