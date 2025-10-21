package service

//go:generate mockgen -destination=subscription_mock.go -source=subscription.go -package=service

import (
	"EffectiveMobile/internal/repository"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type ServicesRepository interface {
	AddService(ctx context.Context, name string) (int, error)
	GetServiceName(ctx context.Context, id int) (string, error)
	GetServiceID(ctx context.Context, name string) (int, error)
	GetOrCreateServiceID(ctx context.Context, name string) (int, error)
	DeleteService(ctx context.Context, id int) error
}

type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, p repository.CreateSubscriptionParams) (int64, error)
	GetSubscription(ctx context.Context, id int64) (repository.Subscription, error)
	UpdateSubscription(ctx context.Context, p repository.UpdateSubscriptionParams) error
	DeleteSubscription(ctx context.Context, id int64) error
	ListSubscriptions(ctx context.Context, p repository.ListSubscriptionsParams) ([]repository.Subscription, int, error)
}

type SubscriptionService struct {
	serviceRepo      ServicesRepository
	subscriptionRepo SubscriptionRepository
	log              *slog.Logger
}

var ErrValidation = errors.New("validation error")

func NewSubscriptionService(serviceRepo ServicesRepository, subscriptionRepo SubscriptionRepository, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		serviceRepo:      serviceRepo,
		subscriptionRepo: subscriptionRepo,
		log:              log,
	}
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, serviceName string, price int, userID uuid.UUID, startDate, endDate string) (int64, error) {
	const op = "service.subscription.CreateSubscription"
	log := s.log.With(slog.String("op", op))

    startDateParsed, err := s.ParseMonth(startDate)
	if err != nil {
        return 0, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	var endDatePtr *time.Time
    if endDate != "" {
		ed, err := s.ParseMonth(endDate)
		if err != nil {
            return 0, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		if ed.Before(startDateParsed) {
            return 0, fmt.Errorf("%w: end date must be after start date", ErrValidation)
		}
		endDatePtr = &ed
	}

	serviceID, err := s.serviceRepo.GetOrCreateServiceID(ctx, serviceName)
	if err != nil {
		log.Error("get or create service failed", slog.String("err", err.Error()))
		return 0, err
	}

	id, err := s.subscriptionRepo.CreateSubscription(ctx, repository.CreateSubscriptionParams{
		UserID:    userID,
		ServiceID: serviceID,
		PriceRub:  price,
		StartDate: startDateParsed,
		EndDate:   endDatePtr,
	})
	if err != nil {
		log.Error("create subscription failed", slog.String("err", err.Error()))
		return 0, err
	}

	return id, nil
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, id int64) (*repository.Subscription, error) {
	const op = "service.subscription.GetSubscription"
	log := s.log.With(slog.String("op", op))

	subscription, err := s.subscriptionRepo.GetSubscription(ctx, id)
	if err != nil {
		log.Error("get subscription failed", slog.String("err", err.Error()))
		return nil, err
	}

	return &subscription, nil
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, id int64, serviceName *string, price *int, startDate, endDate *string) error {
	const op = "service.subscription.UpdateSubscription"
	log := s.log.With(slog.String("op", op))

	updateParams := repository.UpdateSubscriptionParams{
		ID:       id,
		PriceRub: price,
	}

	if serviceName != nil {
		serviceID, err := s.serviceRepo.GetOrCreateServiceID(ctx, *serviceName)
		if err != nil {
			log.Error("get or create service failed", slog.String("err", err.Error()))
			return err
		}
		updateParams.ServiceID = &serviceID
	}

    if startDate != nil {
		startDateParsed, err := s.ParseMonth(*startDate)
		if err != nil {
            return fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		updateParams.StartDate = &startDateParsed
	}

    if endDate != nil {
		if *endDate == "" || *endDate == "null" {
			updateParams.EndDate = &time.Time{}
		} else {
			endDateParsed, err := s.ParseMonth(*endDate)
			if err != nil {
                return fmt.Errorf("%w: %s", ErrValidation, err.Error())
			}
			if updateParams.StartDate != nil && endDateParsed.Before(*updateParams.StartDate) {
                return fmt.Errorf("%w: end date must be after start date", ErrValidation)
			}
			updateParams.EndDate = &endDateParsed
		}
	}

	err := s.subscriptionRepo.UpdateSubscription(ctx, updateParams)
	if err != nil {
		log.Error("update subscription failed", slog.String("err", err.Error()))
		return err
	}

	return nil
}

func (s *SubscriptionService) DeleteSubscription(ctx context.Context, id int64) error {
	const op = "service.subscription.DeleteSubscription"
	log := s.log.With(slog.String("op", op))

	err := s.subscriptionRepo.DeleteSubscription(ctx, id)
	if err != nil {
		log.Error("delete subscription failed", slog.String("err", err.Error()))
		return err
	}
	return nil
}

func (s *SubscriptionService) ParseMonth(monthStr string) (time.Time, error) {
	t, err := time.Parse("01-2006", monthStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format, expected MM-YYYY (e.g., 01-2024), got: %s", monthStr)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, params repository.ListSubscriptionsParams) ([]repository.Subscription, int, error) {
	return s.subscriptionRepo.ListSubscriptions(ctx, params)
}
