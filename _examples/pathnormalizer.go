package main

import (
	"fmt"
	"net/http"

	"github.com/golibry/go-http/http/router/middleware"
)

func main() {
	// Create a simple handler that shows the received path
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Received path: %s\n", r.URL.Path)
		fmt.Printf("Handler received path: %s\n", r.URL.Path)
	})

	// Create the PathNormalizer middleware
	pathNormalizer := middleware.NewPathNormalizer(finalHandler)

	// Create a test server
	server := &http.Server{
		Addr:    ":8080",
		Handler: pathNormalizer,
	}

	fmt.Println("PathNormalizer Example Server")
	fmt.Println("=============================")
	fmt.Println("Server starting on :8080")
	fmt.Println()
	fmt.Println("Test URLs to try:")
	fmt.Println("- http://localhost:8080/api /v1/ users")
	fmt.Println("- http://localhost:8080/api//v1///users")
	fmt.Println("- http://localhost:8080//api //v1/// users //")
	fmt.Println("- http://localhost:8080/   ")
	fmt.Println("- http://localhost:8080/api/v1/users (already normalized)")
	fmt.Println()
	fmt.Println("Watch the console for normalization logs!")
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}