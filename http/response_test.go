package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test helper types
type CustomHTTPError struct {
	message    string
	statusCode int
}

func (e CustomHTTPError) Error() string {
	return e.message
}

func (e CustomHTTPError) StatusCode() int {
	return e.statusCode
}

type ValidationError struct {
	field string
}

func (e ValidationError) Error() string {
	return "validation failed for field: " + e.field
}

type ResponseSuite struct {
	suite.Suite
}

func TestResponseSuite(t *testing.T) {
	suite.Run(t, new(ResponseSuite))
}

func (suite *ResponseSuite) TestResponseWriterCanCacheStatusCode() {
	expectedCode := 234
	baseResponseWriter := httptest.NewRecorder()
	responseWriter := NewResponseWriter(baseResponseWriter)
	responseWriter.WriteHeader(expectedCode)
	suite.Assert().Equal(expectedCode, baseResponseWriter.Code)
	suite.Assert().Equal(expectedCode, responseWriter.statusCode)
}

func (suite *ResponseSuite) TestItCanBuildJSONResponse() {
	recorder := httptest.NewRecorder()
	data := map[string]string{"message": "hello world"}
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusCreated).
		Header("X-Custom", "test").
		JSON().
		Data(data).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusCreated, recorder.Code)
	suite.Assert().Equal("application/json", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal("test", recorder.Header().Get("X-Custom"))
	
	var result map[string]string
	err = json.Unmarshal(recorder.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	suite.Assert().Equal("hello world", result["message"])
}

func (suite *ResponseSuite) TestItCanBuildTextResponse() {
	recorder := httptest.NewRecorder()
	content := "Hello, World!"
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusAccepted).
		Header("X-Custom", "text-test").
		Text().
		ContentString(content).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusAccepted, recorder.Code)
	suite.Assert().Equal("text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal("text-test", recorder.Header().Get("X-Custom"))
	suite.Assert().Equal(content, recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildTextResponseWithBytes() {
	recorder := httptest.NewRecorder()
	content := []byte("Hello, Bytes!")
	
	err := NewResponseBuilder(recorder).
		Text().
		Content(content).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusOK, recorder.Code)
	suite.Assert().Equal("text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal("Hello, Bytes!", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildHTMLResponse() {
	recorder := httptest.NewRecorder()
	htmlContent := "<h1>Hello, HTML!</h1>"
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusOK).
		HTML().
		ContentString(htmlContent).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusOK, recorder.Code)
	suite.Assert().Equal("text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal(htmlContent, recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildHTMLResponseWithBytes() {
	recorder := httptest.NewRecorder()
	htmlContent := []byte("<p>HTML from bytes</p>")
	
	err := NewResponseBuilder(recorder).
		HTML().
		Content(htmlContent).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal("text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal("<p>HTML from bytes</p>", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildTextErrorResponse() {
	recorder := httptest.NewRecorder()
	testError := errors.New("something went wrong")
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusBadRequest).
		Error().
		WithError(testError).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code)
	suite.Assert().Equal("text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Assert().Equal("something went wrong", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildJSONErrorResponse() {
	recorder := httptest.NewRecorder()
	testError := errors.New("json error occurred")
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusInternalServerError).
		Error().
		WithError(testError).
		AsJSON().
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusInternalServerError, recorder.Code)
	suite.Assert().Equal("application/json", recorder.Header().Get("Content-Type"))
	
	var result map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	suite.Assert().Equal("json error occurred", result["error"])
	suite.Assert().Equal(float64(500), result["status"])
}

func (suite *ResponseSuite) TestItCanBuildErrorResponseWithCustomMessage() {
	recorder := httptest.NewRecorder()
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusNotFound).
		Error().
		WithMessage("custom error message").
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusNotFound, recorder.Code)
	suite.Assert().Equal("custom error message", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanBuildErrorResponseWithStatusTextFallback() {
	recorder := httptest.NewRecorder()
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusNotFound).
		Error().
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusNotFound, recorder.Code)
	suite.Assert().Equal("Not Found", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanChainMultipleHeaders() {
	recorder := httptest.NewRecorder()
	
	err := NewResponseBuilder(recorder).
		Header("X-First", "first-value").
		Header("X-Second", "second-value").
		Header("X-Third", "third-value").
		Text().
		ContentString("test").
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal("first-value", recorder.Header().Get("X-First"))
	suite.Assert().Equal("second-value", recorder.Header().Get("X-Second"))
	suite.Assert().Equal("third-value", recorder.Header().Get("X-Third"))
}

func (suite *ResponseSuite) TestItCanHandleHTTPErrorInterface() {
	recorder := httptest.NewRecorder()
	httpError := CustomHTTPError{
		message:    "custom http error",
		statusCode: http.StatusUnauthorized,
	}
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(httpError).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusUnauthorized, recorder.Code)
	suite.Assert().Equal("custom http error", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanHandleHTTPErrorInterfaceWithJSON() {
	recorder := httptest.NewRecorder()
	httpError := CustomHTTPError{
		message:    "json http error",
		statusCode: http.StatusBadRequest,
	}
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(httpError).
		AsJSON().
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code)
	suite.Assert().Equal("application/json", recorder.Header().Get("Content-Type"))
	
	var result map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	suite.Assert().Equal("json http error", result["error"])
	suite.Assert().Equal(float64(400), result["status"])
}

func (suite *ResponseSuite) TestItCanHandleErrorCategories() {
	recorder := httptest.NewRecorder()
	validationError := ValidationError{field: "email"}
	
	// Create error category for validation errors
	validationCategory := NewErrorCategory(http.StatusBadRequest)
	AddErrorType[ValidationError](validationCategory)
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(validationError).
		AddErrorCategory(validationCategory).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code)
	suite.Assert().Equal("validation failed for field: email", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanHandleErrorCategoriesWithSentinelErrors() {
	recorder := httptest.NewRecorder()
	sentinelError := errors.New("database connection failed")
	actualError := sentinelError // In real scenario, this might be wrapped
	
	// Create error category for database errors
	dbCategory := NewErrorCategory(http.StatusServiceUnavailable)
	dbCategory.AddSentinelError(sentinelError)
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(actualError).
		AddErrorCategory(dbCategory).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusServiceUnavailable, recorder.Code)
	suite.Assert().Equal("database connection failed", recorder.Body.String())
}

func (suite *ResponseSuite) TestItCanLogErrorsWithStructuredLogger() {
	recorder := httptest.NewRecorder()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	ctx := context.Background()
	
	testError := errors.New("logged error")
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(testError).
		WithLogger(logger).
		WithContext(ctx).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusInternalServerError, recorder.Code)
	suite.Assert().Equal("logged error", recorder.Body.String())
	
	// Check that error was logged
	logOutput := logBuffer.String()
	suite.Assert().Contains(logOutput, "HTTP Request Error")
	suite.Assert().Contains(logOutput, "logged error")
	suite.Assert().Contains(logOutput, "StatusCode=500")
}

func (suite *ResponseSuite) TestItCanHandleMultipleErrorCategories() {
	recorder := httptest.NewRecorder()
	validationError := ValidationError{field: "username"}
	
	// Create multiple error categories
	validationCategory := NewErrorCategory(http.StatusBadRequest)
	AddErrorType[ValidationError](validationCategory)
	
	authCategory := NewErrorCategory(http.StatusUnauthorized)
	authCategory.AddSentinelError(errors.New("unauthorized"))
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(validationError).
		WithErrorCategories(validationCategory, authCategory).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code)
	suite.Assert().Equal("validation failed for field: username", recorder.Body.String())
}

func (suite *ResponseSuite) TestItPrioritizesHTTPErrorOverCategories() {
	recorder := httptest.NewRecorder()
	httpError := CustomHTTPError{
		message:    "http error priority test",
		statusCode: http.StatusTeapot, // 418
	}
	
	// Create error category that would match if HTTPError wasn't prioritized
	category := NewErrorCategory(http.StatusBadRequest)
	AddErrorType[CustomHTTPError](category)
	
	err := NewResponseBuilder(recorder).
		Error().
		WithError(httpError).
		AddErrorCategory(category).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusTeapot, recorder.Code) // Should use HTTPError status, not category
	suite.Assert().Equal("http error priority test", recorder.Body.String())
}

func (suite *ResponseSuite) TestItFallsBackToExplicitStatusCode() {
	recorder := httptest.NewRecorder()
	regularError := errors.New("regular error")
	
	err := NewResponseBuilder(recorder).
		Status(http.StatusNotFound).
		Error().
		WithError(regularError).
		Send()
	
	suite.Assert().NoError(err)
	suite.Assert().Equal(http.StatusNotFound, recorder.Code)
	suite.Assert().Equal("regular error", recorder.Body.String())
}
