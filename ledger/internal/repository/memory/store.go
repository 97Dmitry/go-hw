package memory

import (
	"context"
	"math"
	"sync"
	"time"

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

func (r *TransactionRepository) SumByCategory(_ context.Context, category string, period domain.Period) (int64, error) {
	var total int64
	for _, t := range r.GetAllTransactionByCategory(category, period) {
		total += int64(math.Round(t.Amount * 100))
	}
	return total, nil
}

func (r *TransactionRepository) Summary(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	txs, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	totals := map[string]int64{}
	for _, t := range txs {
		if !t.Date.Before(from) && t.Date.Before(to) {
			totals[t.Category] += int64(math.Round(t.Amount * 100))
		}
	}
	result := make([]domain.CategoryTotal, 0, len(totals))
	for category, cents := range totals {
		result = append(result, domain.CategoryTotal{Category: category, Total: float64(cents) / 100})
	}
	return result, nil
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)
var _ domain.TransactionRepository = (*TransactionRepository)(nil)
var _ domain.UnitOfWork = (*Store)(nil)
