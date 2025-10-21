//go:generate go run go.uber.org/mock/mockgen@latest -destination=subscription_mock.go -source=subscription.go -package=handlers

package handlers

import (
	"EffectiveMobile/internal/repository"
	serv "EffectiveMobile/internal/service"
	"EffectiveMobile/pkg/api/response"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
    "strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	ErrInvalidArguments      = "invalid arguments"
	ErrInvalidSubscriptionID = "invalid subscription id"
	ErrSubscriptionNotFound  = "subscription not found"
	ErrSubscriptionExists    = "subscription already exists"
	ErrInternalServer        = "internal server error"
	ErrInvalidUserIDFormat   = "invalid user_id format"
)

type SubscriptionService interface {
	CreateSubscription(ctx context.Context, serviceName string, price int, userID uuid.UUID, startDate, endDate string) (int64, error)
	GetSubscription(ctx context.Context, id int64) (*repository.Subscription, error)
	UpdateSubscription(ctx context.Context, id int64, serviceName *string, price *int, startDate, endDate *string) error
	DeleteSubscription(ctx context.Context, id int64) error
	ListSubscriptions(ctx context.Context, params repository.ListSubscriptionsParams) ([]repository.Subscription, int, error)
}

type CreateSubscriptionRequest struct {
	ServiceName string    `json:"service_name" validate:"required"`
	Price       int       `json:"price" validate:"required,min=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required"`
	EndDate     *string   `json:"end_date,omitempty"`
}

func validateCreateSubscriptionRequest(req CreateSubscriptionRequest) error {
	validate := validator.New()

    req.ServiceName = strings.TrimSpace(req.ServiceName)
    if req.ServiceName == "" { 
        return fmt.Errorf("service_name is required")
    }
	
    return validate.Struct(req)
}

type CreateSubscriptionResponse struct {
	Status string `json:"status"`
	ID     int64  `json:"id"`
}

type UpdateSubscriptionRequest struct {
    ServiceName *string `json:"service_name,omitempty" validate:"omitempty,min=1"`
	Price       *int    `json:"price,omitempty" validate:"omitempty,min=0"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

func validateUpdateSubscriptionRequest(req UpdateSubscriptionRequest) error {
	validate := validator.New()
    if req.ServiceName != nil {
        trimmed := strings.TrimSpace(*req.ServiceName)
        if trimmed == "" {
            req.ServiceName = nil
        } else {
            req.ServiceName = &trimmed
        }
    }
	if err := validate.Struct(req); err != nil {
		return err
	}

	if req.ServiceName == nil && req.Price == nil && req.StartDate == nil && req.EndDate == nil {
		return fmt.Errorf("at least one field must be provided")
	}

	return nil
}

type GetSubscriptionResponse struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartDate   string  `json:"start_date" example:"01-2024"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2024"`
}

type ListSubscriptionsItem struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type ListSubscriptionsResponse struct {
	Subscriptions []ListSubscriptionsItem `json:"subscriptions"`
	Total         int                     `json:"total"`
	Limit         int                     `json:"limit"`
	Offset        int                     `json:"offset"`
}

// @Summary      Create subscription
// @Description  Create a new subscription. Date format: MM-YYYY (e.g., "01-2024")
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        input  body      CreateSubscriptionRequest  true  "Create payload"
// @Success      201    {object}  CreateSubscriptionResponse
// @Failure      400    {object}  ErrorResponse  "Invalid request body or validation error"
// @Failure      409    {object}  ErrorResponse  "Subscription already exists"
// @Failure      500    {object}  ErrorResponse  "Internal server error"
// @Router       /subscriptions [post]
func SaveSubscription(subscriptionService SubscriptionService, log *slog.Logger) http.HandlerFunc {
	const op = "handlers.api.subscription.SaveSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req CreateSubscriptionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
		}

		if err := validateCreateSubscriptionRequest(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
		}

		var endDate string
		if req.EndDate != nil {
			endDate = *req.EndDate
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		id, err := subscriptionService.CreateSubscription(ctx, req.ServiceName, req.Price, req.UserID, req.StartDate, endDate)
		if err != nil {
			if errors.Is(err, serv.ErrValidation) {
				response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
				return
			}
			if errors.Is(err, repository.ErrSubscriptionAlreadyExists) {
				response.WriteError(w, http.StatusConflict, ErrSubscriptionExists)
				return
			}
			reqLog.Error("create subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServer)
			return
		}

		w.Header().Set("Location", "/api/v1/subscriptions/"+strconv.FormatInt(id, 10))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateSubscriptionResponse{
			Status: "ok",
			ID:     id,
		})
	}
}

// @Summary      Get subscription
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      int  true  "Subscription ID"
// @Success      200  {object}  GetSubscriptionResponse
// @Failure      400  {object}  ErrorResponse  "Invalid subscription ID"
// @Failure      404  {object}  ErrorResponse  "Subscription not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /subscriptions/{id} [get]
func GetSubscription(subscriptionService SubscriptionService, log *slog.Logger) http.HandlerFunc {
	const op = "handlers.api.subscription.GetSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidSubscriptionID)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		subscription, err := subscriptionService.GetSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, ErrSubscriptionNotFound)
				return
			}
			reqLog.Error("get subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServer)
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

// @Summary      Update subscription
// @Description  Update subscription fields (partial update). Date format: MM-YYYY (e.g., "12-2024"). Can change service_name, price, start_date, and end_date.
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id     path      int                       true  "Subscription ID"
// @Param        input  body      UpdateSubscriptionRequest  true  "Update payload"
// @Success      200    {object}  map[string]string          "Successfully updated"
// @Failure      400    {object}  ErrorResponse              "Invalid request body or validation error"
// @Failure      404    {object}  ErrorResponse              "Subscription not found"
// @Failure      409    {object}  ErrorResponse              "Conflict - duplicate subscription (user_id + service_id + start_date)"
// @Failure      500    {object}  ErrorResponse              "Internal server error"
// @Router       /subscriptions/{id} [put]
func UpdateSubscription(subscriptionService SubscriptionService, log *slog.Logger) http.HandlerFunc {
	const op = "handlers.api.subscription.UpdateSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
        if err != nil || id <= 0 {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidSubscriptionID)
			return
		}

		var req UpdateSubscriptionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			reqLog.Error("failed to decode request", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
		}

		if err := validateUpdateSubscriptionRequest(req); err != nil {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err = subscriptionService.UpdateSubscription(ctx, id, req.ServiceName, req.Price, req.StartDate, req.EndDate)
		if err != nil {
			if errors.Is(err, serv.ErrValidation) {
				response.WriteError(w, http.StatusBadRequest, ErrInvalidArguments)
				return
			}
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, ErrSubscriptionNotFound)
				return
			}
		if errors.Is(err, repository.ErrSubscriptionAlreadyExists) {
			response.WriteError(w, http.StatusConflict, ErrSubscriptionExists)
			return
		}
			reqLog.Error("update subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServer)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
		})
	}
}

// @Summary      Delete subscription
// @Tags         subscriptions
// @Param        id   path  int  true  "Subscription ID"
// @Success      204  {string}  string  "No content"
// @Failure      400  {object}  ErrorResponse  "Invalid subscription ID"
// @Failure      404  {object}  ErrorResponse  "Subscription not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /subscriptions/{id} [delete]
func DeleteSubscription(subscriptionService SubscriptionService, log *slog.Logger) http.HandlerFunc {
	const op = "handlers.api.subscription.DeleteSubscription"
	log = log.With(slog.String("op", op))

	return func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.With(
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, ErrInvalidSubscriptionID)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err = subscriptionService.DeleteSubscription(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrSubscriptionNotFound) {
				response.WriteError(w, http.StatusNotFound, ErrSubscriptionNotFound)
				return
			}
			reqLog.Error("delete subscription failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServer)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// @Summary      List subscriptions
// @Tags         subscriptions
// @Produce      json
// @Param        limit         query     int     false  "limit"   minimum(1)  default(10)
// @Param        offset        query     int     false  "offset"  minimum(0)  default(0)
// @Param        user_id       query     string  false  "user uuid"
// @Param        service_name  query     string  false  "service name"
// @Success      200           {object}  ListSubscriptionsResponse  "List of subscriptions with pagination"
// @Failure      400           {object}  ErrorResponse           "Invalid user_id format"
// @Failure      500           {object}  ErrorResponse           "Internal server error"
// @Router       /subscriptions [get]
func ListSubscriptions(subscriptionService SubscriptionService, log *slog.Logger) http.HandlerFunc {
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
			id, err := uuid.Parse(userIDStr)
			if err != nil {
				response.WriteError(w, http.StatusBadRequest, ErrInvalidUserIDFormat)
				return
			}
			userID = &id
		}

		var serviceNamePtr *string
		if serviceName != "" {
			serviceNamePtr = &serviceName
		}

		subscriptions, total, err := subscriptionService.ListSubscriptions(r.Context(), repository.ListSubscriptionsParams{
			Limit:       limit,
			Offset:      offset,
			UserID:      userID,
			ServiceName: serviceNamePtr,
		})

		if err != nil {
			reqLog.Error("list subscriptions failed", slog.String("err", err.Error()))
			response.WriteError(w, http.StatusInternalServerError, ErrInternalServer)
			return
		}

		items := make([]ListSubscriptionsItem, 0, len(subscriptions))
		for _, s := range subscriptions {
			item := ListSubscriptionsItem{
				ID:          s.ID,
				ServiceName: s.ServiceName,
				Price:       s.Price,
				UserID:      s.UserID.String(),
				StartDate:   s.StartDate.Format("01-2006"),
			}
			if s.EndDate != nil {
				ed := s.EndDate.Format("01-2006")
				item.EndDate = &ed
			}
			items = append(items, item)
		}

		result := ListSubscriptionsResponse{
			Subscriptions: items,
			Total:         total,
			Limit:         limit,
			Offset:        offset,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func GetSubscriptionsRoutes(subscriptionService SubscriptionService, log *slog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Post("/", SaveSubscription(subscriptionService, log))
	r.Get("/", ListSubscriptions(subscriptionService, log))
	r.Get("/{id}", GetSubscription(subscriptionService, log))
	r.Put("/{id}", UpdateSubscription(subscriptionService, log))
	r.Delete("/{id}", DeleteSubscription(subscriptionService, log))
	return r
}
