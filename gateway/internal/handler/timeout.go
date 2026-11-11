package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

const DefaultRequestTimeout = 2 * time.Second
const timeoutMessage = "request timeout"

// TimeoutHandler buffers the response and rejects late writes. Work must still
// observe the request context: Go cannot forcibly stop a handler goroutine.
func requestTimeout(next http.Handler, timeout time.Duration) http.Handler {
	bounded := http.TimeoutHandler(next, timeout, "")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		bounded.ServeHTTP(&deadlineWriter{ResponseWriter: w, ctx: ctx}, r.WithContext(ctx))
	})
}

// Adapt TimeoutHandler's 503/HTML response to the API's 504/JSON contract.
// The outer deadline is also checked when the handler finishes at the deadline.
type deadlineWriter struct {
	http.ResponseWriter
	ctx     context.Context
	expired bool
}

func (w *deadlineWriter) WriteHeader(status int) {
	if w.ctx.Err() != nil {
		w.expired = true
		clear(w.Header())
		log.Printf("request canceled: %v", w.ctx.Err())
		writeError(w.ResponseWriter, http.StatusGatewayTimeout, timeoutMessage)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *deadlineWriter) Write(data []byte) (int, error) {
	if w.expired {
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}

func writeServiceError(w http.ResponseWriter, operation string, err error) {
	log.Printf("%s: %v", operation, err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusGatewayTimeout, timeoutMessage)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
