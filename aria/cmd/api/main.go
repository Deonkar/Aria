package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ariav1 "github.com/Deonkar/Aria/aria/gen/aria/v1"
	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/Deonkar/Aria/aria/internal/db"
	grpcserver "github.com/Deonkar/Aria/aria/internal/grpc"
	"github.com/Deonkar/Aria/aria/internal/handlers"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/Deonkar/Aria/aria/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

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

	rdb, err := cache.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed")
	}
	defer func() {
		_ = rdb.Close()
	}()

	userRepo := db.NewUserRepo(pool)
	convRepo := db.NewConversationRepo(pool)
	oai := ai.NewOpenAIClient(cfg)
	querySvc := ai.NewQueryService(cfg, oai, roPool, rdb)

	oauthCfg := auth.GoogleConfig(cfg)
	secureCookie := strings.HasPrefix(strings.ToLower(cfg.GoogleRedirectURL), "https://")

	chatH := &handlers.ChatHandler{
		UserRepo: userRepo,
		ConvRepo: convRepo,
		QuerySvc: querySvc,
		RDB:      rdb,
	}
	convH := &handlers.ConversationsHandler{ConvRepo: convRepo, RDB: rdb}
	feedbackH := &handlers.FeedbackHandler{
		Pool:       pool,
		ConvRepo:   convRepo,
		RDB:        rdb,
		OA:         oai,
		EmbedModel: cfg.OpenAIEmbedModel,
	}
	adminH := &handlers.AdminHandler{Pool: pool}
	queryH := &handlers.QueryHandler{UserRepo: userRepo, QuerySvc: querySvc}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcserver.RecoveryUnaryInterceptor, grpcserver.LoggingUnaryInterceptor),
		grpc.ChainStreamInterceptor(grpcserver.LoggingStreamInterceptor),
	)
	ariav1.RegisterAIQueryServiceServer(grpcSrv, grpcserver.NewServer(querySvc))
	reflection.Register(grpcSrv)
	grpcLis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal().Err(err).Msg("grpc listen failed")
	}

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

	r.Get("/auth/google", auth.HandleGoogleLogin(oauthCfg, rdb))
	r.Get("/auth/callback", auth.HandleGoogleCallback(oauthCfg, userRepo, rdb, cfg))
	r.Post("/auth/refresh", auth.HandleRefresh(userRepo, rdb, cfg))
	r.Post("/auth/demo-token", auth.HandleDemoToken(userRepo, cfg))

	r.Group(func(pr chi.Router) {
		pr.Use(auth.Authenticate(cfg.JWTSecret, rdb))
		pr.Post("/auth/logout", auth.HandleLogout(rdb, secureCookie))

		pr.Post("/chat", chatH.ServeHTTP)
		pr.Post("/query", queryH.ServeHTTP)

		pr.Get("/conversations", convH.List)
		pr.Get("/conversations/{id}/messages", convH.ListMessages)
		pr.Delete("/conversations/{id}", convH.Delete)

		pr.Post("/messages/{id}/feedback", feedbackH.ServeHTTP)
		pr.Get("/admin/metrics", adminH.Metrics)
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
		go grpcSrv.GracefulStop()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("http server shutdown error")
		}
	}()

	go func() {
		log.Info().Str("addr", ":9090").Msg("grpc server starting")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatal().Err(err).Msg("grpc server failed")
		}
	}()

	log.Info().Str("addr", srv.Addr).Msg("http server starting")
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("http server failed")
	}
}
