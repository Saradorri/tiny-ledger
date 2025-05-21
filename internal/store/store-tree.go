package store

import (
	"errors"
	"sync"
	"time"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"

	"tiny-ledger/internal/models"
)

type userLedgerV2 struct {
	tree    *redblacktree.Tree // key: time, value: TransactionRecord
	balance float64
}

type LedgerStoreV2 struct {
	mu    sync.RWMutex             // for concurrent hashmap and thread-safety, we can move this to user level
	users map[string]*userLedgerV2 //sync.Map is the alternative but limit the lock control and prefer to use lock manually
}

func NewLedgerStoreV2() *LedgerStore {
	return &LedgerStore{
		users: make(map[string]*userLedger),
	}
}

func (s *LedgerStoreV2) AddTransactionV2(userId string, txType models.TransactionType, amount float64, description string) (models.TransactionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ledger, exists := s.users[userId]
	if !exists {
		ledger = &userLedgerV2{
			tree: redblacktree.NewWith(timeComparator),
		}
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

	ledger.tree.Put(tx.Timestamp, tx)

	return tx, nil
}

func (s *LedgerStoreV2) GetPaginatedTransactionsV2(userId string, startTime, endTime *time.Time, page, pageSize int) PaginatedTransactions {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ledger, exists := s.users[userId]
	if !exists {
		return PaginatedTransactions{}
	}

	it := ledger.tree.Iterator()
	transactions := []models.TransactionRecord{}

	// Seek to first valid timestamp
	for it.Next() {
		t := it.Key().(time.Time)
		if startTime != nil && t.Before(*startTime) {
			continue
		}
		if endTime != nil && t.After(*endTime) {
			break
		}
		tx := it.Value().(models.TransactionRecord)
		transactions = append(transactions, tx)
	}

	total := len(transactions)

	// Pagination
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return PaginatedTransactions{
			Transactions: []models.TransactionRecord{},
			TotalCount:   total,
		}
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return PaginatedTransactions{
		Transactions: transactions[start:end],
		TotalCount:   total,
	}
}

func (s *LedgerStoreV2) GetBalanceV2(userId string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ledger, exists := s.users[userId]
	if !exists {
		return 0, nil
	}
	return ledger.balance, nil
}

func timeComparator(a, b interface{}) int {
	t1 := a.(time.Time)
	t2 := b.(time.Time)
	return utils.TimeComparator(t1, t2)
}
