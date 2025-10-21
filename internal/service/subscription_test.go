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

type SubscriptionServiceSuite struct {
	suite.Suite

	ctrl                *gomock.Controller
	serviceRepo         *MockServicesRepository
	subscriptionRepo    *MockSubscriptionRepository
	subscriptionService *SubscriptionService
	logger              *slog.Logger
	ctx                 context.Context
}

func TestSubscriptionService(t *testing.T) {
	suite.Run(t, &SubscriptionServiceSuite{})
}

func (s *SubscriptionServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.serviceRepo = NewMockServicesRepository(s.ctrl)
	s.subscriptionRepo = NewMockSubscriptionRepository(s.ctrl)

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	s.ctx = context.Background()

	s.subscriptionService = NewSubscriptionService(s.serviceRepo, s.subscriptionRepo, s.logger)
}

func (s *SubscriptionServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *SubscriptionServiceSuite) TestCreateSubscription_Success() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"
	serviceID := 1
	subscriptionID := int64(123)

	s.serviceRepo.EXPECT().
		GetOrCreateServiceID(s.ctx, serviceName).
		Return(serviceID, nil)

	s.subscriptionRepo.EXPECT().
		CreateSubscription(gomock.Any(), repository.CreateSubscriptionParams{
			UserID:    userID,
			ServiceID: serviceID,
			PriceRub:  price,
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   nil,
		}).
		Return(subscriptionID, nil)

	result, err := s.subscriptionService.CreateSubscription(s.ctx, serviceName, price, userID, startDate, "")

	s.NoError(err)
	s.Equal(subscriptionID, result)
}

func (s *SubscriptionServiceSuite) TestCreateSubscription_WithEndDate() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"
	endDate := "03-2024"
	serviceID := 1
	subscriptionID := int64(123)

	s.serviceRepo.EXPECT().
		GetOrCreateServiceID(s.ctx, serviceName).
		Return(serviceID, nil)

	s.subscriptionRepo.EXPECT().
		CreateSubscription(gomock.Any(), repository.CreateSubscriptionParams{
			UserID:    userID,
			ServiceID: serviceID,
			PriceRub:  price,
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   &[]time.Time{time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}[0],
		}).
		Return(subscriptionID, nil)

	result, err := s.subscriptionService.CreateSubscription(s.ctx, serviceName, price, userID, startDate, endDate)

	s.NoError(err)
	s.Equal(subscriptionID, result)
}

func (s *SubscriptionServiceSuite) TestCreateSubscription_ServiceError() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"
	serviceError := errors.New("service error")

	s.serviceRepo.EXPECT().
		GetOrCreateServiceID(s.ctx, serviceName).
		Return(0, serviceError)

	result, err := s.subscriptionService.CreateSubscription(s.ctx, serviceName, price, userID, startDate, "")

	s.Error(err)
	s.Equal(int64(0), result)
	s.Contains(err.Error(), "service error")
}

func (s *SubscriptionServiceSuite) TestCreateSubscription_InvalidDate() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "invalid-date"

	result, err := s.subscriptionService.CreateSubscription(s.ctx, serviceName, price, userID, startDate, "")

	s.Error(err)
	s.Equal(int64(0), result)
	s.Contains(err.Error(), "invalid date format")
}

func (s *SubscriptionServiceSuite) TestCreateSubscription_EndDateBeforeStartDate() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "03-2024"
	endDate := "01-2024"

	result, err := s.subscriptionService.CreateSubscription(s.ctx, serviceName, price, userID, startDate, endDate)

	s.Error(err)
	s.Equal(int64(0), result)
	s.Contains(err.Error(), "end date must be after start date")
}

func (s *SubscriptionServiceSuite) TestGetSubscription_Success() {
	subscriptionID := int64(123)
	expectedSubscription := repository.Subscription{
		ID:          subscriptionID,
		UserID:      uuid.New(),
		ServiceName: "Netflix",
		Price:       500,
		StartDate:   time.Now(),
		EndDate:     nil,
	}

	s.subscriptionRepo.EXPECT().
		GetSubscription(s.ctx, subscriptionID).
		Return(expectedSubscription, nil)

	result, err := s.subscriptionService.GetSubscription(s.ctx, subscriptionID)

	s.NoError(err)
	s.Equal(&expectedSubscription, result)
}

func (s *SubscriptionServiceSuite) TestGetSubscription_NotFound() {
	subscriptionID := int64(123)
	notFoundError := repository.ErrSubscriptionNotFound

	s.subscriptionRepo.EXPECT().
		GetSubscription(s.ctx, subscriptionID).
		Return(repository.Subscription{}, notFoundError)

	result, err := s.subscriptionService.GetSubscription(s.ctx, subscriptionID)

	s.Error(err)
	s.Nil(result)
	s.Equal(notFoundError, err)
}

func (s *SubscriptionServiceSuite) TestUpdateSubscription_Success() {
	subscriptionID := int64(123)
	price := 600
	startDate := "02-2024"
	endDate := "04-2024"

	s.subscriptionRepo.EXPECT().
		UpdateSubscription(s.ctx, repository.UpdateSubscriptionParams{
			ID:        subscriptionID,
			ServiceID: nil,
			PriceRub:  &price,
			StartDate: &[]time.Time{time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)}[0],
			EndDate:   &[]time.Time{time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)}[0],
		}).
		Return(nil)

	err := s.subscriptionService.UpdateSubscription(s.ctx, subscriptionID, nil, &price, &startDate, &endDate)

	s.NoError(err)
}

func (s *SubscriptionServiceSuite) TestUpdateSubscription_NotFound() {
	subscriptionID := int64(123)
	price := 600
	startDate := "02-2024"
	notFoundError := repository.ErrSubscriptionNotFound

	s.subscriptionRepo.EXPECT().
		UpdateSubscription(s.ctx, repository.UpdateSubscriptionParams{
			ID:        subscriptionID,
			ServiceID: nil,
			PriceRub:  &price,
			StartDate: &[]time.Time{time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)}[0],
			EndDate:   nil,
		}).
		Return(notFoundError)

	err := s.subscriptionService.UpdateSubscription(s.ctx, subscriptionID, nil, &price, &startDate, nil)

	s.Error(err)
	s.Equal(notFoundError, err)
}

func (s *SubscriptionServiceSuite) TestUpdateSubscription_WithServiceName() {
	subscriptionID := int64(123)
	serviceName := "Spotify"
	serviceID := 5
	price := 700

	s.serviceRepo.EXPECT().
		GetOrCreateServiceID(s.ctx, serviceName).
		Return(serviceID, nil)

	s.subscriptionRepo.EXPECT().
		UpdateSubscription(s.ctx, repository.UpdateSubscriptionParams{
			ID:        subscriptionID,
			ServiceID: &serviceID,
			PriceRub:  &price,
			StartDate: nil,
			EndDate:   nil,
		}).
		Return(nil)

	err := s.subscriptionService.UpdateSubscription(s.ctx, subscriptionID, &serviceName, &price, nil, nil)

	s.NoError(err)
}

func (s *SubscriptionServiceSuite) TestUpdateSubscription_Conflict() {
	subscriptionID := int64(123)
	startDate := "02-2024"
	conflictError := repository.ErrSubscriptionAlreadyExists

	s.subscriptionRepo.EXPECT().
		UpdateSubscription(s.ctx, repository.UpdateSubscriptionParams{
			ID:        subscriptionID,
			ServiceID: nil,
			PriceRub:  nil,
			StartDate: &[]time.Time{time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)}[0],
			EndDate:   nil,
		}).
		Return(conflictError)

	err := s.subscriptionService.UpdateSubscription(s.ctx, subscriptionID, nil, nil, &startDate, nil)

	s.Error(err)
	s.Equal(conflictError, err)
}

func (s *SubscriptionServiceSuite) TestDeleteSubscription_Success() {
	subscriptionID := int64(123)

	s.subscriptionRepo.EXPECT().
		DeleteSubscription(s.ctx, subscriptionID).
		Return(nil)

	err := s.subscriptionService.DeleteSubscription(s.ctx, subscriptionID)

	s.NoError(err)
}

func (s *SubscriptionServiceSuite) TestDeleteSubscription_NotFound() {
	subscriptionID := int64(123)
	notFoundError := repository.ErrSubscriptionNotFound

	s.subscriptionRepo.EXPECT().
		DeleteSubscription(s.ctx, subscriptionID).
		Return(notFoundError)

	err := s.subscriptionService.DeleteSubscription(s.ctx, subscriptionID)

	s.Error(err)
	s.Equal(notFoundError, err)
}

func (s *SubscriptionServiceSuite) TestListSubscriptions_Success() {
	userID := uuid.New()
	limit := 10
	offset := 0

	expectedSubscriptions := []repository.Subscription{
		{
			ID:          1,
			UserID:      userID,
			ServiceName: "Netflix",
			Price:       500,
			StartDate:   time.Now(),
			EndDate:     nil,
		},
	}
	expectedTotal := 1

	s.subscriptionRepo.EXPECT().
		ListSubscriptions(s.ctx, repository.ListSubscriptionsParams{
			UserID: &userID,
			Limit:  limit,
			Offset: offset,
		}).
		Return(expectedSubscriptions, expectedTotal, nil)

	subscriptions, total, err := s.subscriptionService.ListSubscriptions(s.ctx, repository.ListSubscriptionsParams{
		UserID: &userID,
		Limit:  limit,
		Offset: offset,
	})

	s.NoError(err)
	s.Equal(expectedSubscriptions, subscriptions)
	s.Equal(expectedTotal, total)
}

func (s *SubscriptionServiceSuite) TestListSubscriptions_Error() {
	userID := uuid.New()
	limit := 10
	offset := 0
	repoError := errors.New("repository error")

	s.subscriptionRepo.EXPECT().
		ListSubscriptions(s.ctx, repository.ListSubscriptionsParams{
			UserID: &userID,
			Limit:  limit,
			Offset: offset,
		}).
		Return(nil, 0, repoError)

	subscriptions, total, err := s.subscriptionService.ListSubscriptions(s.ctx, repository.ListSubscriptionsParams{
		UserID: &userID,
		Limit:  limit,
		Offset: offset,
	})

	s.Error(err)
	s.Nil(subscriptions)
	s.Equal(0, total)
	s.Equal(repoError, err)
}
