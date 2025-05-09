package models

import (
	"testing"
	"time"
)

func TestNewTransactionRecord(t *testing.T) {
	amount := 100.0
	description := "Test transaction"

	tx := NewTransactionRecord(Deposit, amount, description)

	if tx.Amount != amount {
		t.Errorf("Expected amount %f, got %f", amount, tx.Amount)
	}

	if tx.Type != Deposit {
		t.Errorf("Expected type %s, got %s", Deposit, tx.Type)
	}

	if tx.Description != description {
		t.Errorf("Expected description %s, got %s", description, tx.Description)
	}

	if tx.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Expected a valid UUID, got zero UUID")
	}

	now := time.Now()
	timeDiff := now.Sub(tx.Timestamp)
	if timeDiff < 0 || timeDiff > time.Second {
		t.Errorf("Expected timestamp to be close to now, difference was %v", timeDiff)
	}

	withdrawalTx := NewTransactionRecord(Withdrawal, amount, description)
	if withdrawalTx.Type != Withdrawal {
		t.Errorf("Expected type %s, got %s", Withdrawal, withdrawalTx.Type)
	}
}
