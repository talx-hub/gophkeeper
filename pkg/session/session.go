package session

import (
	"context"
	"fmt"

	"github.com/talx-hub/gophkeeper/internal/model"
)

type TokenStorage interface {
	Add(ctx context.Context, token string) (string, error)
	Get(ctx context.Context, uuid string) error
	Delete(ctx context.Context, uuid string) error
}

type Manager struct {
	tokenStorage TokenStorage
}

func (m *Manager) GenerateTokens(ctx context.Context) (string, string, error) {
	return "", "", nil
}

func (m *Manager) GenerateRefreshToken(ctx context.Context) (string, error) {
	return "", nil
}

func (m *Manager) GenerateAccessToken() (string, error) {
	return "", nil
}

func (m *Manager) CheckRefreshToken(token string) error {
	return nil
}

func (m *Manager) CheckAccessToken(token string) error {
	return nil
}

func (m *Manager) Logout(ctx context.Context, refreshToken string) error {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()

	if err := m.tokenStorage.Delete(ctxTO, refreshToken); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}
