package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSignToken_Valid(t *testing.T) {
	secret := "test_secret"
	jti := "jti-1"
	token, err := SignToken("user-1", "a@example.com", "agent", jti, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	claims, err := VerifyToken(token, secret)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("expected sub user-1, got %q", claims.Subject)
	}
	if claims.ID != jti {
		t.Fatalf("expected jti %q, got %q", jti, claims.ID)
	}
	if claims.Email != "a@example.com" || claims.Role != "agent" {
		t.Fatalf("unexpected claims")
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	secret := "test_secret"
	token, err := SignToken("user-1", "a@example.com", "agent", "jti-1", secret, -1*time.Second)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	_, err = VerifyToken(token, secret)
	if err == nil {
		t.Fatalf("expected error for expired token")
	}
}

func TestVerifyToken_Tampered(t *testing.T) {
	secret := "test_secret"
	token, err := SignToken("user-1", "a@example.com", "agent", "jti-1", secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	token = token + "x"
	_, err = VerifyToken(token, secret)
	if err == nil {
		t.Fatalf("expected error for tampered token")
	}
}

func TestVerifyToken_WrongKey(t *testing.T) {
	token, err := SignToken("user-1", "a@example.com", "agent", "jti-1", "k1", 1*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	_, err = VerifyToken(token, "k2")
	if err == nil {
		t.Fatalf("expected error for wrong key")
	}
}

func TestBlacklist_BlocksToken(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	jti := "jti-x"
	if err := BlacklistToken(ctx, rdb, jti, 10*time.Second); err != nil {
		t.Fatalf("BlacklistToken: %v", err)
	}
	bl, err := IsBlacklisted(ctx, rdb, jti)
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if !bl {
		t.Fatalf("expected blacklisted")
	}
}

func TestBlacklist_ExpiredEntry(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	jti := "jti-y"
	if err := BlacklistToken(ctx, rdb, jti, 1*time.Second); err != nil {
		t.Fatalf("BlacklistToken: %v", err)
	}
	mr.FastForward(2 * time.Second)

	bl, err := IsBlacklisted(ctx, rdb, jti)
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if bl {
		t.Fatalf("expected not blacklisted after expiry")
	}
}

