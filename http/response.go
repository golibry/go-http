package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// HTTPError represents an error with an associated HTTP status code.
type HTTPError interface {
	error
	StatusCode() int
}

// ErrorCategory represents a category of errors with a default status code.
type ErrorCategory struct {
	StatusCode int
	checkFuncs []func(error) bool
}

func NewErrorCategory(statusCode int) *ErrorCategory {
	return &ErrorCategory{
		StatusCode: statusCode,
		checkFuncs: make([]func(error) bool, 0),
	}
}

func (ec *ErrorCategory) AddSentinelError(e error) {
	ec.checkFuncs = append(
		ec.checkFuncs, func(err error) bool {
			return errors.Is(err, e)
		},
	)
}

func (ec *ErrorCategory) Matches(err error) bool {
	for _, check := range ec.checkFuncs {
		if check(err) {
			return true
		}
	}
	return false
}

func AddErrorType[T error](ec *ErrorCategory) {
	ec.checkFuncs = append(
		ec.checkFuncs, func(err error) bool {
			var target T
			return errors.As(err, &target)
		},
	)
}

type AccessResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *AccessResponseWriter {
	return &AccessResponseWriter{w, http.StatusOK}
}

func (lrw *AccessResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *AccessResponseWriter) StatusCode() int {
	return lrw.statusCode
}

// ResponseBuilder provides a base structure for building HTTP responses
type ResponseBuilder struct {
	writer     http.ResponseWriter
	statusCode int
	headers    map[string]string
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(w http.ResponseWriter) *ResponseBuilder {
	return &ResponseBuilder{
		writer:     w,
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}

// Status sets the HTTP status code
func (rb *ResponseBuilder) Status(code int) *ResponseBuilder {
	rb.statusCode = code
	return rb
}

// Header sets a response header
func (rb *ResponseBuilder) Header(key, value string) *ResponseBuilder {
	rb.headers[key] = value
	return rb
}

// writeHeaders writes all headers to the response writer
func (rb *ResponseBuilder) writeHeaders() {
	for key, value := range rb.headers {
		rb.writer.Header().Set(key, value)
	}
	rb.writer.WriteHeader(rb.statusCode)
}

// JSONResponseBuilder builds JSON responses
type JSONResponseBuilder struct {
	*ResponseBuilder
	data interface{}
}

// JSON creates a new JSON response builder
func (rb *ResponseBuilder) JSON() *JSONResponseBuilder {
	rb.Header("Content-Type", "application/json")
	return &JSONResponseBuilder{ResponseBuilder: rb}
}

// Data sets the JSON data to be written
func (jrb *JSONResponseBuilder) Data(data interface{}) *JSONResponseBuilder {
	jrb.data = data
	return jrb
}

// Send writes the JSON response
func (jrb *JSONResponseBuilder) Send() error {
	jrb.writeHeaders()
	return json.NewEncoder(jrb.writer).Encode(jrb.data)
}

// TextResponseBuilder builds plain text responses
type TextResponseBuilder struct {
	*ResponseBuilder
	content []byte
}

// Text creates a new text response builder
func (rb *ResponseBuilder) Text() *TextResponseBuilder {
	rb.Header("Content-Type", "text/plain; charset=utf-8")
	return &TextResponseBuilder{ResponseBuilder: rb}
}

// Content sets the text content to be written
func (trb *TextResponseBuilder) Content(content []byte) *TextResponseBuilder {
	trb.content = content
	return trb
}

// ContentString sets the text content from a string
func (trb *TextResponseBuilder) ContentString(content string) *TextResponseBuilder {
	trb.content = []byte(content)
	return trb
}

// Send writes the text response
func (trb *TextResponseBuilder) Send() error {
	trb.writeHeaders()
	_, err := trb.writer.Write(trb.content)
	return err
}

// HTMLResponseBuilder builds HTML responses
type HTMLResponseBuilder struct {
	*ResponseBuilder
	content []byte
}

// HTML creates a new HTML response builder
func (rb *ResponseBuilder) HTML() *HTMLResponseBuilder {
	rb.Header("Content-Type", "text/html; charset=utf-8")
	return &HTMLResponseBuilder{ResponseBuilder: rb}
}

// Content sets the HTML content to be written
func (hrb *HTMLResponseBuilder) Content(content []byte) *HTMLResponseBuilder {
	hrb.content = content
	return hrb
}

// ContentString sets the HTML content from a string
func (hrb *HTMLResponseBuilder) ContentString(content string) *HTMLResponseBuilder {
	hrb.content = []byte(content)
	return hrb
}

// Send writes the HTML response
func (hrb *HTMLResponseBuilder) Send() error {
	hrb.writeHeaders()
	_, err := hrb.writer.Write(hrb.content)
	return err
}

// ErrorResponseBuilder builds error responses with advanced error handling capabilities
type ErrorResponseBuilder struct {
	*ResponseBuilder
	err        error
	message    string
	isJSON     bool
	ctx        context.Context
	logger     *slog.Logger
	categories []*ErrorCategory
}

// Error creates a new error response builder
func (rb *ResponseBuilder) Error() *ErrorResponseBuilder {
	rb.Header("Content-Type", "text/plain; charset=utf-8")
	return &ErrorResponseBuilder{
		ResponseBuilder: rb,
		categories:      make([]*ErrorCategory, 0),
	}
}

// WithError sets the error to be written
func (erb *ErrorResponseBuilder) WithError(err error) *ErrorResponseBuilder {
	erb.err = err
	return erb
}

// WithMessage sets a custom error message
func (erb *ErrorResponseBuilder) WithMessage(message string) *ErrorResponseBuilder {
	erb.message = message
	return erb
}

// AsJSON configures the error response to be in JSON format
func (erb *ErrorResponseBuilder) AsJSON() *ErrorResponseBuilder {
	erb.Header("Content-Type", "application/json")
	erb.isJSON = true
	return erb
}

// WithLogger sets the structured logger for error logging
func (erb *ErrorResponseBuilder) WithLogger(logger *slog.Logger) *ErrorResponseBuilder {
	erb.logger = logger
	return erb
}

// WithContext sets the context for structured logging
func (erb *ErrorResponseBuilder) WithContext(ctx context.Context) *ErrorResponseBuilder {
	erb.ctx = ctx
	return erb
}

// WithErrorCategories sets the error categories for flexible error classification
func (erb *ErrorResponseBuilder) WithErrorCategories(categories ...*ErrorCategory) *ErrorResponseBuilder {
	erb.categories = append(erb.categories, categories...)
	return erb
}

// AddErrorCategory adds a single error category
func (erb *ErrorResponseBuilder) AddErrorCategory(category *ErrorCategory) *ErrorResponseBuilder {
	erb.categories = append(erb.categories, category)
	return erb
}

// getStatusCode determines the HTTP status code for an error using enhanced classification
func (erb *ErrorResponseBuilder) getStatusCode(err error) int {
	// Check if the error implements HTTPError interface
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode()
	}

	// Check error categories
	for _, category := range erb.categories {
		if category.Matches(err) {
			return category.StatusCode
		}
	}

	// If a status code was explicitly set (not the default 200), use it
	// We only use the explicitly set status code if it's not the default OK status
	if erb.statusCode != 0 && erb.statusCode != http.StatusOK {
		return erb.statusCode
	}

	// Default to internal server error for errors
	return http.StatusInternalServerError
}

// Send writes the error response with enhanced error handling
func (erb *ErrorResponseBuilder) Send() error {
	// Determine the appropriate status code
	var statusCode int
	if erb.err != nil {
		statusCode = erb.getStatusCode(erb.err)
	} else {
		// Use explicitly set status code or default
		statusCode = erb.statusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
	}

	// Update the response builder's status code
	erb.Status(statusCode)

	// Log the error if logger is available and there's an error
	if erb.err != nil {
		if erb.logger != nil {
			logCtx := erb.ctx
			if logCtx == nil {
				logCtx = context.Background()
			}
			erb.logger.ErrorContext(
				logCtx,
				"HTTP Request Error",
				slog.String("Error", erb.err.Error()),
				slog.Int("StatusCode", statusCode),
			)
		} else {
			// Fallback to stderr if no logger available
			_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", erb.err)
		}
	}

	// Determine the message to send
	message := erb.message
	if message == "" && erb.err != nil {
		message = erb.err.Error()
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	if erb.isJSON {
		erb.writeHeaders()
		errorResponse := map[string]interface{}{
			"error":  message,
			"status": statusCode,
		}
		return json.NewEncoder(erb.writer).Encode(errorResponse)
	}

	erb.writeHeaders()
	_, err := erb.writer.Write([]byte(message))
	return err
}
