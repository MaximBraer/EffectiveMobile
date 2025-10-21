//go:generate go run go.uber.org/mock/mockgen@latest -destination=stats_mock.go -source=stats.go -package=handlers

package handlers

import (
	"EffectiveMobile/internal/repository"
	"EffectiveMobile/pkg/api/response"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const (
	ErrInvalidStatsArguments = "invalid arguments"
	ErrInvalidUserID         = "invalid user_id format"
	ErrInternalServerStats   = "internal server error"
)

var ErrValidation = errors.New("validation error")

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

type validatedStatsParams struct {
	UserID      *uuid.UUID
	ServiceName *string
	StartDate   *time.Time
	EndDate     *time.Time
}

func validateStatsParams(req GetTotalStatsRequest, statsService StatsService) (*validatedStatsParams, error) {
	params := &validatedStatsParams{}

	if req.UserID != nil {
		id, err := uuid.Parse(*req.UserID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, ErrInvalidUserID)
		}
		params.UserID = &id
	}

	if req.ServiceName != nil {
		params.ServiceName = req.ServiceName
	}

	if req.StartDate != nil {
		date, err := statsService.ParseMonth(*req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		params.StartDate = &date
	}

	if req.EndDate != nil {
		date, err := statsService.ParseMonth(*req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		params.EndDate = &date
	}

	if params.StartDate != nil && params.EndDate != nil && params.EndDate.Before(*params.StartDate) {
		return nil, fmt.Errorf("%w: end date must be after start date", ErrValidation)
	}

	return params, nil
}

type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Filters struct {
	UserID      *string `json:"user_id,omitempty"`
	ServiceName *string `json:"service_name,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"invalid arguments"`
}

// @Summary      Get total stats
// @Description  Calculate total cost of subscriptions for the period. Date format: MM-YYYY (e.g., "01-2024", "12-2024")
// @Tags         stats
// @Produce      json
// @Param        user_id       query     string  false  "User UUID"                    example(550e8400-e29b-41d4-a716-446655440000)
// @Param        service_name  query     string  false  "Service name"                 example(Netflix)
// @Param        start_date    query     string  false  "Period start (MM-YYYY)"       example(01-2024)
// @Param        end_date      query     string  false  "Period end (MM-YYYY)"         example(12-2024)
// @Success      200           {object}  GetTotalStatsResponse
// @Failure      400           {object}  ErrorResponse  "Invalid arguments or date format"
// @Failure      500           {object}  ErrorResponse  "Internal server error"
// @Router       /stats/total [get]
func GetTotalStats(statsService StatsService, log *slog.Logger) http.HandlerFunc {
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

		params, err := validateStatsParams(req, statsService)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stats, err := statsService.GetTotalCost(ctx, params.UserID, params.ServiceName, params.StartDate, params.EndDate)

		if err != nil {
			reqLog.Error("get total cost failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServerStats)
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

func GetStatRoutes(statsService StatsService, log *slog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Get("/total", GetTotalStats(statsService, log))
	return r
}
