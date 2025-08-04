package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	httplib "github.com/golibry/go-http/http"
)

// Custom error types demonstrating HTTPError interface
type AuthenticationError struct {
	message string
}

func (e AuthenticationError) Error() string {
	return e.message
}

func (e AuthenticationError) StatusCode() int {
	return http.StatusUnauthorized
}

type ValidationError struct {
	field string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field: %s", e.field)
}

// Database connection error (sentinel error)
var ErrDatabaseConnection = errors.New("database connection failed")

func main() {
	fmt.Println("Enhanced Error Handling Example")
	fmt.Println("================================")

	// Setup structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := context.Background()

	// Setup error categories
	validationCategory := httplib.NewErrorCategory(http.StatusBadRequest)
	httplib.AddErrorType[ValidationError](validationCategory)

	dbCategory := httplib.NewErrorCategory(http.StatusServiceUnavailable)
	dbCategory.AddSentinelError(ErrDatabaseConnection)

	fmt.Println("\n1. HTTPError Interface - Automatic Status Code Detection:")
	demonstrateHTTPError(logger, ctx)

	fmt.Println("\n2. Error Categories - Flexible Error Classification:")
	demonstrateErrorCategories(logger, ctx, validationCategory, dbCategory)

	fmt.Println("\n3. Structured Logging - Comprehensive Error Tracking:")
	demonstrateStructuredLogging(logger, ctx)

	fmt.Println("\n4. Combined Features - Real-world Scenario:")
	demonstrateRealWorldScenario(logger, ctx, validationCategory, dbCategory)
}

func demonstrateHTTPError(logger *slog.Logger, ctx context.Context) {
	recorder := httptest.NewRecorder()
	authError := AuthenticationError{message: "invalid credentials"}

	err := httplib.NewResponseBuilder(recorder).
		Error().
		WithError(authError).
		WithLogger(logger).
		WithContext(ctx).
		AsJSON().
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", recorder.Code)
	fmt.Printf("Response: %s\n", recorder.Body.String())
}

func demonstrateErrorCategories(logger *slog.Logger, ctx context.Context, validationCategory, dbCategory *httplib.ErrorCategory) {
	// Validation error example
	recorder1 := httptest.NewRecorder()
	validationErr := ValidationError{field: "email"}

	err := httplib.NewResponseBuilder(recorder1).
		Error().
		WithError(validationErr).
		WithLogger(logger).
		WithContext(ctx).
		AddErrorCategory(validationCategory).
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Validation Error - Status Code: %d, Response: %s\n", 
		recorder1.Code, recorder1.Body.String())

	// Database error example
	recorder2 := httptest.NewRecorder()

	err = httplib.NewResponseBuilder(recorder2).
		Error().
		WithError(ErrDatabaseConnection).
		WithLogger(logger).
		WithContext(ctx).
		AddErrorCategory(dbCategory).
		AsJSON().
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Database Error - Status Code: %d, Response: %s\n", 
		recorder2.Code, recorder2.Body.String())
}

func demonstrateStructuredLogging(logger *slog.Logger, ctx context.Context) {
	recorder := httptest.NewRecorder()
	genericError := errors.New("something went wrong in the system")

	err := httplib.NewResponseBuilder(recorder).
		Error().
		WithError(genericError).
		WithLogger(logger).
		WithContext(ctx).
		WithMessage("A system error occurred").
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", recorder.Code)
	fmt.Printf("Response: %s\n", recorder.Body.String())
}

func demonstrateRealWorldScenario(logger *slog.Logger, ctx context.Context, validationCategory, dbCategory *httplib.ErrorCategory) {
	fmt.Println("Simulating a user registration endpoint with multiple error types...")

	// Scenario 1: Validation error
	recorder1 := httptest.NewRecorder()
	validationErr := ValidationError{field: "password"}

	err := httplib.NewResponseBuilder(recorder1).
		Error().
		WithError(validationErr).
		WithLogger(logger).
		WithContext(ctx).
		WithErrorCategories(validationCategory, dbCategory).
		AsJSON().
		Send()

	if err == nil {
		fmt.Printf("Validation Error Response: %s\n", recorder1.Body.String())
	}

	// Scenario 2: Authentication error (HTTPError interface)
	recorder2 := httptest.NewRecorder()
	authErr := AuthenticationError{message: "session expired"}

	err = httplib.NewResponseBuilder(recorder2).
		Error().
		WithError(authErr).
		WithLogger(logger).
		WithContext(ctx).
		WithErrorCategories(validationCategory, dbCategory).
		AsJSON().
		Send()

	if err == nil {
		fmt.Printf("Authentication Error Response: %s\n", recorder2.Body.String())
	}

	// Scenario 3: Database error (sentinel error)
	recorder3 := httptest.NewRecorder()

	err = httplib.NewResponseBuilder(recorder3).
		Error().
		WithError(ErrDatabaseConnection).
		WithLogger(logger).
		WithContext(ctx).
		WithErrorCategories(validationCategory, dbCategory).
		AsJSON().
		Send()

	if err == nil {
		fmt.Printf("Database Error Response: %s\n", recorder3.Body.String())
	}

	fmt.Println("\nKey Benefits:")
	fmt.Println("- Automatic status code detection from error types")
	fmt.Println("- Flexible error categorization without changing error types")
	fmt.Println("- Structured logging with context for observability")
	fmt.Println("- Easy integration with existing middleware patterns")
	fmt.Println("- Backward compatibility with existing error handling")
}