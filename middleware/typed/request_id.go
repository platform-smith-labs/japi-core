package typed

import (
	"log/slog"
	"net/http"

	"github.com/platform-smith-labs/japi-core/handler"
	httpMiddleware "github.com/platform-smith-labs/japi-core/middleware/http"
)

// WithRequestID enriches HandlerContext with request ID for correlation and tracing.
//
// This middleware:
// - Extracts request ID from request context (set by http.WithRequestID middleware)
// - Stores it in ctx.RequestID for handler access
// - Enriches the logger with request_id field for structured logging
//
// Dependencies: Requires http.WithRequestID middleware to be applied first
// Context modifications: Sets ctx.RequestID, enriches ctx.Logger
// Use: Apply via MakeHandler(..., WithRequestID, ...)
//
// Example:
//
//	// In main.go - Apply HTTP middleware first
//	r := chi.NewRouter()
//	r.Use(http.WithRequestID())
//
//	// In handler definition - Add typed middleware
//	handler := MakeHandler(
//	    Server,
//	    RouteInfo{Method: "POST", Path: "/api/v1/users"},
//	    myHandler,
//	    WithRequestID,  // No type parameters or () needed!
//	    WithLogging,
//	)
//
// Request IDs enable:
// - Correlation of logs across microservices
// - Debugging distributed systems
// - Request tracing and observability
//
// Best Practices:
// - Apply http.WithRequestID() early in chi router middleware chain
// - Apply typed.WithRequestID before logging middleware
// - Access via ctx.RequestID.Value() in handlers
func WithRequestID[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Extract request ID from request context
		requestID := httpMiddleware.GetRequestID(r)

		// Store request ID in HandlerContext if present
		if requestID != "" {
			ctx.RequestID = handler.NewNullable(requestID)

			// Enrich logger with request ID for structured logging
			ctx.Logger = ctx.Logger.With(slog.String("request_id", requestID))
		}

		// Call next handler with enriched context
		return next(ctx, w, r)
	}
}
