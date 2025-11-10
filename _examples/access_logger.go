package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/golibry/go-http/http/router/middleware"
)

// access_logger.go
//
// Demonstrates the HTTP access logger middleware. It logs structured details
// about each request (method, path, status, duration, and optionally client IP).
//
// How to run:
//
//	go run ./_examples/access_logger.go
//
// What to look for:
//
//	The program uses a httptest.ResponseRecorder to execute the handler. The
//	slog logger prints a structured "HTTP Request" log line to stdout.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Your application handler
	mainHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		},
	)

	// Configure the access logger to include client IP in the log
	options := middleware.AccessLogOptions{LogClientIp: true}
	accessLogger := middleware.NewHTTPAccessLogger(mainHandler, logger, options)

	// Execute the middleware-wrapped handler using a test request
	req := httptest.NewRequest(http.MethodPost, "http://example.com/items?limit=10", nil)
	req.RemoteAddr = "203.0.113.10:52345" // will be parsed to client IP by the middleware
	rec := httptest.NewRecorder()

	accessLogger.ServeHTTP(rec, req)

	fmt.Println("Status code captured by recorder:", rec.Code)
	fmt.Println("Body:", rec.Body.String())
}
