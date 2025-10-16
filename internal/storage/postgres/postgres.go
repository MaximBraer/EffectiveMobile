package postgres

import (
	"EffectiveMobile/internal/config"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"time"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, scfg config.Storage) (*Storage, error) {
	const op = "storage.postgres.New"

	// postgres://user:pass@host:port/db?sslmode=disable
	user := url.QueryEscape(scfg.User)
	pass := url.QueryEscape(scfg.Password)
	host := scfg.Address
	db := url.PathEscape(scfg.Database)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, host, db)

	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: parse config: %w", op, err)
	}

	if scfg.MinConns > 0 {
		pcfg.MinConns = int32(scfg.MinConns)
	}
	if scfg.MaxConns > 0 {
		pcfg.MaxConns = int32(scfg.MaxConns)
	}
	if scfg.HealthCheckPeriod > 0 {
		pcfg.HealthCheckPeriod = scfg.HealthCheckPeriod
	}
	if scfg.MaxConnLifetime > 0 {
		pcfg.MaxConnLifetime = scfg.MaxConnLifetime
	}
	if scfg.MaxConnIdleTime > 0 {
		pcfg.MaxConnIdleTime = scfg.MaxConnIdleTime
	}

	if pcfg.ConnConfig.RuntimeParams == nil {
		pcfg.ConnConfig.RuntimeParams = map[string]string{}
	}

	pcfg.ConnConfig.RuntimeParams["application_name"] = "subscriptions"
	pcfg.ConnConfig.RuntimeParams["statement_timeout"] = scfg.StatementTimeout
	pcfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = scfg.IdleInTransactionSessionTimeout

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("%s: new pool: %w", op, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	return &Storage{db: pool}, nil
}

func (s *Storage) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}
