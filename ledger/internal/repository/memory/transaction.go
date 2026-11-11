package memory

import (
	"context"
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

func (r *TransactionRepository) Add(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Transaction{}, err
	}

	r.lastID++
	tx.ID = r.lastID
	r.txs = append(r.txs, tx)

	return tx, nil
}

// List возвращает копию, чтобы вызывающий код не мог изменить внутреннее состояние хранилища.
func (r *TransactionRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return slices.Clone(r.txs), nil
}
