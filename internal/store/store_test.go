package store

import (
	"testing"
	"time"

	"tiny-ledger/internal/models"
)

func TestLedgerStore_AddTransaction(t *testing.T) {
	store := NewLedgerStore()
	userId := "test_user"

	// deposit
	tx, err := store.AddTransaction(userId, models.Deposit, 100.0, "Test deposit")
	if err != nil {
		t.Fatalf("Error adding deposit: %v", err)
	}

	if tx.Amount != 100.0 || tx.Type != models.Deposit {
		t.Errorf("Transaction data incorrect: %+v", tx)
	}

	// withdrawal
	tx, err = store.AddTransaction(userId, models.Withdrawal, 50.0, "Test withdrawal")
	if err != nil {
		t.Fatalf("Error adding withdrawal: %v", err)
	}

	if tx.Amount != 50.0 || tx.Type != models.Withdrawal {
		t.Errorf("Transaction data incorrect: %+v", tx)
	}

	// insufficient funds
	_, err = store.AddTransaction(userId, models.Withdrawal, 100.0, "Excessive withdrawal")
	if err == nil {
		t.Error("Expected insufficient funds error, got none")
	}
}

func TestLedgerStore_GetBalance(t *testing.T) {
	store := NewLedgerStore()
	userId := "balance_test_user"

	// Initial balance should be 0
	balance, err := store.GetBalance(userId)
	if err != nil {
		t.Fatalf("Error getting balance: %v", err)
	}
	if balance != 0.0 {
		t.Errorf("Expected initial balance 0.0, got %.2f", balance)
	}

	_, err = store.AddTransaction(userId, models.Deposit, 100.0, "Test deposit")
	if err != nil {
		t.Fatalf("Error adding deposit: %v", err)
	}

	balance, err = store.GetBalance(userId)
	if err != nil {
		t.Fatalf("Error getting balance: %v", err)
	}
	if balance != 100.0 {
		t.Errorf("Expected balance 100.0, got %.2f", balance)
	}

	_, err = store.AddTransaction(userId, models.Withdrawal, 30.0, "Test withdrawal")
	if err != nil {
		t.Fatalf("Error adding withdrawal: %v", err)
	}

	balance, err = store.GetBalance(userId)
	if err != nil {
		t.Fatalf("Error getting balance: %v", err)
	}
	if balance != 70.0 {
		t.Errorf("Expected balance 70.0, got %.2f", balance)
	}
}

func TestLedgerStore_GetPaginatedTransactions(t *testing.T) {
	store := NewLedgerStore()
	userId := "pagination_test_user"

	// create 25 transactions
	for i := 0; i < 25; i++ {
		_, err := store.AddTransaction(userId, models.Deposit, float64(i+1)*10.0, "Pagination test")
		if err != nil {
			t.Fatalf("Error adding transaction: %v", err)
		}
	}

	// get all transactions (1 page with large page size)
	result := store.GetPaginatedTransactions(userId, nil, nil, 1, 50)
	if len(result.Transactions) != 25 {
		t.Errorf("Expected 25 transactions, got %d", len(result.Transactions))
	}
	if result.TotalCount != 25 {
		t.Errorf("Expected total count 25, got %d", result.TotalCount)
	}

	// first page - 10 per page
	result = store.GetPaginatedTransactions(userId, nil, nil, 1, 10)
	if len(result.Transactions) != 10 {
		t.Errorf("Expected 10 transactions, got %d", len(result.Transactions))
	}
	if result.TotalCount != 25 {
		t.Errorf("Expected total count 25, got %d", result.TotalCount)
	}

	// third page 10 per page - should get 5 items
	result = store.GetPaginatedTransactions(userId, nil, nil, 3, 10)
	if len(result.Transactions) != 5 {
		t.Errorf("Expected 5 transactions, got %d", len(result.Transactions))
	}

	// time filtering
	now := time.Now()
	startTime := now.Add(-24 * time.Hour) // 1 day ago

	tx := models.TransactionRecord{
		ID:          [16]byte{},
		Amount:      999.0,
		Type:        models.Deposit,
		Timestamp:   startTime.Add(1 * time.Hour), // 23 hours ago
		Description: "Timestamped transaction",
	}
	store.AddTransactionWithTime(userId, tx)

	result = store.GetPaginatedTransactions(userId, &startTime, nil, 1, 50)
	if result.TotalCount != 26 { // 25 original + 1 timestamp
		t.Errorf("Expected 26 transactions with time filter, got %d", result.TotalCount)
	}

	// filtter with a future end time - should get all
	endTime := now.Add(24 * time.Hour) // 1 day in future
	result = store.GetPaginatedTransactions(userId, nil, &endTime, 1, 50)
	if result.TotalCount != 26 {
		t.Errorf("Expected 26 transactions with end time filter, got %d", result.TotalCount)
	}

	// a past end time - should get only the timestamp transaction
	pastEndTime := startTime.Add(2 * time.Hour) // 22 hours ago
	result = store.GetPaginatedTransactions(userId, &startTime, &pastEndTime, 1, 50)
	if result.TotalCount != 1 {
		t.Errorf("Expected 1 transaction with narrow time filter, got %d", result.TotalCount)
	}
}
