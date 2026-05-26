package tokens

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/pkg/hash"
)

const AccessTokenExpire = 15 * time.Minute
const RefreshTokenExpireDays = 15

type Generator struct {
	secret []byte
}

func NewGenerator(secret []byte) *Generator {
	return &Generator{
		secret: secret,
	}
}

func (g *Generator) GenerateRefreshToken(_ context.Context,
) ([]byte, time.Time, error) {
	const tokenSize = 32
	random, err := hash.GenerateRandom(tokenSize)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf(
			"failed to generate token: %w", err)
	}

	tokenHash := hash.GenerateSHA256(random)
	return tokenHash,
		time.Now().UTC().AddDate(0, 0, RefreshTokenExpireDays),
		nil
}

func (g *Generator) GenerateAccessToken(_ context.Context, userID model.UserID) (string, error) {
	iat := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(iat.Add(AccessTokenExpire)),
			Subject:   string(userID),
		},
	)
	tokenString, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("fail JWT signing: %w", err)
	}
	return tokenString, nil
}

func (g *Generator) CheckAccessToken(_ context.Context, token string) (model.UserID, error) {
	const jwtLeeway = 10 * time.Second

	claims := &jwt.RegisteredClaims{}
	t, err := jwt.ParseWithClaims(
		token, claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("wrong signing algorithm")
			}
			return g.secret, nil
		},
		jwt.WithLeeway(jwtLeeway),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", jwt.ErrTokenExpired
		}
		return "", fmt.Errorf("invalid token: %w", err)
	}
	if !t.Valid {
		return "", errors.New("invalid token")
	}

	return model.UserID(claims.Subject), nil
}
