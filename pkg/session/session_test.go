package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gophkeeper/pkg/tokens"
)

func TestManager_CreateSession(t *testing.T) {
	secret := []byte("test-secret")
	issuer := tokens.NewGenerator(secret)

	t.Run("success", func(t *testing.T) {
		st := newStorageMock(t).WithSave().Build()
		m := NewManager(st, issuer)

		access, refresh, err := m.CreateSession(context.Background(), "user-1")
		require.NoError(t, err)
		require.NotEmpty(t, access)
		require.NotEmpty(t, refresh)
	})

	t.Run("save fails", func(t *testing.T) {
		st := newStorageMock(t).WithSave().Build()
		m := NewManager(st, issuer)

		access, refresh, err := m.CreateSession(context.Background(), "save-fail")
		require.Error(t, err)
		require.Empty(t, access)
		require.Empty(t, refresh)
	})
}

func TestManager_RefreshSession(t *testing.T) {
	secret := []byte("test-secret")
	issuer := tokens.NewGenerator(secret)

	t.Run("success", func(t *testing.T) {
		st := newStorageMock(t).WithDelete().WithSave().Build()
		m := NewManager(st, issuer)

		access, refresh, err := m.RefreshSession(context.Background(), "user-1", "old-token")
		require.NoError(t, err)
		require.NotEmpty(t, access)
		require.NotEmpty(t, refresh)
	})

	t.Run("delete fails", func(t *testing.T) {
		st := newStorageMock(t).WithDelete().Build()
		m := NewManager(st, issuer)

		access, refresh, err := m.RefreshSession(context.Background(), "user-1", "delete-fail")
		require.Error(t, err)
		require.Empty(t, access)
		require.Empty(t, refresh)
		st.AssertNotCalled(t, "CreateSession", anyCtx(), anyUserID())
	})

	t.Run("create session fails", func(t *testing.T) {
		st := newStorageMock(t).WithDelete().WithSave().Build()
		m := NewManager(st, issuer)

		access, refresh, err := m.RefreshSession(context.Background(), "save-fail", "old-token")
		require.Error(t, err)
		require.Empty(t, access)
		require.Empty(t, refresh)
	})
}

func TestManager_ValidateAccessToken(t *testing.T) {
	secret := []byte("test-secret")
	issuer := tokens.NewGenerator(secret)
	m := NewManager(newStorageMock(t).Build(), issuer)

	t.Run("valid token", func(t *testing.T) {
		token, err := issuer.GenerateAccessToken(context.Background(), "user-123")
		require.NoError(t, err)
		uid, err := m.ValidateAccessToken(context.Background(), token)
		require.NoError(t, err)
		require.Equal(t, "user-123", uid)
	})

	t.Run("invalid token", func(t *testing.T) {
		uid, err := m.ValidateAccessToken(context.Background(), "bad.token.string")
		require.Error(t, err)
		require.Empty(t, uid)
	})
}

func TestManager_ValidateRefreshToken(t *testing.T) {
	issuer := tokens.NewGenerator([]byte("test-secret"))

	t.Run("success", func(t *testing.T) {
		st := newStorageMock(t).WithValidate().Build()
		m := NewManager(st, issuer)
		err := m.ValidateRefreshToken(context.Background(), "token-ok")
		require.NoError(t, err)
	})

	t.Run("validate fails", func(t *testing.T) {
		st := newStorageMock(t).WithValidate().Build()
		m := NewManager(st, issuer)
		err := m.ValidateRefreshToken(context.Background(), "validate-fail")
		require.Error(t, err)
	})
}

func TestManager_RevokeSession(t *testing.T) {
	issuer := tokens.NewGenerator([]byte("test-secret"))

	t.Run("success", func(t *testing.T) {
		st := newStorageMock(t).WithDelete().Build()
		m := NewManager(st, issuer)
		err := m.RevokeSession(context.Background(), "token-ok")
		require.NoError(t, err)
	})

	t.Run("delete fails", func(t *testing.T) {
		st := newStorageMock(t).WithDelete().Build()
		m := NewManager(st, issuer)
		err := m.RevokeSession(context.Background(), "delete-fail")
		require.Error(t, err)
	})
}

func anyCtx() interface{}    { return mock.Anything }
func anyUserID() interface{} { return mock.AnythingOfType("model.UserID") }
