package api

//go:generate mockgen -destination=router_mock.go -source=router.go -package=api

import (
	"EffectiveMobile/internal/api/handlers"
	"EffectiveMobile/internal/api/middleware/logger"
	"EffectiveMobile/internal/service"
	"log/slog"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func NewRouter(log *slog.Logger, repo service.ServicesRepository, subscriptionRepo service.SubscriptionRepository, statsRepo service.StatsRepository) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Use(logger.New(log))

	subscriptionService := service.NewSubscriptionService(repo, subscriptionRepo, log)

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/subscriptions", func(r chi.Router) {
			r.Mount("/", handlers.GetSubscriptionsRoutes(log, subscriptionService))
		})
		statsService := service.NewStatsService(statsRepo, log)
		r.Mount("/stats", handlers.GetStatRoutes(log, statsService))
	})

	return router
}
