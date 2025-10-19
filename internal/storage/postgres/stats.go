package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type GetTotalCostParams struct {
	UserID      *uuid.UUID `json:"user_id"`
	ServiceName *string    `json:"service_name"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

type TotalCostStats struct {
	TotalCost          int        `json:"total_cost"`
	StartDate          *time.Time `json:"start_date"`
	EndDate            *time.Time `json:"end_date"`
	UserID             *uuid.UUID `json:"user_id"`
	ServiceName        *string    `json:"service_name"`
	SubscriptionsCount int        `json:"subscriptions_count"`
}

func (s *Storage) GetTotalCost(ctx context.Context, p GetTotalCostParams) (TotalCostStats, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if p.UserID != nil {
		whereClause += " AND s.user_id = $" + strconv.Itoa(argIndex)
		args = append(args, *p.UserID)
		argIndex++
	}

	if p.ServiceName != nil {
		whereClause += " AND sv.name = $" + strconv.Itoa(argIndex)
		args = append(args, *p.ServiceName)
		argIndex++
	}

	if p.StartDate != nil {
		whereClause += " AND s.start_date >= $" + strconv.Itoa(argIndex)
		args = append(args, *p.StartDate)
		argIndex++
	}

	if p.EndDate != nil {
		whereClause += " AND (s.end_date IS NULL OR s.end_date <= $" + strconv.Itoa(argIndex) + ")"
		args = append(args, *p.EndDate)
		argIndex++
	}

	query := `SELECT 
		COALESCE(SUM(s.price_rub), 0) as total_cost,
		COUNT(*) as subscriptions_count
		FROM subscription s
		JOIN service sv ON s.service_id = sv.id
		` + whereClause

	var stats TotalCostStats
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalCost,
		&stats.SubscriptionsCount,
	)

	if err != nil {
		return TotalCostStats{}, err
	}

	stats.StartDate = p.StartDate
	stats.EndDate = p.EndDate
	stats.UserID = p.UserID
	stats.ServiceName = p.ServiceName

	return stats, nil
}
