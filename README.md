
# go-http

A Go HTTP middleware library providing common functionalities for HTTP request/response handling, error management, access logging, and panic recovery.

Migrated from https://github.com/rsgcata/go-http

## Features

- **Response Writer**: Enhanced HTTP response writer with status code tracking
- **Access Logger**: Structured HTTP access logging with configurable options
- **Error Handler**: Sophisticated error handling with HTTP status code mapping
- **Recoverer**: Panic recovery middleware with structured logging

## Installation

```bash
go get github.com/golibry/go-http
```

## Components

### Response Writer

The `ResponseWriter` wraps the standard `http.ResponseWriter` to track the HTTP status code, which is useful for logging and monitoring.

```go
import httpInternal "github.com/golibry/go-http/http"

func handler(w http.ResponseWriter, r *http.Request) {
    // Wrap the response writer
    rw := httpInternal.NewResponseWriter(w)
    
    // Use it like a normal ResponseWriter
    rw.WriteHeader(http.StatusCreated)
    rw.Write([]byte("Resource created"))
    
    // Get the status code
    statusCode := rw.StatusCode() // Returns 201
}
```

### Access Logger Middleware

Provides structured HTTP access logging with detailed request information.

```go
import (
    "log/slog"
    "net/http"
    "github.com/golibry/go-http/http/router/middleware"
)

func setupAccessLogger() http.Handler {
    logger := slog.Default()
    
    // Your main handler
    mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Hello"))
    })
    
    // Configure access logging options
    options := middleware.AccessLogOptions{
        LogClientIp: true, // Set to false to exclude client IP from logs
    }
    
    // Create access logger middleware
    return middleware.NewHttpAccessLogger(mainHandler, logger, options)
}
```

**Logged Information:**
- HTTP Method
- Host
- Path
- Query parameters
- Protocol version
- User Agent
- Response status code
- Request duration
- Client IP (optional)

### Error Handler Middleware

Error handling with HTTP status code mapping and structured logging.

```go
import (
    "context"
    "errors"
    "log/slog"
    "net/http"
    "github.com/golibry/go-http/http/router/middleware"
)

// Define custom error types
type ValidationError struct {
    Message string
}

func (e ValidationError) Error() string {
    return e.Message
}

// Implement HTTPError interface for custom status codes
func (e ValidationError) StatusCode() int {
    return http.StatusBadRequest
}

// Define a custom handler that returns errors
func myHandler(w http.ResponseWriter, r *http.Request) error {
    if r.URL.Path == "/error" {
        return ValidationError{Message: "Invalid input"}
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Success"))
    return nil
}

func setupErrorHandler() http.Handler {
    logger := slog.Default()
    ctx := context.Background()
    
    // Create error categories for grouping errors by status code
    badRequestCategory := middleware.NewErrorCategory(http.StatusBadRequest)
    badRequestCategory.AddSentinelError(errors.New("validation failed"))
    
    notFoundCategory := middleware.NewErrorCategory(http.StatusNotFound)
    middleware.AddErrorType[*ValidationError](notFoundCategory)
    
    categories := []*middleware.ErrorCategory{
        badRequestCategory,
        notFoundCategory,
    }
    
    return middleware.NewErrorhandler(myHandler, ctx, logger, categories)
}
```

**Error Classification:**
1. **HTTPError Interface**: Errors implementing `StatusCode() int` method
2. **Error Categories**: Group errors by status code using sentinel errors or error types
3. **Default**: Falls back to 500 Internal Server Error

### Recoverer Middleware

Panic recovery middleware that catches panics and converts them to HTTP errors.

```go
import (
    "context"
    "log/slog"
    "net/http"
    "github.com/golibry/go-http/http/router/middleware"
)

func setupRecoverer() http.Handler {
    logger := slog.Default()
    ctx := context.Background()
    
    // Handler that might panic
    panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/panic" {
            panic("Something went wrong!")
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("No panic here"))
    })
    
    // Create recoverer middleware
	recoverer := middleware.NewRecoverer(panicHandler, ctx, logger)
    
    return recoverer
}
```

## Middleware Chaining

You can chain multiple middleware together for comprehensive request handling:

```go
func setupMiddlewareChain() http.Handler {
    logger := slog.Default()
    ctx := context.Background()
    
    // Your main handler
    mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Hello, World!"))
    })
    
    accessLogger := middleware.NewHttpAccessLogger(
        mainHandler,
        logger,
        middleware.AccessLogOptions{LogClientIp: true},
    )

    // Chain middleware: Recoverer -> Access Logger -> Main Handler
    recoverer := middleware.NewRecoverer(accessLogger, ctx, logger)
    
    return recoverer
}
```

## Error Handling Patterns

### Custom HTTP Errors

```go
type NotFoundError struct {
    Resource string
}

func (e NotFoundError) Error() string {
    return fmt.Sprintf("Resource not found: %s", e.Resource)
}

func (e NotFoundError) StatusCode() int {
    return http.StatusNotFound
}
```

### Error Categories

```go
// Create categories for different error types
validationCategory := middleware.NewErrorCategory(http.StatusBadRequest)
validationCategory.AddSentinelError(errors.New("validation error"))

authCategory := middleware.NewErrorCategory(http.StatusUnauthorized)
middleware.AddErrorType[*AuthError](authCategory)
```

## API Reference

### Types

- `ResponseWriter`: Enhanced response writer with status code tracking
- `CustomHandler`: Handler function that returns an error
- `HTTPError`: Interface for errors with HTTP status codes
- `ErrorCategory`: Groups errors by HTTP status code
- `AccessLogOptions`: Configuration for access logging
- `HttpAccessLogger`: Access logging middleware
- `Errorhandler`: Error handling middleware
- `Recoverer`: Panic recovery middleware

### Functions

- `NewResponseWriter(w http.ResponseWriter) *ResponseWriter`
- `NewHttpAccessLogger(next http.Handler, logger *slog.Logger, options AccessLogOptions) *HttpAccessLogger`
- `NewErrorhandler(next CustomHandler, ctx context.Context, logger *slog.Logger, categories []*ErrorCategory) *Errorhandler`
- `NewErrorCategory(statusCode int) *ErrorCategory`
- `AddErrorType[T error](ec *ErrorCategory)`

## Requirements

- Go 1.24.1 or later
- Standard library only (no external dependencies for core functionality)
- `github.com/stretchr/testify` for testing

## License

See the LICENSE file for details.