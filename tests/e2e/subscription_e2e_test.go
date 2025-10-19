package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"EffectiveMobile/internal/api/handlers"
	"EffectiveMobile/internal/config"
	"EffectiveMobile/internal/lib/logger/handlers/slogdiscard"
	"EffectiveMobile/internal/storage/postgres"
)

func TestSubscriptionE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cfg, err := config.MustLoad()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := postgres.New(ctx, cfg.Storage, slogdiscard.NewDiscardLogger())
	require.NoError(t, err)
	defer storage.Close()

	userID := uuid.New()
	serviceName := "Netflix"
	price := 500
	startDate := "01-2024"
	endDate := "12-2024"

	testCases := []struct {
		name           string
		method         string
		path           string
		body           map[string]interface{}
		expectedStatus int
		validate       func(t *testing.T, response map[string]interface{})
		setup          func() int64
	}{
		{
			name:   "Create subscription",
			method: "POST",
			path:   "/api/v1/subscriptions",
			body: map[string]interface{}{
				"service_name": serviceName,
				"price":        price,
				"user_id":      userID,
				"start_date":   startDate,
				"end_date":     endDate,
			},
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "ok", response["status"])
				assert.Greater(t, int64(response["id"].(float64)), int64(0))
			},
		},
		{
			name:   "Get subscription",
			method: "GET",
			path:   "/api/v1/subscriptions/1",
			body:   nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, serviceName, response["service_name"])
				assert.Equal(t, float64(price), response["price"])
				assert.Equal(t, userID.String(), response["user_id"])
				assert.Equal(t, startDate, response["start_date"])
				assert.Equal(t, endDate, response["end_date"])
			},
		},
		{
			name:   "Update subscription",
			method: "PUT",
			path:   "/api/v1/subscriptions/1",
			body: map[string]interface{}{
				"price": 600,
			},
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "ok", response["status"])
			},
		},
		{
			name:   "Get updated subscription",
			method: "GET",
			path:   "/api/v1/subscriptions/1",
			body:   nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, serviceName, response["service_name"])
				assert.Equal(t, float64(600), response["price"])
				assert.Equal(t, userID.String(), response["user_id"])
				assert.Equal(t, startDate, response["start_date"])
				assert.Equal(t, endDate, response["end_date"])
			},
		},
		{
			name:   "List subscriptions",
			method: "GET",
			path:   "/api/v1/subscriptions",
			body:   nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				subscriptions := response["subscriptions"].([]interface{})
				assert.Len(t, subscriptions, 1)
				subscription := subscriptions[0].(map[string]interface{})
				assert.Equal(t, serviceName, subscription["service_name"])
				assert.Equal(t, float64(600), subscription["price"])
			},
		},
		{
			name:   "Get stats",
			method: "GET",
			path:   "/api/v1/stats/total",
			body:   nil,
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, float64(600), response["total_cost"])
				assert.Equal(t, float64(1), response["subscriptions_count"])
			},
		},
		{
			name:   "Delete subscription",
			method: "DELETE",
			path:   "/api/v1/subscriptions/1",
			body:   nil,
			expectedStatus: http.StatusNoContent,
			validate: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:   "Empty service name",
			method: "POST",
			path:   "/api/v1/subscriptions",
			body: map[string]interface{}{
				"service_name": "",
				"price":        500,
				"user_id":      uuid.New(),
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:   "Negative price",
			method: "POST",
			path:   "/api/v1/subscriptions",
			body: map[string]interface{}{
				"service_name": "Netflix",
				"price":        -100,
				"user_id":      uuid.New(),
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:   "Invalid date format",
			method: "POST",
			path:   "/api/v1/subscriptions",
			body: map[string]interface{}{
				"service_name": "Netflix",
				"price":        500,
				"user_id":      uuid.New(),
				"start_date":   "invalid-date",
			},
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:   "Duplicate subscription",
			method: "POST",
			path:   "/api/v1/subscriptions",
			body: map[string]interface{}{
				"service_name": "Netflix",
				"price":        500,
				"user_id":      userID,
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusConflict,
			validate: func(t *testing.T, response map[string]interface{}) {
			},
		},
	}

	var subscriptionID int64
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error
			if tc.body != nil {
				body, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			path := tc.path
			if tc.name == "Get subscription" || tc.name == "Update subscription" || tc.name == "Get updated subscription" || tc.name == "Delete subscription" {
				path = fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID)
			}

			req := httptest.NewRequest(tc.method, path, bytes.NewReader(body))
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()

			switch tc.method {
			case "POST":
				handlers.SaveSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
			case "GET":
				if tc.name == "List subscriptions" {
					handlers.ListSubscriptions(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
				} else if tc.name == "Get stats" {
					handlers.GetTotalStats(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
				} else {
					handlers.GetSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
				}
			case "PUT":
				handlers.UpdateSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
			case "DELETE":
				handlers.DeleteSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
			}

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus != http.StatusNoContent && w.Body.Len() > 0 {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tc.validate(t, response)

				if tc.name == "Create subscription" {
					subscriptionID = int64(response["id"].(float64))
				}
			}
		})
	}
}
