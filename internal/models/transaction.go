package models

import (
	"github.com/google/uuid"
	"time"
)

type TransactionType string

const (
	Deposit    TransactionType = "deposit"
	Withdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	UserID      string          `json:"user_id"`
	Amount      float64         `json:"amount"`
	Type        TransactionType `json:"type"`
	Description string          `json:"description,omitempty"`
}

type TransactionRecord struct {
	ID          uuid.UUID       `json:"id"`
	Amount      float64         `json:"amount"`
	Type        TransactionType `json:"type"`
	Timestamp   time.Time       `json:"timestamp"`
	Description string          `json:"description,omitempty"`
}

func NewTransactionRecord(transactionType TransactionType, amount float64, description string) TransactionRecord {
	return TransactionRecord{
		ID:          uuid.New(),
		Amount:      amount,
		Type:        transactionType,
		Timestamp:   time.Now(),
		Description: description,
	}
}
