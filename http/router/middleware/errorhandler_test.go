package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ErrorhandlerSuite struct {
	suite.Suite
}

func TestErrorhandlerSuite(t *testing.T) {
	suite.Run(t, new(ErrorhandlerSuite))
}

type domainError struct {
	message string
}

func (e domainError) Error() string {
	return e.message
}

// Custom errors for testing
var (
	testValidationError     = errors.New("test validation error")
	testAuthenticationError = errors.New("test authentication error")
	testAuthorizationError  = errors.New("test authorization error")
	testNotFoundError       = errors.New("test not found error")
	testConflictError       = errors.New("test conflict error")
	testRateLimitError      = errors.New("test rate limit error")
	testServerError         = errors.New("test server error")
)

// Test HTTPError implementation
type testHTTPError struct {
	message    string
	statusCode int
}

func (e *testHTTPError) Error() string {
	return e.message
}

func (e *testHTTPError) StatusCode() int {
	return e.statusCode
}

func (suite *ErrorhandlerSuite) TestNoErrorHandlingNeeded() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create a handler that doesn't return an error
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) error {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// Create an errorhandler with handler
	errorHandler := NewErrorhandler(handler, context.Background(), nil, nil)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().True(handlerCalled, "Handler should be called")
	suite.Assert().Equal(http.StatusOK, recorder.Code, "Status code should be 200 OK")
}

func (suite *ErrorhandlerSuite) TestHandleServerError() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that returns a server error
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return testServerError
	}

	// Create an errorhandler with handler
	errorHandler := NewErrorhandler(handler, context.Background(), logger, nil)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
	suite.Assert().Contains(
		outputBuffer.String(),
		testServerError.Error(),
		"Log should contain error message",
	)
}

func (suite *ErrorhandlerSuite) TestHandleValidationError() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a validation error category with our test error
	validationCategory := NewErrorCategory(http.StatusBadRequest)
	validationCategory.AddSentinelError(testValidationError)

	// Create a handler that returns a validation error
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return testValidationError
	}

	// Create an errorhandler with handler
	errorHandler := NewErrorhandler(
		handler,
		context.Background(),
		logger,
		[]*ErrorCategory{validationCategory},
	)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusBadRequest,
		recorder.Code,
		"Status code should be 400 Bad Request",
	)
	suite.Assert().Contains(
		outputBuffer.String(),
		testValidationError.Error(),
		"Log should contain error message",
	)
}

func (suite *ErrorhandlerSuite) TestHandleErrorWithoutLogger() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create a handler that returns an error
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return testServerError
	}

	// Create an errorhandler with handler but without logger
	errorHandler := NewErrorhandler(handler, context.Background(), nil, nil)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
}

func (suite *ErrorhandlerSuite) TestHTTPErrorInterface() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create a handler that returns an HTTPError
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return &testHTTPError{
			message:    "custom http error",
			statusCode: http.StatusTeapot,
		}
	}

	// Create an errorhandler
	errorHandler := NewErrorhandler(handler, context.Background(), nil, nil)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusTeapot,
		recorder.Code,
		"Status code should be 418 I'm a teapot",
	)
}

func (suite *ErrorhandlerSuite) TestCustomErrorMapping() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create error categories for custom mapping
	authCategory := NewErrorCategory(http.StatusUnauthorized)
	authCategory.AddSentinelError(testAuthenticationError)

	notFoundCategory := NewErrorCategory(http.StatusNotFound)
	notFoundCategory.AddSentinelError(testNotFoundError)

	// Create a handler that returns an authentication error
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return testAuthenticationError
	}

	// Create an errorhandler with custom categories
	errorHandler := NewErrorhandler(
		handler,
		context.Background(),
		nil,
		[]*ErrorCategory{authCategory, notFoundCategory},
	)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusUnauthorized,
		recorder.Code,
		"Status code should be 401 Unauthorized",
	)
}

func (suite *ErrorhandlerSuite) TestMultipleErrorCategories() {
	testCases := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{
			name:           "Authentication Error",
			error:          testAuthenticationError,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Authorization Error",
			error:          testAuthorizationError,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Not Found Error",
			error:          testNotFoundError,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Conflict Error",
			error:          testConflictError,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Rate Limit Error",
			error:          testRateLimitError,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "Domain Error",
			error:          domainError{"domain error 1"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Domain Error",
			error:          domainError{"domain error 2"},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.Run(
			tc.name, func() {
				// Setup
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/test", nil)

				// Create error categories
				authCategory := NewErrorCategory(http.StatusUnauthorized)
				authCategory.AddSentinelError(testAuthenticationError)

				authzCategory := NewErrorCategory(http.StatusForbidden)
				authzCategory.AddSentinelError(testAuthorizationError)

				notFoundCategory := NewErrorCategory(http.StatusNotFound)
				notFoundCategory.AddSentinelError(testNotFoundError)

				conflictCategory := NewErrorCategory(http.StatusConflict)
				conflictCategory.AddSentinelError(testConflictError)

				rateLimitCategory := NewErrorCategory(http.StatusTooManyRequests)
				rateLimitCategory.AddSentinelError(testRateLimitError)

				domainCategory := NewErrorCategory(http.StatusBadRequest)
				AddErrorType[domainError](domainCategory)

				categories := []*ErrorCategory{
					authCategory,
					authzCategory,
					notFoundCategory,
					conflictCategory,
					rateLimitCategory,
					domainCategory,
				}

				// Create a handler that returns the test error
				handler := func(w http.ResponseWriter, r *http.Request) error {
					return tc.error
				}

				// Create an errorhandler with categories
				errorHandler := NewErrorhandler(
					handler,
					context.Background(),
					nil,
					categories,
				)

				// Execute
				errorHandler.ServeHTTP(recorder, request)

				// Assert
				suite.Assert().Equal(
					tc.expectedStatus,
					recorder.Code,
					fmt.Sprintf("Status code should be %d for %s", tc.expectedStatus, tc.name),
				)
			},
		)
	}
}

func (suite *ErrorhandlerSuite) TestErrorPriorityOrder() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create error categories with different priorities
	// HTTPError should take precedence over categories
	httpErrorCategory := NewErrorCategory(http.StatusBadRequest)
	httpErrorCategory.AddSentinelError(testValidationError)

	// Create a handler that returns an HTTPError that also matches a category
	handler := func(w http.ResponseWriter, r *http.Request) error {
		// This error implements HTTPError interface and should take precedence
		return &testHTTPError{
			message:    "http error with custom status",
			statusCode: http.StatusTeapot,
		}
	}

	// Create an errorhandler with categories
	errorHandler := NewErrorhandler(
		handler,
		context.Background(),
		nil,
		[]*ErrorCategory{httpErrorCategory},
	)

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert - HTTPError interface should take precedence
	suite.Assert().Equal(
		http.StatusTeapot,
		recorder.Code,
		"HTTPError interface should take precedence over categories",
	)
}
