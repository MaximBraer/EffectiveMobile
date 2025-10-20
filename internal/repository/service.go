package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func AddService(ctx context.Context, db *sql.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("empty service name")
	}

	query, args, err := squirrel.Insert("service").
		Columns("name").
		Values(name).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int
	err = db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return 0, ErrServiceNameExists
			}
		}
		return 0, err
	}
	return id, nil
}

func GetServiceName(ctx context.Context, db *sql.DB, id int) (string, error) {
	query, args, err := squirrel.Select("name").
		From("service").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("could not build query: %w", err)
	}

	var name string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrServiceNotFound
		}
		return "", err
	}
	return name, nil
}

func GetServiceID(ctx context.Context, db *sql.DB, name string) (int, error) {
	query, args, err := squirrel.Select("id").
		From("service").
		Where(squirrel.Eq{"name": name}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrServiceNotFound
		}
		return 0, err
	}
	return id, nil
}

func GetOrCreateServiceID(ctx context.Context, db *sql.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("empty service name")
	}

	query, args, err := squirrel.Insert("service").
		Columns("name").
		Values(name).
		Suffix("ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int
	err = db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func DeleteService(ctx context.Context, db *sql.DB, id int) error {
	query, args, err := squirrel.Delete("service").
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
		return ErrServiceNotFound
	}

	return nil
}
