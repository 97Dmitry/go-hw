package memory

import (
	"slices"
	"sync"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

// TransactionRepository — потокобезопасное in-memory хранилище транзакций.
// ID выдаёт только репозиторий, начиная с 1: ID == 0 означает «ещё не сохранена».
type TransactionRepository struct {
	mu     sync.RWMutex
	txs    []domain.Transaction
	lastID int64
}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{}
}

func (r *TransactionRepository) Add(tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	tx.ID = r.lastID
	r.txs = append(r.txs, tx)

	return tx, nil
}

// List возвращает копию, чтобы вызывающий код не мог изменить внутреннее состояние хранилища.
func (r *TransactionRepository) List() []domain.Transaction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Clone(r.txs)
}

func (r *TransactionRepository) GetAllTransactionByCategory(category string, period domain.Period) []domain.Transaction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Transaction, 0)

	for _, tx := range r.txs {
		if tx.Category != category {
			continue
		}

		if period.InPeriod(tx.Date) {
			result = append(result, tx)
		}
	}

	return result
}
