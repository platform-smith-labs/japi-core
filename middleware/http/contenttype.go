package http

import (
	"net/http"
)

// WithContentType sets the Content-Type header for all responses.
//
// This middleware adds a Content-Type header to every HTTP response.
// Useful for APIs that always return the same content type.
//
// Dependencies: None
// Context modifications: None
// Use: Apply to chi router via r.Use(WithContentType("application/json"))
//
// Example:
//
//	r := chi.NewRouter()
//	r.Use(WithContentType("application/json"))
func WithContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}
