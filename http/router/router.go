package router

import (
	"net/http"
)

// NamedMiddleware represents middleware with an identifier
type NamedMiddleware struct {
	Name       string
	Middleware func(http.Handler) http.Handler
}

// WithNamedMiddlewares applies named middlewares with selective override capability
func WithNamedMiddlewares(
	handler http.Handler,
	namedMiddlewares []NamedMiddleware,
	overrides []NamedMiddleware,
) http.Handler {
	// Create a map of override middleware names to functions for a quick lookup
	overrideMap := make(map[string]func(http.Handler) http.Handler)

	// Add override middlewares to the map
	if overrides != nil {
		for _, override := range overrides {
			overrideMap[override.Name] = override.Middleware
		}
	}

	// Apply middlewares in the order they appear in namedMiddlewares
	// This preserves the intended middleware chain order
	for _, namedMw := range namedMiddlewares {
		if overrideMiddleware, exists := overrideMap[namedMw.Name]; exists {
			// Use override middleware if available
			handler = overrideMiddleware(handler)
		} else {
			// Use original middleware
			handler = namedMw.Middleware(handler)
		}
	}

	// Apply any additional middlewares from overrides that weren't in the original list
	// This maintains ordering for leftover overrides
	if overrides != nil {
		for _, override := range overrides {
			// Check if this middleware name was not in the original list
			found := false
			for _, namedMw := range namedMiddlewares {
				if namedMw.Name == override.Name {
					found = true
					break
				}
			}
			if !found {
				handler = override.Middleware(handler)
			}
		}
	}

	return handler
}

type ServerMuxWrapper struct {
	http.ServeMux
	defaultNamedMiddlewares []NamedMiddleware
}

// NewServerMuxWrapper creates a new ServerMuxWrapper with named middlewares
func NewServerMuxWrapper(namedMiddlewares []NamedMiddleware) *ServerMuxWrapper {
	return &ServerMuxWrapper{
		ServeMux:                http.ServeMux{},
		defaultNamedMiddlewares: namedMiddlewares,
	}
}

func (mux *ServerMuxWrapper) Handle(pattern string, handler http.Handler) {
	finalHandler := WithNamedMiddlewares(handler, mux.defaultNamedMiddlewares, nil)
	mux.ServeMux.Handle(pattern, finalHandler)
}

// HandleWithCustomMiddlewares allows selective override of default middlewares
// while preserving non-overridden defaults
func (mux *ServerMuxWrapper) HandleWithCustomMiddlewares(
	pattern string,
	handler http.Handler,
	overrides []NamedMiddleware,
) {
	finalHandler := WithNamedMiddlewares(handler, mux.defaultNamedMiddlewares, overrides)
	mux.ServeMux.Handle(pattern, finalHandler)
}
