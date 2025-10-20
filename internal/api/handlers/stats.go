package handlers

import (
	"EffectiveMobile/internal/service"
	"EffectiveMobile/pkg/api/response"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

type GetTotalStatsRequest struct {
	UserID      *string `json:"user_id,omitempty"`
	ServiceName *string `json:"service_name,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

type GetTotalStatsResponse struct {
	TotalCost          int     `json:"total_cost"`
	Period             Period  `json:"period"`
	Filters            Filters `json:"filters"`
	SubscriptionsCount int     `json:"subscriptions_count"`
}

func getStringParam(r *http.Request, key string) *string {
	if value := r.URL.Query().Get(key); value != "" {
		return &value
	}
	return nil
}

type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Filters struct {
	UserID      *string `json:"user_id,omitempty"`
	ServiceName *string `json:"service_name,omitempty"`
}

func GetTotalStats(log *slog.Logger, statsService *service.StatsService) http.HandlerFunc {
	const op = "handlers.api.stats.GetTotalStats"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		req := GetTotalStatsRequest{
			UserID:      getStringParam(r, "user_id"),
			ServiceName: getStringParam(r, "service_name"),
			StartDate:   getStringParam(r, "start_date"),
			EndDate:     getStringParam(r, "end_date"),
		}

		var userID *uuid.UUID
		var serviceName *string
		var startDate, endDate *time.Time

		if req.UserID != nil {
			if id, err := uuid.Parse(*req.UserID); err == nil {
				userID = &id
			}
		}

		if req.ServiceName != nil {
			serviceName = req.ServiceName
		}

		if req.StartDate != nil {
			if date, err := service.ParseMonth(*req.StartDate); err == nil {
				startDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, "invalid start_date format")
				return
			}
		}

		if req.EndDate != nil {
			if date, err := service.ParseMonth(*req.EndDate); err == nil {
				endDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, "invalid end_date format")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stats, err := statsService.GetTotalCost(ctx, userID, serviceName, startDate, endDate)

		if err != nil {
			reqLog.Error("get total cost failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		statsResponse := GetTotalStatsResponse{
			TotalCost: stats.TotalCost,
			Period: Period{
				Start: service.FormatDate(stats.StartDate),
				End:   service.FormatDate(stats.EndDate),
			},
			Filters: Filters{
				UserID:      service.FormatUUID(stats.UserID),
				ServiceName: stats.ServiceName,
			},
			SubscriptionsCount: stats.SubscriptionsCount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(statsResponse)
	}
}

func GetStatRoutes(log *slog.Logger, statsService *service.StatsService) chi.Router {
	r := chi.NewRouter()
	r.Get("/total", GetTotalStats(log, statsService))
	return r
}
