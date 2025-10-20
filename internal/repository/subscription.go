package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionNotCreated    = errors.New("subscription not created")
)

type CreateSubscriptionParams struct {
	UserID    uuid.UUID
	ServiceID int
	PriceRub  int
	StartDate time.Time
	EndDate   *time.Time
}

type UpdateSubscriptionParams struct {
	ID        int64
	PriceRub  *int
	StartDate *time.Time
	EndDate   *time.Time
}

type ListSubscriptionsParams struct {
	Limit       int
	Offset      int
	UserID      *uuid.UUID
	ServiceName *string
}

type Subscription struct {
	ID          int64
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartDate   time.Time
	EndDate     *time.Time
}

type SubscriptionRepository struct {
	provider Provider
	logger   Logger
}

func NewSubscriptionRepository(provider Provider, logger Logger) *SubscriptionRepository {
	return &SubscriptionRepository{
		provider: provider,
		logger:   logger,
	}
}

func (r *SubscriptionRepository) CreateSubscription(ctx context.Context, p CreateSubscriptionParams, log *slog.Logger) (int64, error) {
	const op = "repository.subscription.CreateSubscription"
	log = log.With(slog.String("op", op))

	checkQuery, checkArgs, err := squirrel.Select("id").
		From("subscription").
		Where(squirrel.Eq{
			"user_id":    p.UserID,
			"service_id": p.ServiceID,
			"start_date": p.StartDate,
		}).
		Where(func() squirrel.Eq {
			if p.EndDate != nil {
				return squirrel.Eq{"end_date": p.EndDate}
			}
			return squirrel.Eq{"end_date": nil}
		}()).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build check query: %w", err)
	}

	var existingID int64
	err = r.provider.GetConn().QueryRowContext(ctx, checkQuery, checkArgs...).Scan(&existingID)
	if err == nil {
		return 0, ErrSubscriptionAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("could not check existing subscription: %w", err)
	}

	query, args, err := squirrel.Insert("subscription").
		Columns("user_id", "service_id", "price_rub", "start_date", "end_date").
		Values(p.UserID, p.ServiceID, p.PriceRub, p.StartDate, p.EndDate).
		PlaceholderFormat(squirrel.Dollar).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int64
	if err := r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrSubscriptionAlreadyExists
		}
		return 0, err
	}

	if id == 0 {
		return 0, ErrSubscriptionNotCreated
	}

	return id, nil
}

func (r *SubscriptionRepository) GetSubscription(ctx context.Context, id int64) (Subscription, error) {
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
	err = r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(
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

func (r *SubscriptionRepository) UpdateSubscription(ctx context.Context, p UpdateSubscriptionParams) error {
	queryBuilder := squirrel.Update("subscription")

	if p.PriceRub != nil {
		queryBuilder = queryBuilder.Set("price_rub", *p.PriceRub)
	}

	if p.StartDate != nil {
		queryBuilder = queryBuilder.Set("start_date", *p.StartDate)
	}

	if p.EndDate != nil {
		queryBuilder = queryBuilder.Set("end_date", *p.EndDate)
	}

	queryBuilder = queryBuilder.Where(squirrel.Eq{"id": p.ID})

	query, args, err := queryBuilder.PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query: %w", err)
	}

	result, err := r.provider.GetConn().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("result.RowsAffected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (r *SubscriptionRepository) DeleteSubscription(ctx context.Context, id int64) error {
	query, args, err := squirrel.Delete("subscription").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query: %w", err)
	}

	result, err := r.provider.GetConn().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("result.RowsAffected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (r *SubscriptionRepository) ListSubscriptions(ctx context.Context, p ListSubscriptionsParams) ([]Subscription, int, error) {
    countBuilder := squirrel.Select("COUNT(*)").
        From("subscription s").
        Join("service sv ON s.service_id = sv.id").
        PlaceholderFormat(squirrel.Dollar)

    if p.UserID != nil {
        countBuilder = countBuilder.Where(squirrel.Eq{"s.user_id": *p.UserID})
    }
    if p.ServiceName != nil {
        countBuilder = countBuilder.Where(squirrel.Eq{"sv.name": *p.ServiceName})
    }

    countQuery, countArgs, err := countBuilder.ToSql()
    if err != nil {
        return nil, 0, fmt.Errorf("could not build count query: %w", err)
    }

    var total int
    if err := r.provider.GetConn().QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("failed to get count: %w", err)
    }

    // Данные
    dataBuilder := squirrel.Select(
        "s.id", "sv.name", "s.price_rub", "s.user_id", "s.start_date", "s.end_date",
    ).
        From("subscription s").
        Join("service sv ON s.service_id = sv.id").
        PlaceholderFormat(squirrel.Dollar)

    if p.UserID != nil {
        dataBuilder = dataBuilder.Where(squirrel.Eq{"s.user_id": *p.UserID})
    }
    if p.ServiceName != nil {
        dataBuilder = dataBuilder.Where(squirrel.Eq{"sv.name": *p.ServiceName})
    }

    if p.Limit > 0 {
        dataBuilder = dataBuilder.Limit(uint64(p.Limit))
    }
    if p.Offset >= 0 {
        dataBuilder = dataBuilder.Offset(uint64(p.Offset))
    }
    dataBuilder = dataBuilder.OrderBy("s.id")

    query, args, err := dataBuilder.ToSql()
    if err != nil {
        return nil, 0, fmt.Errorf("could not build query: %w", err)
    }

    rows, err := r.provider.GetConn().QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to execute query: %w", err)
    }
    defer rows.Close()

    var subscriptions []Subscription
    for rows.Next() {
        var subscription Subscription
        err = rows.Scan(
            &subscription.ID,
            &subscription.ServiceName,
            &subscription.Price,
            &subscription.UserID,
            &subscription.StartDate,
            &subscription.EndDate,
        )
        if err != nil {
            return nil, 0, fmt.Errorf("failed to scan row: %w", err)
        }
        subscriptions = append(subscriptions, subscription)
    }

    return subscriptions, total, nil
}
