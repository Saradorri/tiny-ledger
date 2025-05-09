package services

import (
	"errors"
	"regexp"
	"time"

	"tiny-ledger/internal/models"
	"tiny-ledger/internal/store"
)

type PaginatedTransactions struct {
	Transactions []models.TransactionRecord
	TotalCount   int
	Page         int
	PageSize     int
	TotalPages   int
}

type LedgerService interface {
	RecordTransaction(userId string, txType models.TransactionType, amount float64, description string) (models.TransactionRecord, error)
	GetPaginatedTransactionHistory(userId string, startTime, endTime *time.Time, page, pageSize int) (PaginatedTransactions, error)
	GetCurrentBalance(userId string) (float64, error)
}

type ledgerService struct {
	store *store.LedgerStore
}

func NewLedgerService(store *store.LedgerStore) LedgerService {
	return &ledgerService{
		store: store,
	}
}

var userIdRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)

func (s *ledgerService) RecordTransaction(userId string, txType models.TransactionType, amount float64, description string) (models.TransactionRecord, error) {
	if userId == "" {
		return models.TransactionRecord{}, errors.New("user ID is required")
	}

	if !userIdRegex.MatchString(userId) {
		return models.TransactionRecord{}, errors.New("invalid user ID format: must be 3-50 alphanumeric characters, underscores, dots, or hyphens")
	}

	if amount <= 0 {
		return models.TransactionRecord{}, errors.New("amount must be positive")
	}

	const maxAmount = 1000000.0
	if amount > maxAmount {
		return models.TransactionRecord{}, errors.New("amount exceeds maximum allowed")
	}

	if txType != models.Deposit && txType != models.Withdrawal {
		return models.TransactionRecord{}, errors.New("invalid transaction type")
	}

	if len(description) > 500 {
		return models.TransactionRecord{}, errors.New("description exceeds maximum length of 500 characters")
	}

	tx, err := s.store.AddTransaction(userId, txType, amount, description)
	if err != nil {
		return models.TransactionRecord{}, err
	}
	return tx, nil
}

func (s *ledgerService) GetPaginatedTransactionHistory(userId string, startTime, endTime *time.Time, page, pageSize int) (PaginatedTransactions, error) {
	if userId == "" {
		return PaginatedTransactions{}, errors.New("user ID is required")
	}

	if !userIdRegex.MatchString(userId) {
		return PaginatedTransactions{}, errors.New("invalid user ID format")
	}

	if page < 1 {
		page = 1 // default page num
	}

	if pageSize < 1 {
		pageSize = 10 // default page size
	}

	const maxPageSize = 100
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return PaginatedTransactions{}, errors.New("start time cannot be after end time")
	}

	result := s.store.GetPaginatedTransactions(userId, startTime, endTime, page, pageSize)

	totalPages := (result.TotalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginatedTransactions{
		Transactions: result.Transactions,
		TotalCount:   result.TotalCount,
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
	}, nil
}

func (s *ledgerService) GetCurrentBalance(userId string) (float64, error) {
	if userId == "" {
		return 0, errors.New("user ID is required")
	}

	if !userIdRegex.MatchString(userId) {
		return 0, errors.New("invalid user ID format")
	}

	balance, err := s.store.GetBalance(userId)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
