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
// Middlewares execute in the same order they appear in namedMiddlewares.
// Overrides replace matching default middlewares by name. Overrides with new
// names are appended after the defaults and execute in their provided order.
func WithNamedMiddlewares(
	handler http.Handler,
	namedMiddlewares []NamedMiddleware,
	overrides []NamedMiddleware,
) http.Handler {
	overrideMap := make(map[string]func(http.Handler) http.Handler)
	if overrides != nil {
		for _, override := range overrides {
			overrideMap[override.Name] = override.Middleware
		}
	}

	resolvedMiddlewares := make([]NamedMiddleware, 0, len(namedMiddlewares)+len(overrides))
	for _, namedMw := range namedMiddlewares {
		if overrideMiddleware, exists := overrideMap[namedMw.Name]; exists {
			resolvedMiddlewares = append(resolvedMiddlewares, NamedMiddleware{
				Name:       namedMw.Name,
				Middleware: overrideMiddleware,
			})
		} else {
			resolvedMiddlewares = append(resolvedMiddlewares, namedMw)
		}
	}

	if overrides != nil {
		for _, override := range overrides {
			found := false
			for _, namedMw := range namedMiddlewares {
				if namedMw.Name == override.Name {
					found = true
					break
				}
			}
			if !found {
				resolvedMiddlewares = append(resolvedMiddlewares, override)
			}
		}
	}

	for i := len(resolvedMiddlewares) - 1; i >= 0; i-- {
		handler = resolvedMiddlewares[i].Middleware(handler)
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
