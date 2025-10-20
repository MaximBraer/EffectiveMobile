//go:generate go run go.uber.org/mock/mockgen@latest -destination=stats_mock.go -source=stats.go -package=handlers

package handlers

import (
	"EffectiveMobile/internal/repository"
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

const (
	ErrInvalidArguments = "invalid arguments"
)

type StatsService interface {
	GetTotalCost(ctx context.Context, userID *uuid.UUID, serviceName *string, startDate, endDate *time.Time) (*repository.TotalCostStats, error)
	ParseMonth(s string) (time.Time, error)
	FormatDate(date *time.Time) string
	FormatUUID(uuid *uuid.UUID) *string
}

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

// @Summary      Get total stats
// @Tags         stats
// @Produce      json
// @Param        user_id       query     string  false  "user uuid"
// @Param        service_name  query     string  false  "service name"
// @Param        start_date    query     string  false  "MM-YYYY"
// @Param        end_date      query     string  false  "MM-YYYY"
// @Success      200           {object}  GetTotalStatsResponse
// @Failure      400           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /stats/total [get]
func GetTotalStats(subscriptionService SubscriptionService, statsService StatsService, log *slog.Logger) http.HandlerFunc {
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
			if date, err := statsService.ParseMonth(*req.StartDate); err == nil {
				startDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
				return
			}
		}

		if req.EndDate != nil {
			if date, err := statsService.ParseMonth(*req.EndDate); err == nil {
				endDate = &date
			} else {
				response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
				return
			}
		}

		if startDate != nil && endDate != nil && endDate.Before(*startDate) {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
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
				Start: statsService.FormatDate(stats.StartDate),
				End:   statsService.FormatDate(stats.EndDate),
			},
			Filters: Filters{
				UserID:      statsService.FormatUUID(stats.UserID),
				ServiceName: stats.ServiceName,
			},
			SubscriptionsCount: stats.SubscriptionsCount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(statsResponse)
	}
}

func GetStatRoutes(subscriptionService SubscriptionService, statsService StatsService, log *slog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Get("/total", GetTotalStats(subscriptionService, statsService, log))
	return r
}
