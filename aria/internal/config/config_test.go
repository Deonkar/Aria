package config

import (
	"os"
	"testing"
)

func setEnv(t *testing.T, key, val string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	if err := os.Setenv(key, val); err != nil {
		t.Fatalf("setenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	setEnv(t, "DATABASE_URL", "postgres://aria:aria@localhost:5432/aria?sslmode=disable")
	setEnv(t, "DATABASE_URL_READONLY", "postgres://aria_readonly:aria_ro@localhost:5432/aria?sslmode=disable")
	setEnv(t, "REDIS_URL", "redis://localhost:6379/0")
	setEnv(t, "JWT_SECRET", "test_secret")
	setEnv(t, "OPENAI_API_KEY", "test_key")
	setEnv(t, "GOOGLE_CLIENT_ID", "test_client_id")
	setEnv(t, "GOOGLE_CLIENT_SECRET", "test_client_secret")
	setEnv(t, "GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")
}

func TestConfig_LoadValid(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.JWTSecret != "test_secret" {
		t.Fatalf("unexpected JWTSecret: %q", cfg.JWTSecret)
	}
}

func TestConfig_MissingJWTSecret(t *testing.T) {
	setRequiredEnv(t)
	setEnv(t, "JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfig_MissingOpenAIKey(t *testing.T) {
	setRequiredEnv(t)
	setEnv(t, "OPENAI_API_KEY", "")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
}

