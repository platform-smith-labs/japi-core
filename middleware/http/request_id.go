package http

import (
	"context"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// RequestIDHeader is the HTTP header name for request IDs.
const RequestIDHeader = "X-Request-ID"

// contextKey is the type of this package's context keys, so they cannot
// collide with a bare string key set by any other package.
type contextKey string

// RequestIDContextKey is the context key for storing request IDs. Read it
// with GetRequestID / GetRequestIDFromContext and set it with
// ContextWithRequestID; ctx.Value("request_id") does not match it.
const RequestIDContextKey contextKey = "request_id"

// ContextWithRequestID returns ctx carrying requestID, which GetRequestID
// then returns in preference to chi's request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDContextKey, requestID)
}

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
			ctx := ContextWithRequestID(r.Context(), requestID)

			// Continue with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID extracts the request ID from the request context.
//
// It understands BOTH sources of a request ID, which is the whole point:
//
//  1. this package's WithRequestID, and
//  2. chi's middleware.RequestID — which router.NewChiRouter installs by default,
//     and which stores the ID under a private key of its own.
//
// Consulting only (1) was a latent bug: the default router installs (2), so
// GetRequestID returned "" on every request for any consumer that had not also
// added WithRequestID by hand. The ID was generated and then discarded.
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
	return GetRequestIDFromContext(r.Context())
}

// GetRequestIDFromContext is GetRequestID for callers that hold a context rather
// than a *http.Request — handlers whose context has outlived the request value,
// and background work started from one.
func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// This package's own middleware takes precedence: if both ran, WithRequestID
	// is the more specific choice, having been added deliberately.
	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok && requestID != "" {
		return requestID
	}
	// Fall back to chi's middleware.RequestID, which the default router installs.
	return chimiddleware.GetReqID(ctx)
}

// WithRequestIDHeader echoes the request ID onto the RESPONSE as X-Request-ID.
//
// chi's middleware.RequestID reads the inbound header (or generates an ID) into the
// request context and deliberately writes nothing back, so without this the caller
// never learns the ID of the request it just made — which is precisely what a person
// holding a failed response needs in order to ask about it.
//
// Install it AFTER chi's middleware.RequestID, so the ID exists by the time it runs.
// It writes the header before the handler executes, so the value survives a handler
// that writes its own status or panics into Recoverer.
//
// It is a no-op when no request ID is present, and never overwrites a header a
// handler has already set deliberately.
func WithRequestIDHeader() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if w.Header().Get(RequestIDHeader) == "" {
				if requestID := GetRequestID(r); requestID != "" {
					w.Header().Set(RequestIDHeader, requestID)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
