package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/golibry/go-http/http/router/middleware"
	"github.com/golibry/go-http/http/session"
	"github.com/golibry/go-http/http/session/storage"
)

// session_management.go
//
// Demonstrates setting up the session manager, wiring the session middleware,
// and using sessions in handlers (attributes and flash messages). This example
// uses the in-memory storage implementation.
//
// How to run:
//   go run ./_examples/session_management.go
//
// What to look for:
//   The first request creates a session, sets attributes and a flash message.
//   The second request reads the stored attribute and consumes the flash.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// 1) Configure the manager
	store := storage.NewMemoryStorage()
	options := session.DefaultOptions()
	manager := session.NewManager(store, ctx, logger, options)

	// Start background GC (good practice for long-running services)
	manager.StartGC(ctx)
	defer manager.StopGC()

	// 2) Build a handler that uses sessions
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get or create the session for this request
		sess, err := middleware.GetOrCreateSession(r.Context(), w, r, manager)
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}

		// On first path, set some data
		if r.URL.Path == "/set" {
			sess.Set("user_id", "12345")
			sess.AddFlash("Welcome back!")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("set"))
			return
		}

		// On second path, read and consume the data
		if r.URL.Path == "/get" {
			if v, ok := sess.Get("user_id"); ok {
				_, _ = fmt.Fprintf(w, "user_id=%v\n", v)
			} else {
				_, _ = fmt.Fprintln(w, "user_id missing")
			}
			flashes := sess.GetFlashes() // reading removes them
			_, _ = fmt.Fprintf(w, "flashes=%v\n", flashes)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	// 3) Wrap with session middleware so sessions are saved automatically
	chain := middleware.NewSessionMiddleware(app, ctx, logger, manager)

	// Simulate two requests: set then get
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "http://example.com/set", nil)
	chain.ServeHTTP(rec1, req1)
	fmt.Println("/set ->", rec1.Code, rec1.Body.String())

	// Reuse the cookie to access the same session
	cookie := rec1.Result().Cookies()[0]
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/get", nil)
	req2.AddCookie(cookie)
	chain.ServeHTTP(rec2, req2)
	fmt.Println("/get ->", rec2.Code)
	fmt.Println(rec2.Body.String())
}
