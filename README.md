
# go-http

A Go HTTP middleware library providing common functionalities for HTTP request/response handling, error management, access logging, and panic recovery.

Migrated from https://github.com/rsgcata/go-http

## Features

- **Response Writer**: Enhanced HTTP response writer with status code tracking
- **Access Logger**: Structured HTTP access logging with configurable options
- **Recoverer**: Panic recovery middleware with structured logging
- **Session Management**: Comprehensive session handling with attributes, flash messages, and encryption

## Recent Changes (2025-10-27)

- Error handling: Added per-category logging control for `ErrorCategory`.
  - New methods on `ErrorCategory`: `WithLogging(enabled bool)`, `DisableLogging()`, `EnableLogging()`, `IsLoggingEnabled()`.
  - `ErrorResponseBuilder.Send()` now honors the matched category’s logging flag: when a category matches the error, it logs only if logging is enabled for that category. If no category matches, default behavior remains (logs when logger provided; otherwise writes to stderr).
  - See updated examples in the Error Handling Patterns section below.

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

### Response Builder

The `ResponseBuilder` provides a fluent API for constructing HTTP responses with support for JSON, Text, HTML, and structured error responses.

#### JSON Responses

```go
import httplib "github.com/golibry/go-http/http"

func jsonHandler(w http.ResponseWriter, r *http.Request) {
    data := map[string]interface{}{
        "message": "Hello, World!",
        "status":  "success",
        "data":    []string{"item1", "item2", "item3"},
    }

    err := httplib.NewResponseBuilder(w).
        Status(http.StatusCreated).
        Header("X-API-Version", "v1.0").
        Header("X-Request-ID", "12345").
        JSON().
        Data(data).
        Send()

    if err != nil {
        // Handle error
        log.Printf("Error sending JSON response: %v", err)
    }
}
```

#### Text Responses

```go
func textHandler(w http.ResponseWriter, r *http.Request) {
    err := httplib.NewResponseBuilder(w).
        Status(http.StatusOK).
        Header("X-Custom-Header", "custom-value").
        Text().
        ContentString("This is a plain text response").
        Send()

    if err != nil {
        log.Printf("Error sending text response: %v", err)
    }
}
```

#### HTML Responses

```go
func htmlHandler(w http.ResponseWriter, r *http.Request) {
    htmlContent := `<!DOCTYPE html>
<html>
<head><title>Example</title></head>
<body><h1>Hello from Response Builder!</h1></body>
</html>`

    err := httplib.NewResponseBuilder(w).
        Status(http.StatusOK).
        HTML().
        ContentString(htmlContent).
        Send()

    if err != nil {
        log.Printf("Error sending HTML response: %v", err)
    }
}
```

#### Error Responses

The ResponseBuilder provides sophisticated error handling with automatic status code detection, structured logging, and flexible error categorization.

```go
import (
    "context"
    "errors"
    "log/slog"
    "net/http"
    httplib "github.com/golibry/go-http/http"
)

// Custom error type implementing HTTPError interface
type ValidationError struct {
    Field string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field: %s", e.Field)
}

func (e ValidationError) StatusCode() int {
    return http.StatusBadRequest
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
    logger := slog.Default()
    ctx := context.Background()
    
    // Setup error categories
    validationCategory := httplib.NewErrorCategory(http.StatusBadRequest)
    httplib.AddErrorType[ValidationError](validationCategory)
    
    // Example 1: Simple error response
    err := httplib.NewResponseBuilder(w).
        Status(http.StatusBadRequest).
        Error().
        WithError(errors.New("validation failed")).
        Send()

    // Example 2: JSON error response with logging
    err = httplib.NewResponseBuilder(w).
        Error().
        WithError(ValidationError{Field: "email"}).
        WithLogger(logger).
        WithContext(ctx).
        AddErrorCategory(validationCategory).
        AsJSON().
        Send()

    // Example 3: Custom message error response
    err = httplib.NewResponseBuilder(w).
        Status(http.StatusNotFound).
        Error().
        WithMessage("The requested resource was not found").
        AsJSON().
        Send()

    if err != nil {
        log.Printf("Error sending error response: %v", err)
    }
}
```

**Error Response Features:**
- **Automatic Status Code Detection**: Errors implementing `HTTPError` interface automatically set the correct status code
- **Error Categories**: Flexible error classification using `ErrorCategory` for grouping errors by status code
- **Structured Logging**: Integration with `slog.Logger` for comprehensive error tracking
- **Context Support**: Request context preservation for observability
- **Multiple Formats**: Support for both text and JSON error responses

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
    return middleware.NewHTTPAccessLogger(mainHandler, logger, options)
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

### Session Management

Comprehensive session management with support for attributes, flash messages, encryption, and automatic garbage collection.

For detailed documentation including configuration options, security considerations, custom storage implementations, and complete examples, see the [Session Management Documentation](http/session/README.md).


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
    
    accessLogger := middleware.NewHTTPAccessLogger(
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
import (
    "errors"
    "net/http"
    httplib "github.com/golibry/go-http/http"
)

// Create categories for different error types
validationCategory := httplib.NewErrorCategory(http.StatusBadRequest)
validationCategory.AddSentinelError(errors.New("validation error"))

authCategory := httplib.NewErrorCategory(http.StatusUnauthorized)
httplib.AddErrorType[*AuthError](authCategory)
```

#### Per-Category Logging Control

You can enable or disable logging for specific error categories. When an error matches a category, `ErrorResponseBuilder.Send()` will only log if logging is enabled for that matched category. If no category matches, the default behavior applies (logs when a logger is provided; otherwise writes to stderr).

```go
// Suppose you want to avoid logging validation errors (noisy, expected user mistakes)
validationCategory := httplib.NewErrorCategory(http.StatusBadRequest)
httplib.AddErrorType[ValidationError](validationCategory)
validationCategory.DisableLogging()

// Later in your handler
_ = httplib.NewResponseBuilder(w).
    Error().
    WithError(ValidationError{Field: "email"}).
    AddErrorCategory(validationCategory).
    WithLogger(logger).
    WithContext(r.Context()).
    Send()

// The response will be 400 with the error message, but no log will be emitted for this matched category
```

To re-enable logging for a category or toggle it dynamically:

```go
category := httplib.NewErrorCategory(http.StatusInternalServerError)
category.EnableLogging()               // explicit enable
_ = category.WithLogging(false)        // disable via fluent method
_ = category.WithLogging(true)         // enable via fluent method
_ = category.IsLoggingEnabled()        // query current setting
```

## API Reference

### Types

- `ResponseWriter`: Enhanced response writer with status code tracking
- `ResponseBuilder`: Fluent API for building HTTP responses
- `JSONResponseBuilder`: Builder for JSON responses
- `TextResponseBuilder`: Builder for text responses
- `HTMLResponseBuilder`: Builder for HTML responses
- `ErrorResponseBuilder`: Builder for structured error responses
- `HTTPError`: Interface for errors with HTTP status codes
- `ErrorCategory`: Groups errors by HTTP status code
- `AccessLogOptions`: Configuration for access logging
- `HTTPAccessLogger`: Access logging middleware
- `Recoverer`: Panic recovery middleware

### Functions

#### Response Building
- `NewResponseBuilder(w http.ResponseWriter) *ResponseBuilder`
- `(rb *ResponseBuilder) Status(code int) *ResponseBuilder`
- `(rb *ResponseBuilder) Header(key, value string) *ResponseBuilder`
- `(rb *ResponseBuilder) JSON() *JSONResponseBuilder`
- `(rb *ResponseBuilder) Text() *TextResponseBuilder`
- `(rb *ResponseBuilder) HTML() *HTMLResponseBuilder`
- `(rb *ResponseBuilder) Error() *ErrorResponseBuilder`

#### JSON Response Building
- `(jrb *JSONResponseBuilder) Data(data interface{}) *JSONResponseBuilder`
- `(jrb *JSONResponseBuilder) Send() error`

#### Text Response Building
- `(trb *TextResponseBuilder) Content(content []byte) *TextResponseBuilder`
- `(trb *TextResponseBuilder) ContentString(content string) *TextResponseBuilder`
- `(trb *TextResponseBuilder) Send() error`

#### HTML Response Building
- `(hrb *HTMLResponseBuilder) Content(content []byte) *HTMLResponseBuilder`
- `(hrb *HTMLResponseBuilder) ContentString(content string) *HTMLResponseBuilder`
- `(hrb *HTMLResponseBuilder) Send() error`

#### Error Response Building
- `(erb *ErrorResponseBuilder) WithError(err error) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) WithMessage(message string) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) AsJSON() *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) WithLogger(logger *slog.Logger) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) WithContext(ctx context.Context) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) WithErrorCategories(categories ...*ErrorCategory) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) AddErrorCategory(category *ErrorCategory) *ErrorResponseBuilder`
- `(erb *ErrorResponseBuilder) Send() error`

#### Core Functions
- `NewResponseWriter(w http.ResponseWriter) *ResponseWriter`
- `NewHTTPAccessLogger(next http.Handler, logger *slog.Logger, options AccessLogOptions) *HTTPAccessLogger`
- `NewErrorCategory(statusCode int) *ErrorCategory`
- `AddErrorType[T error](ec *ErrorCategory)`

#### ErrorCategory Methods
- `(ec *ErrorCategory) AddSentinelError(e error)`
- `(ec *ErrorCategory) WithLogging(enabled bool) *ErrorCategory`
- `(ec *ErrorCategory) DisableLogging() *ErrorCategory`
- `(ec *ErrorCategory) EnableLogging() *ErrorCategory`
- `(ec *ErrorCategory) IsLoggingEnabled() bool`

## Requirements

- Go 1.24.1 or later
- Standard library only (no external dependencies for core functionality)
- `github.com/stretchr/testify` for testing

## License

See the LICENSE file for details.