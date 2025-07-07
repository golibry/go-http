package middleware

import (
	"bytes"
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type RecovererSuite struct {
	suite.Suite
}

func TestRecovererSuite(t *testing.T) {
	suite.Run(t, new(RecovererSuite))
}

func (suite *RecovererSuite) TestNoRecoveryNeeded() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create a handler that doesn't panic
	handlerCalled := false
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
	)

	// Create a recoverer with handler
	chain := &Recoverer{
		ctx:    context.Background(),
		logger: nil,
		next:   handler,
	}

	// Execute
	chain.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().True(handlerCalled, "Handler should be called")
	suite.Assert().Equal(http.StatusOK, recorder.Code, "Status code should be 200 OK")
}

func (suite *RecovererSuite) TestRecoverFromStringPanic() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that panics with a string
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic("test panic string")
		},
	)

	// Create a recoverer with handler
	chain := &Recoverer{
		ctx:    context.Background(),
		logger: logger,
		next:   handler,
	}

	// Execute
	chain.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
	suite.Assert().Contains(
		outputBuffer.String(),
		"test panic string",
		"Log should contain panic message",
	)
}

func (suite *RecovererSuite) TestRecoverFromErrorPanic() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that panics with an error
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic(errors.New("test error panic"))
		},
	)

	// Create a recoverer with handler
	chain := &Recoverer{
		ctx:    context.Background(),
		logger: logger,
		next:   handler,
	}

	// Execute
	chain.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
	suite.Assert().Contains(
		outputBuffer.String(),
		"test error panic",
		"Log should contain error message",
	)
}

func (suite *RecovererSuite) TestRecoverFromOtherPanic() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that panics with an integer
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic(123) // Panic with an integer
		},
	)

	// Create a recoverer with handler
	chain := &Recoverer{
		ctx:    context.Background(),
		logger: logger,
		next:   handler,
	}

	// Execute
	chain.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
	suite.Assert().Contains(outputBuffer.String(), "123", "Log should contain panic value")
}

func (suite *RecovererSuite) TestRecoverWithoutLogger() {
	// Setup
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create a handler that panics
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic("test panic without logger")
		},
	)

	// Create a recoverer with handler
	chain := &Recoverer{
		ctx:    context.Background(),
		logger: nil,
		next:   handler,
	}

	// Execute
	chain.ServeHTTP(recorder, request)

	// Assert
	suite.Assert().Equal(
		http.StatusInternalServerError,
		recorder.Code,
		"Status code should be 500 Internal Server Error",
	)
}
