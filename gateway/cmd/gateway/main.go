package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
	timeout := handler.DefaultRequestTimeout
	if value := os.Getenv("HTTP_REQUEST_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("HTTP_REQUEST_TIMEOUT must be a positive duration")
		}
		timeout = parsed
	}

	store, closeFn, err := ledger.Open(context.Background())
	if err != nil {
		return err
	}
	defer closeFn()
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           handler.NewRouterWithTimeout(store, timeout),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return srv.ListenAndServe()
}
