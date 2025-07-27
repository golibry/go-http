package middleware

import (
	"context"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type PathNormalizerSuite struct {
	suite.Suite
}

func TestPathNormalizerSuite(t *testing.T) {
	suite.Run(t, new(PathNormalizerSuite))
}

func (suite *PathNormalizerSuite) TestItCanNormalizePathWithSpaces() {
	testCases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "path with spaces",
			inputPath:    "/api /v1/ users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "path with multiple spaces",
			inputPath:    "/api  /v1  /  users  ",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "path with only spaces",
			inputPath:    "   ",
			expectedPath: "/",
		},
	}

	for _, tc := range testCases {
		suite.Run(
			tc.name, func() {
				var capturedPath string
				testHandler := http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						capturedPath = r.URL.Path
						w.WriteHeader(http.StatusOK)
					},
				)

				middleware := NewPathNormalizer(testHandler, context.Background())
				request := httptest.NewRequest("GET", "http://example.com/", nil)
				request.URL.Path = tc.inputPath
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				suite.Assert().Equal(tc.expectedPath, capturedPath)
			},
		)
	}
}

func (suite *PathNormalizerSuite) TestItCanNormalizePathWithMultipleSlashes() {
	testCases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "path with double slashes",
			inputPath:    "/api//v1//users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "path with multiple consecutive slashes",
			inputPath:    "/api///v1////users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "path with trailing slashes",
			inputPath:    "/api/v1/users///",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "root path with multiple slashes",
			inputPath:    "///",
			expectedPath: "/",
		},
	}

	for _, tc := range testCases {
		suite.Run(
			tc.name, func() {
				var capturedPath string
				testHandler := http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						capturedPath = r.URL.Path
						w.WriteHeader(http.StatusOK)
					},
				)

				middleware := NewPathNormalizer(testHandler, context.Background())
				request := httptest.NewRequest("GET", "http://example.com"+tc.inputPath, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				suite.Assert().Equal(tc.expectedPath, capturedPath)
			},
		)
	}
}

func (suite *PathNormalizerSuite) TestItCanHandleEdgeCases() {
	testCases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "empty path",
			inputPath:    "",
			expectedPath: "/",
		},
		{
			name:         "root path",
			inputPath:    "/",
			expectedPath: "/",
		},
		{
			name:         "already normalized path",
			inputPath:    "/api/v1/users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "complex mixed case",
			inputPath:    "//api //v1/// users //",
			expectedPath: "/api/v1/users",
		},
	}

	for _, tc := range testCases {
		suite.Run(
			tc.name, func() {
				var capturedPath string
				testHandler := http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						capturedPath = r.URL.Path
						w.WriteHeader(http.StatusOK)
					},
				)

				middleware := NewPathNormalizer(testHandler, context.Background())
				request := httptest.NewRequest("GET", "http://example.com/", nil)
				request.URL.Path = tc.inputPath
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				suite.Assert().Equal(tc.expectedPath, capturedPath)
			},
		)
	}
}

func (suite *PathNormalizerSuite) TestItCanChainWithOtherMiddleware() {
	var capturedPath string
	finalHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		},
	)

	// Create a simple middleware that adds a header
	headerMiddleware := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "middleware-chain")
			finalHandler.ServeHTTP(w, r)
		},
	)

	// Chain PathNormalizer with the header middleware
	pathNormalizer := NewPathNormalizer(headerMiddleware, context.Background())

	inputPath := "/api //v1/// users"
	expectedPath := "/api/v1/users"

	request := httptest.NewRequest("GET", "http://example.com/", nil)
	request.URL.Path = inputPath
	recorder := httptest.NewRecorder()

	pathNormalizer.ServeHTTP(recorder, request)

	// Verify path was normalized
	suite.Assert().Equal(expectedPath, capturedPath)
	// Verify the chain worked and the header was set
	suite.Assert().Equal("middleware-chain", recorder.Header().Get("X-Test"))
	suite.Assert().Equal(http.StatusOK, recorder.Code)
}

func (suite *PathNormalizerSuite) TestNormalizePathFunction() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "/"},
		{"root path", "/", "/"},
		{"simple path", "/users", "/users"},
		{"path with spaces", "/api /users", "/api/users"},
		{"path with multiple slashes", "/api//users", "/api/users"},
		{"complex case", "//api //v1/// users //", "/api/v1/users"},
		{"only slashes", "///", "/"},
		{"only spaces", "   ", "/"},
		{"mixed spaces and slashes", " // / // ", "/"},
	}

	for _, tc := range testCases {
		suite.Run(
			tc.name, func() {
				result := normalizePath(tc.input)
				suite.Assert().Equal(tc.expected, result)
			},
		)
	}
}
