package handler_test

import (
	"context"
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

type bulkResponse struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Errors   []struct {
		Index int    `json:"index"`
		Error string `json:"error"`
	} `json:"errors"`
}

func TestBulkTransactions(t *testing.T) {
	router := handler.NewRouter(ledger.New())
	budget := createBudget(t, router)
	date := budget.Period.Start.Format(time.RFC3339)
	body := fmt.Sprintf(`[
		{"amount":2000,"category":"еда","description":"обед","date":%q},
		{"amount":6000,"category":"еда","date":%q},
		{"amount":1,"category":"нет бюджета","date":%q},
		{"amount":-1,"category":"еда","date":%q},
		{"amount":1,"category":"еда"}
	]`, date, date, date, date)
	var summary bulkResponse
	request(t, router, http.MethodPost, "/api/transactions/bulk?workers=4", body, http.StatusOK, &summary)
	var indices []int
	for _, failure := range summary.Errors {
		indices = append(indices, failure.Index)
		if failure.Error == "" {
			t.Fatalf("empty reason for item %d", failure.Index)
		}
	}
	if summary.Accepted != 1 || summary.Rejected != 4 || !reflect.DeepEqual(indices, []int{1, 2, 3, 4}) {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	var saved []api.TransactionResponse
	request(t, router, http.MethodGet, "/api/transactions", "", http.StatusOK, &saved)
	if len(saved) != 1 || saved[0].Amount != 2000 || saved[0].Description != "обед" {
		t.Fatalf("unexpected saved transactions: %+v", saved)
	}
}

func TestBulkTransactionsEmpty(t *testing.T) {
	router := handler.NewRouter(ledger.New())
	var summary bulkResponse
	request(t, router, http.MethodPost, "/api/transactions/bulk", `[]`, http.StatusOK, &summary)
	if summary.Accepted != 0 || summary.Rejected != 0 || summary.Errors == nil || len(summary.Errors) != 0 {
		t.Fatalf("unexpected empty summary: %+v", summary)
	}
}

func TestBulkTransactionsBadRequest(t *testing.T) {
	for _, tc := range []struct{ query, body string }{
		{"", `null`},
		{"", `{}`},
		{"", `[] []`},
		{"", `[{"date":"yesterday"}]`},
		{"?workers=", `[]`},
		{"?workers=0", `[]`},
		{"?workers=33", `[]`},
		{"?workers=two", `[]`},
	} {
		var problem map[string]string
		request(t, handler.NewRouter(ledger.New()), http.MethodPost, "/api/transactions/bulk"+tc.query, tc.body, http.StatusBadRequest, &problem)
		if problem["error"] == "" {
			t.Fatalf("missing error for %q %s", tc.query, tc.body)
		}
	}
}

func TestBulkTransactionsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/transactions/bulk", strings.NewReader(`[]`))
	rec := httptest.NewRecorder()
	handler.NewRouter(ledger.New()).ServeHTTP(rec, req)
	if rec.Code != http.StatusGatewayTimeout || strings.TrimSpace(rec.Body.String()) != `{"error":"request timeout"}` {
		t.Fatalf("unexpected cancellation response: %d %s", rec.Code, rec.Body.String())
	}
}
