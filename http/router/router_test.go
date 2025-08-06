package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
}

func TestRouterSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}

// Helper function to create a middleware that adds a header
func createTestMiddleware(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Middleware", name)
			next.ServeHTTP(w, r)
		})
	}
}

// Helper handler that writes "OK"
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

func (suite *RouterTestSuite) TestItCanApplyNamedMiddlewaresInOrder() {
	// Arrange
	namedMiddlewares := []NamedMiddleware{
		{Name: "first", Middleware: createTestMiddleware("first")},
		{Name: "second", Middleware: createTestMiddleware("second")},
		{Name: "third", Middleware: createTestMiddleware("third")},
	}
	
	handler := WithNamedMiddlewares(testHandler(), namedMiddlewares, nil)
	
	// Act
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	middlewareHeaders := recorder.Header().Values("X-Middleware")
	// Middlewares are applied in reverse order (last wraps first)
	assert.Equal(suite.T(), []string{"third", "second", "first"}, middlewareHeaders)
}

func (suite *RouterTestSuite) TestItCanOverrideNamedMiddlewares() {
	// Arrange
	namedMiddlewares := []NamedMiddleware{
		{Name: "first", Middleware: createTestMiddleware("first")},
		{Name: "second", Middleware: createTestMiddleware("second")},
		{Name: "third", Middleware: createTestMiddleware("third")},
	}
	
	overrides := []NamedMiddleware{
		{Name: "second", Middleware: createTestMiddleware("overridden-second")},
	}
	
	handler := WithNamedMiddlewares(testHandler(), namedMiddlewares, overrides)
	
	// Act
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	middlewareHeaders := recorder.Header().Values("X-Middleware")
	// Middlewares are applied in reverse order (last wraps first)
	assert.Equal(suite.T(), []string{"third", "overridden-second", "first"}, middlewareHeaders)
}

func (suite *RouterTestSuite) TestItCanAddLeftoverOverridesWithOrdering() {
	// Arrange
	namedMiddlewares := []NamedMiddleware{
		{Name: "first", Middleware: createTestMiddleware("first")},
		{Name: "second", Middleware: createTestMiddleware("second")},
	}
	
	overrides := []NamedMiddleware{
		{Name: "second", Middleware: createTestMiddleware("overridden-second")},
		{Name: "extra", Middleware: createTestMiddleware("extra")},
		{Name: "bonus", Middleware: createTestMiddleware("bonus")},
	}
	
	handler := WithNamedMiddlewares(testHandler(), namedMiddlewares, overrides)
	
	// Act
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	middlewareHeaders := recorder.Header().Values("X-Middleware")
	
	// With slice-based overrides, leftover overrides should maintain their order
	// Expected order: bonus, extra, overridden-second, first (reverse due to wrapping)
	assert.Equal(suite.T(), []string{"bonus", "extra", "overridden-second", "first"}, middlewareHeaders)
}

func (suite *RouterTestSuite) TestItHandlesEmptyOverrides() {
	// Arrange
	namedMiddlewares := []NamedMiddleware{
		{Name: "first", Middleware: createTestMiddleware("first")},
		{Name: "second", Middleware: createTestMiddleware("second")},
	}
	
	handler := WithNamedMiddlewares(testHandler(), namedMiddlewares, nil)
	
	// Act
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	middlewareHeaders := recorder.Header().Values("X-Middleware")
	// Middlewares are applied in reverse order (last wraps first)
	assert.Equal(suite.T(), []string{"second", "first"}, middlewareHeaders)
}