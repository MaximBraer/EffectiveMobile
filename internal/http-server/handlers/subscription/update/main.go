package update

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Request struct {
	Price     *int    `json:"price,omitempty"     validate:"omitempty,gte=0"`
	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime=01-2006"`
	EndDate   *string `json:"end_date,omitempty"   validate:"omitempty,datetime=01-2006"`
}

func New(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.subscription.update.New"
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

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		if err := validator.New().Struct(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		if req.Price == nil && req.StartDate == nil && req.EndDate == nil {
			response.WriteError(w, http.StatusBadRequest, "at least one field must be provided")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		existing, err := s.GetSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, storage.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, "subscription not found")
				return
			}
			reqLog.Error("get subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		updateParams := postgres.UpdateSubscriptionParams{
			ID: id,
		}

		if req.Price != nil {
			updateParams.PriceRub = req.Price
		} else {
			updateParams.PriceRub = &existing.Price
		}

		if req.StartDate != nil {
			startDate, err := parseMonth(*req.StartDate)
			if err != nil {
				response.WriteError(w, http.StatusBadRequest, "invalid arguments")
				return
			}
			updateParams.StartDate = &startDate
		} else {
			updateParams.StartDate = &existing.StartDate
		}

		if req.EndDate != nil {
			endDate, err := parseMonth(*req.EndDate)
			if err != nil {
				response.WriteError(w, http.StatusBadRequest, "invalid arguments")
				return
			}
			if endDate.Before(*updateParams.StartDate) {
				response.WriteError(w, http.StatusBadRequest, "invalid arguments")
				return
			}
			updateParams.EndDate = &endDate
		} else {
			updateParams.EndDate = existing.EndDate
		}

		err = s.UpdateSubscription(ctx, updateParams)
		if err != nil {
			if errors.Is(err, storage.ErrSubscriptionAlreadyExists) {
				response.WriteError(w, http.StatusConflict, "invalid arguments")
				return
			}
			reqLog.Error("update subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
		})
	}
}

func parseMonth(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

