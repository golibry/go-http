package middleware

import (
	"net/http"
	"strings"
)

// PathNormalizer middleware strips empty spaces and trailing extra slashes from the URL path
type PathNormalizer struct {
	next http.Handler
}

// NewPathNormalizer creates new PathNormalizer middleware
func NewPathNormalizer(next http.Handler) *PathNormalizer {
	return &PathNormalizer{
		next: next,
	}
}

// ServeHTTP implements the http.Handler interface
func (pn *PathNormalizer) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	originalPath := rq.URL.Path
	normalizedPath := normalizePath(originalPath)

	// Only modify the request if the path actually changed
	if originalPath != normalizedPath {
		// Update the request URL path
		rq.URL.Path = normalizedPath
	}

	// Continue with the next handler
	pn.next.ServeHTTP(rw, rq)
}

// normalizePath strips spaces and normalizes slashes in the URL path
func normalizePath(path string) string {
	// Strip all spaces from the path
	path = strings.ReplaceAll(path, " ", "")

	// Handle empty path or root path
	if path == "" || path == "/" {
		return "/"
	}

	// Split the path into segments, removing empty segments (which represent multiple slashes)
	segments := strings.Split(path, "/")
	var cleanSegments []string

	for _, segment := range segments {
		if segment != "" {
			cleanSegments = append(cleanSegments, segment)
		}
	}

	// Rebuild the path
	if len(cleanSegments) == 0 {
		return "/"
	}

	return "/" + strings.Join(cleanSegments, "/")
}
