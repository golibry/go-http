package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	httplib "github.com/golibry/go-http/http"
)

func main() {
	// Example 1: JSON Response
	fmt.Println("=== JSON Response Example ===")
	recorder1 := httptest.NewRecorder()
	data := map[string]interface{}{
		"message": "Hello, World!",
		"status":  "success",
		"data":    []string{"item1", "item2", "item3"},
	}

	err := httplib.NewResponseBuilder(recorder1).
		Status(http.StatusCreated).
		Header("X-API-Version", "v1.0").
		Header("X-Request-ID", "12345").
		JSON().
		Data(data).
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder1.Code)
		fmt.Printf("Content-Type: %s\n", recorder1.Header().Get("Content-Type"))
		fmt.Printf("X-API-Version: %s\n", recorder1.Header().Get("X-API-Version"))
		fmt.Printf("Body: %s\n", recorder1.Body.String())
	}

	// Example 2: Text Response
	fmt.Println("\n=== Text Response Example ===")
	recorder2 := httptest.NewRecorder()

	err = httplib.NewResponseBuilder(recorder2).
		Status(http.StatusOK).
		Header("X-Custom-Header", "custom-value").
		Text().
		ContentString("This is a plain text response with custom headers").
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder2.Code)
		fmt.Printf("Content-Type: %s\n", recorder2.Header().Get("Content-Type"))
		fmt.Printf("Body: %s\n", recorder2.Body.String())
	}

	// Example 3: HTML Response
	fmt.Println("\n=== HTML Response Example ===")
	recorder3 := httptest.NewRecorder()
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Response Builder Example</title>
</head>
<body>
    <h1>Hello from Response Builder!</h1>
    <p>This HTML was generated using the builder pattern.</p>
</body>
</html>`

	err = httplib.NewResponseBuilder(recorder3).
		Status(http.StatusOK).
		HTML().
		ContentString(htmlContent).
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder3.Code)
		fmt.Printf("Content-Type: %s\n", recorder3.Header().Get("Content-Type"))
		fmt.Printf("Body length: %d characters\n", len(recorder3.Body.String()))
	}

	// Example 4: Text Error Response
	fmt.Println("\n=== Text Error Response Example ===")
	recorder4 := httptest.NewRecorder()
	testError := errors.New("validation failed: missing required field 'email'")

	err = httplib.NewResponseBuilder(recorder4).
		Status(http.StatusBadRequest).
		Header("X-Error-Code", "VALIDATION_ERROR").
		Error().
		WithError(testError).
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder4.Code)
		fmt.Printf("Content-Type: %s\n", recorder4.Header().Get("Content-Type"))
		fmt.Printf("X-Error-Code: %s\n", recorder4.Header().Get("X-Error-Code"))
		fmt.Printf("Body: %s\n", recorder4.Body.String())
	}

	// Example 5: JSON Error Response
	fmt.Println("\n=== JSON Error Response Example ===")
	recorder5 := httptest.NewRecorder()
	apiError := errors.New("internal server error occurred")

	err = httplib.NewResponseBuilder(recorder5).
		Status(http.StatusInternalServerError).
		Header("X-Request-ID", "req-67890").
		Error().
		WithError(apiError).
		AsJSON().
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder5.Code)
		fmt.Printf("Content-Type: %s\n", recorder5.Header().Get("Content-Type"))
		fmt.Printf("X-Request-ID: %s\n", recorder5.Header().Get("X-Request-ID"))
		fmt.Printf("Body: %s\n", recorder5.Body.String())
	}

	// Example 6: Custom Message Error Response
	fmt.Println("\n=== Custom Message Error Response Example ===")
	recorder6 := httptest.NewRecorder()

	err = httplib.NewResponseBuilder(recorder6).
		Status(http.StatusNotFound).
		Error().
		WithMessage("The requested resource was not found").
		AsJSON().
		Send()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", recorder6.Code)
		fmt.Printf("Content-Type: %s\n", recorder6.Header().Get("Content-Type"))
		fmt.Printf("Body: %s\n", recorder6.Body.String())
	}

	fmt.Println("\n=== Response Builder Pattern Examples Complete ===")
}