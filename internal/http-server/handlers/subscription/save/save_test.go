package save_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"EffectiveMobile/internal/http-server/handlers/subscription/save"
	"EffectiveMobile/internal/http-server/handlers/subscription/save/mocks"
	"EffectiveMobile/internal/lib/logger/handlers/slogdiscard"
	"EffectiveMobile/internal/storage"
	"EffectiveMobile/internal/storage/postgres"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name           string
		request        save.Request
		serviceID      int
		subscriptionID int64
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Success",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
			},
			serviceID:      1,
			subscriptionID: 123,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Success with end date",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
				EndDate:     stringPtr("12-2024"),
			},
			serviceID:      1,
			subscriptionID: 124,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Validation Error - empty service name",
			request: save.Request{
				ServiceName: "",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid arguments",
		},
		{
			name: "Validation Error - negative price",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       -100,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid arguments",
		},
		{
			name: "Validation Error - nil user ID",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.Nil,
				StartDate:   "01-2024",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid arguments",
		},
		{
			name: "Validation Error - invalid date format",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "invalid-date",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid arguments",
		},
		{
			name: "Validation Error - end date before start date",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "12-2024",
				EndDate:     stringPtr("01-2024"),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid arguments",
		},
		{
			name: "Subscription Already Exists",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
			},
			serviceID:      1,
			mockError:      storage.ErrSubscriptionAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedError:  "invalid arguments",
		},
		{
			name: "Internal Server Error",
			request: save.Request{
				ServiceName: "Netflix",
				Price:       500,
				UserID:      uuid.New(),
				StartDate:   "01-2024",
			},
			serviceID:      1,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storageMock := mocks.NewSubscriptionStorage(t)

			if tc.request.ServiceName != "" && tc.request.UserID != uuid.Nil {
				storageMock.On("GetOrCreateServiceID", mock.Anything, tc.request.ServiceName).
					Return(tc.serviceID, nil).Maybe()
			}

			if tc.mockError == nil && tc.expectedStatus == http.StatusCreated {
				storageMock.On("CreateSubscription", mock.Anything, mock.MatchedBy(func(p postgres.CreateSubscriptionParams) bool {
					return p.UserID == tc.request.UserID &&
						p.ServiceID == tc.serviceID &&
						p.PriceRub == tc.request.Price
				}), mock.Anything).
					Return(tc.subscriptionID, nil).Once()
			} else if tc.mockError != nil && tc.expectedStatus != http.StatusBadRequest {
				storageMock.On("CreateSubscription", mock.Anything, mock.AnythingOfType("postgres.CreateSubscriptionParams"), mock.Anything).
					Return(int64(0), tc.mockError).Once()
			}

			r := chi.NewRouter()
			r.Post("/subscriptions", save.New(slogdiscard.NewDiscardLogger(), storageMock))

			reqBody, _ := json.Marshal(tc.request)
			req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedError, response["error"])
			} else if tc.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "ok", response["status"])
				assert.Equal(t, float64(tc.subscriptionID), response["id"])
			}

			storageMock.AssertExpectations(t)
		})
	}
}

func TestSaveHandler_InvalidJSON(t *testing.T) {
	storageMock := mocks.NewSubscriptionStorage(t)

	r := chi.NewRouter()
	r.Post("/subscriptions", save.New(slogdiscard.NewDiscardLogger(), storageMock))

	req := httptest.NewRequest("POST", "/subscriptions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid arguments", response["error"])
}

func stringPtr(s string) *string {
	return &s
}
