package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"tiny-ledger/internal/models"
)

type userLedger struct {
	transactions []models.TransactionRecord
	balance      float64 // based on float is not accurate it's better not to float!!
}

type PaginatedTransactions struct {
	Transactions []models.TransactionRecord
	TotalCount   int
}

type LedgerStore struct {
	mu    sync.RWMutex           // for concurrent hashmap and thread-safety
	users map[string]*userLedger //sync.Map is the alternative but limit the lock control and prefer to use lock manually
}

func NewLedgerStore() *LedgerStore {
	return &LedgerStore{
		users: make(map[string]*userLedger),
	}
}

func (s *LedgerStore) AddTransaction(userId string, txType models.TransactionType, amount float64, description string) (models.TransactionRecord, error) {
	s.mu.Lock() // Lock for writing
	defer s.mu.Unlock()

	ledger, exists := s.users[userId]
	if !exists {
		ledger = &userLedger{}
		s.users[userId] = ledger
	}

	if txType == models.Withdrawal && ledger.balance < amount {
		return models.TransactionRecord{}, errors.New("insufficient funds")
	}

	tx := models.NewTransactionRecord(txType, amount, description)

	if txType == models.Deposit {
		ledger.balance += amount
	} else {
		ledger.balance -= amount
	}

	ledger.transactions = append(ledger.transactions, tx)
	// sort when inserting help optimize get transaction history between 2 dates based on the current structure
	sort.SliceStable(ledger.transactions, func(i, j int) bool {
		return ledger.transactions[i].Timestamp.Before(ledger.transactions[j].Timestamp)
	})

	return tx, nil
}

// AddTransactionWithTime add transaction with specific time just for test purpose
func (s *LedgerStore) AddTransactionWithTime(userId string, tx models.TransactionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ledger, exists := s.users[userId]
	if !exists {
		ledger = &userLedger{}
		s.users[userId] = ledger
	}

	if tx.Type == models.Deposit {
		ledger.balance += tx.Amount
	} else if tx.Type == models.Withdrawal {
		ledger.balance -= tx.Amount
	}

	ledger.transactions = append(ledger.transactions, tx)
	sort.SliceStable(ledger.transactions, func(i, j int) bool {
		return ledger.transactions[i].Timestamp.Before(ledger.transactions[j].Timestamp)
	})
}

func (s *LedgerStore) GetPaginatedTransactions(userId string, startTime, endTime *time.Time, page, pageSize int) PaginatedTransactions {
	s.mu.RLock() // RLock for reading
	defer s.mu.RUnlock()

	ledger, exists := s.users[userId]
	if !exists {
		return PaginatedTransactions{
			Transactions: []models.TransactionRecord{},
			TotalCount:   0,
		}
	}

	n := len(ledger.transactions)

	//  first of all: apply time filterings that start index ≥ startTime
	startIdx := 0
	if startTime != nil {
		startIdx = sort.Search(n, func(i int) bool {
			return !ledger.transactions[i].Timestamp.Before(*startTime)
		})
	}

	// end index > endTime
	endIdx := n
	if endTime != nil {
		endIdx = sort.Search(n, func(i int) bool {
			return ledger.transactions[i].Timestamp.After(*endTime)
		})
	}

	filteredCount := endIdx - startIdx

	if page < 1 {
		page = 1
	}

	pageStartIdx := startIdx + (page-1)*pageSize
	pageEndIdx := pageStartIdx + pageSize

	if pageStartIdx >= endIdx {
		return PaginatedTransactions{
			Transactions: []models.TransactionRecord{},
			TotalCount:   filteredCount,
		}
	}

	if pageEndIdx > endIdx {
		pageEndIdx = endIdx
	}

	// deep copy of the subset for thread safety
	pageTransactions := make([]models.TransactionRecord, pageEndIdx-pageStartIdx)
	copy(pageTransactions, ledger.transactions[pageStartIdx:pageEndIdx])

	return PaginatedTransactions{
		Transactions: pageTransactions,
		TotalCount:   filteredCount,
	}
}

func (s *LedgerStore) GetBalance(userId string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ledger, exists := s.users[userId]
	if !exists {
		return 0, nil
	}
	return ledger.balance, nil
}
