package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/ledger"
)

type slowReport struct {
	ledger.Service
	exited chan struct{}
}

func (s slowReport) GetReportSummary(ctx context.Context, _, _ time.Time) ([]ledger.CategoryTotal, error) {
	defer close(s.exited)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestReportTimeout(t *testing.T) {
	exited := make(chan struct{})
	router := NewRouterWithTimeout(slowReport{ledger.New(), exited}, 10*time.Millisecond)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/reports/summary?from=2026-01-01&to=2026-02-01", nil))
	if w.Code != 504 || strings.TrimSpace(w.Body.String()) != "{\"error\":\"request timeout\"}" {
		t.Fatalf("timeout: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content type: %v", w.Header())
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("service did not stop")
	}
}

func TestTimeoutDiscardsLateResponse(t *testing.T) {
	exited := make(chan struct{})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(exited)
		w.WriteHeader(201)
		w.Write([]byte("partial"))
		<-r.Context().Done()
		w.Write([]byte("late"))
	})
	w := httptest.NewRecorder()
	requestTimeout(next, time.Millisecond).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 504 || strings.TrimSpace(w.Body.String()) != "{\"error\":\"request timeout\"}" {
		t.Fatalf("partial response leaked: %d %s", w.Code, w.Body.String())
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("handler did not stop")
	}
}

func TestCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	NewRouter(ledger.New()).ServeHTTP(w, httptest.NewRequest("GET", "/api/budgets", nil).WithContext(ctx))
	if w.Code != 504 || strings.TrimSpace(w.Body.String()) != "{\"error\":\"request timeout\"}" {
		t.Fatalf("cancellation: %d %s", w.Code, w.Body.String())
	}
}

func TestTimeoutAppliesToEveryAPIRoute(t *testing.T) {
	for _, path := range []string{"/api/budgets", "/api/transactions", "/api/reports/summary?from=2026-01-01&to=2026-02-01"} {
		w := httptest.NewRecorder()
		NewRouterWithTimeout(ledger.New(), time.Nanosecond).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 504 {
			t.Fatalf("%s: want 504, got %d", path, w.Code)
		}
	}
}

func TestTimeoutPreservesNormalResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("missing request deadline")
		}
		writeError(w, http.StatusServiceUnavailable, "temporarily unavailable")
	})
	w := httptest.NewRecorder()
	requestTimeout(next, time.Second).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 || strings.TrimSpace(w.Body.String()) != "{\"error\":\"temporarily unavailable\"}" {
		t.Fatalf("normal response changed: %d %s", w.Code, w.Body.String())
	}
}
