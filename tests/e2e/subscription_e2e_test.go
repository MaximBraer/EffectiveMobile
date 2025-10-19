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

func TestSubscriptionE2E_FullFlow(t *testing.T) {
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

	requestBody := map[string]interface{}{
		"service_name": serviceName,
		"price":        price,
		"user_id":      userID,
		"start_date":   startDate,
		"end_date":     endDate,
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	subscriptionID := int64(response["id"].(float64))

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), nil)
	w = httptest.NewRecorder()
	handlers.GetSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	require.NoError(t, err)
	assert.Equal(t, serviceName, getResponse["service_name"])
	assert.Equal(t, float64(price), getResponse["price"])
	assert.Equal(t, userID.String(), getResponse["user_id"])
	assert.Equal(t, startDate, getResponse["start_date"])
	assert.Equal(t, endDate, getResponse["end_date"])

	updateBody := map[string]interface{}{
		"price": 600,
	}
	updateJson, err := json.Marshal(updateBody)
	require.NoError(t, err)

	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), bytes.NewReader(updateJson))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.UpdateSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), nil)
	w = httptest.NewRecorder()
	handlers.GetSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var updatedResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &updatedResponse)
	require.NoError(t, err)
	assert.Equal(t, serviceName, updatedResponse["service_name"])
	assert.Equal(t, float64(600), updatedResponse["price"])
	assert.Equal(t, userID.String(), updatedResponse["user_id"])
	assert.Equal(t, startDate, updatedResponse["start_date"])
	assert.Equal(t, endDate, updatedResponse["end_date"])

	req = httptest.NewRequest("GET", "/api/v1/subscriptions", nil)
	w = httptest.NewRecorder()
	handlers.ListSubscriptions(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	subscriptions := listResponse["subscriptions"].([]interface{})
	assert.Len(t, subscriptions, 1)
	subscription := subscriptions[0].(map[string]interface{})
	assert.Equal(t, serviceName, subscription["service_name"])
	assert.Equal(t, float64(600), subscription["price"])

	req = httptest.NewRequest("GET", "/api/v1/stats/total", nil)
	w = httptest.NewRecorder()
	handlers.GetTotalStats(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var statsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(t, err)
	assert.Equal(t, float64(600), statsResponse["total_cost"])
	assert.Equal(t, float64(1), statsResponse["subscriptions_count"])

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/subscriptions/%d", subscriptionID), nil)
	w = httptest.NewRecorder()
	handlers.DeleteSubscription(slogdiscard.NewDiscardLogger(), storage).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSubscriptionE2E_ValidationErrors(t *testing.T) {
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

	handler := handlers.SaveSubscription(slogdiscard.NewDiscardLogger(), storage)

	userID := uuid.New()

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		createFirst    bool
	}{
		{
			name: "Empty service name",
			requestBody: map[string]interface{}{
				"service_name": "",
				"price":        500,
				"user_id":      uuid.New(),
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Negative price",
			requestBody: map[string]interface{}{
				"service_name": "Netflix",
				"price":        -100,
				"user_id":      uuid.New(),
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid date format",
			requestBody: map[string]interface{}{
				"service_name": "Netflix",
				"price":        500,
				"user_id":      uuid.New(),
				"start_date":   "invalid-date",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate subscription",
			requestBody: map[string]interface{}{
				"service_name": "Netflix",
				"price":        500,
				"user_id":      userID,
				"start_date":   "01-2024",
			},
			expectedStatus: http.StatusConflict,
			createFirst:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.createFirst {
				jsonBody, err := json.Marshal(tc.requestBody)
				require.NoError(t, err)

				req := httptest.NewRequest("POST", "/api/v1/subscriptions", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusCreated, w.Code)
			}

			jsonBody, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/subscriptions", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
