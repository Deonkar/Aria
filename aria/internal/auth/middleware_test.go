package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMiddleware_NoHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	h := Authenticate("secret", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call next")
	}))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")

	h := Authenticate("secret", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call next")
	}))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	secret := "secret"
	tok, err := SignToken("u", "e", "agent", "jti", secret, -1*time.Second)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	h := Authenticate(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call next")
	}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_BlacklistedJTI(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	secret := "secret"
	tok, err := SignToken("u", "e", "agent", "jti-black", secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	if err := BlacklistToken(context.Background(), rdb, "jti-black", 1*time.Hour); err != nil {
		t.Fatalf("BlacklistToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	h := Authenticate(secret, rdb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call next")
	}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	secret := "secret"
	tok, err := SignToken("u1", "e1", "agent", "jti-ok", secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	called := false
	h := Authenticate(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims == nil {
			t.Fatalf("expected claims in context")
		}
		if claims.Subject != "u1" {
			t.Fatalf("unexpected sub: %q", claims.Subject)
		}
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected next called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestClaimsFromContext_Missing(t *testing.T) {
	claims, ok := ClaimsFromContext(context.Background())
	if ok || claims != nil {
		t.Fatalf("expected missing claims")
	}
}

