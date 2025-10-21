package api

import (
	"EffectiveMobile/internal/api/handlers"
	"EffectiveMobile/internal/api/middleware/logger"
	"EffectiveMobile/internal/repository"
	"EffectiveMobile/internal/service"
	"log/slog"
	"net/http"

	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(log *slog.Logger, serviceRepo *repository.ServiceRepository, subscriptionRepo *repository.SubscriptionRepository, statsRepo *repository.StatsRepository) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(10 * time.Second))

	router.Use(logger.New(log))

	subscriptionService := service.NewSubscriptionService(serviceRepo, subscriptionRepo, log)
	statsService := service.NewStatsService(statsRepo, log)

	fs := http.FileServer(http.Dir(".static/swagger"))
	router.Handle("/static/swagger/*", http.StripPrefix("/static/swagger", fs))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/static/swagger/swagger.json"),
	))

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/subscriptions", func(r chi.Router) {
			r.Mount("/", handlers.GetSubscriptionsRoutes(subscriptionService, statsService, log))
		})
		r.Mount("/stats", handlers.GetStatRoutes(statsService, log))
	})

	return router
}
