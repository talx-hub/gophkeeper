package session

import (
	"context"
	"fmt"
	"time"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/pkg/tokens"
)

type RefreshTokenStorage interface {
	Save(ctx context.Context, tokenID, userID string, expiresAt time.Time) error
	Validate(ctx context.Context, tokenID string) error
	Delete(ctx context.Context, tokenID string) error
}

type Manager struct {
	storage   RefreshTokenStorage
	generator *tokens.Generator
}

func (m *Manager) CreateSession(ctx context.Context, userID string,
) (string, string, error) {
	accessToken, err := m.generator.GenerateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, expiresAt, err := m.generator.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	err = m.storage.Save(ctxTO, refreshToken, userID, expiresAt)
	if err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (m *Manager) RefreshSession(ctx context.Context, refreshToken string,
) (string, string, error) {
	return "", "", nil
}

func (m *Manager) ValidateAccessToken(token string) (string, error) {
	userID, err := m.generator.CheckAccessToken(token)
	if err != nil {
		//nolint:wrapcheck // the ValidateAccessToken is just proxy for tokens.CheckAccessToken
		return "", err
	}
	return userID, nil
}

func (m *Manager) ValidateRefreshToken(ctx context.Context, token string) error {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	err := m.storage.Validate(ctxTO, token)
	if err != nil {
		//nolint:wrapcheck // the ValidateRefreshToken is just proxy for storage.Validate
		return err
	}
	return nil
}

func (m *Manager) RevokeSession(ctx context.Context, refreshToken string,
) error {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()

	if err := m.storage.Delete(ctxTO, refreshToken); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}
