package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/golibry/go-http/http/router/middleware"
)

// csrf_middleware.go
//
// Demonstrates CSRF protection via a deliberate custom header on unsafe methods.
// Defaults require a header: X-Deliberate-Request: 1 for POST/PUT/PATCH/DELETE.
//
// How to run:
//
//	go run ./_examples/csrf_middleware.go
//
// What to look for:
//
//	 The first request misses the header and gets 403. The second request includes the
//	header and succeeds with 200.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// A simple handler that would normally perform a write operation
	mainHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		},
	)

	csrf := middleware.NewCSRFMiddleware(mainHandler, logger, middleware.CSRFOptions{})

	// 1) Missing deliberate header: should be forbidden
	req1 := httptest.NewRequest(http.MethodPost, "http://example.com/api/items", nil)
	rec1 := httptest.NewRecorder()
	csrf.ServeHTTP(rec1, req1)
	fmt.Println("1) Missing header -> status:", rec1.Code, "body:", rec1.Body.String())

	// 2) With header: should succeed
	req2 := httptest.NewRequest(http.MethodPost, "http://example.com/api/items", nil)
	req2.Header.Set("X-Deliberate-Request", "1")
	rec2 := httptest.NewRecorder()
	csrf.ServeHTTP(rec2, req2)
	fmt.Println("2) With header -> status:", rec2.Code, "body:", rec2.Body.String())
}
