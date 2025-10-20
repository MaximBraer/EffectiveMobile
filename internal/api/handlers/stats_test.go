package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"EffectiveMobile/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type StatsHandlersSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	subscriptionService *MockSubscriptionService
	statsService *MockStatsService
	logger       *slog.Logger
	ctx          context.Context
}

func TestStatsHandlers(t *testing.T) {
	suite.Run(t, &StatsHandlersSuite{})
}

func (s *StatsHandlersSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.subscriptionService = NewMockSubscriptionService(s.ctrl)
	s.statsService = NewMockStatsService(s.ctrl)
	
	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	s.ctx = context.Background()
}

func (s *StatsHandlersSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *StatsHandlersSuite) TestGetTotalStats_Success() {
	userID := uuid.New()
	serviceName := "Netflix"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedStats := &repository.TotalCostStats{
		TotalCost:           5000,
		UserID:              &userID,
		ServiceName:         &serviceName,
		StartDate:           &startDate,
		EndDate:             &endDate,
		SubscriptionsCount:  10,
	}

	req := httptest.NewRequest("GET", "/stats/total?user_id="+userID.String()+"&service_name=Netflix&start_date=01-2024&end_date=12-2024", nil)
	w := httptest.NewRecorder()

	s.statsService.EXPECT().
		ParseMonth("01-2024").
		Return(startDate, nil)

	s.statsService.EXPECT().
		ParseMonth("12-2024").
		Return(endDate, nil)

	s.statsService.EXPECT().
		GetTotalCost(gomock.Any(), &userID, &serviceName, &startDate, &endDate).
		Return(expectedStats, nil)

	s.statsService.EXPECT().
		FormatDate(&startDate).
		Return("2024-01-01")

	s.statsService.EXPECT().
		FormatDate(&endDate).
		Return("2024-12-31")

	s.statsService.EXPECT().
		FormatUUID(&userID).
		Return(&[]string{userID.String()}[0])

	GetTotalStats(s.subscriptionService, s.statsService, s.logger)(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response GetTotalStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(expectedStats.TotalCost, response.TotalCost)
	s.Equal(expectedStats.SubscriptionsCount, response.SubscriptionsCount)
}

func (s *StatsHandlersSuite) TestGetTotalStats_ServiceError() {
	userID := uuid.New()
	serviceName := "Netflix"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	req := httptest.NewRequest("GET", "/stats/total?user_id="+userID.String()+"&service_name=Netflix&start_date=01-2024&end_date=12-2024", nil)
	w := httptest.NewRecorder()

	s.statsService.EXPECT().
		ParseMonth("01-2024").
		Return(startDate, nil)

	s.statsService.EXPECT().
		ParseMonth("12-2024").
		Return(endDate, nil)

	s.statsService.EXPECT().
		GetTotalCost(gomock.Any(), &userID, &serviceName, &startDate, &endDate).
		Return(nil, repository.ErrSubscriptionNotCreated)

	GetTotalStats(s.subscriptionService, s.statsService, s.logger)(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)
}

func (s *StatsHandlersSuite) TestGetTotalStats_InvalidDate() {
	userID := uuid.New()

	req := httptest.NewRequest("GET", "/stats/total?user_id="+userID.String()+"&start_date=invalid-date", nil)
	w := httptest.NewRecorder()

	s.statsService.EXPECT().
		ParseMonth("invalid-date").
		Return(time.Time{}, repository.ErrInvalidDateFormat)

	GetTotalStats(s.subscriptionService, s.statsService, s.logger)(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *StatsHandlersSuite) TestGetTotalStats_EndDateBeforeStartDate() {
	userID := uuid.New()
	startDate := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	req := httptest.NewRequest("GET", "/stats/total?user_id="+userID.String()+"&start_date=12-2024&end_date=01-2024", nil)
	w := httptest.NewRecorder()

	s.statsService.EXPECT().
		ParseMonth("12-2024").
		Return(startDate, nil)

	s.statsService.EXPECT().
		ParseMonth("01-2024").
		Return(endDate, nil)

	GetTotalStats(s.subscriptionService, s.statsService, s.logger)(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}
