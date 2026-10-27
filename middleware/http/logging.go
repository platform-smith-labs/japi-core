// Package http provides standard HTTP middleware that works with http.Handler.
// These middleware can be applied globally to Chi routers via r.Use().
package http

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// WithLogging creates structured logging middleware for Chi.
//
// This middleware logs HTTP requests and responses with structured logging.
// It captures the status code by wrapping the response writer.
//
// Logged keys: method on the request line; method, http.route (the chi route
// pattern, or "unmatched") and status on the response line. The raw URL path,
// query string, client address and user agent are never logged. Both lines are
// logged with the request context, so a logger built on otel.TraceHandler adds
// trace_id and span_id when tracing middleware runs outside this one.
//
// Dependencies: *slog.Logger
// Context modifications: None
// Use: Apply to chi router via r.Use(WithLogging(logger))
//
// Example:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	r := chi.NewRouter()
//	r.Use(WithLogging(logger))
func WithLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a wrapped response writer to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request. The route is not known yet (chi matches it inside
			// next), and the raw path, query string, client address and user
			// agent are deliberately not logged.
			logger.InfoContext(r.Context(), "HTTP Request",
				"method", r.Method,
			)

			// Call next handler
			next.ServeHTTP(ww, r)

			// Log response
			logger.InfoContext(r.Context(), "HTTP Response",
				"method", r.Method,
				"http.route", routePattern(r),
				"status", ww.statusCode,
			)
		})
	}
}

// unmatchedRoute is logged as http.route when no chi route pattern matched.
// It equals otel.UnmatchedRoute.
const unmatchedRoute = "unmatched"

// routePattern returns the matched chi route pattern, or unmatchedRoute.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return unmatchedRoute
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Unwrap returns the wrapped writer so http.ResponseController can reach
// optional methods (SetWriteDeadline, EnableFullDuplex, ...) through this wrapper.
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Flush flushes the wrapped writer when it supports flushing; otherwise it is a no-op.
func (w *responseWriter) Flush() { _ = w.FlushError() }

// FlushError is Flush returning the wrapped writer's error, so
// http.ResponseController.Flush reports failures through this wrapper.
func (w *responseWriter) FlushError() error {
	return http.NewResponseController(w.ResponseWriter).Flush()
}

// Hijack hijacks the wrapped writer's connection, or returns an error wrapping
// http.ErrNotSupported when the wrapped writer cannot be hijacked.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}
