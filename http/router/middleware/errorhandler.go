package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// CustomHandler is like http.Handler but returns an error.
type CustomHandler func(w http.ResponseWriter, r *http.Request) error

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

// ErrorMapper allows custom error-to-status-code mapping.
type ErrorMapper map[error]int

type Errorhandler struct {
	next       CustomHandler
	ctx        context.Context
	logger     *slog.Logger
	categories []*ErrorCategory
}

// NewErrorhandler creates a new Errorhandler with all struct properties as arguments.
func NewErrorhandler(
	next CustomHandler,
	ctx context.Context,
	logger *slog.Logger,
	categories []*ErrorCategory,
) *Errorhandler {
	return &Errorhandler{
		next:       next,
		ctx:        ctx,
		logger:     logger,
		categories: categories,
	}
}

func (handler *Errorhandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	err := handler.next(rw, rq)
	if err != nil {
		if handler.logger != nil {
			handler.logger.ErrorContext(
				handler.ctx,
				"HTTP Request Error",
				slog.String("Error", err.Error()),
				slog.String("URL Path", rq.URL.Path),
			)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		}

		// Determine status code using enhanced error classification
		statusCode := handler.getStatusCode(err)
		http.Error(rw, http.StatusText(statusCode), statusCode)
	}
}

// getStatusCode determines the HTTP status code for an error using the enhanced classification system.
func (handler *Errorhandler) getStatusCode(err error) int {
	// Check if the error implements HTTPError interface
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode()
	}

	// Check error categories
	for _, category := range handler.categories {
		if errIsInCategory(err, category) {
			return category.StatusCode
		}
	}

	// Default to internal server error
	return http.StatusInternalServerError
}

// errIsInCategory checks if an error belongs to a specific category.
func errIsInCategory(err error, category *ErrorCategory) bool {
	if category.Matches(err) {
		return true
	}
	return false
}
