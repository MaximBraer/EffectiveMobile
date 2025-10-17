package get

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/middleware"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	ID          int64  `json:"id"`
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func New(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.subscription.get.New"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := r.URL.Path[len("/api/v1/subscriptions/"):]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			response.WriteError(w, http.StatusBadRequest, "invalid subscription id")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		subscription, err := s.GetSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, storage.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, "subscription not found")
				return
			}
			reqLog.Error("get subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Response{
			ID:          subscription.ID,
			ServiceName: subscription.ServiceName,
			Price:       subscription.Price,
			UserID:      subscription.UserID.String(),
			StartDate:   subscription.StartDate.Format("01-2006"),
			EndDate:     formatEndDate(subscription.EndDate),
		})
	}
}

func formatEndDate(endDate *time.Time) *string {
	if endDate == nil {
		return nil
	}
	formatted := endDate.Format("01-2006")
	return &formatted
}

