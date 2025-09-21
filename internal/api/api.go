package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
		grpcServer: nil,
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

	userRepo, tokenRepo, keeperRepo := prepareRepos(pool, s.log)
	sessionManager := session.NewManager(
		tokenRepo,
		tokens.NewGenerator([]byte(s.cfg.SecretKey)),
	)
	authInterceptor := NewAuthInterceptor(sessionManager, s.log)

	s.grpcServer = grpc.NewServer(grpc.ChainUnaryInterceptor(
		authInterceptor.Interceptor(),
	))
	healthpb.RegisterHealthServiceServer(s.grpcServer,
		v1.NewHealthService(
			s.log,
			pool,
		))
	authpb.RegisterAuthServiceServer(s.grpcServer,
		v1.NewAuthService(
			s.log,
			userRepo,
			sessionManager,
		))
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

type AuthInterceptor struct {
	log            *slog.Logger
	sessionManager v1.SessionManager
}

func NewAuthInterceptor(manager v1.SessionManager, log *slog.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		log:            log,
		sessionManager: manager,
	}
}

func (i *AuthInterceptor) Interceptor() grpc.UnaryServerInterceptor {
	interceptorFoo := func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			i.log.ErrorContext(ctx, "metadata not found in context")
			return nil, status.Error(codes.Unauthenticated, "authentication error")
		}

		accessTokens := md.Get(session.MDKeyAuthorisation)
		accessToken := strings.TrimPrefix(accessTokens[0], session.AuthTokenPrefix)

		userID, err := i.sessionManager.ValidateAccessToken(
			context.Background(), accessToken)
		if err != nil {
			if !errors.Is(err, jwt.ErrTokenExpired) {
				i.log.InfoContext(ctx, "invalid access token",
					"token", accessToken,
					model.KeyLoggerError, err,
				)
			}
			return nil, status.Error(codes.Unauthenticated, "authentication error")
		}

		userCtx := context.WithValue(ctx, model.ContextKeyUserID, userID)
		return handler(userCtx, req)
	}

	return interceptorFoo
}
