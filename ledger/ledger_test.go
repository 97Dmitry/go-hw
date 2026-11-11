package ledger_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/ledger"
)

func TestTransactionValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		amount   float64
		category string
		wantErr  bool
	}{
		{"valid", 100, "еда", false},
		{"zero_amount", 0, "еда", true},
		{"negative_amount", -1, "еда", true},
		{"empty_category", 100, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tx := ledger.Transaction{Amount: tc.amount, Category: tc.category}
			if err := tx.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestBudgetValidate(t *testing.T) {
	t.Parallel()
	period := ledger.MonthPeriod(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name     string
		limit    float64
		category string
		period   ledger.Period
		wantErr  bool
	}{
		{"valid", 5000, "еда", period, false},
		{"zero_limit", 0, "еда", period, true},
		{"negative_limit", -1, "еда", period, true},
		{"empty_category", 5000, "", period, true},
		{"invalid_period", 5000, "еда", ledger.Period{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := ledger.Budget{Limit: tc.limit, Category: tc.category, Period: tc.period}
			if err := b.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestLedgerBudgetLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		amount  float64
		wantErr error
	}{
		{"within_limit", 2000, nil},
		{"exact_limit", 4000, nil},
		{"exceeded", 4001, ledger.ErrBudgetExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := ledger.New()
			date := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
			if _, err := store.SetBudget(context.Background(), ledger.Budget{Category: "еда", Limit: 5000, Period: ledger.MonthPeriod(date)}); err != nil {
				t.Fatal(err)
			}
			first, err := store.AddTransaction(context.Background(), ledger.Transaction{Amount: 1000, Category: "еда", Date: date})
			if err != nil {
				t.Fatal(err)
			}
			saved, err := store.AddTransaction(context.Background(), ledger.Transaction{Amount: tc.amount, Category: "еда", Date: date})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("AddTransaction() error = %v, want %v", err, tc.wantErr)
			}
			want := []ledger.Transaction{first}
			if tc.wantErr == nil {
				if saved.ID <= first.ID {
					t.Fatalf("invalid saved ID: %d", saved.ID)
				}
				want = append(want, saved)
			}
			if got, err := store.ListTransactions(context.Background()); err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("transactions = %+v, want %+v", got, want)
			}
		})
	}
}

func TestLedgerIsolation(t *testing.T) {
	first, second := ledger.New(), ledger.New()
	date := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	if _, err := first.SetBudget(context.Background(), ledger.Budget{Category: "еда", Limit: 5000, Period: ledger.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	if _, err := first.AddTransaction(context.Background(), ledger.Transaction{Amount: 100, Category: "еда", Date: date}); err != nil {
		t.Fatal(err)
	}
	budgets, err := second.ListBudgets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	txs, err := second.ListTransactions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(budgets) != 0 || len(txs) != 0 {
		t.Fatal("ledger instances share state")
	}
}

func TestMoneyPrecision(t *testing.T) {
	for _, amount := range []float64{0.001, 1.005, 1e14, 1e-8, 0.010000001} {
		tx := ledger.Transaction{Amount: amount, Category: "food"}
		b := ledger.Budget{Limit: amount, Category: "food", Period: ledger.MonthPeriod(time.Now())}
		if tx.Validate() == nil || b.Validate() == nil {
			t.Fatalf("unrepresentable NUMERIC(14,2) value accepted: %v", amount)
		}
	}
}

func TestMoneyValidCents(t *testing.T) {
	for _, amount := range []float64{0.01, 0.29, 670546325022.57, 999999999999.99} {
		tx := ledger.Transaction{Amount: amount, Category: "food"}
		if err := tx.Validate(); err != nil {
			t.Fatalf("valid cents %v rejected: %v", amount, err)
		}
	}
}

func TestExactCentsLimit(t *testing.T) {
	store := ledger.New()
	date := time.Now()
	if _, err := store.SetBudget(context.Background(), ledger.Budget{Category: "cents", Limit: 0.30, Period: ledger.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []float64{0.10, 0.20} {
		if _, err := store.AddTransaction(context.Background(), ledger.Transaction{Category: "cents", Amount: amount, Date: date}); err != nil {
			t.Fatalf("exact cents rejected: %v", err)
		}
	}
	if _, err := store.AddTransaction(context.Background(), ledger.Transaction{Category: "cents", Amount: 0.01, Date: date}); !errors.Is(err, ledger.ErrBudgetExceeded) {
		t.Fatalf("extra cent accepted: %v", err)
	}
	txs, err := store.ListTransactions(context.Background())
	if err != nil || len(txs) != 2 {
		t.Fatalf("rejected expense saved: %v %v", txs, err)
	}
}

func TestConcurrentExpenses(t *testing.T) {
	store := ledger.New()
	date := time.Now()
	if _, err := store.SetBudget(context.Background(), ledger.Budget{Category: "concurrent", Limit: 100, Period: ledger.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 20)
	for range 20 {
		go func() {
			_, err := store.AddTransaction(context.Background(), ledger.Transaction{Category: "concurrent", Amount: 60, Date: date})
			results <- err
		}()
	}
	successes := 0
	for range 20 {
		err := <-results
		if err == nil {
			successes++
		} else if !errors.Is(err, ledger.ErrBudgetExceeded) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("want one success, got %d", successes)
	}
}

func TestReportPeriod(t *testing.T) {
	store := ledger.New()
	ctx := context.Background()
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	for _, category := range []string{"food", "transport"} {
		if _, err := store.SetBudget(ctx, ledger.Budget{Category: category, Limit: 100, Period: ledger.MonthPeriod(from)}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tx := range []ledger.Transaction{
		{Category: "food", Amount: 0.10, Date: from.AddDate(0, 0, -1)},
		{Category: "food", Amount: 0.20, Date: from},
		{Category: "transport", Amount: 0.30, Date: from},
		{Category: "food", Amount: 0.40, Date: to},
	} {
		if _, err := store.AddTransaction(ctx, tx); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetReportSummary(ctx, from, to)
	want := []ledger.CategoryTotal{{Category: "food", Total: 0.20}, {Category: "transport", Total: 0.30}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("report interval: %v %v", got, err)
	}
	empty, err := store.GetReportSummary(ctx, to.AddDate(0, 0, 1), to.AddDate(0, 0, 2))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty report: %v %v", empty, err)
	}
}

func TestCanceledServiceOperations(t *testing.T) {
	store := ledger.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.SetBudget(ctx, ledger.Budget{Category: "food", Limit: 100, Period: ledger.MonthPeriod(time.Now())}); !errors.Is(err, context.Canceled) {
		t.Fatalf("set: %v", err)
	}
	if _, err := store.AddTransaction(ctx, ledger.Transaction{Category: "food", Amount: 1, Date: time.Now()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("add: %v", err)
	}
	if _, err := store.ListBudgets(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("budgets: %v", err)
	}
	if _, err := store.ListTransactions(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("transactions: %v", err)
	}
	budgets, err := store.ListBudgets(context.Background())
	if err != nil || len(budgets) != 0 {
		t.Fatalf("canceled write persisted: %v %v", budgets, err)
	}
}
