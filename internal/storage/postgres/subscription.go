package postgres

import (
	"EffectiveMobile/internal/storage"
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"strconv"
	"time"
)

type CreateSubscriptionParams struct {
	UserID    uuid.UUID
	ServiceID int
	PriceRub  int
	StartDate time.Time
	EndDate   *time.Time
}

func (s *Storage) CreateSubscription(ctx context.Context, p CreateSubscriptionParams, log *slog.Logger) (int64, error) {
	const op = "storage.postgres.CreateSubscription"
	log = log.With(slog.String("op", op))

	const q = `INSERT INTO subscription (user_id, service_id, price_rub, start_date, end_date)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id;`

	var id int64
	err := s.db.QueryRow(ctx, q,
		p.UserID,
		p.ServiceID,
		p.PriceRub,
		p.StartDate,
		p.EndDate,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return 0, storage.ErrSubscriptionAlreadyExists
			case pgerrcode.ForeignKeyViolation:
				return 0, storage.ErrServiceNotFound
			case pgerrcode.CheckViolation:
				return 0, err
			default:
				return 0, err
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storage.ErrSubscriptionNotCreated
		}
		return 0, err
	}

	return id, nil
}

type Subscription struct {
	ID          int64      `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

func (s *Storage) GetSubscription(ctx context.Context, id int64) (Subscription, error) {
	const q = `SELECT s.id, sv.name, s.price_rub, s.user_id, s.start_date, s.end_date
		FROM subscription s
		JOIN service sv ON s.service_id = sv.id
		WHERE s.id = $1`

	var sub Subscription
	err := s.db.QueryRow(ctx, q, id).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, storage.ErrSubscriptionNotFound
		}
		return Subscription{}, err
	}

	return sub, nil
}

type UpdateSubscriptionParams struct {
	ID        int64      `json:"id"`
	PriceRub  *int       `json:"price_rub"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

func (s *Storage) UpdateSubscription(ctx context.Context, p UpdateSubscriptionParams) error {
	const q = `UPDATE subscription 
		SET price_rub = $2, start_date = $3, end_date = $4
		WHERE id = $1`

	ct, err := s.db.Exec(ctx, q, p.ID, p.PriceRub, p.StartDate, p.EndDate)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return storage.ErrSubscriptionAlreadyExists
			case pgerrcode.CheckViolation:
				return err
			default:
				return err
			}
		}
		return err
	}

	if ct.RowsAffected() == 0 {
		return storage.ErrSubscriptionNotFound
	}

	return nil
}

func (s *Storage) DeleteSubscription(ctx context.Context, id int64) error {
	const q = `DELETE FROM subscription WHERE id = $1`

	ct, err := s.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return storage.ErrSubscriptionNotFound
	}

	return nil
}

type ListSubscriptionsParams struct {
	Limit       int        `json:"limit"`
	Offset      int        `json:"offset"`
	UserID      *uuid.UUID `json:"user_id"`
	ServiceName *string    `json:"service_name"`
}

func (s *Storage) ListSubscriptions(ctx context.Context, p ListSubscriptionsParams) ([]Subscription, int, error) {
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

	countQuery := `SELECT COUNT(*) FROM subscription s JOIN service sv ON s.service_id = sv.id ` + whereClause
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT s.id, sv.name, s.price_rub, s.user_id, s.start_date, s.end_date
		FROM subscription s
		JOIN service sv ON s.service_id = sv.id
		` + whereClause + `
		ORDER BY s.id
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	args = append(args, p.Limit, p.Offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&sub.StartDate,
			&sub.EndDate,
		)
		if err != nil {
			return nil, 0, err
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, total, nil
}
