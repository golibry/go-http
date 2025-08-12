package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TimeoutSuite struct {
	suite.Suite
}

func TestTimeoutSuite(t *testing.T) {
	suite.Run(t, new(TimeoutSuite))
}

type timeoutLog struct {
	Level   string  `json:"level"`
	Msg     string  `json:"msg"`
	Method  string  `json:"method"`
	Path    string  `json:"path"`
	Timeout float64 `json:"timeout"`
}

func (suite *TimeoutSuite) TestItCanHandleRequestWithinTimeout() {
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that responds quickly
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		},
	)

	options := TimeoutOptions{
		Timeout:      1 * time.Second,
		ErrorMessage: "Request timed out",
	}

	middleware := NewTimeoutMiddleware(handler, logger, options)

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	middleware.ServeHTTP(recorder, req)

	suite.Equal(http.StatusOK, recorder.Code)
	suite.Equal("success", recorder.Body.String())
	// Should not have any timeout logs
	suite.Empty(outputBuffer.String())
}

func (suite *TimeoutSuite) TestItCanHandleRequestTimeout() {
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	// Create a handler that takes longer than the timeout
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("should not reach here"))
		},
	)

	options := TimeoutOptions{
		Timeout:      50 * time.Millisecond,
		ErrorMessage: "Custom timeout message",
	}

	middleware := NewTimeoutMiddleware(handler, logger, options)

	req := httptest.NewRequest("GET", "/timeout-test", nil)
	recorder := httptest.NewRecorder()

	middleware.ServeHTTP(recorder, req)

	suite.Equal(http.StatusRequestTimeout, recorder.Code)
	suite.Equal("Custom timeout message", recorder.Body.String())
	suite.Equal("text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))

	// Verify timeout was logged
	loggedEntry := timeoutLog{}
	err := json.Unmarshal(outputBuffer.Bytes(), &loggedEntry)
	suite.NoError(err)
	suite.Equal("WARN", loggedEntry.Level)
	suite.Equal("Request timeout", loggedEntry.Msg)
	suite.Equal("GET", loggedEntry.Method)
	suite.Equal("/timeout-test", loggedEntry.Path)
}

func (suite *TimeoutSuite) TestItCanUseDefaultValues() {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	// Create middleware with empty options to test defaults
	middleware := NewTimeoutMiddleware(handler, nil, TimeoutOptions{})

	suite.Equal(30*time.Second, middleware.options.Timeout)
	suite.Equal("Request timeout", middleware.options.ErrorMessage)
}

func (suite *TimeoutSuite) TestItCanHandlePanicInHandler() {
	logger := slog.New(slog.NewJSONHandler(new(bytes.Buffer), &slog.HandlerOptions{}))

	// Create a handler that panics
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		},
	)

	options := TimeoutOptions{
		Timeout:      1 * time.Second,
		ErrorMessage: "Request timed out",
	}

	middleware := NewTimeoutMiddleware(handler, logger, options)

	req := httptest.NewRequest("GET", "/panic-test", nil)
	recorder := httptest.NewRecorder()

	// Should re-panic
	suite.Panics(
		func() {
			middleware.ServeHTTP(recorder, req)
		},
	)
}

func (suite *TimeoutSuite) TestItCanHandleNilLogger() {
	// Create a handler that takes longer than the timeout
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	)

	options := TimeoutOptions{
		Timeout:      50 * time.Millisecond,
		ErrorMessage: "Timeout occurred",
	}

	// Pass nil logger
	middleware := NewTimeoutMiddleware(handler, nil, options)

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// Should not panic even with nil logger
	suite.NotPanics(
		func() {
			middleware.ServeHTTP(recorder, req)
		},
	)

	suite.Equal(http.StatusRequestTimeout, recorder.Code)
	suite.Equal("Timeout occurred", recorder.Body.String())
}

func (suite *TimeoutSuite) TestItCanHandleCustomTimeoutAndMessage() {
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	)

	customTimeout := 100 * time.Millisecond
	customMessage := "Custom timeout error message"

	options := TimeoutOptions{
		Timeout:      customTimeout,
		ErrorMessage: customMessage,
	}

	middleware := NewTimeoutMiddleware(handler, logger, options)

	req := httptest.NewRequest("POST", "/custom", nil)
	recorder := httptest.NewRecorder()

	middleware.ServeHTTP(recorder, req)

	suite.Equal(http.StatusRequestTimeout, recorder.Code)
	suite.Equal(customMessage, recorder.Body.String())

	// Verify custom timeout was logged
	loggedEntry := timeoutLog{}
	err := json.Unmarshal(outputBuffer.Bytes(), &loggedEntry)
	suite.NoError(err)
	suite.Equal("POST", loggedEntry.Method)
	suite.Equal("/custom", loggedEntry.Path)
}

func (suite *TimeoutSuite) TestItCanHandleQuickSuccessfulRequest() {
	logger := slog.New(slog.NewJSONHandler(new(bytes.Buffer), &slog.HandlerOptions{}))

	// Handler that responds immediately
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test-value")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("quick response"))
		},
	)

	options := TimeoutOptions{
		Timeout:      5 * time.Second, // Long timeout
		ErrorMessage: "Should not timeout",
	}

	middleware := NewTimeoutMiddleware(handler, logger, options)

	req := httptest.NewRequest("PUT", "/quick", nil)
	recorder := httptest.NewRecorder()

	middleware.ServeHTTP(recorder, req)

	suite.Equal(http.StatusCreated, recorder.Code)
	suite.Equal("quick response", recorder.Body.String())
	suite.Equal("test-value", recorder.Header().Get("X-Custom"))
}
