package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Provider interface {
	GetConn() *sql.DB
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type PostgresStorage struct {
	provider Provider
	logger   Logger
}

var (
	ErrServiceNotFound           = errors.New("service not found")
	ErrServiceNameExists         = errors.New("service name already exists")
	ErrServiceInUse              = errors.New("service is referenced by subscriptions")
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

type GetTotalCostParams struct {
	UserID      *uuid.UUID
	ServiceName *string
	StartDate   *time.Time
	EndDate     *time.Time
}

type Subscription struct {
	ID          int64
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartDate   time.Time
	EndDate     *time.Time
}

type TotalCostStats struct {
	TotalCost          int
	StartDate          *time.Time
	EndDate            *time.Time
	UserID             *uuid.UUID
	ServiceName        *string
	SubscriptionsCount int
}

func New(provider Provider, logger Logger) *PostgresStorage {
	return &PostgresStorage{
		provider: provider,
		logger:   logger,
	}
}

func (s *PostgresStorage) Close() error {
	return s.provider.GetConn().Close()
}

func (s *PostgresStorage) AddService(ctx context.Context, name string) (int, error) {
	return AddService(ctx, s.provider.GetConn(), name)
}

func (s *PostgresStorage) GetServiceName(ctx context.Context, id int) (string, error) {
	return GetServiceName(ctx, s.provider.GetConn(), id)
}

func (s *PostgresStorage) GetServiceID(ctx context.Context, name string) (int, error) {
	return GetServiceID(ctx, s.provider.GetConn(), name)
}

func (s *PostgresStorage) GetOrCreateServiceID(ctx context.Context, name string) (int, error) {
	return GetOrCreateServiceID(ctx, s.provider.GetConn(), name)
}

func (s *PostgresStorage) DeleteService(ctx context.Context, id int) error {
	return DeleteService(ctx, s.provider.GetConn(), id)
}

func (s *PostgresStorage) CreateSubscription(ctx context.Context, p CreateSubscriptionParams, log *slog.Logger) (int64, error) {
	return CreateSubscription(ctx, s.provider.GetConn(), p, log)
}

func (s *PostgresStorage) GetSubscription(ctx context.Context, id int64) (Subscription, error) {
	return GetSubscription(ctx, s.provider.GetConn(), id)
}

func (s *PostgresStorage) UpdateSubscription(ctx context.Context, p UpdateSubscriptionParams) error {
	return UpdateSubscription(ctx, s.provider.GetConn(), p)
}

func (s *PostgresStorage) DeleteSubscription(ctx context.Context, id int64) error {
	return DeleteSubscription(ctx, s.provider.GetConn(), id)
}

func (s *PostgresStorage) ListSubscriptions(ctx context.Context, p ListSubscriptionsParams) ([]Subscription, int, error) {
	return ListSubscriptions(ctx, s.provider.GetConn(), p)
}

func (s *PostgresStorage) GetTotalCost(ctx context.Context, p GetTotalCostParams) (TotalCostStats, error) {
	return GetTotalCost(ctx, s.provider.GetConn(), p)
}
