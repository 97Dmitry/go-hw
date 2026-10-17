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

type ledgerService interface {
	AddTransaction(ledger.Transaction) (ledger.Transaction, error)
	ListTransactions() []ledger.Transaction
	SetBudget(ledger.Budget) (ledger.Budget, error)
	ListBudgets() []ledger.Budget
}

func NewRouter(store ledgerService) http.Handler {
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
		saved, err := store.AddTransaction(tx)
		if err != nil {
			switch {
			case errors.Is(err, ledger.ErrBudgetExceeded):
				writeError(w, http.StatusConflict, "budget exceeded")
			case errors.Is(err, ledger.ErrBudgetNotFound):
				writeError(w, http.StatusConflict, "budget not found")
			default:
				log.Printf("add transaction: %v", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}
		writeJSON(w, http.StatusCreated, api.TransactionFromDomain(saved))
	})

	mux.HandleFunc("GET /api/transactions", func(w http.ResponseWriter, _ *http.Request) {
		result := make([]api.TransactionResponse, 0)
		for _, tx := range store.ListTransactions() {
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
		saved, err := store.SetBudget(budget)
		if err != nil {
			log.Printf("set budget: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusCreated, api.BudgetFromDomain(saved))
	})

	mux.HandleFunc("GET /api/budgets", func(w http.ResponseWriter, _ *http.Request) {
		result := make([]api.BudgetResponse, 0)
		for _, b := range store.ListBudgets() {
			result = append(result, api.BudgetFromDomain(b))
		}
		writeJSON(w, http.StatusOK, result)
	})
	return logging(jsonContentType(mux))
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
	if err := json.NewEncoder(w).Encode(value); err != nil {
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
