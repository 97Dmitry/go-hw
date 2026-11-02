package main

import (
	"context"
	"github.com/97Dmitry/go-hw/ledger"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := ledger.Open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	log.Print("Ledger ready; Gateway embeds its own Ledger instance using the same database")
	<-ctx.Done()
	return nil
}
