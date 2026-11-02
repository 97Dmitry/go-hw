package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/cache"
	"github.com/97Dmitry/go-hw/ledger/internal/db"
	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/postgres"
)

type CategoryTotal = domain.CategoryTotal

// Open connects production storage. New remains an isolated in-memory fixture.
func Open(ctx context.Context) (*Ledger, error) {
	conn, err := db.Open(ctx)
	if err != nil {
		return nil, err
	}
	log.Print("Ledger connected to PostgreSQL")
	client, err := cache.Open(ctx)
	if err != nil {
		conn.Close()
		return nil, err
	}
	log.Print("Ledger connected to Redis")
	return &Ledger{sql: &postgres.Store{DB: conn}, cache: client}, nil
}

func (l *Ledger) Close() error {
	var errs []error
	if l.cache != nil {
		errs = append(errs, l.cache.Close())
	}
	if l.sql != nil {
		errs = append(errs, l.sql.DB.Close())
	}
	return errors.Join(errs...)
}

// GetReportSummary uses UTC dates and a half-open interval [from,to).
// Cached reports may lag successful writes by up to 30 seconds.
func (l *Ledger) GetReportSummary(ctx context.Context, from, to time.Time) ([]CategoryTotal, error) {
	from = from.UTC()
	to = to.UTC()
	if !from.Before(to) || from.Format("15:04:05.999999999") != "00:00:00" || to.Format("15:04:05.999999999") != "00:00:00" {
		return nil, fmt.Errorf("report requires UTC date boundaries with from < to")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := "report:summary:" + from.Format("2006-01-02") + ":" + to.Format("2006-01-02")
	if l.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		data, err := l.cache.Get(cacheCtx, key).Bytes()
		cancel()
		var result []CategoryTotal
		if err == nil && json.Unmarshal(data, &result) == nil && result != nil {
			log.Printf("report cache hit: %s", key)
			return result, nil
		}
	}
	log.Printf("report cache miss: %s", key)
	var result []CategoryTotal
	var err error
	if l.sql != nil {
		result, err = l.sql.Summary(ctx, from, to)
	} else {
		result = make([]CategoryTotal, 0)
		totals := map[string]float64{}
		for _, t := range l.transactions.ListTransactions() {
			if !t.Date.Before(from) && t.Date.Before(to) {
				totals[t.Category] += t.Amount
			}
		}
		for category, total := range totals {
			result = append(result, CategoryTotal{Category: category, Total: total})
		}
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Category < result[j].Category })
	if l.cache != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := l.cache.Set(cacheCtx, key, data, 30*time.Second).Err(); err != nil {
			log.Printf("report cache set failed: %v", err)
		}
	}
	return result, nil
}
