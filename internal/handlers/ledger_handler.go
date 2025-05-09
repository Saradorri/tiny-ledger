package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"tiny-ledger/internal/models"
	"tiny-ledger/internal/services"

	"github.com/gorilla/mux"
)

type LedgerHandler struct {
	service services.LedgerService
}

func NewLedgerHandler(s services.LedgerService) *LedgerHandler {
	return &LedgerHandler{service: s}
}

func (h *LedgerHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/users/{userId}/transactions", h.handleTransaction).Methods("POST")
	r.HandleFunc("/users/{userId}/balance", h.handleBalance).Methods("GET")
	r.HandleFunc("/users/{userId}/transactions", h.handleTransactionsHistory).Methods("GET")
}

type transactionRequest struct {
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"type"`
	Description     string  `json:"description,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func sendErrorResponse(w http.ResponseWriter, status int, message string) {
	sendJSONResponse(w, status, ErrorResponse{Error: message})
}

func (h *LedgerHandler) handleTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["userId"]

	if userId == "" {
		sendErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	var req transactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "invalid request format: "+err.Error())
		return
	}

	const maxTransactionAmount = 1000000.0 // this is just for limiting the amont of transactions
	if req.Amount > maxTransactionAmount {
		sendErrorResponse(w, http.StatusBadRequest, "transaction amount exceeds maximum allowed")
		return
	}

	tx, err := h.service.RecordTransaction(userId, models.TransactionType(req.TransactionType), req.Amount, req.Description)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	sendJSONResponse(w, http.StatusCreated, tx)
}

func (h *LedgerHandler) handleBalance(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]
	if userId == "" {
		sendErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	balance, err := h.service.GetCurrentBalance(userId)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]float64{"balance": balance})
}

func (h *LedgerHandler) handleTransactionsHistory(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]
	if userId == "" {
		sendErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	// Default values for page and pagesize
	page := 1
	pageSize := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	var startTime, endTime *time.Time
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &t
		} else {
			sendErrorResponse(w, http.StatusBadRequest, "invalid start time format, use RFC3339")
			return
		}
	}

	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &t
		} else {
			sendErrorResponse(w, http.StatusBadRequest, "invalid end time format, use RFC3339")
			return
		}
	}

	result, err := h.service.GetPaginatedTransactionHistory(userId, startTime, endTime, page, pageSize)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"transactions": result.Transactions,
		"pagination": map[string]interface{}{
			"page":       result.Page,
			"pageSize":   result.PageSize,
			"totalItems": result.TotalCount,
			"totalPages": result.TotalPages,
		},
	}

	sendJSONResponse(w, http.StatusOK, response)
}
