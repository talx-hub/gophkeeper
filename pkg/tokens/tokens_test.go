package tokens

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gophkeeper/internal/model"
)

func TestNewGenerator(t *testing.T) {
	secret := []byte("secret")
	g := NewGenerator(secret)
	require.NotNil(t, g)
	require.Equal(t, secret, g.secret)
}

func TestGenerator_GenerateRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		g := NewGenerator([]byte("secret"))
		token, exp, err := g.GenerateRefreshToken(context.Background())
		require.NoError(t, err)
		require.NotEmpty(t, token)
		require.WithinDuration(t, time.Now().AddDate(0, 0, RefreshTokenExpireDays), exp, time.Second*2)
	})
}

func TestGenerator_GenerateAccessToken(t *testing.T) {
	g := NewGenerator([]byte("secret"))
	token, err := g.GenerateAccessToken(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestGenerator_CheckAccessToken(t *testing.T) {
	secret := []byte("secret")
	g := NewGenerator(secret)

	t.Run("valid token", func(t *testing.T) {
		tokenStr, err := g.GenerateAccessToken(context.Background(), "user-123")
		require.NoError(t, err)
		subj, err := g.CheckAccessToken(context.Background(), tokenStr)
		require.NoError(t, err)
		require.Equal(t, model.UserID("user-123"), subj)
	})

	t.Run("wrong signing algorithm", func(t *testing.T) {
		claims := jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
			Subject:   "user-1",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
		tokenStr, err := token.SignedString(secret)
		require.NoError(t, err)
		subj, err := g.CheckAccessToken(context.Background(), tokenStr)
		require.Error(t, err)
		require.Empty(t, subj)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			Subject:   "user-expired",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(secret)
		require.NoError(t, err)
		subj, err := g.CheckAccessToken(context.Background(), tokenStr)
		require.ErrorIs(t, err, jwt.ErrTokenExpired)
		require.Empty(t, subj)
	})

	t.Run("invalid token format", func(t *testing.T) {
		subj, err := g.CheckAccessToken(context.Background(), "invalid.token.string")
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "invalid token"))
		require.Empty(t, subj)
	})

	t.Run("token invalid after parse", func(t *testing.T) {
		// sign with wrong secret to make it invalid
		tokenStr, err := NewGenerator([]byte("wrong")).GenerateAccessToken(context.Background(), "user-x")
		require.NoError(t, err)
		subj, err := g.CheckAccessToken(context.Background(), tokenStr)
		require.Error(t, err)
		require.Empty(t, subj)
	})
}
