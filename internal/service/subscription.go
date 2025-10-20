package service

//go:generate mockgen -destination=subscription_mock.go -source=subscription.go -package=service

import (
	"EffectiveMobile/internal/repository"
	"context"
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
	CreateSubscription(ctx context.Context, p repository.CreateSubscriptionParams, log *slog.Logger) (int64, error)
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
		return 0, err
	}

	var endDatePtr *time.Time
	if endDate != "" {
		ed, err := s.ParseMonth(endDate)
		if err != nil {
			return 0, err
		}
		if ed.Before(startDateParsed) {
			return 0, fmt.Errorf("end date must be after start date")
		}
		endDatePtr = &ed
	}

	serviceID, err := s.serviceRepo.GetOrCreateServiceID(ctx, serviceName)
	if err != nil {
		log.Error("get or create service failed", slog.String("err", err.Error()))
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, err := s.subscriptionRepo.CreateSubscription(ctx, repository.CreateSubscriptionParams{
		UserID:    userID,
		ServiceID: serviceID,
		PriceRub:  price,
		StartDate: startDateParsed,
		EndDate:   endDatePtr,
	}, log)
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

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, id int64, price *int, startDate, endDate *string) error {
	const op = "service.subscription.UpdateSubscription"
	log := s.log.With(slog.String("op", op))

	existing, err := s.subscriptionRepo.GetSubscription(ctx, id)
	if err != nil {
		log.Error("get subscription failed", slog.String("err", err.Error()))
		return err
	}

	updateParams := repository.UpdateSubscriptionParams{
		ID: id,
	}

	if price != nil {
		updateParams.PriceRub = price
	} else {
		updateParams.PriceRub = &existing.Price
	}

	if startDate != nil {
		startDateParsed, err := s.ParseMonth(*startDate)
		if err != nil {
			return err
		}
		updateParams.StartDate = &startDateParsed
	} else {
		updateParams.StartDate = &existing.StartDate
	}

	if endDate != nil {
		endDateParsed, err := s.ParseMonth(*endDate)
		if err != nil {
			return err
		}
		if endDateParsed.Before(*updateParams.StartDate) {
			return err
		}
		updateParams.EndDate = &endDateParsed
	} else {
		updateParams.EndDate = existing.EndDate
	}

	err = s.subscriptionRepo.UpdateSubscription(ctx, updateParams)
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
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func formatEndDate(endDate *time.Time) *string {
	if endDate == nil {
		return nil
	}
	formatted := endDate.Format("01-2006")
	return &formatted
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, params repository.ListSubscriptionsParams) ([]repository.Subscription, int, error) {
	return s.subscriptionRepo.ListSubscriptions(ctx, params)
}
