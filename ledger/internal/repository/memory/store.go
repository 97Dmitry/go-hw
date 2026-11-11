package memory

import (
	"context"
	"math"
	"sync"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type Store struct {
	Budgets      *BudgetRepository
	Transactions *TransactionRepository
	mu           sync.Mutex
}

func New() *Store {
	return &Store{Budgets: NewBudgetRepository(), Transactions: NewTransactionRepository()}
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(domain.BudgetRepository, domain.TransactionRepository) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(s.Budgets, s.Transactions)
}

func (r *TransactionRepository) SumByCategory(ctx context.Context, category string, period domain.Period) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var total int64
	for _, t := range r.txs {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if t.Category == category && period.InPeriod(t.Date) {
			total += int64(math.Round(t.Amount * 100))
		}
	}
	return total, nil
}

func (r *TransactionRepository) Categories(ctx context.Context, period domain.Period) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	categories := map[string]struct{}{}
	for _, t := range r.txs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if period.InPeriod(t.Date) {
			categories[t.Category] = struct{}{}
		}
	}
	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}
	return result, nil
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)
var _ domain.TransactionRepository = (*TransactionRepository)(nil)
var _ domain.UnitOfWork = (*Store)(nil)
