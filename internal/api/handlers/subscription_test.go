package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"EffectiveMobile/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SubscriptionHandlersSuite struct {
	suite.Suite

	ctrl                *gomock.Controller
	subscriptionService *MockSubscriptionService
	statsService        *MockStatsService
	logger              *slog.Logger
	ctx                 context.Context
}

func TestSubscriptionHandlers(t *testing.T) {
	suite.Run(t, &SubscriptionHandlersSuite{})
}

func (s *SubscriptionHandlersSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.subscriptionService = NewMockSubscriptionService(s.ctrl)
	s.statsService = NewMockStatsService(s.ctrl)

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	s.ctx = context.Background()
}

func (s *SubscriptionHandlersSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *SubscriptionHandlersSuite) TestSaveSubscription_Success() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"
	subscriptionID := int64(123)

	requestBody := CreateSubscriptionRequest{
		ServiceName: serviceName,
		Price:       price,
		UserID:      userID,
		StartDate:   startDate,
	}

	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		CreateSubscription(gomock.Any(), serviceName, price, userID, startDate, "").
		Return(subscriptionID, nil)

    SaveSubscription(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusCreated, w.Code)

	var response CreateSubscriptionResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(subscriptionID, response.ID)
}

func (s *SubscriptionHandlersSuite) TestSaveSubscription_InvalidJSON() {
	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

    SaveSubscription(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *SubscriptionHandlersSuite) TestSaveSubscription_ServiceError() {
	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"

	requestBody := CreateSubscriptionRequest{
		ServiceName: serviceName,
		Price:       price,
		UserID:      userID,
		StartDate:   startDate,
	}

	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)

	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		CreateSubscription(gomock.Any(), serviceName, price, userID, startDate, "").
		Return(int64(0), repository.ErrSubscriptionNotCreated)

    SaveSubscription(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)
}

func (s *SubscriptionHandlersSuite) TestGetSubscription_Success() {
	subscriptionID := int64(123)
	expectedSubscription := &repository.Subscription{
		ID:          subscriptionID,
		UserID:      uuid.New(),
		ServiceName: "Netflix",
		Price:       500,
		StartDate:   time.Now(),
		EndDate:     nil,
	}

    router := chi.NewRouter()
    router.Get("/subscriptions/{id}", GetSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("GET", "/subscriptions/123", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		GetSubscription(gomock.Any(), subscriptionID).
		Return(expectedSubscription, nil)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response GetSubscriptionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(expectedSubscription.ID, response.ID)
	s.Equal(expectedSubscription.ServiceName, response.ServiceName)
	s.Equal(expectedSubscription.Price, response.Price)
}

func (s *SubscriptionHandlersSuite) TestGetSubscription_NotFound() {
	subscriptionID := int64(123)

    router := chi.NewRouter()
    router.Get("/subscriptions/{id}", GetSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("GET", "/subscriptions/123", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		GetSubscription(gomock.Any(), subscriptionID).
		Return(nil, repository.ErrSubscriptionNotFound)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *SubscriptionHandlersSuite) TestUpdateSubscription_Success() {
	subscriptionID := int64(123)
	price := 600
	startDate := "02-2024"
	endDate := "04-2024"

	requestBody := UpdateSubscriptionRequest{
		Price:     &price,
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)

    router := chi.NewRouter()
    router.Put("/subscriptions/{id}", UpdateSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("PUT", "/subscriptions/123", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		UpdateSubscription(gomock.Any(), subscriptionID, nil, &price, &startDate, &endDate).
		Return(nil)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
}

func (s *SubscriptionHandlersSuite) TestUpdateSubscription_InvalidJSON() {
	req := httptest.NewRequest("PUT", "/subscriptions/123", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

    UpdateSubscription(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *SubscriptionHandlersSuite) TestUpdateSubscription_NotFound() {
	subscriptionID := int64(123)
	price := 600

	requestBody := UpdateSubscriptionRequest{
		Price: &price,
	}

	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)

    router := chi.NewRouter()
    router.Put("/subscriptions/{id}", UpdateSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("PUT", "/subscriptions/123", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		UpdateSubscription(gomock.Any(), subscriptionID, nil, &price, nil, nil).
		Return(repository.ErrSubscriptionNotFound)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *SubscriptionHandlersSuite) TestUpdateSubscription_WithServiceName() {
	subscriptionID := int64(123)
	serviceName := "Spotify"
	price := 700
	
	requestBody := UpdateSubscriptionRequest{
		ServiceName: &serviceName,
		Price:       &price,
	}
	
	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)
	
    router := chi.NewRouter()
    router.Put("/subscriptions/{id}", UpdateSubscription(s.subscriptionService, s.logger))
	
	req := httptest.NewRequest("PUT", "/subscriptions/123", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	s.subscriptionService.EXPECT().
		UpdateSubscription(gomock.Any(), subscriptionID, &serviceName, &price, nil, nil).
		Return(nil)
	
	router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
}

func (s *SubscriptionHandlersSuite) TestUpdateSubscription_Conflict() {
	subscriptionID := int64(123)
	startDate := "02-2024"
	
	requestBody := UpdateSubscriptionRequest{
		StartDate: &startDate,
	}
	
	jsonBody, err := json.Marshal(requestBody)
	s.Require().NoError(err)
	
    router := chi.NewRouter()
    router.Put("/subscriptions/{id}", UpdateSubscription(s.subscriptionService, s.logger))
	
	req := httptest.NewRequest("PUT", "/subscriptions/123", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	s.subscriptionService.EXPECT().
		UpdateSubscription(gomock.Any(), subscriptionID, nil, nil, &startDate, nil).
		Return(repository.ErrSubscriptionAlreadyExists)
	
	router.ServeHTTP(w, req)
	
	s.Equal(http.StatusConflict, w.Code)
}

func (s *SubscriptionHandlersSuite) TestDeleteSubscription_Success() {
	subscriptionID := int64(123)

    router := chi.NewRouter()
    router.Delete("/subscriptions/{id}", DeleteSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("DELETE", "/subscriptions/123", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		DeleteSubscription(gomock.Any(), subscriptionID).
		Return(nil)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusNoContent, w.Code)
}

func (s *SubscriptionHandlersSuite) TestDeleteSubscription_NotFound() {
	subscriptionID := int64(123)

    router := chi.NewRouter()
    router.Delete("/subscriptions/{id}", DeleteSubscription(s.subscriptionService, s.logger))

	req := httptest.NewRequest("DELETE", "/subscriptions/123", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		DeleteSubscription(gomock.Any(), subscriptionID).
		Return(repository.ErrSubscriptionNotFound)

	router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *SubscriptionHandlersSuite) TestListSubscriptions_Success() {
	userID := uuid.New()
	limit := 10
	offset := 0
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedSubscriptions := []repository.Subscription{
		{
			ID:          1,
			UserID:      userID,
			ServiceName: "Netflix",
			Price:       500,
			StartDate:   fixedTime,
			EndDate:     nil,
		},
	}
	expectedTotal := 1

	req := httptest.NewRequest("GET", "/api/v1/subscriptions?user_id="+userID.String()+"&limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		ListSubscriptions(gomock.Any(), repository.ListSubscriptionsParams{
			UserID: &userID,
			Limit:  limit,
			Offset: offset,
		}).
		Return(expectedSubscriptions, expectedTotal, nil)

    ListSubscriptions(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusOK, w.Code)

    var response struct {
        Subscriptions []struct{
            ID          int64   `json:"id"`
            ServiceName string  `json:"service_name"`
            Price       int     `json:"price"`
            UserID      string  `json:"user_id"`
            StartDate   string  `json:"start_date"`
            EndDate     *string `json:"end_date,omitempty"`
        } `json:"subscriptions"`
        Total         int `json:"total"`
    }
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
    s.Len(response.Subscriptions, 1)
    s.Equal(expectedSubscriptions[0].ID, response.Subscriptions[0].ID)
    s.Equal(expectedSubscriptions[0].ServiceName, response.Subscriptions[0].ServiceName)
    s.Equal(expectedSubscriptions[0].Price, response.Subscriptions[0].Price)
    s.Equal(expectedSubscriptions[0].UserID.String(), response.Subscriptions[0].UserID)
    s.Equal("01-2024", response.Subscriptions[0].StartDate)
	s.Equal(expectedTotal, response.Total)
}

func (s *SubscriptionHandlersSuite) TestListSubscriptions_ServiceError() {
	userID := uuid.New()
	limit := 10
	offset := 0

	req := httptest.NewRequest("GET", "/api/v1/subscriptions?user_id="+userID.String()+"&limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	s.subscriptionService.EXPECT().
		ListSubscriptions(gomock.Any(), repository.ListSubscriptionsParams{
			UserID: &userID,
			Limit:  limit,
			Offset: offset,
		}).
		Return(nil, 0, repository.ErrSubscriptionNotCreated)

    ListSubscriptions(s.subscriptionService, s.logger)(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)
}
