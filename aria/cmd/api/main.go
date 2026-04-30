package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/Deonkar/Aria/aria/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config invalid")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("database connection failed")
	}
	defer pool.Close()

	roPool, err := db.NewPool(ctx, cfg.DatabaseURLReadonly)
	if err != nil {
		log.Fatal().Err(err).Msg("readonly database connection failed")
	}
	defer roPool.Close()
	_ = roPool // wired in later phases (AI query execution)

	rdb, err := cache.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed")
	}
	defer func() {
		_ = rdb.Close()
	}()

	userRepo := db.NewUserRepo(pool)
	oauthCfg := auth.GoogleConfig(cfg)
	secureCookie := strings.HasPrefix(strings.ToLower(cfg.GoogleRedirectURL), "https://")

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(logging.RequestLogger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendOrigin},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	// Public auth routes
	r.Get("/auth/google", auth.HandleGoogleLogin(oauthCfg, rdb))
	r.Get("/auth/callback", auth.HandleGoogleCallback(oauthCfg, userRepo, rdb, cfg))
	r.Post("/auth/refresh", auth.HandleRefresh(userRepo, rdb, cfg))

	// Protected routes
	r.Group(func(pr chi.Router) {
		pr.Use(auth.Authenticate(cfg.JWTSecret, rdb))
		pr.Post("/auth/logout", auth.HandleLogout(rdb, secureCookie))
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("http server shutdown error")
		}
	}()

	log.Info().Str("addr", srv.Addr).Msg("http server starting")
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("http server failed")
	}
}

