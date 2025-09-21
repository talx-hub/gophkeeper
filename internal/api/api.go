package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	v1 "github.com/talx-hub/gophkeeper/internal/api/v1"
	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/repo/db"
	"github.com/talx-hub/gophkeeper/internal/repo/router"
	"github.com/talx-hub/gophkeeper/internal/service/server/keeper"
	"github.com/talx-hub/gophkeeper/pkg/config"
	"github.com/talx-hub/gophkeeper/pkg/session"
	"github.com/talx-hub/gophkeeper/pkg/tokens"
	authpb "github.com/talx-hub/gophkeeper/proto/v1/auth"
	healthpb "github.com/talx-hub/gophkeeper/proto/v1/health"
	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type DBManager interface {
	GetPool() (*pgxpool.Pool, error)
}

type Server struct {
	cfg        *config.Config
	dbManager  DBManager
	grpcServer *grpc.Server
	log        *slog.Logger
}

func NewServer(cfg *config.Config, dbManager DBManager, log *slog.Logger) *Server {
	return &Server{
		cfg:        cfg,
		dbManager:  dbManager,
		grpcServer: grpc.NewServer(grpc.ChainUnaryInterceptor()),
		log:        log,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.cfg.RunAddr)
	if err != nil {
		return fmt.Errorf(
			"failed to start listening on address %s: %w", s.cfg.RunAddr, err)
	}

	pool, err := s.dbManager.GetPool()
	if err != nil {
		msg := "get pgxpool.Pool"
		s.log.ErrorContext(context.Background(), msg, model.KeyLoggerError, err)
		return fmt.Errorf("%s: %w", msg, err)
	}

	healthpb.RegisterHealthServiceServer(s.grpcServer,
		v1.NewHealthService(
			s.log,
			pool,
		))

	userRepo, tokenRepo, keeperRepo := prepareRepos(pool, s.log)
	authpb.RegisterAuthServiceServer(s.grpcServer,
		v1.NewAuthService(s.log, userRepo,
			session.NewManager(
				tokenRepo,
				tokens.NewGenerator([]byte(s.cfg.SecretKey)),
			),
		),
	)

	keeperpb.RegisterKeeperServer(s.grpcServer,
		v1.NewKeeperGRPCService(s.log, keeperRepo))

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

func prepareRepos(pool *pgxpool.Pool, log *slog.Logger) (
	v1.UserRepository,
	session.RefreshTokenStorage,
	v1.KeeperUseCase,
) {
	userRepo := db.NewUserRepository(pool, log)
	tokenRepo := db.NewTokenRepository(pool, log)
	metadataRepo := db.NewMetadataRepository(pool, log)
	objectsStorage := db.NewObjectRepository(pool, log)

	//nolint:exhaustive // reason: missing UnspecifiedKey -- is OK
	reposByType := map[model.DataType]keeper.ObjectRepo{
		model.DataTypeBinary:                    objectsStorage,
		model.DataTypeCard:                      objectsStorage,
		model.DataTypeAuthenticationCredentials: objectsStorage,
	}
	objectsRouter := router.New(reposByType)

	useCase := keeper.NewService(objectsRouter, metadataRepo)
	return userRepo, tokenRepo, useCase
}
