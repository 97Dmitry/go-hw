package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/memory"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

type failingTransactions struct {
	domain.TransactionRepository
	err error
}

func (r failingTransactions) SumByCategory(context.Context, string, domain.Period) (int64, error) {
	return 0, r.err
}
func (r failingTransactions) Add(context.Context, domain.Transaction) (domain.Transaction, error) {
	panic("must not insert after sum failure")
}

type work struct {
	budgets      domain.BudgetRepository
	transactions domain.TransactionRepository
}

func (w work) WithinTransaction(ctx context.Context, fn func(domain.BudgetRepository, domain.TransactionRepository) error) error {
	return fn(w.budgets, w.transactions)
}

func TestSumFailureDoesNotInsert(t *testing.T) {
	store := memory.New()
	date := time.Now()
	if _, err := store.Budgets.Set(context.Background(), domain.Budget{Category: "food", Limit: 1, Period: domain.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("sum failed")
	txs := failingTransactions{TransactionRepository: store.Transactions, err: failure}
	// Reads must use the transaction-bound repository, not the outer repository.
	app := service.New(store.Budgets, store.Transactions, work{store.Budgets, txs}, nil)
	if _, err := app.AddTransaction(domain.Transaction{Category: "food", Amount: 0.01, Date: date}); !errors.Is(err, failure) {
		t.Fatalf("want repository error, got %v", err)
	}
	saved, err := app.ListTransactions()
	if err != nil || len(saved) != 0 {
		t.Fatalf("saved after failure: %v %v", saved, err)
	}
}

func TestValidationBeforeRepositories(t *testing.T) {
	app := service.New(nil, nil, nil, nil)
	if _, err := app.AddTransaction(domain.Transaction{Amount: 0, Category: "food"}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("invalid transaction: %v", err)
	}
	if _, err := app.SetBudget(domain.Budget{}); err == nil {
		t.Fatal("invalid budget accepted")
	}
}
