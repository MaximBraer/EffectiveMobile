package handlers

import (
	"EffectiveMobile/internal/repository"
	"EffectiveMobile/internal/service"
	"EffectiveMobile/pkg/api/response"
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

type CreateSubscriptionRequest struct {
	ServiceName string    `json:"service_name" validate:"required"`
	Price       int       `json:"price" validate:"required,min=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required"`
	EndDate     *string   `json:"end_date,omitempty"`
}

type CreateSubscriptionResponse struct {
	Status string `json:"status"`
	ID     int64  `json:"id"`
}

type UpdateSubscriptionRequest struct {
	Price     *int    `json:"price,omitempty" validate:"omitempty,min=0"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

type GetSubscriptionResponse struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func SaveSubscription(log *slog.Logger, service *service.SubscriptionService) http.HandlerFunc {
	const op = "handlers.api.subscription.SaveSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req CreateSubscriptionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var endDate string
		if req.EndDate != nil {
			endDate = *req.EndDate
		}

		id, err := service.CreateSubscription(ctx, req.ServiceName, req.Price, req.UserID, req.StartDate, endDate)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionAlreadyExists) {
				response.WriteError(w, http.StatusConflict, "subscription already exists")
				return
			}
			reqLog.Error("create subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Location", "/subscriptions/"+strconv.FormatInt(id, 10))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateSubscriptionResponse{
			Status: "ok",
			ID:     id,
		})
	}
}

func GetSubscription(log *slog.Logger, service *service.SubscriptionService) http.HandlerFunc {
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

		subscription, err := service.GetSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, "subscription not found")
				return
			}
			reqLog.Error("get subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		subscriptionResponse := GetSubscriptionResponse{
			ID:          subscription.ID,
			ServiceName: subscription.ServiceName,
			Price:       subscription.Price,
			UserID:      subscription.UserID.String(),
			StartDate:   subscription.StartDate.Format("01-2006"),
		}
		if subscription.EndDate != nil {
			endDate := subscription.EndDate.Format("01-2006")
			subscriptionResponse.EndDate = &endDate
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(subscriptionResponse)
	}
}

func UpdateSubscription(log *slog.Logger, service *service.SubscriptionService) http.HandlerFunc {
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

		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid arguments")
			return
		}

		if req.Price == nil && req.StartDate == nil && req.EndDate == nil {
			response.WriteError(w, http.StatusBadRequest, "at least one field must be provided")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err = service.UpdateSubscription(ctx, id, req.Price, req.StartDate, req.EndDate)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, "subscription not found")
				return
			}
			if errors.Is(err, repository.ErrSubscriptionAlreadyExists) {
				response.WriteError(w, http.StatusConflict, "subscription already exists")
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

func DeleteSubscription(log *slog.Logger, service *service.SubscriptionService) http.HandlerFunc {
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

		err = service.DeleteSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
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

func ListSubscriptions(log *slog.Logger, service *service.SubscriptionService) http.HandlerFunc {
	const op = "handlers.api.subscription.ListSubscriptions"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")
		userIDStr := r.URL.Query().Get("user_id")
		serviceName := r.URL.Query().Get("service_name")

		limit := 10
		offset := 0

		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		if offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		var userID *uuid.UUID
		if userIDStr != "" {
			if id, err := uuid.Parse(userIDStr); err == nil {
				userID = &id
			}
		}

		var serviceNamePtr *string
		if serviceName != "" {
			serviceNamePtr = &serviceName
		}

		subscriptions, total, err := service.ListSubscriptions(r.Context(), repository.ListSubscriptionsParams{
			Limit:       limit,
			Offset:      offset,
			UserID:      userID,
			ServiceName: serviceNamePtr,
		})

		if err != nil {
			reqLog.Error("list subscriptions failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		result := map[string]interface{}{
			"subscriptions": subscriptions,
			"total":         total,
			"limit":         limit,
			"offset":        offset,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func GetSubscriptionsRoutes(log *slog.Logger, service *service.SubscriptionService) chi.Router {
	r := chi.NewRouter()
	r.Post("/", SaveSubscription(log, service))
	r.Get("/", ListSubscriptions(log, service))
	r.Get("/{id}", GetSubscription(log, service))
	r.Put("/{id}", UpdateSubscription(log, service))
	r.Delete("/{id}", DeleteSubscription(log, service))
	return r
}
