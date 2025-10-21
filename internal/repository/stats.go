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

type SubscriptionCost struct {
	ID          int64
	StartDate   time.Time
	EndDate     *time.Time
	PriceRub    int
	UserID      uuid.UUID
	ServiceName string
}

type TotalCostStats struct {
	TotalCost          int
	Subscriptions      []SubscriptionCost
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
			"s.id",
			"s.start_date",
			"s.end_date",
			"s.price_rub",
			"s.user_id",
			"sv.name",
		).
		ToSql()
	if err != nil {
		return TotalCostStats{}, fmt.Errorf("could not build query: %w", err)
	}

	rows, err := r.provider.GetConn().QueryContext(ctx, query, args...)
	if err != nil {
		return TotalCostStats{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var subscriptions []SubscriptionCost
	for rows.Next() {
		var id int64
		var startDate, endDate time.Time
		var priceRub int
		var userID uuid.UUID
		var serviceName string
		var endDatePtr *time.Time

		err := rows.Scan(&id, &startDate, &endDate, &priceRub, &userID, &serviceName)
		if err != nil {
			return TotalCostStats{}, fmt.Errorf("failed to scan row: %w", err)
		}

		if !endDate.IsZero() {
			endDatePtr = &endDate
		}

		subscriptions = append(subscriptions, SubscriptionCost{
			ID:          id,
			StartDate:   startDate,
			EndDate:     endDatePtr,
			PriceRub:    priceRub,
			UserID:      userID,
			ServiceName: serviceName,
		})
	}

	if err = rows.Err(); err != nil {
		return TotalCostStats{}, fmt.Errorf("error iterating rows: %w", err)
	}

	stats := TotalCostStats{
		Subscriptions:      subscriptions,
		UserID:             p.UserID,
		ServiceName:        p.ServiceName,
		StartDate:          p.StartDate,
		EndDate:            p.EndDate,
		SubscriptionsCount: len(subscriptions),
	}

	return stats, nil
}
