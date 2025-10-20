package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
)

func GetTotalCost(ctx context.Context, db *sql.DB, p GetTotalCostParams) (TotalCostStats, error) {
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
		whereConditions = append(whereConditions, squirrel.Or{
			squirrel.Eq{"s.end_date": nil},
			squirrel.LtOrEq{"s.end_date": *p.EndDate},
		})
	}

	if len(whereConditions) > 0 {
		baseQuery = baseQuery.Where(whereConditions)
	}

	query := baseQuery.Columns(
		"COALESCE(SUM(s.price_rub), 0) as total_cost",
		"MIN(s.start_date) as start_date",
		"MAX(COALESCE(s.end_date, s.start_date)) as end_date",
		"COUNT(*) as subscriptions_count",
	)

	querySql, queryArgs, err := query.ToSql()
	if err != nil {
		return TotalCostStats{}, fmt.Errorf("could not build query: %w", err)
	}

	var stats TotalCostStats
	err = db.QueryRowContext(ctx, querySql, queryArgs...).Scan(
		&stats.TotalCost,
		&stats.StartDate,
		&stats.EndDate,
		&stats.SubscriptionsCount,
	)

	// Set UserID and ServiceName from parameters
	stats.UserID = p.UserID
	stats.ServiceName = p.ServiceName

	if err != nil {
		return TotalCostStats{}, err
	}

	return stats, nil
}
