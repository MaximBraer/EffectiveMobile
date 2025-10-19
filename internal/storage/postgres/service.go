package postgres

import (
	"EffectiveMobile/internal/storage"
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Storage) AddService(ctx context.Context, name string) (int, error) {
	const q = `INSERT INTO service(name) VALUES($1) RETURNING id`
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("empty service name")
	}

	var id int
	if err := s.db.QueryRow(ctx, q, name).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, storage.ErrServiceNameExists
		}
		return 0, err
	}
	return id, nil
}

func (s *Storage) GetServiceName(ctx context.Context, id int) (string, error) {
	const q = `SELECT name FROM service WHERE id = $1`

	var name string
	if err := s.db.QueryRow(ctx, q, id).Scan(&name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", storage.ErrServiceNotFound
		}
		return "", err
	}
	return name, nil
}

func (s *Storage) GetServiceID(ctx context.Context, name string) (int, error) {
	const q = `SELECT id FROM service WHERE name = $1`

	var id int
	if err := s.db.QueryRow(ctx, q, name).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storage.ErrServiceNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *Storage) GetOrCreateServiceID(ctx context.Context, name string) (int, error) {
	id, err := s.GetServiceID(ctx, name)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, storage.ErrServiceNotFound) {
		return 0, err
	}
	return s.AddService(ctx, name)
}

func (s *Storage) DeleteService(ctx context.Context, id int) error {
	const q = `DELETE FROM service WHERE id = $1`

	ct, err := s.db.Exec(ctx, q, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return storage.ErrServiceInUse
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return storage.ErrServiceNotFound
	}
	return nil
}
