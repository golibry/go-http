package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	httpInternal "github.com/golibry/go-http/http"
)

// response_writer.go
//
// Demonstrates the enhanced ResponseWriter, which tracks the status code even
// if the handler doesn't write a body. Useful for logging and metrics.
//
// How to run:
//
//	go run ./_examples/response_writer.go
//
// What to look for:
//
//	The wrapped writer captures the status code set by the handler.
func main() {
	// A handler that sets a custom status and writes no body
	h := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			rw := httpInternal.NewResponseWriter(w)
			rw.WriteHeader(http.StatusAccepted) // 202
			// No body written
		},
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)

	// Wrap the ResponseWriter so we can inspect the resulting status code
	rw := httpInternal.NewResponseWriter(rec)
	h.ServeHTTP(rw, req)

	fmt.Println("Recorder status:", rec.Code)       // what client would see
	fmt.Println("Tracked status:", rw.StatusCode()) // via our wrapper
}
