package list

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Pagination    Pagination     `json:"pagination"`
}

type Subscription struct {
	ID          int64  `json:"id"`
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type Pagination struct {
	Total    int  `json:"total"`
	Limit    int  `json:"limit"`
	Offset   int  `json:"offset"`
	HasMore  bool `json:"has_more"`
}

func New(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.subscription.list.New"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		limit := 10
		offset := 0
		var userID *uuid.UUID
		var serviceName *string

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
			if id, err := uuid.Parse(userIDStr); err == nil {
				userID = &id
			} else {
				response.WriteError(w, http.StatusBadRequest, "invalid user_id format")
				return
			}
		}

		if serviceNameStr := r.URL.Query().Get("service_name"); serviceNameStr != "" {
			serviceName = &serviceNameStr
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		subscriptions, total, err := s.ListSubscriptions(ctx, postgres.ListSubscriptionsParams{
			Limit:       limit,
			Offset:      offset,
			UserID:      userID,
			ServiceName: serviceName,
		})

		if err != nil {
			reqLog.Error("list subscriptions failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		response := Response{
			Subscriptions: make([]Subscription, len(subscriptions)),
			Pagination: Pagination{
				Total:    total,
				Limit:    limit,
				Offset:   offset,
				HasMore:  offset+limit < total,
			},
		}

		for i, sub := range subscriptions {
			response.Subscriptions[i] = Subscription{
				ID:          sub.ID,
				ServiceName: sub.ServiceName,
				Price:       sub.Price,
				UserID:      sub.UserID.String(),
				StartDate:   sub.StartDate.Format("01-2006"),
				EndDate:     formatEndDate(sub.EndDate),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func formatEndDate(endDate *time.Time) *string {
	if endDate == nil {
		return nil
	}
	formatted := endDate.Format("01-2006")
	return &formatted
}

