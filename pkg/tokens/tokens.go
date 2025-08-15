package tokens

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const AccessTokenExpire = 15 * time.Minute
const RefreshTokenExpireDays = 15

var ErrTokenExpired = errors.New("token expired")

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type Generator struct {
	secret []byte
}

func (g *Generator) GenerateRefreshToken() (token string, expiresAt time.Time, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("uuid generation failed: %v\n%s", r, debug.Stack())
		}
	}()

	token = uuid.New().String() // may panic
	return token,
		time.Now().UTC().AddDate(0, 0, RefreshTokenExpireDays),
		nil
}

func (g *Generator) GenerateAccessToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(AccessTokenExpire)),
			},
			UserID: userID,
		},
	)
	tokenString, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("fail JWT signing: %w", err)
	}
	return tokenString, nil
}

func (g *Generator) CheckAccessToken(token string) (string, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(
		token, claims,
		func(token *jwt.Token) (interface{}, error) {
			return g.secret, nil
		})
	if err != nil {
		return "", fmt.Errorf("failed to parse token %w", err)
	}
	tokenExpired := claims.ExpiresAt.Before(time.Now().UTC())
	if tokenExpired {
		return "", ErrTokenExpired
	}

	return claims.UserID, nil
}
