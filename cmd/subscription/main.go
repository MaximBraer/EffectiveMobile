package main

import (
	"EffectiveMobile/internal/config"
	stats_total "EffectiveMobile/internal/http-server/handlers/stats/total"
	subscription_delete "EffectiveMobile/internal/http-server/handlers/subscription/delete"
	subscription_get "EffectiveMobile/internal/http-server/handlers/subscription/get"
	subscription_list "EffectiveMobile/internal/http-server/handlers/subscription/list"
	subscription_save "EffectiveMobile/internal/http-server/handlers/subscription/save"
	subscription_update "EffectiveMobile/internal/http-server/handlers/subscription/update"
	"EffectiveMobile/internal/http-server/middleware/logger"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	storage, err := postgres.New(context.Background(), cfg.Storage, log)
	if err != nil {
		log.Error("failed to init storage", slog.String("err", err.Error()))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/subscriptions", func(r chi.Router) {
			r.Post("/", subscription_save.New(log, storage))
			r.Get("/{id}", subscription_get.New(log, storage))
			r.Put("/{id}", subscription_update.New(log, storage))
			r.Delete("/{id}", subscription_delete.New(log, storage))
			r.Get("/", subscription_list.New(log, storage))
		})
		r.Get("/stats/total", stats_total.New(log, storage))
	})

	log.Info("starting server", slog.String("env", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", slog.String("err", err.Error()))
	}

	log.Error("stopping server", slog.String("env", cfg.Env))
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
