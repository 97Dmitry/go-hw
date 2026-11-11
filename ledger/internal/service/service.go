package service

import (
	"context"
	"math"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type Service interface {
	SetBudget(context.Context, domain.Budget) (domain.Budget, error)
	ListBudgets(context.Context) ([]domain.Budget, error)
	AddTransaction(context.Context, domain.Transaction) (domain.Transaction, error)
	ListTransactions(context.Context) ([]domain.Transaction, error)
	GetReportSummary(context.Context, time.Time, time.Time) ([]domain.CategoryTotal, error)
}

type Application struct {
	budgets      domain.BudgetRepository
	transactions domain.TransactionRepository
	work         domain.UnitOfWork
	cache        domain.ReportCache
}

func New(b domain.BudgetRepository, t domain.TransactionRepository, work domain.UnitOfWork, cache domain.ReportCache) *Application {
	return &Application{budgets: b, transactions: t, work: work, cache: cache}
}

func (s *Application) SetBudget(ctx context.Context, b domain.Budget) (saved domain.Budget, err error) {
	if err = b.Validate(); err != nil {
		return
	}
	b.Period = domain.MonthPeriod(b.Period.Start)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = s.work.WithinTransaction(ctx, func(budgets domain.BudgetRepository, _ domain.TransactionRepository) error {
		saved, err = budgets.Set(ctx, b)
		return err
	})
	return
}

func (s *Application) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.budgets.List(ctx)
}

func (s *Application) AddTransaction(ctx context.Context, t domain.Transaction) (saved domain.Transaction, err error) {
	if err = t.Validate(); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = s.work.WithinTransaction(ctx, func(budgets domain.BudgetRepository, transactions domain.TransactionRepository) error {
		budget, err := budgets.GetByCategory(ctx, t.Category, t.Date)
		if err != nil {
			return err
		}
		spent, err := transactions.SumByCategory(ctx, t.Category, budget.Period)
		if err != nil {
			return err
		}
		if spent+int64(math.Round(t.Amount*100)) > int64(math.Round(budget.Limit*100)) {
			return domain.ErrBudgetExceeded
		}
		saved, err = transactions.Add(ctx, t)
		return err
	})
	return
}

func (s *Application) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.transactions.List(ctx)
}

var _ Service = (*Application)(nil)
