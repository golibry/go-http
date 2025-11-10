package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/golibry/go-http/http/router"
	"github.com/golibry/go-http/http/router/middleware"
)

// router_chain.go
//
// Demonstrates chaining multiple middlewares around a handler using the
// router.ServerMuxWrapper and named middlewares. The example wraps a base
// handler with Recoverer and Access Logger, then serves a test request.
//
// How to run:
//   go run ./_examples/router_chain.go
//
// What to look for:
//   Observe that the request is processed successfully and a structured
//   access log is printed. You can adapt this pattern to your own mux/routes.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// Base application handler
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	// Named middlewares for the default chain
	middlewares := []router.NamedMiddleware{
		{
			Name: "recoverer",
			Middleware: func(next http.Handler) http.Handler {
				return middleware.NewRecoverer(next, ctx, logger)
			},
		},
		{
			Name: "access",
			Middleware: func(next http.Handler) http.Handler {
				return middleware.NewHTTPAccessLogger(
					next,
					logger,
					middleware.AccessLogOptions{LogClientIp: true},
				)
			},
		},
	}

	mux := router.NewServerMuxWrapper(middlewares)
	mux.Handle("/", base)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()

	// Serve the request through the mux (and middleware chain)
	mux.ServeHTTP(rec, req)

	fmt.Println("Status:", rec.Code)
	fmt.Println("Body:", rec.Body.String())
}
