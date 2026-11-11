package domain

import (
	"context"
	"time"
)

// BudgetRepository stores monthly budgets. Set creates or updates a budget.
type BudgetRepository interface {
	Add(context.Context, Budget) (Budget, error)
	Set(context.Context, Budget) (Budget, error)
	GetByCategory(context.Context, string, time.Time) (Budget, error)
	List(context.Context) ([]Budget, error)
}

type TransactionRepository interface {
	Add(context.Context, Transaction) (Transaction, error)
	List(context.Context) ([]Transaction, error)
	SumByCategory(context.Context, string, Period) (int64, error) // integer cents
	Summary(context.Context, time.Time, time.Time) ([]CategoryTotal, error)
}

// UnitOfWork serializes budget changes and expenses for the same monthly budget.
// The callback must use its supplied repositories; errors roll back SQL writes.
type UnitOfWork interface {
	WithinTransaction(context.Context, func(BudgetRepository, TransactionRepository) error) error
}

type ReportCache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
}
