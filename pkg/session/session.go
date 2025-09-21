package session

import (
	"context"
	"fmt"
	"time"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/pkg/tokens"
)

const MDKeyAuthorisation = "authorization"
const AuthTokenPrefix = "Bearer "

type RefreshTokenStorage interface {
	Save(ctx context.Context, tokenHash []byte, userID model.UserID, expiresAt time.Time) error
	Validate(ctx context.Context, tokenHash []byte, userID model.UserID) error
	Delete(ctx context.Context, tokenHash []byte) error
}

type Manager struct {
	storage RefreshTokenStorage
	issuer  *tokens.Generator
}

func NewManager(storage RefreshTokenStorage, issuer *tokens.Generator) *Manager {
	return &Manager{
		storage: storage,
		issuer:  issuer,
	}
}

func (m *Manager) CreateSession(ctx context.Context, userID model.UserID,
) (string, []byte, error) {
	accessToken, err := m.issuer.GenerateAccessToken(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, expiresAt, err := m.issuer.GenerateRefreshToken(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	err = m.storage.Save(ctxTO, refreshToken, userID, expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (m *Manager) RefreshSession(ctx context.Context, userID model.UserID, oldRefreshToken []byte,
) (string, []byte, error) {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	if err := m.storage.Delete(ctxTO, oldRefreshToken); err != nil {
		return "", nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	accessToken, refreshToken, err := m.CreateSession(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create new session: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (m *Manager) ValidateAccessToken(ctx context.Context, token string) (model.UserID, error) {
	userID, err := m.issuer.CheckAccessToken(ctx, token)
	if err != nil {
		//nolint:wrapcheck // the ValidateAccessToken is just proxy for tokens.CheckAccessToken
		return "", err
	}
	return userID, nil
}

func (m *Manager) ValidateRefreshToken(ctx context.Context, token []byte, userID model.UserID) error {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	err := m.storage.Validate(ctxTO, token, userID)
	if err != nil {
		//nolint:wrapcheck // the ValidateRefreshToken is just proxy for storage.Validate
		return err
	}
	return nil
}

func (m *Manager) RevokeSession(ctx context.Context, refreshToken []byte) error {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()

	if err := m.storage.Delete(ctxTO, refreshToken); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}
