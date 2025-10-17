package save

import (
	resp "EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type Request struct {
	ServiceName string    `json:"service_name" validate:"required,min=1"`
	Price       int       `json:"price"        validate:"required,gte=0"`
	UserID      uuid.UUID `json:"user_id"      validate:"required"`
	StartDate   string    `json:"start_date"   validate:"required,datetime=01-2006"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,datetime=01-2006"`
}

func New(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.subscription.save.New"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			resp.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		if req.ServiceName == "" || req.UserID == uuid.Nil {
			resp.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}
		reqLog.Info("decoded request body", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			resp.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		serviceID, err := s.GetOrCreateServiceID(r.Context(), req.ServiceName)
		if err != nil {
			reqLog.Error("get or create service failed", slog.String("err", err.Error()))
			resp.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		startDate, err := parseMonth(req.StartDate)
		if err != nil {
			resp.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		var endDatePtr *time.Time
		if req.EndDate != nil {
			ed, err := parseMonth(*req.EndDate)
			if err != nil || ed.Before(startDate) {
				resp.WriteError(w, http.StatusBadRequest, "invalid arguments")
				return
			}
			endDatePtr = &ed
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		id, err := s.CreateSubscription(ctx, postgres.CreateSubscriptionParams{
			UserID:    req.UserID,
			ServiceID: serviceID,
			PriceRub:  req.Price,
			StartDate: startDate,
			EndDate:   endDatePtr,
		}, log)
		if err != nil {
			if errors.Is(err, storage.ErrSubscriptionAlreadyExists) {
				resp.WriteError(w, http.StatusConflict, "invalid arguments")
				return
			}
			reqLog.Error("create subscription failed", slog.String("err", err.Error()))
			resp.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Location", "/subscriptions/"+int64ToStr(id))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"id":     id,
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

func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

