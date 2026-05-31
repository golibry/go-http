package middleware

import (
	"log/slog"
	"net/http"
	"strings"
)

// CSRFMiddleware provides CSRF protection by validating a custom request header
// This middleware is intended for APIs/SPAs where a deliberate client-side
// action adds a specific header to unsafe HTTP methods.
type CSRFMiddleware struct {
	next    http.Handler
	logger  *slog.Logger
	options CSRFOptions
}

// CSRFOptions configures the CSRF middleware behavior
//
// HeaderName:  name of the header to validate (default: "X-Deliberate-Request")
// HeaderValue: required value; if empty, only header presence is validated (default: "true")
// ErrorMessage: response message when validation fails (default: "CSRF validation failed")
// UnsafeMethods: list of methods to validate; if empty defaults to POST, PUT, PATCH, DELETE
// ExcludedPaths: exact URL paths that bypass CSRF validation
//
// Notes:
// - Header comparison for value is case-sensitive; header name lookup is case-insensitive
// per HTTP spec.
type CSRFOptions struct {
	HeaderName    string
	HeaderValue   string
	ErrorMessage  string
	UnsafeMethods []string
	ExcludedPaths []string
}

// NewCSRFMiddleware creates a new CSRF middleware instance
func NewCSRFMiddleware(
	next http.Handler,
	logger *slog.Logger,
	options CSRFOptions,
) *CSRFMiddleware {
	if options.HeaderName == "" {
		options.HeaderName = "X-Deliberate-Request"
	}
	if options.HeaderValue == "" {
		options.HeaderValue = "1"
	}
	if options.ErrorMessage == "" {
		options.ErrorMessage = "Forbidden"
	}
	if len(options.UnsafeMethods) == 0 {
		options.UnsafeMethods = []string{"POST", "PUT", "PATCH", "DELETE"}
	}
	return &CSRFMiddleware{next: next, logger: logger, options: options}
}

// ServeHTTP implements the middleware logic
func (cm *CSRFMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cm.isExcludedPath(r.URL.Path) {
		cm.next.ServeHTTP(w, r)
		return
	}

	if !cm.shouldValidateMethod(r.Method) {
		cm.next.ServeHTTP(w, r)
		return
	}

	reqHeader := r.Header.Get(cm.options.HeaderName)
	if !cm.isValidHeader(reqHeader) {
		if cm.logger != nil {
			cm.logger.WarnContext(
				r.Context(),
				"CSRF header validation failed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("header", cm.options.HeaderName),
			)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(cm.options.ErrorMessage))
		return
	}

	cm.next.ServeHTTP(w, r)
}

func (cm *CSRFMiddleware) isExcludedPath(path string) bool {
	for _, excludedPath := range cm.options.ExcludedPaths {
		if path == excludedPath {
			return true
		}
	}

	return false
}

func (cm *CSRFMiddleware) shouldValidateMethod(method string) bool {
	m := strings.ToUpper(method)
	for _, um := range cm.options.UnsafeMethods {
		if m == strings.ToUpper(um) {
			return true
		}
	}
	return false
}

func (cm *CSRFMiddleware) isValidHeader(value string) bool {
	return value == cm.options.HeaderValue
}
