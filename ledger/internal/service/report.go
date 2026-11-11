package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

// GetReportSummary uses UTC dates and a half-open interval [from,to).
// Cached reports may lag successful writes by up to 30 seconds.
func (s *Application) GetReportSummary(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	from = from.UTC()
	to = to.UTC()
	if !from.Before(to) || from.Format("15:04:05.999999999") != "00:00:00" || to.Format("15:04:05.999999999") != "00:00:00" {
		return nil, fmt.Errorf("report requires UTC date boundaries with from < to")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := "report:summary:" + from.Format("2006-01-02") + ":" + to.Format("2006-01-02")
	if s.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		data, err := s.cache.Get(cacheCtx, key)
		cancel()
		var result []domain.CategoryTotal
		if err == nil && json.Unmarshal(data, &result) == nil && result != nil {
			log.Printf("report cache hit: %s", key)
			return result, nil
		}
	}
	log.Printf("report cache miss: %s", key)
	result, err := s.transactions.Summary(ctx, from, to)
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Category < result[j].Category })
	if s.cache != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		cacheCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := s.cache.Set(cacheCtx, key, data, 30*time.Second); err != nil {
			log.Printf("report cache set failed: %v", err)
		}
	}
	return result, nil
}
