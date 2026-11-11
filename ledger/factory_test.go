package ledger_test

import (
	"context"
	"errors"
	"testing"

	"github.com/97Dmitry/go-hw/ledger"
)

func TestOpenCanceledContext(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:1/cashapp?sslmode=disable")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app, closeFn, err := ledger.Open(ctx)
	if !errors.Is(err, context.Canceled) || app != nil || closeFn != nil {
		t.Fatalf("want cancellation and no resources, got %v %v %v", app, closeFn != nil, err)
	}
}
