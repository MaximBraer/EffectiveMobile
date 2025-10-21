package main

import (
	"EffectiveMobile/internal/api"
	"EffectiveMobile/internal/config"
	"EffectiveMobile/internal/repository"
	"EffectiveMobile/pkg/postgres"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

// @title           Subscription API
// @version         1.0
// @description     This is a server subscription API.
// @contact.name    Maksim Braer
// @contact.email   braer.maks@gmail.com
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		slog.Error("failed to load config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	log := setupLogger(cfg.Env)

	provider := postgres.New(
		cfg.SQLDataBase.User,
		cfg.SQLDataBase.Password,
		cfg.SQLDataBase.DataBaseInfo,
		log,
	)

	if err := provider.Open(); err != nil {
		log.Error("failed to open provider", slog.String("err", err.Error()))
		os.Exit(1)
	}

	serviceRepo := repository.NewServiceRepository(provider, log)
	subscriptionRepo := repository.NewSubscriptionRepository(provider, log)
	statsRepo := repository.NewStatsRepository(provider, log)

	router := api.NewRouter(log, serviceRepo, subscriptionRepo, statsRepo)

	log.Info("starting server", slog.String("addr", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	log.Info("server started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	sig := <-quit
	log.Info("received shutdown signal", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Info("shutting down server...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", slog.String("err", err.Error()))
	}

	log.Info("closing database connections...")
	err = provider.Close()
	if err != nil {
		log.Error("failed to close database connections", slog.String("err", err.Error()))
	}

	log.Info("server stopped")
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
