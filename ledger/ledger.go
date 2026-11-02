package ledger

import (
	"context"
	"sync"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/memory"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/postgres"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
	"github.com/redis/go-redis/v9"
)

type Transaction = domain.Transaction
type Budget = domain.Budget
type Period = domain.Period

var ErrBudgetExceeded = domain.ErrBudgetExceeded
var ErrBudgetNotFound = domain.ErrBudgetNotFound

var MonthPeriod = domain.MonthPeriod

type Ledger struct {
	sql          *postgres.Store
	cache        *redis.Client
	mu           sync.Mutex
	transactions *service.TransactionService
	budgets      *service.BudgetService
}

func New() *Ledger {
	budgets := service.NewBudgetService(memory.NewBudgetRepository())
	transactions := service.NewTransactionService(memory.NewTransactionRepository(), budgets)
	return &Ledger{transactions: transactions, budgets: budgets}
}

func (l *Ledger) AddTransaction(tx Transaction) (Transaction, error) {
	if l.sql != nil {
		if err := tx.Validate(); err != nil {
			return Transaction{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return l.sql.AddTransaction(ctx, tx)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.transactions.AddTransaction(tx)
}
func (l *Ledger) ListTransactions() ([]Transaction, error) {
	if l.sql != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return l.sql.ListTransactions(ctx)
	}
	return l.transactions.ListTransactions(), nil
}
func (l *Ledger) SetBudget(b Budget) (Budget, error) {
	if l.sql != nil {
		if err := b.Validate(); err != nil {
			return Budget{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return l.sql.SetBudget(ctx, b)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.budgets.Set(b)
}
func (l *Ledger) ListBudgets() ([]Budget, error) {
	if l.sql != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return l.sql.ListBudgets(ctx)
	}
	return l.budgets.List(), nil
}
