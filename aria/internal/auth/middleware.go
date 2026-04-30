package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/redis/go-redis/v9"
)

type ctxKey int

const claimsKey ctxKey = iota

func Authenticate(jwtSecret string, rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			tokenString := strings.TrimSpace(h[len("Bearer "):])
			if tokenString == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims, err := VerifyToken(tokenString, jwtSecret)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if rdb != nil {
				bl, berr := IsBlacklisted(r.Context(), rdb, claims.ID)
				if berr != nil {
					// Fail closed: if we cannot verify blacklist status, deny.
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
				if bl {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(claimsKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Claims)
	return c, ok
}

