package ledger_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/ledger"
)

func TestPersistence(t *testing.T) {
	if os.Getenv("LEDGER_INTEGRATION") != "1" {
		t.Skip("set LEDGER_INTEGRATION=1 against migrated isolated PostgreSQL and Redis")
	}
	ctx := context.Background()
	store, err := ledger.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	date := time.Date(2100+int(time.Now().UnixNano()%7000), 2, 10, 12, 0, 0, 0, time.UTC)
	category := fmt.Sprintf("test-%d", time.Now().UnixNano())
	b := ledger.Budget{Category: category, Limit: 100, Period: ledger.MonthPeriod(date)}
	zone := time.FixedZone("test-offset", -5*60*60)
	b.Period.Start = b.Period.Start.In(zone)
	b.Period.End = b.Period.End.In(zone)
	first, err := store.SetBudget(b)
	if err != nil {
		t.Fatal(err)
	}
	b.Limit = 90
	updated, err := store.SetBudget(b)
	if err != nil || updated.ID != first.ID {
		t.Fatalf("upsert: %+v %v", updated, err)
	}
	b.Period = ledger.MonthPeriod(date.AddDate(0, 1, 0))
	if _, err := store.SetBudget(b); err != nil {
		t.Fatal(err)
	}
	other, err := ledger.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, writer := range []*ledger.Ledger{store, other} {
		wg.Go(func() {
			_, err := writer.AddTransaction(ledger.Transaction{Amount: 60, Category: category, Date: date})
			results <- err
		})
	}
	wg.Wait()
	close(results)
	successes, exceeded := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ledger.ErrBudgetExceeded) {
			exceeded++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || exceeded != 1 {
		t.Fatalf("concurrent budget: %d successes, %d exceeded", successes, exceeded)
	}
	if _, err := store.AddTransaction(ledger.Transaction{Amount: 1, Category: category, Date: date.AddDate(0, 2, 0)}); !errors.Is(err, ledger.ErrBudgetNotFound) {
		t.Fatalf("missing monthly budget: %v", err)
	}
	from, to := ledger.MonthPeriod(date).Start, ledger.MonthPeriod(date).End
	report, err := store.GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range report {
		if row.Category == category && row.Total == 60 {
			found = true
		}
	}
	if !found {
		t.Fatalf("report: %+v", report)
	}
	if _, err := store.GetReportSummary(ctx, from, to); err != nil {
		t.Fatal(err)
	}
	reopened, err := ledger.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	txs, err := reopened.ListTransactions()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, tx := range txs {
		if tx.Category == category {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("persisted count: %d", count)
	}
}
