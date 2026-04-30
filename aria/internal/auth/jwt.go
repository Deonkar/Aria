package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func SignToken(userID, email, role, jti, secret string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return s, nil
}

func VerifyToken(tokenString, secret string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	tok, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func BlacklistToken(ctx context.Context, rdb *redis.Client, jti string, remainingTTL time.Duration) error {
	if jti == "" {
		return errors.New("missing jti")
	}
	if remainingTTL <= 0 {
		// No point storing an already-expired token.
		return nil
	}
	key := fmt.Sprintf("jwt_blacklist:%s", jti)
	if err := rdb.Set(ctx, key, "1", remainingTTL).Err(); err != nil {
		return fmt.Errorf("redis set blacklist: %w", err)
	}
	return nil
}

func IsBlacklisted(ctx context.Context, rdb *redis.Client, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	key := fmt.Sprintf("jwt_blacklist:%s", jti)
	n, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists blacklist: %w", err)
	}
	return n > 0, nil
}

