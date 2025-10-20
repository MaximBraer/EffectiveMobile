package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type GetTotalCostParams struct {
	UserID      *uuid.UUID
	ServiceName *string
	StartDate   *time.Time
	EndDate     *time.Time
}

type TotalCostStats struct {
	TotalCost          int
	StartDate          *time.Time
	EndDate            *time.Time
	UserID             *uuid.UUID
	ServiceName        *string
	SubscriptionsCount int
}

type StatsRepository struct {
	provider Provider
	logger   Logger
}

func NewStatsRepository(provider Provider, logger Logger) *StatsRepository {
	return &StatsRepository{
		provider: provider,
		logger:   logger,
	}
}

func (r *StatsRepository) GetTotalCost(ctx context.Context, p GetTotalCostParams) (TotalCostStats, error) {
	baseQuery := squirrel.Select().
		From("subscription s").
		Join("service sv ON s.service_id = sv.id").
		PlaceholderFormat(squirrel.Dollar)

	whereConditions := squirrel.And{}

	if p.UserID != nil {
		whereConditions = append(whereConditions, squirrel.Eq{"s.user_id": *p.UserID})
	}

	if p.ServiceName != nil {
		whereConditions = append(whereConditions, squirrel.Eq{"sv.name": *p.ServiceName})
	}

	if p.StartDate != nil {
		whereConditions = append(whereConditions, squirrel.GtOrEq{"s.start_date": *p.StartDate})
	}

	if p.EndDate != nil {
		whereConditions = append(whereConditions, squirrel.LtOrEq{"s.start_date": *p.EndDate})
	}

	if len(whereConditions) > 0 {
		baseQuery = baseQuery.Where(whereConditions)
	}

	query, args, err := baseQuery.
		Columns(
			"COALESCE(SUM(s.price_rub), 0) as total_cost",
			"COUNT(*) as subscriptions_count",
		).
		ToSql()
	if err != nil {
		return TotalCostStats{}, fmt.Errorf("could not build query: %w", err)
	}

	var stats TotalCostStats
	err = r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalCost,
		&stats.SubscriptionsCount,
	)
	if err != nil {
		return TotalCostStats{}, fmt.Errorf("failed to execute query: %w", err)
	}

	// Устанавливаем параметры запроса в результат
	stats.UserID = p.UserID
	stats.ServiceName = p.ServiceName
	stats.StartDate = p.StartDate
	stats.EndDate = p.EndDate

	return stats, nil
}
