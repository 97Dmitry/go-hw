package ledger

import (
	"sync"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/memory"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

type Transaction = domain.Transaction
type Budget = domain.Budget
type Period = domain.Period

var ErrBudgetExceeded = domain.ErrBudgetExceeded
var ErrBudgetNotFound = domain.ErrBudgetNotFound

var MonthPeriod = domain.MonthPeriod

type Ledger struct {
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
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.transactions.AddTransaction(tx)
}
func (l *Ledger) ListTransactions() []Transaction { return l.transactions.ListTransactions() }
func (l *Ledger) SetBudget(b Budget) (Budget, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.budgets.Set(b)
}
func (l *Ledger) ListBudgets() []Budget { return l.budgets.List() }
