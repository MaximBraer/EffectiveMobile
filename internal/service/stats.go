package service

//go:generate mockgen -destination=stats_mock.go -source=stats.go -package=service

import (
	"EffectiveMobile/internal/repository"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Repository interface
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

	return &stats, nil
}

func ParseMonth(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func FormatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("2006-01-02")
}

func FormatUUID(uuid *uuid.UUID) *string {
	if uuid == nil {
		return nil
	}
	str := uuid.String()
	return &str
}
