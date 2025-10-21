package service

//go:generate mockgen -destination=stats_mock.go -source=stats.go -package=service

import (
	"EffectiveMobile/internal/repository"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type StatsRepository interface {
	GetTotalCost(ctx context.Context, p repository.GetTotalCostParams) (repository.TotalCostStats, error)
}

type StatsService struct {
	statsRepo StatsRepository
	log       *slog.Logger
}

func NewStatsService(statsRepo StatsRepository, log *slog.Logger) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		log:       log,
	}
}

func (s *StatsService) GetTotalCost(ctx context.Context, userID *uuid.UUID, serviceName *string, startDate, endDate *time.Time) (*repository.TotalCostStats, error) {
	const op = "service.stats.GetTotalCost"
	log := s.log.With(slog.String("op", op))

	stats, err := s.statsRepo.GetTotalCost(ctx, repository.GetTotalCostParams{
		UserID:      userID,
		ServiceName: serviceName,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		log.Error("get total cost failed", slog.String("err", err.Error()))
		return nil, err
	}

	totalCost := s.calculateTotalCost(stats.Subscriptions, startDate, endDate)
	stats.TotalCost = totalCost

	return &stats, nil
}

func (s *StatsService) ParseMonth(monthStr string) (time.Time, error) {
	t, err := time.Parse("01-2006", monthStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format, expected MM-YYYY (e.g., 01-2024), got: %s", monthStr)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func (s *StatsService) FormatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("01-2006")
}

func (s *StatsService) FormatUUID(uuid *uuid.UUID) *string {
	if uuid == nil {
		return nil
	}
	str := uuid.String()
	return &str
}

func (s *StatsService) calculateTotalCost(subscriptions []repository.SubscriptionCost, periodStart, periodEnd *time.Time) int {
	totalCost := 0

	for _, sub := range subscriptions {
		months := s.calculateIntersectionMonths(sub.StartDate, sub.EndDate, periodStart, periodEnd)
		totalCost += sub.PriceRub * months
	}

	return totalCost
}

func (s *StatsService) calculateIntersectionMonths(
	subscriptionStart time.Time,
	subscriptionEnd *time.Time,
	periodStart *time.Time,
	periodEnd *time.Time,
) int {
	if periodStart == nil && periodEnd == nil {
		if subscriptionEnd == nil {
			now := time.Now()
			return s.monthsBetween(subscriptionStart, now)
		}
		return s.monthsBetween(subscriptionStart, *subscriptionEnd)
	}

	var intersectionStart, intersectionEnd time.Time

	if periodStart == nil {
		intersectionStart = subscriptionStart
	} else {
		if subscriptionStart.After(*periodStart) {
			intersectionStart = subscriptionStart
		} else {
			intersectionStart = *periodStart
		}
	}

	if subscriptionEnd == nil {
		if periodEnd == nil {
			now := time.Now()
			intersectionEnd = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		} else {
			intersectionEnd = *periodEnd
		}
	} else {
		if periodEnd == nil {
			intersectionEnd = *subscriptionEnd
		} else {
			if subscriptionEnd.Before(*periodEnd) {
				intersectionEnd = *subscriptionEnd
			} else {
				intersectionEnd = *periodEnd
			}
		}
	}

	if intersectionStart.After(intersectionEnd) {
		return 0
	}

	return s.monthsBetween(intersectionStart, intersectionEnd)
}

func (s *StatsService) monthsBetween(start, end time.Time) int {
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, end.Location())

	months := 0
	for start.Before(end) || start.Equal(end) {
		start = start.AddDate(0, 1, 0)
		months++
	}

	return months
}

