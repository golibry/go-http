package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golibry/go-http/http/router/middleware"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Example 1: Route with 5-second timeout
	fastHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second) // Simulate work
		fmt.Fprintf(w, "Fast route completed in 1 second")
	})

	fastRouteWithTimeout := middleware.NewTimeoutMiddlewareWithDuration(
		fastHandler,
		ctx,
		logger,
		5*time.Second, // 5-second timeout
	)

	// Example 2: Route with 2-second timeout and custom message
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // This will timeout
		fmt.Fprintf(w, "This should not be reached")
	})

	slowRouteOptions := middleware.TimeoutOptions{
		Timeout: 2 * time.Second,
		Message: "This route timed out after 2 seconds",
	}

	slowRouteWithTimeout := middleware.NewTimeoutMiddleware(
		slowHandler,
		ctx,
		logger,
		slowRouteOptions,
	)

	// Example 3: Route with default timeout (30 seconds)
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintf(w, "Normal route completed")
	})

	normalRouteWithTimeout := middleware.NewTimeoutMiddleware(
		normalHandler,
		ctx,
		logger,
		middleware.DefaultTimeoutOptions(),
	)

	// Set up routes
	http.Handle("/fast", fastRouteWithTimeout)
	http.Handle("/slow", slowRouteWithTimeout)
	http.Handle("/normal", normalRouteWithTimeout)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try these routes:")
	fmt.Println("  GET /fast   - 5s timeout, completes in 1s")
	fmt.Println("  GET /slow   - 2s timeout, tries to take 3s (will timeout)")
	fmt.Println("  GET /normal - 30s timeout, completes in 0.5s")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("Server failed to start", "error", err)
	}
}