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

func TestImportTransactionsRespectsBudgetConcurrently(t *testing.T) {
	store := memory.New()
	app := service.New(store.Budgets, store.Transactions, store, nil)
	date := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	if _, err := app.SetBudget(t.Context(), domain.Budget{Category: "food", Limit: 1, Period: domain.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	transactions := make([]domain.Transaction, 20)
	for i := range transactions {
		transactions[i] = domain.Transaction{Amount: 0.25, Category: "food", Date: date}
	}
	summary, err := app.ImportTransactions(t.Context(), transactions, 8)
	if err != nil || summary.Accepted != 4 || summary.Rejected != 16 || len(summary.Errors) != 16 {
		t.Fatalf("unexpected summary: %+v, error: %v", summary, err)
	}
	for i, failure := range summary.Errors {
		if failure.Index < 0 || failure.Index >= len(transactions) || (i > 0 && failure.Index <= summary.Errors[i-1].Index) {
			t.Fatalf("invalid or duplicate index: %+v", failure)
		}
		if failure.Error != domain.ErrBudgetExceeded.Error() {
			t.Fatalf("unexpected rejection: %+v", failure)
		}
	}
	saved, err := app.ListTransactions(t.Context())
	if err != nil || len(saved) != 4 {
		t.Fatalf("unexpected saved transactions: %v, error: %v", saved, err)
	}
}

type cancelAfterCommit struct {
	domain.UnitOfWork
	cancel context.CancelFunc
}

func (w cancelAfterCommit) WithinTransaction(ctx context.Context, fn func(domain.BudgetRepository, domain.TransactionRepository) error) error {
	err := w.UnitOfWork.WithinTransaction(ctx, fn)
	w.cancel()
	return err
}

func TestImportTransactionsPreservesCommittedWorkOnCancellation(t *testing.T) {
	store := memory.New()
	date := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	if _, err := store.Budgets.Set(t.Context(), domain.Budget{Category: "food", Limit: 100, Period: domain.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := service.New(store.Budgets, store.Transactions, cancelAfterCommit{store, cancel}, nil)
	tx := domain.Transaction{Amount: 1, Category: "food", Date: date}
	summary, err := app.ImportTransactions(ctx, []domain.Transaction{tx, tx, tx}, 1)
	if !errors.Is(err, context.Canceled) || summary.Accepted != 1 || summary.Rejected != 0 || len(summary.Errors) != 0 {
		t.Fatalf("unexpected partial summary: %+v, error: %v", summary, err)
	}
	saved, err := app.ListTransactions(t.Context())
	if err != nil || len(saved) != 1 {
		t.Fatalf("unexpected saved transactions: %v, error: %v", saved, err)
	}
}

type blockingImportWork struct {
	started chan struct{}
	stopped chan struct{}
}

func (w blockingImportWork) WithinTransaction(ctx context.Context, _ func(domain.BudgetRepository, domain.TransactionRepository) error) error {
	w.started <- struct{}{}
	defer func() { w.stopped <- struct{}{} }()
	<-ctx.Done()
	return ctx.Err()
}

func TestImportTransactionsCancelsAndJoinsActiveWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	work := blockingImportWork{started: make(chan struct{}, 3), stopped: make(chan struct{}, 3)}
	app := service.New(nil, nil, work, nil)
	tx := domain.Transaction{Amount: 1, Category: "food", Date: time.Now()}
	type outcome struct {
		summary service.ImportSummary
		err     error
	}
	result := make(chan outcome, 1)
	go func() {
		summary, err := app.ImportTransactions(ctx, []domain.Transaction{tx, tx, tx}, 2)
		result <- outcome{summary, err}
	}()
	startedDeadline := time.After(time.Second)
	for range 2 {
		select {
		case <-work.started:
		case <-startedDeadline:
			t.Fatal("two workers did not reach storage concurrently")
		}
	}
	cancel()
	select {
	case got := <-result:
		if !errors.Is(got.err, context.Canceled) || got.summary.Accepted != 0 || got.summary.Rejected != 0 {
			t.Fatalf("unexpected cancellation result: %+v", got)
		}
		if len(work.stopped) != 2 || len(work.started) != 0 {
			t.Fatalf("unexpected worker activity: %d stopped, %d extra started", len(work.stopped), len(work.started))
		}
	case <-time.After(time.Second):
		t.Fatal("import did not stop on cancellation")
	}
}
