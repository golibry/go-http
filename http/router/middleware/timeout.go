package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// TimeoutMiddleware provides request timeout handling middleware
//
// The middleware buffers handler output until the handler completes. This keeps
// timeout responses from being mixed with late handler writes. Streaming,
// hijacking, and server-sent event handlers should bypass this middleware.
type TimeoutMiddleware struct {
	next    http.Handler
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

	timeoutWriter := newTimeoutResponseWriter()

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

		tm.next.ServeHTTP(timeoutWriter, r)
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		// Request completed successfully
		if panicValue != nil {
			// Re-panic if there was a panic in the handler
			panic(panicValue)
		}
		timeoutWriter.WriteTo(w)
		return

	case <-ctx.Done():
		// Request timed out
		if tm.logger != nil {
			tm.logger.WarnContext(
				r.Context(),
				"Request timeout",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("timeout", tm.options.Timeout),
			)
		}

		timeoutWriter.MarkTimedOut()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(tm.options.ErrorMessage))
		return
	}
}

type timeoutResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	statusCode  int
	wroteHeader bool
	timedOut    bool
	mu          sync.Mutex
}

func newTimeoutResponseWriter() *timeoutResponseWriter {
	return &timeoutResponseWriter{
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (tw *timeoutResponseWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutResponseWriter) WriteHeader(statusCode int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.wroteHeader {
		return
	}

	tw.statusCode = statusCode
	tw.wroteHeader = true
}

func (tw *timeoutResponseWriter) Write(data []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return len(data), nil
	}

	if !tw.wroteHeader {
		tw.statusCode = http.StatusOK
		tw.wroteHeader = true
	}

	return tw.body.Write(data)
}

func (tw *timeoutResponseWriter) MarkTimedOut() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.timedOut = true
}

func (tw *timeoutResponseWriter) WriteTo(w http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return
	}

	for key, values := range tw.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(tw.statusCode)
	_, _ = w.Write(tw.body.Bytes())
}
