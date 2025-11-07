package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CSRFSuite struct {
	suite.Suite
}

func TestCSRFSuite(t *testing.T) {
	suite.Run(t, new(CSRFSuite))
}

type csrfWarnLog struct {
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Header string `json:"header"`
}

func (s *CSRFSuite) TestItAllowsSafeMethodsWithoutHeader() {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		},
	)

	mw := NewCSRFMiddleware(handler, nil, CSRFOptions{})

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req := httptest.NewRequest(method, "/safe", nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
		s.Equal(http.StatusOK, rr.Code, "method %s should pass without header", method)
		s.Equal("ok", rr.Body.String())
	}
}

func (s *CSRFSuite) TestItBlocksUnsafeMethodsWithoutHeader() {
	output := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{}))

	mw := NewCSRFMiddleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		), logger, CSRFOptions{},
	)

	for _, method := range []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	} {
		req := httptest.NewRequest(method, "/blocked", nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)

		s.Equal(http.StatusForbidden, rr.Code, "method %s should be blocked without header", method)
		s.Equal("text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
		s.Equal("Forbidden", rr.Body.String())
	}

	// Ensure a warning was logged at least once
	s.NotEmpty(output.Bytes())
}

func (s *CSRFSuite) TestItAllowsUnsafeWithValidHeader() {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		},
	)

	mw := NewCSRFMiddleware(
		handler,
		nil,
		CSRFOptions{},
	) // defaults require X-Deliberate-Request: true

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("X-Deliberate-Request", "1")
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	s.Equal(http.StatusCreated, rr.Code)
	s.Equal("created", rr.Body.String())
}

func (s *CSRFSuite) TestItUsesCustomHeaderNameAndValue() {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	opts := CSRFOptions{HeaderName: "X-Custom-Deliberate", HeaderValue: "yes"}
	mw := NewCSRFMiddleware(handler, nil, opts)

	// Missing header -> forbidden
	req1 := httptest.NewRequest(http.MethodDelete, "/custom", nil)
	rr1 := httptest.NewRecorder()
	mw.ServeHTTP(rr1, req1)
	s.Equal(http.StatusForbidden, rr1.Code)

	// Wrong value -> forbidden
	req2 := httptest.NewRequest(http.MethodDelete, "/custom", nil)
	req2.Header.Set("X-Custom-Deliberate", "no")
	rr2 := httptest.NewRecorder()
	mw.ServeHTTP(rr2, req2)
	s.Equal(http.StatusForbidden, rr2.Code)

	// Correct value -> ok
	req3 := httptest.NewRequest(http.MethodDelete, "/custom", nil)
	req3.Header.Set("X-Custom-Deliberate", "yes")
	rr3 := httptest.NewRecorder()
	mw.ServeHTTP(rr3, req3)
	s.Equal(http.StatusOK, rr3.Code)
}

func (s *CSRFSuite) TestItRespectsCustomUnsafeMethods() {
	// Only validate PATCH
	opts := CSRFOptions{UnsafeMethods: []string{http.MethodPatch}}
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	mw := NewCSRFMiddleware(handler, nil, opts)

	// POST should pass (not in a custom list)
	req1 := httptest.NewRequest(http.MethodPost, "/not-validated", nil)
	rr1 := httptest.NewRecorder()
	mw.ServeHTTP(rr1, req1)
	s.Equal(http.StatusOK, rr1.Code)

	// PATCH should be validated and blocked without a header
	req2 := httptest.NewRequest(http.MethodPatch, "/validated", nil)
	rr2 := httptest.NewRecorder()
	mw.ServeHTTP(rr2, req2)
	s.Equal(http.StatusForbidden, rr2.Code)

	// PATCH with the required default header / value should pass
	req3 := httptest.NewRequest(http.MethodPatch, "/validated", nil)
	req3.Header.Set("X-Deliberate-Request", "1")
	rr3 := httptest.NewRecorder()
	mw.ServeHTTP(rr3, req3)
	s.Equal(http.StatusOK, rr3.Code)
}

func (s *CSRFSuite) TestItLogsWarningOnFailure() {
	output := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := NewCSRFMiddleware(handler, logger, CSRFOptions{})

	req := httptest.NewRequest(http.MethodPost, "/warn", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	s.Equal(http.StatusForbidden, rr.Code)

	// Parse log
	entry := csrfWarnLog{}
	err := json.Unmarshal(output.Bytes(), &entry)
	s.NoError(err)
	s.Equal("WARN", entry.Level)
	s.Equal("CSRF header validation failed", entry.Msg)
	s.Equal("POST", entry.Method)
	s.Equal("/warn", entry.Path)
	s.Equal("X-Deliberate-Request", entry.Header)
}
