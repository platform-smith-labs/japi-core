package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header name for request IDs
	RequestIDHeader = "X-Request-ID"

	// RequestIDContextKey is the context key for storing request IDs
	RequestIDContextKey = "request_id"
)

// WithRequestID generates or propagates request IDs for correlation and tracing.
//
// This middleware:
// - Reads X-Request-ID from incoming request headers
// - Generates a new UUID if no request ID is present
// - Stores the request ID in the request context
// - Adds X-Request-ID to the response headers
//
// Dependencies: None
// Context modifications: Adds request_id to context
// Use: Apply to chi router via r.Use(WithRequestID())
//
// Example:
//
//	r := chi.NewRouter()
//	r.Use(WithRequestID())
//
// Request IDs enable:
// - Correlation of logs across microservices
// - Debugging distributed systems
// - Request tracing and observability
//
// Best Practices:
// - Apply early in middleware chain (before logging)
// - Use typed middleware to add to HandlerContext
// - Include request ID in all log statements
func WithRequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to read existing request ID from header
			requestID := r.Header.Get(RequestIDHeader)

			// Generate new UUID if no request ID present
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Add request ID to response header
			w.Header().Set(RequestIDHeader, requestID)

			// Store request ID in context for downstream use
			ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)

			// Continue with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID extracts the request ID from the request context.
//
// Returns empty string if no request ID is found.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    requestID := http.GetRequestID(r)
//	    log.Printf("Handling request %s", requestID)
//	}
func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}
