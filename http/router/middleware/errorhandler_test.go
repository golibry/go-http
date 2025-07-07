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

// Custom errors for testing
var (
	testClientError = errors.New("test client error")
	testServerError = errors.New("test server error")
)

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
	errorHandler := &Errorhandler{
		ctx:        context.Background(),
		logger:     nil,
		next:       handler,
		clientErrs: []error{testClientError},
	}

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
	errorHandler := &Errorhandler{
		ctx:        context.Background(),
		logger:     logger,
		next:       handler,
		clientErrs: []error{testClientError},
	}

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

func (suite *ErrorhandlerSuite) TestHandleClientError() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that returns a client error
	handler := func(w http.ResponseWriter, r *http.Request) error {
		return testClientError
	}

	// Create an errorhandler with handler
	errorHandler := &Errorhandler{
		ctx:        context.Background(),
		logger:     logger,
		next:       handler,
		clientErrs: []error{testClientError},
	}

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
		testClientError.Error(),
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
	errorHandler := &Errorhandler{
		ctx:        context.Background(),
		logger:     nil,
		next:       handler,
		clientErrs: []error{testClientError},
	}

	// Execute
	errorHandler.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
}

func (suite *ErrorhandlerSuite) TestErrIsAny() {
	// Test when error is in the list
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	errList := []error{err1, err2}

	suite.Assert().True(errIsAny(err1, errList), "Should return true when error is in the list")

	// Test when error is not in the list
	err3 := errors.New("error 3")
	suite.Assert().False(errIsAny(err3, errList), "Should return false when error is not in the list")

	// Test with wrapped error
	wrappedErr := fmt.Errorf("wrapped: %w", err1)
	suite.Assert().True(errIsAny(wrappedErr, errList), "Should return true when wrapped error is in the list")
}
