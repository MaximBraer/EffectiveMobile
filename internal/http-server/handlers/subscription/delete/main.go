package delete

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
)

func New(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.subscription.delete.New"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := r.URL.Path[len("/api/v1/subscriptions/"):]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid subscription id")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err = s.DeleteSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, storage.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, "subscription not found")
				return
			}
			reqLog.Error("delete subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
