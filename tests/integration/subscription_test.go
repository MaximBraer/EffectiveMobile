package integration

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	slogformatter "github.com/samber/slog-formatter"
	"github.com/stretchr/testify/suite"

	"EffectiveMobile/internal/config"
	"EffectiveMobile/internal/repository"
	"EffectiveMobile/pkg/postgres"
)

type SubscriptionSuite struct {
	Suite

	logger *slog.Logger
	DB     *sql.DB
}

func TestSubscription(t *testing.T) {
	if !isIntegrationTestsRun() {
		t.Skip()
		return
	}

	suite.Run(t, &SubscriptionSuite{})
}

func (s *SubscriptionSuite) SetupTest() {
	s.Suite.SetupTest()

	s.logger = slog.New(
		slogformatter.NewFormatterHandler(
			slogformatter.TimezoneConverter(time.UTC),
			slogformatter.TimeFormatter(time.DateTime, nil),
		)(slog.NewJSONHandler(os.Stdout, nil)))

	slog.SetDefault(s.logger)

	s.initDatabase()
}

func (s *SubscriptionSuite) TearDownTest() {
	s.logger = nil
	if s.DB != nil {
		s.Require().NoError(s.DB.Close())
	}
}

func (s *SubscriptionSuite) initDatabase() {
	cfg := &config.Config{
		SQLDataBase: config.SQLConnection{
			User:     "postgres",
			Password: "postgres",
			DataBaseInfo: postgres.SQLDataBase{
				Server:          "localhost",
				Database:        "subscriptions",
				MaxIdleCons:     10,
				MaxOpenCons:     10,
				ConnMaxLifetime: 2,
				Port:            "5433",
			},
		},
	}

	provider := postgres.New(cfg.SQLDataBase.User, cfg.SQLDataBase.Password, cfg.SQLDataBase.DataBaseInfo, s.logger)
	s.Require().NoError(provider.Open())

	s.DB = provider.GetConn()
}


func (s *SubscriptionSuite) clearDatabase() {
	_, err := s.DB.Exec(`TRUNCATE TABLE subscription CASCADE`)
	s.NoError(err)
	_, err = s.DB.Exec(`TRUNCATE TABLE service CASCADE`)
	s.NoError(err)
}

func (s *SubscriptionSuite) TestCreateSubscription() {
	s.clearDatabase()

	userID := uuid.New()
	subscriptionID := s.createSubscription("Netflix", 500, userID, "01-2024", "")

	s.Eventually(
		func() bool {
			var subscription repository.Subscription
			err := s.DB.QueryRow(
				`SELECT s.id, sv.name, s.price_rub, s.user_id, s.start_date, s.end_date 
				 FROM subscription s 
				 JOIN service sv ON s.service_id = sv.id 
				 WHERE s.id = $1`,
				subscriptionID,
			).Scan(&subscription.ID, &subscription.ServiceName, &subscription.Price, 
				&subscription.UserID, &subscription.StartDate, &subscription.EndDate)

			if errors.Is(err, sql.ErrNoRows) {
				return false
			}

			s.NoError(err)
			s.Equal("Netflix", subscription.ServiceName)
			s.Equal(500, subscription.Price)
			s.Equal(userID, subscription.UserID)
			s.NotZero(subscription.StartDate)
			s.Nil(subscription.EndDate)

			return true
		},
		time.Second*5,
		time.Millisecond*100,
	)
}

func (s *SubscriptionSuite) TestGetSubscription() {
	s.clearDatabase()

	userID := uuid.New()
	subscriptionID := s.createSubscription("Netflix", 500, userID, "01-2024", "")

	respBody, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), nil, nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var subscription struct {
		ID          int64      `json:"id"`
		ServiceName string     `json:"service_name"`
		Price       int        `json:"price"`
		UserID      uuid.UUID  `json:"user_id"`
		StartDate   string     `json:"start_date"`
		EndDate     *string    `json:"end_date"`
	}
	err = jsoniter.Unmarshal(respBody, &subscription)
	s.NoError(err)

	s.Equal(subscriptionID, subscription.ID)
	s.Equal("Netflix", subscription.ServiceName)
	s.Equal(500, subscription.Price)
	s.Equal(userID, subscription.UserID)
	s.NotEmpty(subscription.StartDate)
}

func (s *SubscriptionSuite) TestUpdateSubscription() {
	s.clearDatabase()

	userID := uuid.New()
	subscriptionID := s.createSubscription("Netflix", 500, userID, "01-2024", "")

	updateData := []byte(`{"price":600,"end_date":"12-2024"}`)
	respBody, resp, err := doRequest(http.MethodPut, mainHost, fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), updateData, nil, nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var response struct{ Status string `json:"status"` }
	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)
	s.Equal("ok", response.Status)

	var subscription repository.Subscription
	err = s.DB.QueryRow(
		`SELECT s.id, sv.name, s.price_rub, s.user_id, s.start_date, s.end_date 
		 FROM subscription s 
		 JOIN service sv ON s.service_id = sv.id 
		 WHERE s.id = $1`,
		subscriptionID,
	).Scan(&subscription.ID, &subscription.ServiceName, &subscription.Price, 
		&subscription.UserID, &subscription.StartDate, &subscription.EndDate)

	s.NoError(err)
	s.Equal(600, subscription.Price)
	s.NotNil(subscription.EndDate)
}

func (s *SubscriptionSuite) TestDeleteSubscription() {
	s.clearDatabase()

	userID := uuid.New()
	subscriptionID := s.createSubscription("Netflix", 500, userID, "01-2024", "")

	resp, err := deleteAPIResponse(mainHost, fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), nil)
	s.NoError(err)
	s.Equal(204, resp.StatusCode)

	var count int
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM subscription WHERE id = $1`, subscriptionID).Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *SubscriptionSuite) TestListSubscriptions() {
	s.clearDatabase()

	userID := uuid.New()
	_ = s.createSubscription("Netflix", 500, userID, "01-2024", "")
	_ = s.createSubscription("Spotify", 300, userID, "02-2024", "")

	respBody, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/subscriptions?user_id=%s&limit=%d&offset=%d", userID.String(), 10, 0), nil, nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var response struct {
		Subscriptions []struct {
			ID          int64      `json:"ID"`
			ServiceName string     `json:"ServiceName"`
			Price       int        `json:"Price"`
			UserID      uuid.UUID  `json:"UserID"`
			StartDate   time.Time  `json:"StartDate"`
			EndDate     *time.Time `json:"EndDate"`
		} `json:"subscriptions"`
		Total int `json:"total"`
	}
	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)

	s.Len(response.Subscriptions, 2)
	s.Equal(2, response.Total)

	s.Require().GreaterOrEqual(len(response.Subscriptions), 2)
	s.Equal("Netflix", response.Subscriptions[0].ServiceName)
	s.Equal(500, response.Subscriptions[0].Price)
	s.Equal("Spotify", response.Subscriptions[1].ServiceName)
	s.Equal(300, response.Subscriptions[1].Price)
}

func (s *SubscriptionSuite) TestGetTotalStats() {
	s.clearDatabase()

	userID := uuid.New()
	_ = s.createSubscription("Netflix", 500, userID, "01-2024", "")
	_ = s.createSubscription("Spotify", 300, userID, "02-2024", "")

	respBody, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/stats/total?user_id=%s&start_date=01-2024&end_date=12-2024", userID.String()), nil, nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var stats struct {
		TotalCost          int `json:"total_cost"`
		SubscriptionsCount int `json:"subscriptions_count"`
	}
	err = jsoniter.Unmarshal(respBody, &stats)
	s.NoError(err)

	s.Equal(9300, stats.TotalCost)
	s.Equal(2, stats.SubscriptionsCount)
}

func (s *SubscriptionSuite) createSubscription(serviceName string, price int, userID uuid.UUID, startDate, endDate string) int64 {
	requestBody := fmt.Sprintf(`{
		"service_name": "%s",
		"price": %d,
		"user_id": "%s",
		"start_date": "%s",
		"end_date": "%s"
	}`, serviceName, price, userID.String(), startDate, endDate)

	respBody, resp, err := postAPIResponse(mainHost, "/api/v1/subscriptions", []byte(requestBody), nil, nil)
	s.NoError(err)
	s.Equal(201, resp.StatusCode)

	var response struct {
		ID int64 `json:"id"`
	}

	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)
	s.NotZero(response.ID)

	return response.ID
}