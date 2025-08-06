package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golibry/go-http/http/router"
	"github.com/golibry/go-http/http/router/middleware"
)

func main() {
	// Create a structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	// Example 1: Standalone timeout middleware usage
	log.Println("=== Example 1: Standalone Timeout Middleware ===")
	
	// Create a slow handler for demonstration
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handler started for %s", r.URL.Path)
		time.Sleep(2 * time.Second) // Simulate slow processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Slow response completed"))
	})

	// Create timeout middleware with 1 second timeout
	timeoutOptions := middleware.TimeoutOptions{
		Timeout:      1 * time.Second,
		ErrorMessage: "Request took too long to process",
	}
	
	_ = middleware.NewTimeoutMiddleware(
		slowHandler,
		ctx,
		logger,
		timeoutOptions,
	)

	// Example 2: Integration with router system
	log.Println("\n=== Example 2: Router Integration ===")

	// Create timeout middleware wrapper function
	timeoutWrapper := func(next http.Handler) http.Handler {
		return middleware.NewTimeoutMiddleware(
			next,
			ctx,
			logger,
			middleware.TimeoutOptions{
				Timeout:      3 * time.Second,
				ErrorMessage: "API request timeout",
			},
		)
	}

	// Create logging middleware for comparison
	loggingWrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Printf("Request started: %s %s", r.Method, r.URL.Path)
			
			next.ServeHTTP(w, r)
			
			log.Printf("Request completed: %s %s (took %v)", 
				r.Method, r.URL.Path, time.Since(start))
		})
	}

	// Define named middlewares including timeout
	namedMiddlewares := []router.NamedMiddleware{
		{Name: "timeout", Middleware: timeoutWrapper},
		{Name: "logging", Middleware: loggingWrapper},
	}

	// Create router with named middlewares
	mux := router.NewServerMuxWrapper(namedMiddlewares)

	// Fast handler (completes within timeout)
	fastHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Fast processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Fast response"))
	})

	// Slow handler (exceeds timeout)
	verySlowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Very slow processing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should timeout"))
	})

	// Register routes with default middlewares (including timeout)
	mux.Handle("/fast", fastHandler)
	mux.Handle("/slow", verySlowHandler)

	// Register route with custom timeout (longer timeout)
	customTimeoutWrapper := func(next http.Handler) http.Handler {
		return middleware.NewTimeoutMiddleware(
			next,
			ctx,
			logger,
			middleware.TimeoutOptions{
				Timeout:      10 * time.Second,
				ErrorMessage: "Extended timeout reached",
			},
		)
	}

	overrides := map[string]func(http.Handler) http.Handler{
		"timeout": customTimeoutWrapper,
	}
	mux.HandleWithCustomMiddlewares("/slow-with-long-timeout", verySlowHandler, overrides)

	// Example 3: Different timeout configurations
	log.Println("\n=== Example 3: Different Timeout Configurations ===")

	// Short timeout for API endpoints
	apiTimeoutOptions := middleware.TimeoutOptions{
		Timeout:      2 * time.Second,
		ErrorMessage: "API timeout - please try again",
	}

	// Long timeout for file uploads
	uploadTimeoutOptions := middleware.TimeoutOptions{
		Timeout:      30 * time.Second,
		ErrorMessage: "Upload timeout - file too large or connection too slow",
	}

	// Create different timeout middlewares
	apiTimeout := middleware.NewTimeoutMiddleware(fastHandler, ctx, logger, apiTimeoutOptions)
	uploadTimeout := middleware.NewTimeoutMiddleware(slowHandler, ctx, logger, uploadTimeoutOptions)

	// Register with different timeouts
	mux.Handle("/api/data", apiTimeout)
	mux.Handle("/upload", uploadTimeout)

	log.Println("Server starting on :8080")
	log.Println("Routes:")
	log.Println("  /fast - Fast handler with 3s timeout (should succeed)")
	log.Println("  /slow - Slow handler with 3s timeout (should timeout)")
	log.Println("  /slow-with-long-timeout - Slow handler with 10s timeout (should succeed)")
	log.Println("  /api/data - API endpoint with 2s timeout")
	log.Println("  /upload - Upload endpoint with 30s timeout")
	log.Println("")
	log.Println("Try these commands:")
	log.Println("  curl http://localhost:8080/fast")
	log.Println("  curl http://localhost:8080/slow")
	log.Println("  curl http://localhost:8080/slow-with-long-timeout")

	log.Fatal(http.ListenAndServe(":8080", mux))
}