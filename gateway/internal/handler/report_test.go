package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/97Dmitry/go-hw/ledger"
)

type brokenStore struct{ *ledger.Ledger }

func (brokenStore) ListBudgets() ([]ledger.Budget, error) {
	return nil, errors.New("database unavailable")
}
func (brokenStore) ListTransactions() ([]ledger.Transaction, error) {
	return nil, errors.New("database unavailable")
}

func TestReadFailures(t *testing.T) {
	router := NewRouter(brokenStore{ledger.New()})
	for _, path := range []string{"/api/budgets", "/api/transactions"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 500 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
}
func TestReportDates(t *testing.T) {
	router := NewRouter(ledger.New())
	for _, tc := range []struct {
		query  string
		status int
	}{
		{"", 400}, {"?from=2026-02-30&to=2026-03-01", 400}, {"?from=2026-03-01&to=2026-03-01", 400}, {"?from=2026-03-01&to=2026-02-01", 400}, {"?from=2026-02-01&to=2026-03-01", 200},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/reports/summary"+tc.query, nil))
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.query, w.Code)
		}
	}
}
