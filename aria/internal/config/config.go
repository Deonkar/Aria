package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	DatabaseURLReadonly string
	RedisURL            string
	JWTSecret           string
	OpenAIAPIKey        string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	FrontendOrigin      string
}

func Load() (*Config, error) {
	// Best-effort load of .env in local development. In docker, env_file handles it.
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DatabaseURLReadonly: strings.TrimSpace(os.Getenv("DATABASE_URL_READONLY")),
		RedisURL:            strings.TrimSpace(os.Getenv("REDIS_URL")),
		JWTSecret:           strings.TrimSpace(os.Getenv("JWT_SECRET")),
		OpenAIAPIKey:        strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		GoogleClientID:      strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:  strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:   strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")),
		FrontendOrigin:      strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN")),
	}

	var missing []string
	require := func(envKey, val string) {
		if val == "" {
			missing = append(missing, envKey)
		}
	}

	require("DATABASE_URL", cfg.DatabaseURL)
	require("DATABASE_URL_READONLY", cfg.DatabaseURLReadonly)
	require("REDIS_URL", cfg.RedisURL)
	require("JWT_SECRET", cfg.JWTSecret)
	require("OPENAI_API_KEY", cfg.OpenAIAPIKey)
	require("GOOGLE_CLIENT_ID", cfg.GoogleClientID)
	require("GOOGLE_CLIENT_SECRET", cfg.GoogleClientSecret)
	require("GOOGLE_REDIRECT_URL", cfg.GoogleRedirectURL)

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

