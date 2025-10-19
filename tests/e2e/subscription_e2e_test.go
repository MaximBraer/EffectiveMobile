package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"EffectiveMobile/internal/config"
	"EffectiveMobile/internal/http-server/handlers/subscription/save"
	"EffectiveMobile/internal/lib/logger/handlers/slogdiscard"
	"EffectiveMobile/internal/storage/postgres"
)

func TestSubscriptionE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	cfg := config.MustLoad()

	storage, err := postgres.New(ctx, cfg.Storage, slogdiscard.NewDiscardLogger())
	require.NoError(t, err)
	defer storage.Close()

	handler := save.New(slogdiscard.NewDiscardLogger(), storage)

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

	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	subscriptionID := int64(response["id"].(float64))
	assert.Greater(t, subscriptionID, int64(0))

	subscription, err := storage.GetSubscription(ctx, subscriptionID)
	require.NoError(t, err)

	assert.Equal(t, serviceName, subscription.ServiceName)
	assert.Equal(t, price, subscription.Price)
	assert.Equal(t, userID, subscription.UserID)
	assert.Equal(t, startDate, subscription.StartDate.Format("01-2006"))

	if subscription.EndDate != nil {
		assert.Equal(t, endDate, subscription.EndDate.Format("01-2006"))
	}

	t.Logf("E2E test passed: subscription %d created successfully", subscriptionID)
}

func TestSubscriptionE2E_ValidationErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	cfg := config.MustLoad()

	storage, err := postgres.New(ctx, cfg.Storage, slogdiscard.NewDiscardLogger())
	require.NoError(t, err)
	defer storage.Close()

	handler := save.New(slogdiscard.NewDiscardLogger(), storage)

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestSubscriptionE2E_DuplicateSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	cfg := config.MustLoad()

	storage, err := postgres.New(ctx, cfg.Storage, slogdiscard.NewDiscardLogger())
	require.NoError(t, err)
	defer storage.Close()

	handler := save.New(slogdiscard.NewDiscardLogger(), storage)

	userID := uuid.New()
	requestBody := map[string]interface{}{
		"service_name": "Netflix",
		"price":        500,
		"user_id":      userID,
		"start_date":   "01-2024",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	req2 := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(jsonBody))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}
