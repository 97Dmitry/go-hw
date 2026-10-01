package main

import (
	"log"
	"net/http"
	"time"

	"github.com/97Dmitry/go-hw/gateway/internal/handler"
)

func main() {
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           handler.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
