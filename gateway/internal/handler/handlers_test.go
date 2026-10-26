package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/gateway/internal/api"
	"github.com/97Dmitry/go-hw/gateway/internal/handler"
	"github.com/97Dmitry/go-hw/ledger"
)

func request(t *testing.T, router http.Handler, method, path, body string, status int, dst any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != status {
		t.Fatalf("%s %s: status %d, want %d; body %s", method, path, rec.Code, status, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v; body %s", err, rec.Body.String())
	}
}

func createBudget(t *testing.T, router http.Handler) api.BudgetResponse {
	t.Helper()
	var b api.BudgetResponse
	request(t, router, "POST", "/api/budgets", `{"category":"еда","limit":5000}`, 201, &b)
	if b.ID <= 0 || b.Category != "еда" || b.Limit != 5000 || b.Period.Start.IsZero() || !reflect.DeepEqual(b.Period, ledger.MonthPeriod(b.Period.Start)) {
		t.Fatalf("created budget = %+v", b)
	}
	return b
}

func TestBudgets(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		router := handler.NewRouter(ledger.New())
		b := createBudget(t, router)
		var list []api.BudgetResponse
		request(t, router, "GET", "/api/budgets", "", 200, &list)
		if !reflect.DeepEqual(list, []api.BudgetResponse{b}) {
			t.Fatalf("budgets = %+v, want %+v", list, b)
		}
	})
	for _, tc := range []struct{ name, body string }{
		{"bad_json", `{"category":`},
		{"zero_limit", `{"category":"еда","limit":0}`},
		{"negative_limit", `{"category":"еда","limit":-1}`},
		{"empty_category", `{"category":"","limit":5000}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := handler.NewRouter(ledger.New())
			var problem map[string]string
			request(t, router, "POST", "/api/budgets", tc.body, 400, &problem)
			if problem["error"] == "" {
				t.Fatalf("missing error: %v", problem)
			}
			var list []api.BudgetResponse
			request(t, router, "GET", "/api/budgets", "", 200, &list)
			if list == nil || len(list) != 0 {
				t.Fatalf("budgets after rejection = %+v", list)
			}
		})
	}
}

func TestTransactions(t *testing.T) {
	for _, name := range []string{"ok", "exceeded", "bad_json", "zero_amount", "missing_budget"} {
		t.Run(name, func(t *testing.T) {
			router := handler.NewRouter(ledger.New())
			b := createBudget(t, router)
			date := b.Period.Start.AddDate(0, 0, 1)
			body := func(amount int, category string) string {
				return fmt.Sprintf(`{"amount":%d,"category":%q,"description":"обед","date":%q}`, amount, category, date.Format(time.RFC3339))
			}
			var first api.TransactionResponse
			request(t, router, "POST", "/api/transactions", body(1000, "еда"), 201, &first)
			if first.ID <= 0 || first.Amount != 1000 || first.Category != "еда" || first.Description != "обед" || !first.Date.Equal(date) {
				t.Fatalf("created transaction = %+v", first)
			}
			if name != "ok" {
				payload, status, wantError := body(4001, "еда"), 409, "budget exceeded"
				switch name {
				case "bad_json":
					payload, status, wantError = `{"amount":`, 400, ""
				case "zero_amount":
					payload, status, wantError = body(0, "еда"), 400, ""
				case "missing_budget":
					payload, wantError = body(100, "транспорт"), "budget not found"
				}
				var problem map[string]string
				request(t, router, "POST", "/api/transactions", payload, status, &problem)
				if problem["error"] == "" || (wantError != "" && problem["error"] != wantError) {
					t.Fatalf("error = %v, want %q", problem, wantError)
				}
			}
			var list []api.TransactionResponse
			request(t, router, "GET", "/api/transactions", "", 200, &list)
			if !reflect.DeepEqual(list, []api.TransactionResponse{first}) {
				t.Fatalf("transactions = %+v, want only %+v", list, first)
			}
		})
	}
}
