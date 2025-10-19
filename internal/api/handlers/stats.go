package handlers

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

type Response struct {
	TotalCost          int     `json:"total_cost"`
	Period             Period  `json:"period"`
	Filters            Filters `json:"filters"`
	SubscriptionsCount int     `json:"subscriptions_count"`
}

type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Filters struct {
	UserID      *string `json:"user_id,omitempty"`
	ServiceName *string `json:"service_name,omitempty"`
}

func GetTotalStats(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.stats.GetTotalStats"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userID *uuid.UUID
		var serviceName *string
		var startDate, endDate *time.Time

		if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
			if id, err := uuid.Parse(userIDStr); err == nil {
				userID = &id
			}
		}

		if serviceNameStr := r.URL.Query().Get("service_name"); serviceNameStr != "" {
			serviceName = &serviceNameStr
		}

		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			if date, err := parseMonth(startDateStr); err == nil {
				startDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, "invalid start_date format")
				return
			}
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			if date, err := parseMonth(endDateStr); err == nil {
				endDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, "invalid end_date format")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stats, err := s.GetTotalCost(ctx, postgres.GetTotalCostParams{
			UserID:      userID,
			ServiceName: serviceName,
			StartDate:   startDate,
			EndDate:     endDate,
		})

		if err != nil {
			reqLog.Error("get total cost failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		response := Response{
			TotalCost: stats.TotalCost,
			Period: Period{
				Start: formatDate(stats.StartDate),
				End:   formatDate(stats.EndDate),
			},
			Filters: Filters{
				UserID:      formatUUID(stats.UserID),
				ServiceName: stats.ServiceName,
			},
			SubscriptionsCount: stats.SubscriptionsCount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func parseMonth(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func formatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("2006-01-02")
}

func formatUUID(uuid *uuid.UUID) *string {
	if uuid == nil {
		return nil
	}
	str := uuid.String()
	return &str
}

func GetStatRoutes(log *slog.Logger, s *postgres.Storage) chi.Router {
	r := chi.NewRouter()
	r.Get("/total", GetTotalStats(log, s))
	return r
}
