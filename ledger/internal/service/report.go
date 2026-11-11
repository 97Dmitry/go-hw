package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

// GetReportSummary uses UTC dates and [from,to). Cache entries may lag writes
// by 30 seconds. Every storage operation inherits the caller's cancellation.
func (s *Application) GetReportSummary(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	from, to = from.UTC(), to.UTC()
	if !from.Before(to) || from.Format("15:04:05.999999999") != "00:00:00" || to.Format("15:04:05.999999999") != "00:00:00" {
		return nil, fmt.Errorf("report requires UTC date boundaries with from < to")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := "report:summary:" + from.Format("2006-01-02") + ":" + to.Format("2006-01-02")
	if s.cache != nil {
		cacheCtx, stop := context.WithTimeout(ctx, time.Second)
		data, err := s.cache.Get(cacheCtx, key)
		stop()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var result []domain.CategoryTotal
		if err == nil && json.Unmarshal(data, &result) == nil && result != nil {
			log.Printf("report cache hit: %s", key)
			return result, nil
		}
	}
	log.Printf("report cache miss: %s", key)
	result, err := s.calculateSummary(ctx, domain.Period{Start: from, End: to})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Category < result[j].Category })
	if s.cache != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		cacheCtx, stop := context.WithTimeout(ctx, time.Second)
		err = s.cache.Set(cacheCtx, key, data, 30*time.Second)
		stop()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			log.Printf("report cache set failed: %v", err)
		}
	}
	return result, nil
}

func (s *Application) calculateSummary(ctx context.Context, period domain.Period) ([]domain.CategoryTotal, error) {
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	var heartbeat sync.WaitGroup
	heartbeat.Add(1)
	go func() {
		defer heartbeat.Done()
		ticker := time.NewTicker(400 * time.Millisecond)
		defer func() { ticker.Stop(); log.Print("report heartbeat stopped") }()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				log.Print("report calculation in progress")
			}
		}
	}()
	defer func() { close(done); heartbeat.Wait() }()
	defer func() {
		if err := ctx.Err(); err != nil {
			log.Printf("report calculation canceled: %v", err)
		}
	}()
	log.Print("report calculation started")
	categories, err := s.transactions.Categories(ctx, period)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	results := make([]domain.CategoryTotal, len(categories))
	failures := make([]error, len(categories))
	var workers sync.WaitGroup
	for i, category := range categories {
		if ctx.Err() != nil {
			break
		}
		workers.Go(func() {
			if err := ctx.Err(); err != nil {
				failures[i] = err
				return
			}
			cents, err := s.transactions.SumByCategory(ctx, category, period)
			if err != nil {
				failures[i] = err
				cancel()
				return
			}
			// Each worker owns one index; no concurrent append or shared map writes.
			results[i] = domain.CategoryTotal{Category: category, Total: float64(cents) / 100}
		})
	}
	workers.Wait()
	if err := parent.Err(); err != nil {
		return nil, err
	}
	// Preserve the storage failure that triggered peer cancellation.
	for _, err := range failures {
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	log.Print("report calculation completed")
	return results, nil
}
