package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/97Dmitry/go-hw/gateway/internal/handler"
	"github.com/97Dmitry/go-hw/ledger"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	store, closeFn, err := ledger.Open(context.Background())
	if err != nil {
		return err
	}
	defer closeFn()
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           handler.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return srv.ListenAndServe()
}
