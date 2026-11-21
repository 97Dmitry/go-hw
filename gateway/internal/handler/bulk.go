package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/97Dmitry/go-hw/gateway/internal/api"
	"github.com/97Dmitry/go-hw/ledger"
)

func importTransactions(store ledger.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workers := ledger.DefaultImportWorkers
		query := r.URL.Query()
		if query.Has("workers") {
			var err error
			workers, err = strconv.Atoi(query.Get("workers"))
			if err != nil {
				writeError(w, http.StatusBadRequest, ledger.ErrInvalidImportWorkers.Error())
				return
			}
		}

		var req []api.CreateTransactionRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req == nil {
			writeError(w, http.StatusBadRequest, "request body must be a JSON array")
			return
		}
		transactions := make([]ledger.Transaction, len(req))
		for i, item := range req {
			transactions[i] = item.ToDomain()
		}
		summary, err := store.ImportTransactions(r.Context(), transactions, workers)
		if err != nil {
			if errors.Is(err, ledger.ErrInvalidImportWorkers) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeServiceError(w, "import transactions", err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}
