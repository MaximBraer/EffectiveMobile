package api

import (
	"EffectiveMobile/internal/api/handlers"
	"EffectiveMobile/internal/api/middleware/logger"
	"EffectiveMobile/internal/storage/postgres"
	"log/slog"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func NewRouter(log *slog.Logger, storage *postgres.Storage) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Use(logger.New(log))

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/subscriptions", func(r chi.Router) {
			r.Mount("/", handlers.GetSubscriptionsRoutes(log, storage))
		})
		r.Mount("/stats", handlers.GetStatRoutes(log, storage))
	})

	return router
}
