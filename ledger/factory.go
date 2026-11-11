package ledger

import (
	"context"
	"errors"
	"log"

	"github.com/97Dmitry/go-hw/ledger/internal/cache"
	"github.com/97Dmitry/go-hw/ledger/internal/db"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/memory"
	"github.com/97Dmitry/go-hw/ledger/internal/repository/postgres"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

// New creates an isolated in-memory service for tests, without infrastructure.
func New() Service {
	store := memory.New()
	return service.New(store.Budgets, store.Transactions, store, nil)
}

// Open assembles a ready service. Migrations must be applied separately.
func Open(ctx context.Context) (Service, func() error, error) {
	conn, err := db.Open(ctx)
	if err != nil {
		return nil, nil, err
	}
	log.Print("Ledger connected to PostgreSQL")
	client, err := cache.Open(ctx)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	log.Print("Ledger connected to Redis")
	store := &postgres.Store{DB: conn}
	closeFn := func() error { return errors.Join(client.Close(), conn.Close()) }
	return service.New(store.Budgets(), store.Transactions(), store, cache.Reports{Client: client}), closeFn, nil
}
