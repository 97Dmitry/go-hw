package ledger_test

import (
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
			if _, err := store.SetBudget(ledger.Budget{Category: "еда", Limit: 5000, Period: ledger.MonthPeriod(date)}); err != nil {
				t.Fatal(err)
			}
			first, err := store.AddTransaction(ledger.Transaction{Amount: 1000, Category: "еда", Date: date})
			if err != nil {
				t.Fatal(err)
			}
			saved, err := store.AddTransaction(ledger.Transaction{Amount: tc.amount, Category: "еда", Date: date})
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
			if got := store.ListTransactions(); !reflect.DeepEqual(got, want) {
				t.Fatalf("transactions = %+v, want %+v", got, want)
			}
		})
	}
}

func TestLedgerIsolation(t *testing.T) {
	first, second := ledger.New(), ledger.New()
	date := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	if _, err := first.SetBudget(ledger.Budget{Category: "еда", Limit: 5000, Period: ledger.MonthPeriod(date)}); err != nil {
		t.Fatal(err)
	}
	if _, err := first.AddTransaction(ledger.Transaction{Amount: 100, Category: "еда", Date: date}); err != nil {
		t.Fatal(err)
	}
	if len(second.ListBudgets()) != 0 || len(second.ListTransactions()) != 0 {
		t.Fatal("ledger instances share state")
	}
}
