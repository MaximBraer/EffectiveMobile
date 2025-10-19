package handlers

import (
	"EffectiveMobile/internal/lib/api/response"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func DeleteSubscription(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.subscription.DeleteSubscription"
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

type GetSubscriptionResponse struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func GetSubscription(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.subscription.GetSubscription"
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
		_ = json.NewEncoder(w).Encode(GetSubscriptionResponse{
			ID:          subscription.ID,
			ServiceName: subscription.ServiceName,
			Price:       subscription.Price,
			UserID:      subscription.UserID.String(),
			StartDate:   subscription.StartDate.Format("01-2006"),
			EndDate:     formatEndDate(subscription.EndDate),
		})
	}
}

type ListSubscriptionsResponse struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Pagination    Pagination     `json:"pagination"`
}

type Subscription struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

func ListSubscriptions(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.subscription.ListSubscriptions"
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

		response := ListSubscriptionsResponse{
			Subscriptions: make([]Subscription, len(subscriptions)),
			Pagination: Pagination{
				Total:   total,
				Limit:   limit,
				Offset:  offset,
				HasMore: offset+limit < total,
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

type SaveSubscriptionRequest struct {
	ServiceName string    `json:"service_name" validate:"required,min=1"`
	Price       int       `json:"price"        validate:"required,gte=0"`
	UserID      uuid.UUID `json:"user_id"      validate:"required"`
	StartDate   string    `json:"start_date"   validate:"required,datetime=01-2006"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,datetime=01-2006"`
}

type SaveSubscriptionResponse struct {
	response.Response
	ID int64 `json:"id"`
}

func SaveSubscription(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.subscription.SaveSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req SaveSubscriptionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		if req.ServiceName == "" || req.UserID == uuid.Nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}
		reqLog.Info("decoded request body", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		serviceID, err := s.GetOrCreateServiceID(r.Context(), req.ServiceName)
		if err != nil {
			reqLog.Error("get or create service failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		startDate, err := parseMonth(req.StartDate)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		var endDatePtr *time.Time
		if req.EndDate != nil {
			ed, err := parseMonth(*req.EndDate)
			if err != nil || ed.Before(startDate) {
				response.WriteError(w, http.StatusBadRequest, "invalid arguments")
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
				response.WriteError(w, http.StatusConflict, "invalid arguments")
				return
			}
			reqLog.Error("create subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
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

type UpdateSubscriptionRequest struct {
	Price     *int    `json:"price,omitempty"     validate:"omitempty,gte=0"`
	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime=01-2006"`
	EndDate   *string `json:"end_date,omitempty"   validate:"omitempty,datetime=01-2006"`
}

func UpdateSubscription(log *slog.Logger, s *postgres.Storage) http.HandlerFunc {
	const op = "handlers.api.subscription.UpdateSubscription"
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

		var req UpdateSubscriptionRequest
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

func GetSubscriptionsRoutes(log *slog.Logger, s *postgres.Storage) chi.Router {
	r := chi.NewRouter()
	r.Post("/", SaveSubscription(log, s))
	r.Get("/{id}", GetSubscription(log, s))
	r.Put("/{id}", UpdateSubscription(log, s))
	r.Delete("/{id}", DeleteSubscription(log, s))
	r.Get("/", ListSubscriptions(log, s))
	return r
}

func formatEndDate(endDate *time.Time) *string {
	if endDate == nil {
		return nil
	}
	formatted := endDate.Format("01-2006")
	return &formatted
}

func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
