package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

const DefaultImportWorkers = 4
const maxImportWorkers = 32

var ErrInvalidImportWorkers = fmt.Errorf("workers must be an integer between 1 and %d", maxImportWorkers)
var errImportDateRequired = errors.New("date is required")

type ImportSummary struct {
	Accepted int64         `json:"accepted"`
	Rejected int64         `json:"rejected"`
	Errors   []ImportError `json:"errors"`
}

type ImportError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// ImportTransactions returns completed outcomes and ctx.Err() on cancellation.
// Each accepted expense is committed independently through AddTransaction.
func (s *Application) ImportTransactions(ctx context.Context, transactions []domain.Transaction, workers int) (ImportSummary, error) {
	if workers < 1 || workers > maxImportWorkers {
		return ImportSummary{}, ErrInvalidImportWorkers
	}
	if len(transactions) == 0 || ctx.Err() != nil {
		return ImportSummary{Errors: []ImportError{}}, ctx.Err()
	}

	type job struct {
		index int
		tx    domain.Transaction
	}
	type result struct {
		index   int
		message string
	}
	jobs := make(chan job)
	results := make(chan result)
	collected := make(chan ImportSummary)
	go func() {
		var accepted, rejected atomic.Int64
		failures := make([]ImportError, 0)
		for item := range results {
			if item.message == "" {
				accepted.Add(1)
				continue
			}
			rejected.Add(1)
			failures = append(failures, ImportError{Index: item.index, Error: item.message})
		}
		sort.Slice(failures, func(i, j int) bool { return failures[i].Index < failures[j].Index })
		collected <- ImportSummary{Accepted: accepted.Load(), Rejected: rejected.Load(), Errors: failures}
	}()

	var pool sync.WaitGroup
	for range min(len(transactions), workers) {
		pool.Go(func() {
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				var err error
				if job.tx.Date.IsZero() {
					err = errImportDateRequired
				} else {
					_, err = s.AddTransaction(ctx, job.tx)
				}
				item := result{index: job.index}
				switch err {
				case nil:
				case errImportDateRequired, domain.ErrInvalidAmount, domain.ErrInvalidCategory, domain.ErrBudgetNotFound, domain.ErrBudgetExceeded:
					item.message = err.Error()
				default:
					if ctx.Err() != nil {
						return
					}
					log.Printf("import transaction at index %d: %v", job.index, err)
					item.message = "could not save transaction"
				}
				// Publish completed outcomes even if the request was canceled in the meantime.
				results <- item
			}
		})
	}

enqueue:
	for index, tx := range transactions {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- job{index: index, tx: tx}:
		}
	}
	close(jobs)
	pool.Wait()
	close(results)
	summary := <-collected
	return summary, ctx.Err()
}
