package service_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

type reportRepository struct {
	domain.TransactionRepository
	categories func(context.Context, domain.Period) ([]string, error)
	sum        func(context.Context, string, domain.Period) (int64, error)
}

func (r reportRepository) Categories(ctx context.Context, p domain.Period) ([]string, error) {
	return r.categories(ctx, p)
}
func (r reportRepository) SumByCategory(ctx context.Context, c string, p domain.Period) (int64, error) {
	return r.sum(ctx, c, p)
}

func TestReportRunsCategoriesConcurrently(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := make(chan string, 2)
	release := make(chan struct{})
	period := domain.MonthPeriod(time.Now())
	repo := reportRepository{
		categories: func(ctx context.Context, p domain.Period) ([]string, error) {
			if p != period {
				t.Errorf("period: %v", p)
			}
			return []string{"transport", "food"}, nil
		},
		sum: func(ctx context.Context, c string, p domain.Period) (int64, error) {
			started <- c
			select {
			case <-release:
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			if c == "food" {
				return 12345, nil
			}
			return 6789, nil
		},
	}
	app := service.New(nil, repo, nil, nil)
	type response struct {
		rows []domain.CategoryTotal
		err  error
	}
	done := make(chan response, 1)
	go func() { rows, err := app.GetReportSummary(ctx, period.Start, period.End); done <- response{rows, err} }()
	for range 2 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("category calculations did not run concurrently")
		}
	}
	close(release)
	result := <-done
	if result.err != nil || len(result.rows) != 2 || result.rows[0] != (domain.CategoryTotal{Category: "food", Total: 123.45}) || result.rows[1].Total != 67.89 {
		t.Fatalf("report: %+v %v", result.rows, result.err)
	}
}

func TestReportCancellationJoinsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 2)
	var active atomic.Int32
	repo := reportRepository{
		categories: func(context.Context, domain.Period) ([]string, error) { return []string{"a", "b"}, nil },
		sum: func(ctx context.Context, _ string, _ domain.Period) (int64, error) {
			active.Add(1)
			defer active.Add(-1)
			started <- struct{}{}
			<-ctx.Done()
			return 0, ctx.Err()
		},
	}
	app := service.New(nil, repo, nil, nil)
	period := domain.MonthPeriod(time.Now())
	done := make(chan error, 1)
	go func() { _, err := app.GetReportSummary(ctx, period.Start, period.End); done <- err }()
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers not started")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("workers did not exit")
	}
	if active.Load() != 0 {
		t.Fatal("workers still running after report returned")
	}
}

func TestReportWorkerFailureCancelsPeers(t *testing.T) {
	failure := errors.New("sum unavailable")
	started := make(chan struct{})
	repo := reportRepository{
		categories: func(context.Context, domain.Period) ([]string, error) { return []string{"slow", "broken"}, nil },
		sum: func(ctx context.Context, c string, _ domain.Period) (int64, error) {
			if c == "slow" {
				close(started)
				<-ctx.Done()
				return 0, ctx.Err()
			}
			select {
			case <-started:
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			return 0, failure
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	period := domain.MonthPeriod(time.Now())
	_, err := service.New(nil, repo, nil, nil).GetReportSummary(ctx, period.Start, period.End)
	if !errors.Is(err, failure) {
		t.Fatalf("want original failure, got %v", err)
	}
}

// The heartbeat is observed through its real log output; no production delay is
// added just to make timeouts visible.
type heartbeatLog struct {
	tick    chan struct{}
	stopped chan struct{}
}

func (w heartbeatLog) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("report calculation in progress")) {
		select {
		case w.tick <- struct{}{}:
		default:
		}
	}
	if bytes.Contains(p, []byte("report heartbeat stopped")) {
		select {
		case w.stopped <- struct{}{}:
		default:
		}
	}
	return len(p), nil
}
func TestReportHeartbeatStopsOnCancellation(t *testing.T) {
	output := heartbeatLog{tick: make(chan struct{}, 1), stopped: make(chan struct{}, 1)}
	previous := log.Writer()
	log.SetOutput(output)
	defer log.SetOutput(previous)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := reportRepository{
		categories: func(context.Context, domain.Period) ([]string, error) { return []string{"slow"}, nil },
		sum:        func(ctx context.Context, _ string, _ domain.Period) (int64, error) { <-ctx.Done(); return 0, ctx.Err() },
	}
	period := domain.MonthPeriod(time.Now())
	done := make(chan error, 1)
	go func() {
		_, err := service.New(nil, repo, nil, nil).GetReportSummary(ctx, period.Start, period.End)
		done <- err
	}()
	select {
	case <-output.tick:
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat did not tick")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("report did not stop")
	}
	select {
	case <-output.stopped:
	default:
		t.Fatal("report returned before stopping heartbeat")
	}
}

func TestCanceledReportDoesNotAccessRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	period := domain.MonthPeriod(time.Now())
	_, err := service.New(nil, nil, nil, nil).GetReportSummary(ctx, period.Start, period.End)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

type cancelingCache struct {
	domain.ReportCache
	cancel context.CancelFunc
}

func (c cancelingCache) Get(context.Context, string) ([]byte, error) {
	c.cancel()
	return []byte(`[{"category":"food","total":1}]`), nil
}
func TestCanceledCacheReadDoesNotReturnSuccessOrRecalculate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	period := domain.MonthPeriod(time.Now())
	// A nil repository also catches attempts to fall back after cancellation.
	app := service.New(nil, nil, nil, cancelingCache{cancel: cancel})
	_, err := app.GetReportSummary(ctx, period.Start, period.End)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled cached report: %v", err)
	}
}

func TestCategoryQueryCancellationReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := reportRepository{categories: func(context.Context, domain.Period) ([]string, error) {
		cancel()
		return nil, errors.New("driver interrupted")
	}}
	period := domain.MonthPeriod(time.Now())
	_, err := service.New(nil, repo, nil, nil).GetReportSummary(ctx, period.Start, period.End)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want cancellation, got %v", err)
	}
}
