package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/golibry/go-http/http/session"
)

const sessionContextKey string = "session"

// SessionMiddleware provides session handling middleware
type SessionMiddleware struct {
	next    http.Handler
	ctx     context.Context
	logger  *slog.Logger
	manager session.Manager
}

// NewSessionMiddleware creates new session middleware
func NewSessionMiddleware(
	next http.Handler,
	ctx context.Context,
	logger *slog.Logger,
	manager session.Manager,
) *SessionMiddleware {
	return &SessionMiddleware{
		next:    next,
		ctx:     ctx,
		logger:  logger,
		manager: manager,
	}
}

// ServeHTTP implements the middleware logic
func (sm *SessionMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to get an existing session
	sess, err := sm.manager.GetSession(sm.ctx, r)
	if err != nil && errors.Is(err, session.ErrSessionNotFound) {
		if sm.logger != nil {
			sm.logger.ErrorContext(sm.ctx, "Failed to get session", "error", err)
		}
	}

	// Add session to request context
	ctx := context.WithValue(r.Context(), sessionContextKey, sess)
	r = r.WithContext(ctx)

	sm.next.ServeHTTP(w, r)

	// Save a session if it exists and is dirty
	if sess != nil {
		if err := sess.Save(sm.ctx); err != nil && sm.logger != nil {
			sm.logger.ErrorContext(sm.ctx, "Failed to save session", "error", err)
		}
	}
}

// GetSessionFromContext retrieves session from request context
func GetSessionFromContext(ctx context.Context) (session.Session, bool) {
	sess, ok := ctx.Value(sessionContextKey).(session.Session)
	return sess, ok
}

// GetOrCreateSession gets an existing session or creates a new one
func GetOrCreateSession(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	manager session.Manager,
) (session.Session, error) {
	// Try to get the existing session from context first
	if sess, ok := GetSessionFromContext(ctx); ok && sess != nil {
		return sess, nil
	}

	// Try to get an existing session from request
	sess, err := manager.GetSession(ctx, r)
	if err == nil {
		return sess, nil
	}

	// Create a new session if none exists
	return manager.NewSession(ctx, w, r)
}