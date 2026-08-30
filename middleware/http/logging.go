// Package http provides standard HTTP middleware that works with http.Handler.
// These middleware can be applied globally to Chi routers via r.Use().
package http

import (
	"log/slog"
	"net/http"
)

// WithLogging creates structured logging middleware for Chi.
//
// This middleware logs HTTP requests and responses with structured logging.
// It captures the status code by wrapping the response writer.
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

			// Log request
			logger.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)

			// Call next handler
			next.ServeHTTP(ww, r)

			// Log response
			logger.Info("HTTP Response",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
			)
		})
	}
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
