package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"EffectiveMobile/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type StatsServiceSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	statsRepo    *MockStatsRepository
	statsService *StatsService
	logger       *slog.Logger
	ctx          context.Context
}

func TestStatsService(t *testing.T) {
	suite.Run(t, &StatsServiceSuite{})
}

func (s *StatsServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.statsRepo = NewMockStatsRepository(s.ctrl)

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	s.ctx = context.Background()

	s.statsService = NewStatsService(s.statsRepo, s.logger)
}

func (s *StatsServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *StatsServiceSuite) TestGetTotalCost_Success() {
	userID := uuid.New()
	serviceName := "Netflix"
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	subscriptions := []repository.SubscriptionCost{
		{
			ID:          1,
			StartDate:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     timePtr(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)),
			PriceRub:    400,
			UserID:      userID,
			ServiceName: serviceName,
		},
		{
			ID:          2,
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     timePtr(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
			PriceRub:    500,
			UserID:      userID,
			ServiceName: serviceName,
		},
	}

	repoStats := repository.TotalCostStats{
		Subscriptions:      subscriptions,
		UserID:             &userID,
		ServiceName:        &serviceName,
		StartDate:          &startDate,
		EndDate:            &endDate,
		SubscriptionsCount: 2,
	}

	expectedStats := repository.TotalCostStats{
		TotalCost:          3100,
		Subscriptions:      subscriptions,
		UserID:             &userID,
		ServiceName:        &serviceName,
		StartDate:          &startDate,
		EndDate:            &endDate,
		SubscriptionsCount: 2,
	}

	s.statsRepo.EXPECT().
		GetTotalCost(s.ctx, repository.GetTotalCostParams{
			UserID:      &userID,
			ServiceName: &serviceName,
			StartDate:   &startDate,
			EndDate:     &endDate,
		}).
		Return(repoStats, nil)

	result, err := s.statsService.GetTotalCost(s.ctx, &userID, &serviceName, &startDate, &endDate)

	s.NoError(err)
	s.Equal(&expectedStats, result)
}

func (s *StatsServiceSuite) TestGetTotalCost_WithPartialAndOutsidePeriod() {
	userID := uuid.New()
	serviceName := "Netflix"
	startDate := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	subscriptions := []repository.SubscriptionCost{
		{
			ID:          1,
			StartDate:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     timePtr(time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)),
			PriceRub:    100,
			UserID:      userID,
			ServiceName: serviceName,
		},
		{
			ID:          2,
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     timePtr(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)),
			PriceRub:    200,
			UserID:      userID,
			ServiceName: serviceName,
		},
		{
			ID:          3,
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     timePtr(time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)),
			PriceRub:    300,
			UserID:      userID,
			ServiceName: serviceName,
		},
	}

	repoStats := repository.TotalCostStats{
		Subscriptions:      subscriptions,
		UserID:             &userID,
		ServiceName:        &serviceName,
		StartDate:          &startDate,
		EndDate:            &endDate,
		SubscriptionsCount: 3,
	}

	expectedStats := repository.TotalCostStats{
		TotalCost:          700,
		Subscriptions:      subscriptions,
		UserID:             &userID,
		ServiceName:        &serviceName,
		StartDate:          &startDate,
		EndDate:            &endDate,
		SubscriptionsCount: 3,
	}

	s.statsRepo.EXPECT().
		GetTotalCost(s.ctx, repository.GetTotalCostParams{
			UserID:      &userID,
			ServiceName: &serviceName,
			StartDate:   &startDate,
			EndDate:     &endDate,
		}).
		Return(repoStats, nil)

	result, err := s.statsService.GetTotalCost(s.ctx, &userID, &serviceName, &startDate, &endDate)

	s.NoError(err)
	s.Equal(&expectedStats, result)
}

func (s *StatsServiceSuite) TestGetTotalCost_RepositoryError() {
	userID := uuid.New()
	serviceName := "Netflix"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	repoError := errors.New("database error")

	s.statsRepo.EXPECT().
		GetTotalCost(s.ctx, repository.GetTotalCostParams{
			UserID:      &userID,
			ServiceName: &serviceName,
			StartDate:   &startDate,
			EndDate:     &endDate,
		}).
		Return(repository.TotalCostStats{}, repoError)

	result, err := s.statsService.GetTotalCost(s.ctx, &userID, &serviceName, &startDate, &endDate)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "database error")
}

func (s *StatsServiceSuite) TestParseMonth_Success() {
	dateStr := "01-2024"
	expectedDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	result, err := s.statsService.ParseMonth(dateStr)

	s.NoError(err)
	s.Equal(expectedDate, result)
}

func (s *StatsServiceSuite) TestParseMonth_InvalidFormat() {
	dateStr := "invalid-date"

	result, err := s.statsService.ParseMonth(dateStr)

	s.Error(err)
	s.Zero(result)
	s.Contains(err.Error(), "invalid date format")
}

func (s *StatsServiceSuite) TestParseMonth_InvalidMonth() {
	dateStr := "13-2024"

	result, err := s.statsService.ParseMonth(dateStr)

	s.Error(err)
	s.Zero(result)
	s.Contains(err.Error(), "invalid date format")
}

func (s *StatsServiceSuite) TestFormatDate_Success() {
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expected := "01-2024"

	result := s.statsService.FormatDate(&date)

	s.Equal(expected, result)
}

func (s *StatsServiceSuite) TestFormatDate_NilDate() {
	result := s.statsService.FormatDate(nil)

	s.Equal("", result)
}

func (s *StatsServiceSuite) TestFormatUUID_Success() {
	uid := uuid.New()
	expected := uid.String()

	result := s.statsService.FormatUUID(&uid)

	s.Equal(&expected, result)
}

func (s *StatsServiceSuite) TestFormatUUID_NilUUID() {
	result := s.statsService.FormatUUID(nil)

	s.Nil(result)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
