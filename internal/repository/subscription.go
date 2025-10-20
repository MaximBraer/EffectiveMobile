package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateSubscription(ctx context.Context, db *sql.DB, p CreateSubscriptionParams, log *slog.Logger) (int64, error) {
	const op = "repository.postgres.CreateSubscription"
	log = log.With(slog.String("op", op))

	query, args, err := squirrel.Insert("subscription").
		Columns("user_id", "service_id", "price_rub", "start_date", "end_date").
		Values(p.UserID, p.ServiceID, p.PriceRub, p.StartDate, p.EndDate).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int64
	err = db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return 0, ErrSubscriptionAlreadyExists
			case pgerrcode.ForeignKeyViolation:
				return 0, ErrServiceNotFound
			case pgerrcode.CheckViolation:
				return 0, err
			}
		}
		return 0, err
	}

	if id == 0 {
		return 0, ErrSubscriptionNotCreated
	}

	return id, nil
}

func GetSubscription(ctx context.Context, db *sql.DB, id int64) (Subscription, error) {
	query, args, err := squirrel.Select(
		"s.id", "sv.name", "s.price_rub", "s.user_id", "s.start_date", "s.end_date",
	).
		From("subscription s").
		Join("service sv ON s.service_id = sv.id").
		Where(squirrel.Eq{"s.id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return Subscription{}, fmt.Errorf("could not build query: %w", err)
	}

	var subscription Subscription
	err = db.QueryRowContext(ctx, query, args...).Scan(
		&subscription.ID,
		&subscription.ServiceName,
		&subscription.Price,
		&subscription.UserID,
		&subscription.StartDate,
		&subscription.EndDate,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, ErrSubscriptionNotFound
		}
		return Subscription{}, err
	}

	return subscription, nil
}

func UpdateSubscription(ctx context.Context, db *sql.DB, p UpdateSubscriptionParams) error {
	queryBuilder := squirrel.Update("subscription").
		Where(squirrel.Eq{"id": p.ID}).
		PlaceholderFormat(squirrel.Dollar)

	if p.PriceRub != nil {
		queryBuilder = queryBuilder.Set("price_rub", *p.PriceRub)
	}

	if p.StartDate != nil {
		queryBuilder = queryBuilder.Set("start_date", *p.StartDate)
	}

	if p.EndDate != nil {
		queryBuilder = queryBuilder.Set("end_date", *p.EndDate)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query: %w", err)
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func DeleteSubscription(ctx context.Context, db *sql.DB, id int64) error {
	query, args, err := squirrel.Delete("subscription").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query: %w", err)
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func ListSubscriptions(ctx context.Context, db *sql.DB, p ListSubscriptionsParams) ([]Subscription, int, error) {
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

	if len(whereConditions) > 0 {
		baseQuery = baseQuery.Where(whereConditions)
	}

	// Count query
	countQuery := baseQuery.Columns("COUNT(*)")
	countSql, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("could not build count query: %w", err)
	}

	var total int
	if err := db.QueryRowContext(ctx, countSql, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := baseQuery.Columns(
		"s.id", "sv.name", "s.price_rub", "s.user_id", "s.start_date", "s.end_date",
	).OrderBy("s.id").Limit(uint64(p.Limit)).Offset(uint64(p.Offset))

	query, args, err := dataQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("could not build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var subscription Subscription
		if err := rows.Scan(
			&subscription.ID,
			&subscription.ServiceName,
			&subscription.Price,
			&subscription.UserID,
			&subscription.StartDate,
			&subscription.EndDate,
		); err != nil {
			return nil, 0, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, total, nil
}
