package http

import (
	"context"
	"net/http"

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
// then returns.
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

// GetRequestID returns the request ID stored by WithRequestID,
// ContextWithRequestID or otel.RequestIDFromTrace, or "".
//
// chi's middleware.RequestID is no longer read: this package does not import
// chi's middleware package, which registers net/http/pprof handlers on
// http.DefaultServeMux as an import side effect.
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
	requestID, _ := ctx.Value(RequestIDContextKey).(string)
	return requestID
}

// WithRequestIDHeader echoes the request ID onto the RESPONSE as X-Request-ID.
//
// It is for chains that put a request ID in the context without writing the
// header (WithRequestID already writes it). Install it after whatever stores the ID.
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
