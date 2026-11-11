package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/97Dmitry/go-hw/gateway/internal/api"
	"github.com/97Dmitry/go-hw/ledger"
)

func NewRouter(store ledger.Service) http.Handler {
	return NewRouterWithTimeout(store, DefaultRequestTimeout)
}

func NewRouterWithTimeout(store ledger.Service, timeout time.Duration) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	mux.HandleFunc("POST /api/transactions", func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateTransactionRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		tx := req.ToDomain()
		if tx.Date.IsZero() {
			writeError(w, http.StatusBadRequest, "date is required")
			return
		}
		if err := tx.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		saved, err := store.AddTransaction(r.Context(), tx)
		if err != nil {
			switch {
			case errors.Is(err, ledger.ErrBudgetExceeded):
				writeError(w, http.StatusConflict, "budget exceeded")
			case errors.Is(err, ledger.ErrBudgetNotFound):
				writeError(w, http.StatusConflict, "budget not found")
			default:
				writeServiceError(w, "add transaction", err)
			}
			return
		}
		writeJSON(w, http.StatusCreated, api.TransactionFromDomain(saved))
	})

	mux.HandleFunc("GET /api/transactions", func(w http.ResponseWriter, r *http.Request) {
		result := make([]api.TransactionResponse, 0)
		txs, err := store.ListTransactions(r.Context())
		if err != nil {
			writeServiceError(w, "list transactions", err)
			return
		}
		for _, tx := range txs {
			result = append(result, api.TransactionFromDomain(tx))
		}
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("POST /api/budgets", func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateBudgetRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		budget := req.ToDomain(time.Now())
		if err := budget.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		saved, err := store.SetBudget(r.Context(), budget)
		if err != nil {
			writeServiceError(w, "set budget", err)
			return
		}
		writeJSON(w, http.StatusCreated, api.BudgetFromDomain(saved))
	})

	mux.HandleFunc("GET /api/budgets", func(w http.ResponseWriter, r *http.Request) {
		result := make([]api.BudgetResponse, 0)
		budgets, err := store.ListBudgets(r.Context())
		if err != nil {
			writeServiceError(w, "list budgets", err)
			return
		}
		for _, b := range budgets {
			result = append(result, api.BudgetFromDomain(b))
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /api/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		from, e1 := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, e2 := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if e1 != nil || e2 != nil || !from.Before(to) {
			writeError(w, 400, "from and to must be YYYY-MM-DD with from < to (exclusive)")
			return
		}
		result, err := store.GetReportSummary(r.Context(), from, to)
		if err != nil {
			writeServiceError(w, "report summary", err)
			return
		}
		writeJSON(w, 200, result)
	})
	return logging(requestTimeout(jsonContentType(mux), timeout))
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.URL.Path == "/api/transactions" || r.URL.Path == "/api/budgets" {
			if r.Method != http.MethodGet && r.Method != http.MethodPost {
				w.Header().Set("Allow", "GET, POST")
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
