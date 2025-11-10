package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/golibry/go-http/http/router/middleware"
)

// recoverer_middleware.go
//
// Demonstrates panic recovery middleware. Any panic in downstream handlers is
// intercepted, logged, and converted to a 500 Internal Server Error response.
//
// How to run:
//
//	go run ./_examples/recoverer_middleware.go
//
// What to look for:
//
//	The first request to /panic triggers a panic and returns 500 while logging
//	the error. The second request to /ok returns 200.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// Handler that may panic based on the path
	panicHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/panic" {
				panic("boom")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("no panic"))
		},
	)

	recoverer := middleware.NewRecoverer(panicHandler, ctx, logger)

	// 1) Panicking request
	req1 := httptest.NewRequest(http.MethodGet, "http://example.com/panic", nil)
	rec1 := httptest.NewRecorder()
	recoverer.ServeHTTP(rec1, req1)
	fmt.Println("1) /panic -> status:", rec1.Code, "body:", rec1.Body.String())

	// 2) Healthy request
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/ok", nil)
	rec2 := httptest.NewRecorder()
	recoverer.ServeHTTP(rec2, req2)
	fmt.Println("2) /ok -> status:", rec2.Code, "body:", rec2.Body.String())
}
