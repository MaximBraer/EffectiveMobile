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

var (
	ErrServiceNotFound   = errors.New("service not found")
	ErrServiceNameExists = errors.New("service name already exists")
	ErrServiceInUse      = errors.New("service is referenced by subscriptions")
	ErrInvalidDateFormat = errors.New("invalid date format")
)

type Provider interface {
	GetConn() *sql.DB
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type ServiceRepository struct {
	provider Provider
	logger   Logger
}

func NewServiceRepository(provider Provider, logger Logger) *ServiceRepository {
	return &ServiceRepository{
		provider: provider,
		logger:   logger,
	}
}

func (r *ServiceRepository) AddService(ctx context.Context, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("empty service name")
	}

	query, args, err := squirrel.Insert("service").
		Columns("name").
		Values(name).
		PlaceholderFormat(squirrel.Dollar).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int
	if err := r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrServiceNameExists
		}
		return 0, err
	}
	return id, nil
}

func (r *ServiceRepository) GetServiceName(ctx context.Context, id int) (string, error) {
	query, args, err := squirrel.Select("name").
		From("service").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("could not build query: %w", err)
	}

	var name string
	if err := r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrServiceNotFound
		}
		return "", err
	}
	return name, nil
}

func (r *ServiceRepository) GetServiceID(ctx context.Context, name string) (int, error) {
	query, args, err := squirrel.Select("id").
		From("service").
		Where(squirrel.Eq{"name": name}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query: %w", err)
	}

	var id int
	if err := r.provider.GetConn().QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrServiceNotFound
		}
		return 0, err
	}
	return id, nil
}

func (r *ServiceRepository) GetOrCreateServiceID(ctx context.Context, name string) (int, error) {
	id, err := r.GetServiceID(ctx, name)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, ErrServiceNotFound) {
		return 0, err
	}
	return r.AddService(ctx, name)
}

func (r *ServiceRepository) DeleteService(ctx context.Context, id int) error {
	checkQuery, checkArgs, err := squirrel.Select("COUNT(*)").
		From("subscription").
		Where(squirrel.Eq{"service_id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build check query: %w", err)
	}

	var count int
	if err := r.provider.GetConn().QueryRowContext(ctx, checkQuery, checkArgs...).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return ErrServiceInUse
	}

	query, args, err := squirrel.Delete("service").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query: %w", err)
	}

	result, err := r.provider.GetConn().ExecContext(ctx, query, args...)
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
