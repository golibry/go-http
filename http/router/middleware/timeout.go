package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// TimeoutMiddleware provides request timeout handling middleware
type TimeoutMiddleware struct {
	next    http.Handler
	ctx     context.Context
	logger  *slog.Logger
	options TimeoutOptions
}

// TimeoutOptions configures the timeout middleware behavior
type TimeoutOptions struct {
	Timeout      time.Duration // Request timeout duration
	ErrorMessage string        // Custom error message for timeout
}

// NewTimeoutMiddleware creates new timeout middleware
func NewTimeoutMiddleware(
	next http.Handler,
	ctx context.Context,
	logger *slog.Logger,
	options TimeoutOptions,
) *TimeoutMiddleware {
	// Set default timeout if not specified
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	// Set default error message if not specified
	if options.ErrorMessage == "" {
		options.ErrorMessage = "Request timeout"
	}

	return &TimeoutMiddleware{
		next:    next,
		ctx:     ctx,
		logger:  logger,
		options: options,
	}
}

// ServeHTTP implements the middleware logic
func (tm *TimeoutMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), tm.options.Timeout)
	defer cancel()

	// Create a new request with the timeout context
	r = r.WithContext(ctx)

	// Channel to signal completion
	done := make(chan struct{})
	var panicValue interface{}

	// Run the next handler in a goroutine
	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicValue = p
			}
			close(done)
		}()

		tm.next.ServeHTTP(w, r)
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		// Request completed successfully
		if panicValue != nil {
			// Re-panic if there was a panic in the handler
			panic(panicValue)
		}
		return

	case <-ctx.Done():
		// Request timed out
		if tm.logger != nil {
			tm.logger.WarnContext(
				tm.ctx,
				"Request timeout",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("timeout", tm.options.Timeout),
			)
		}

		// Check if the response has already been written
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusRequestTimeout)
			_, _ = w.Write([]byte(tm.options.ErrorMessage))
		}
		return
	}
}
