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
	OpenAIBaseURL       string
	OpenAIChatModel     string
	OpenAIEmbedModel    string
	OpenAIAPIKeyHeader  string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	FrontendOrigin      string
	// AllowDemoAuth enables POST /auth/demo-token (never enable in production).
	AllowDemoAuth bool
	DemoUserID    string
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
		OpenAIBaseURL:       strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		OpenAIChatModel:     strings.TrimSpace(os.Getenv("OPENAI_CHAT_MODEL")),
		OpenAIEmbedModel:    strings.TrimSpace(os.Getenv("OPENAI_EMBED_MODEL")),
		OpenAIAPIKeyHeader:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY_HEADER")),
		GoogleClientID:      strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:  strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:   strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")),
		FrontendOrigin:      strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN")),
	}
	if cfg.OpenAIChatModel == "" {
		cfg.OpenAIChatModel = "gpt-4o"
	}
	if cfg.OpenAIEmbedModel == "" {
		cfg.OpenAIEmbedModel = "text-embedding-3-small"
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

	if cfg.FrontendOrigin == "" {
		cfg.FrontendOrigin = "http://localhost:3000"
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("ALLOW_DEMO_AUTH"))) {
	case "1", "true", "yes", "on":
		cfg.AllowDemoAuth = true
	}
	cfg.DemoUserID = strings.TrimSpace(os.Getenv("DEMO_USER_ID"))
	if cfg.DemoUserID == "" {
		cfg.DemoUserID = "10000000-0000-0000-0000-000000000001"
	}

	return cfg, nil
}

