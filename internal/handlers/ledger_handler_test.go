package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"tiny-ledger/internal/services"
	"tiny-ledger/internal/store"

	"github.com/gorilla/mux"
)

func setupTestHandler() *LedgerHandler {
	ledgerStore := store.NewLedgerStore()
	ledgerService := services.NewLedgerService(ledgerStore)
	return NewLedgerHandler(ledgerService)
}

func TestHandleTransaction(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	tests := []struct {
		name           string
		userId         string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:   "Valid deposit",
			userId: "test_user",
			requestBody: map[string]interface{}{
				"amount":      100.0,
				"type":        "deposit",
				"description": "Test deposit",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "Zero amount",
			userId: "test_user",
			requestBody: map[string]interface{}{
				"amount":      0.0,
				"type":        "deposit",
				"description": "Zero amount",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Negative amount",
			userId: "test_user",
			requestBody: map[string]interface{}{
				"amount":      -50.0,
				"type":        "deposit",
				"description": "Negative amount",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Excessive amount",
			userId: "test_user",
			requestBody: map[string]interface{}{
				"amount":      2000000.0,
				"type":        "deposit",
				"description": "Too much money",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Invalid transaction type",
			userId: "test_user",
			requestBody: map[string]interface{}{
				"amount":      100.0,
				"type":        "invalid_type",
				"description": "Invalid type",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(test.requestBody)
			req, _ := http.NewRequest("POST", "/users/"+test.userId+"/transactions", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != test.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
					status, test.expectedStatus, rr.Body.String())
			}
		})
	}
}

func TestHandleBalance(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	userId := "balance_test_user"
	depositBody := map[string]interface{}{
		"amount":      100.0,
		"type":        "deposit",
		"description": "Initial deposit",
	}
	jsonBody, _ := json.Marshal(depositBody)
	req, _ := http.NewRequest("POST", "/users/"+userId+"/transactions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	req, _ = http.NewRequest("GET", "/users/"+userId+"/balance", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]float64
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("could not parse response: %v", err)
	}

	if balance, exists := response["balance"]; !exists || balance != 100.0 {
		t.Errorf("unexpected balance: got %v want %v", balance, 100.0)
	}
}

func TestHandleTransactionHistory(t *testing.T) {
	handler := setupTestHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	userId := "history_test_user"

	for i := 0; i < 15; i++ {
		depositBody := map[string]interface{}{
			"amount":      float64(i+1) * 10.0,
			"type":        "deposit",
			"description": "Deposit " + string(rune('A'+i)),
		}
		jsonBody, _ := json.Marshal(depositBody)
		req, _ := http.NewRequest("POST", "/users/"+userId+"/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}

	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "Default pagination (page 1, 10 items)",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
		},
		{
			name:           "Page 2 (5 items)",
			queryParams:    "?page=2",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
		},
		{
			name:           "Custom page size",
			queryParams:    "?pageSize=5",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
		},
		{
			name:           "No results on page 3",
			queryParams:    "?page=3",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/users/"+userId+"/transactions"+tc.queryParams, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("could not parse response: %v", err)
			}

			transactions, ok := response["transactions"].([]interface{})
			if !ok {
				t.Errorf("unexpected response format, transactions not found or not array")
				return
			}

			if len(transactions) != tc.expectedCount {
				t.Errorf("unexpected transaction count: got %v want %v", len(transactions), tc.expectedCount)
			}

			pagination, ok := response["pagination"].(map[string]interface{})
			if !ok {
				t.Errorf("pagination information not found in response")
				return
			}

			if _, exists := pagination["page"]; !exists {
				t.Errorf("page number not found in pagination data")
			}

			if _, exists := pagination["totalItems"]; !exists {
				t.Errorf("totalItems not found in pagination data")
			}

			if totalItems, ok := pagination["totalItems"].(float64); !ok || int(totalItems) != 15 {
				t.Errorf("unexpected totalItems: got %v want %v", totalItems, 15)
			}

			if totalPages, ok := pagination["totalPages"].(float64); !ok {
				t.Errorf("totalPages not found or not a number")
			} else {
				pageSize := 10
				if tc.queryParams == "?pageSize=5" {
					pageSize = 5
				}
				expectedPages := (15 + pageSize - 1) / pageSize
				if int(totalPages) != expectedPages {
					t.Errorf("unexpected totalPages: got %v want %v", totalPages, expectedPages)
				}
			}
		})
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	yesterdayStr := yesterday.Format(time.RFC3339)
	tomorrowStr := tomorrow.Format(time.RFC3339)

	pastTx := map[string]interface{}{
		"amount":      150.0,
		"type":        "deposit",
		"description": "Past transaction",
	}
	pastTxBody, _ := json.Marshal(pastTx)
	pastReq, _ := http.NewRequest("POST", "/users/"+userId+"/transactions", bytes.NewBuffer(pastTxBody))
	pastReq.Header.Set("Content-Type", "application/json")
	pastRr := httptest.NewRecorder()
	router.ServeHTTP(pastRr, pastReq)

	timeRangeTests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "Filter by start time only",
			queryParams:    "?start=" + url.QueryEscape(yesterdayStr),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by end time only",
			queryParams:    "?end=" + url.QueryEscape(tomorrowStr),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by time range",
			queryParams:    "?start=" + url.QueryEscape(yesterdayStr) + "&end=" + url.QueryEscape(tomorrowStr),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid start time format",
			queryParams:    "?start=invalid-time",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range timeRangeTests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/users/"+userId+"/transactions"+tc.queryParams, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, response: %s",
					status, tc.expectedStatus, rr.Body.String())
			}
		})
	}
}
