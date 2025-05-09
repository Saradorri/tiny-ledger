package services

import (
	"sync"
	"testing"
	"time"

	"tiny-ledger/internal/models"
	"tiny-ledger/internal/store"
)

// test concurrency race conditions
func TestConcurrentTransactions_NoRace(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	userId := "user123"
	numGoroutines := 100
	depositAmount := 10.0

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			description := "test deposit"
			_, err := svc.RecordTransaction(userId, models.Deposit, depositAmount, description)
			if err != nil {
				t.Errorf("unexpected error in goroutine %d: %v", i, err)
			}
		}(i)
	}

	wg.Wait()

	balance, err := svc.GetCurrentBalance(userId)
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}

	expectedBalance := float64(numGoroutines) * depositAmount
	if balance != expectedBalance {
		t.Errorf("expected balance %.2f, got %.2f", expectedBalance, balance)
	}

	//get all transactions with a large pagesize
	result, err := svc.GetPaginatedTransactionHistory(userId, nil, nil, 1, numGoroutines)
	if err != nil {
		t.Fatalf("failed to get paginated transaction history: %v", err)
	}

	if len(result.Transactions) != numGoroutines {
		t.Errorf("expected %d transactions, got %d", numGoroutines, len(result.Transactions))
	}

	if result.TotalCount != numGoroutines {
		t.Errorf("expected total count %d, got %d", numGoroutines, result.TotalCount)
	}

	// check transaction sorting
	for i := 1; i < len(result.Transactions); i++ {
		if result.Transactions[i-1].Timestamp.After(result.Transactions[i].Timestamp) {
			t.Errorf("transactions not sorted: tx %d timestamp after tx %d", i-1, i)
		}
	}
}

func TestInputValidation(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	tests := []struct {
		name        string
		userId      string
		txType      models.TransactionType
		amount      float64
		description string
		expectError bool
	}{
		{"Valid deposit", "validUser123", models.Deposit, 100.0, "Valid deposit", false},
		{"Valid withdrawal", "validUser123", models.Withdrawal, 50.0, "Valid withdrawal", false},
		{"Empty user ID", "", models.Deposit, 100.0, "Valid deposit", true},
		{"Invalid user ID", "user@invalid", models.Deposit, 100.0, "Valid deposit", true},
		{"Zero amount", "validUser123", models.Deposit, 0.0, "Zero amount", true},
		{"Negative amount", "validUser123", models.Deposit, -50.0, "Negative amount", true},
		{"Excessive amount", "validUser123", models.Deposit, 2000000.0, "Too much money", true},
		{"Invalid transaction type", "validUser123", "invalid_type", 100.0, "Invalid type", true},
		{"Very long description", "validUser123", models.Deposit, 100.0, string(make([]byte, 1000)), true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.RecordTransaction(test.userId, test.txType, test.amount, test.description)

			if test.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !test.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestInsufficientFunds(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	userId := "test_user"

	_, err := svc.RecordTransaction(userId, models.Deposit, 100.0, "Initial deposit")
	if err != nil {
		t.Fatalf("failed to add initial deposit: %v", err)
	}

	_, err = svc.RecordTransaction(userId, models.Withdrawal, 150.0, "Excessive withdrawal")
	if err == nil {
		t.Errorf("expected insufficient funds error but got none")
	}

	balance, err := svc.GetCurrentBalance(userId)
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}

	if balance != 100.0 {
		t.Errorf("expected balance to remain 100.0, got %.2f", balance)
	}
}

func TestTimeRangeFiltering(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	userId := "time_test_user"

	timePoints := make([]time.Time, 5)
	for i := 0; i < 5; i++ {
		timePoints[i] = time.Date(2023, time.January, i+1, 12, 0, 0, 0, time.UTC)
	}

	ledger := s
	for i, tp := range timePoints {
		amount := float64(i+1) * 10
		tx := models.TransactionRecord{
			ID:          [16]byte{},
			Amount:      amount,
			Type:        models.Deposit,
			Timestamp:   tp,
			Description: "Test transaction",
		}
		ledger.AddTransactionWithTime(userId, tx)
	}

	testCases := []struct {
		name      string
		startTime *time.Time
		endTime   *time.Time
		expected  int
	}{
		{
			name:      "All transactions",
			startTime: nil,
			endTime:   nil,
			expected:  5,
		},
		{
			name:      "Start from second day",
			startTime: timePtr(timePoints[1]),
			endTime:   nil,
			expected:  4,
		},
		{
			name:      "Only first three days",
			startTime: nil,
			endTime:   timePtr(timePoints[2]),
			expected:  3,
		},
		{
			name:      "Specific range (day 2-3)",
			startTime: timePtr(timePoints[1]),
			endTime:   timePtr(timePoints[3]),
			expected:  3,
		},
		{
			name:      "No transactions in range",
			startTime: timePtr(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
			endTime:   timePtr(time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)),
			expected:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Legacy", func(t *testing.T) {
			txs, err := svc.GetPaginatedTransactionHistory(userId, tc.startTime, tc.endTime, 1, 10)
			if err != nil {
				t.Fatalf("failed to get transaction history: %v", err)
			}

			if len(txs.Transactions) != tc.expected {
				t.Errorf("expected %d transactions, got %d", tc.expected, len(txs.Transactions))
			}
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Paginated", func(t *testing.T) {
			result, err := svc.GetPaginatedTransactionHistory(userId, tc.startTime, tc.endTime, 1, 100)
			if err != nil {
				t.Fatalf("failed to get paginated transaction history: %v", err)
			}

			if len(result.Transactions) != tc.expected {
				t.Errorf("expected %d transactions, got %d", tc.expected, len(result.Transactions))
			}

			if result.TotalCount != tc.expected {
				t.Errorf("expected total count %d, got %d", tc.expected, result.TotalCount)
			}
		})
	}
}

func TestPagination(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	userId := "pagination_test_user"

	for i := 0; i < 25; i++ {
		amount := float64(i+1) * 10
		_, err := svc.RecordTransaction(userId, models.Deposit, amount, "Pagination test tx")
		if err != nil {
			t.Fatalf("failed to create test transaction: %v", err)
		}
	}

	testCases := []struct {
		name          string
		page          int
		pageSize      int
		expectedCount int
		expectedPage  int
		expectedSize  int
		expectedTotal int
	}{
		{
			name:          "First page with default size",
			page:          1,
			pageSize:      10,
			expectedCount: 10,
			expectedPage:  1,
			expectedSize:  10,
			expectedTotal: 25,
		},
		{
			name:          "Second page with default size",
			page:          2,
			pageSize:      10,
			expectedCount: 10,
			expectedPage:  2,
			expectedSize:  10,
			expectedTotal: 25,
		},
		{
			name:          "Third page with default size (partial)",
			page:          3,
			pageSize:      10,
			expectedCount: 5,
			expectedPage:  3,
			expectedSize:  10,
			expectedTotal: 25,
		},
		{
			name:          "Page beyond results",
			page:          4,
			pageSize:      10,
			expectedCount: 0,
			expectedPage:  4,
			expectedSize:  10,
			expectedTotal: 25,
		},
		{
			name:          "Custom small page size",
			page:          1,
			pageSize:      5,
			expectedCount: 5,
			expectedPage:  1,
			expectedSize:  5,
			expectedTotal: 25,
		},
		{
			name:          "Invalid page number corrected",
			page:          0,
			pageSize:      10,
			expectedCount: 10,
			expectedPage:  1,
			expectedSize:  10,
			expectedTotal: 25,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.GetPaginatedTransactionHistory(userId, nil, nil, tc.page, tc.pageSize)
			if err != nil {
				t.Fatalf("failed to get paginated transactions: %v", err)
			}

			if len(result.Transactions) != tc.expectedCount {
				t.Errorf("expected %d transactions, got %d", tc.expectedCount, len(result.Transactions))
			}

			if result.Page != tc.expectedPage {
				t.Errorf("expected page %d, got %d", tc.expectedPage, result.Page)
			}

			if result.PageSize != tc.expectedSize {
				t.Errorf("expected page size %d, got %d", tc.expectedSize, result.PageSize)
			}

			if result.TotalCount != tc.expectedTotal {
				t.Errorf("expected total count %d, got %d", tc.expectedTotal, result.TotalCount)
			}

			expectedTotalPages := (tc.expectedTotal + tc.expectedSize - 1) / tc.expectedSize
			if result.TotalPages != expectedTotalPages {
				t.Errorf("expected total pages %d, got %d", expectedTotalPages, result.TotalPages)
			}
		})
	}
}

func TestMultiUserIsolation(t *testing.T) {
	s := store.NewLedgerStore()
	svc := NewLedgerService(s)

	user1 := "user_one"
	user2 := "user_two"

	_, err := svc.RecordTransaction(user1, models.Deposit, 100.0, "User 1 deposit")
	if err != nil {
		t.Fatalf("failed to record transaction for user 1: %v", err)
	}

	_, err = svc.RecordTransaction(user2, models.Deposit, 200.0, "User 2 deposit")
	if err != nil {
		t.Fatalf("failed to record transaction for user 2: %v", err)
	}

	balance1, err := svc.GetCurrentBalance(user1)
	if err != nil {
		t.Fatalf("failed to get balance for user 1: %v", err)
	}

	balance2, err := svc.GetCurrentBalance(user2)
	if err != nil {
		t.Fatalf("failed to get balance for user 2: %v", err)
	}

	if balance1 != 100.0 {
		t.Errorf("expected user 1 balance to be 100.0, got %.2f", balance1)
	}

	if balance2 != 200.0 {
		t.Errorf("expected user 2 balance to be 200.0, got %.2f", balance2)
	}

	result1, err := svc.GetPaginatedTransactionHistory(user1, nil, nil, 1, 10)
	if err != nil {
		t.Fatalf("failed to get transaction history for user 1: %v", err)
	}

	result2, err := svc.GetPaginatedTransactionHistory(user2, nil, nil, 1, 10)
	if err != nil {
		t.Fatalf("failed to get transaction history for user 2: %v", err)
	}

	if len(result1.Transactions) != 1 || result1.Transactions[0].Amount != 100.0 {
		firstAmount := 0.0
		if len(result1.Transactions) > 0 {
			firstAmount = result1.Transactions[0].Amount
		}
		t.Errorf("expected user 1 to have 1 transaction of amount 100.0, got %d transactions with first amount %.2f",
			len(result1.Transactions), firstAmount)
	}

	if len(result2.Transactions) != 1 || result2.Transactions[0].Amount != 200.0 {
		firstAmount := 0.0
		if len(result2.Transactions) > 0 {
			firstAmount = result2.Transactions[0].Amount
		}
		t.Errorf("expected user 2 to have 1 transaction of amount 200.0, got %d transactions with first amount %.2f",
			len(result2.Transactions), firstAmount)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
