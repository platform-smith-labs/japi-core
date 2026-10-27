package handler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/platform-smith-labs/japi-core/v4/core"
	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
)

// AdaptHandler converts Handler[ParamTypeT, BodyTypeT, ResponseBodyT] to http.HandlerFunc.
//
// This adapter bridges the gap between the typed generic handler system and the standard
// http.HandlerFunc interface used by Chi router. It injects application dependencies
// (database, logger) into the handler context and handles error responses.
//
// Parameters:
//   - db: Database connection to inject into handler context
//   - logger: Logger instance to inject into handler context
//   - handler: The typed handler to adapt
//
// Returns: http.HandlerFunc compatible with Chi router
//
// Example:
//
//	handler := MakeHandler(myHandler, ParseParams, ResponseJSON)
//	r.Get("/users/{id}", AdaptHandler(db, logger, handler))
func AdaptHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return adaptHandler(db, logger, nil, handler)
}

// AdaptHandlerWithServices converts a typed Handler to http.HandlerFunc with service injection.
// Like AdaptHandler, but populates ctx.Services with the provided value.
//
// Parameters:
//   - db: Database connection to inject into handler context
//   - logger: Logger instance to inject into handler context
//   - services: Application-defined dependencies (opaque to japi-core)
//   - handler: The typed handler to adapt
//
// Returns: http.HandlerFunc compatible with Chi router
func AdaptHandlerWithServices[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	services any,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return adaptHandler(db, logger, services, handler)
}

// adaptHandler is the shared implementation for both AdaptHandler and AdaptHandlerWithServices.
func adaptHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	services any,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract request context for cancellation and timeout support
		requestCtx := r.Context()

		route := routePattern(requestCtx)

		// Resolve the request ID once, here, so every typed handler can read it from
		// ctx.RequestID without a per-handler middleware. It is not added to the
		// logger: log lines are correlated by trace ID instead (see below).
		requestID := httpmiddleware.GetRequestIDFromContext(requestCtx)

		// Enrich the logger rather than replacing it: everything the caller configured
		// (handlers, attrs, groups) is preserved, with the trace correlation keys of the
		// request's span added when one is present (e.g. from otel.Middleware).
		ctxLogger := withTraceAttrs(logger, requestCtx)

		// Log database connection status for debugging
		ctxLogger.Debug("AdaptHandler creating context",
			"db_nil", db == nil,
			"http.route", route,
		)

		// Create handler context with application dependencies and request context
		ctx := HandlerContext[ParamTypeT, BodyTypeT]{
			Context:     requestCtx, // Propagate HTTP request context
			DB:          db,
			Logger:      ctxLogger,
			Services:    services,
			UserUUID:    Nil[uuid.UUID](), // No auth by default
			CompanyUUID: Nil[uuid.UUID](), // No auth by default
			RequestID:   requestIDNullable(requestID),
		}

		// Execute the handler and handle response/errors
		_, err := handler(ctx, w, r)
		if err != nil {
			// Handle context-specific errors
			if errors.Is(err, context.Canceled) {
				// Client disconnected - don't write response
				ctxLogger.Info("Request cancelled by client", "http.route", route)
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				// Request timeout
				ctxLogger.Error("Request timeout", "http.route", route)
				core.WriteAPIError(w, r, *core.NewAPIError(
					http.StatusGatewayTimeout,
					"Request timeout",
				))
				return
			}

			// Log the error for debugging
			// TODO: err.Error() may echo user input (e.g. a rejected value quoted
			// by a validation or driver error); log a classification instead.
			ctxLogger.Error("Handler error", "error", err.Error(), "http.route", route)

			// Write appropriate error response based on error type
			if apiErr, ok := err.(*core.APIError); ok {
				core.WriteAPIError(w, r, *apiErr)
			} else {
				// Fallback for unexpected errors
				core.Error(w, r, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Success: Response handling is now delegated to middleware (e.g., ResponseJSON)
		// The handler chain is responsible for writing the response
	}
}

// unmatchedRoute is logged as http.route when no chi route pattern matched.
// It equals otel.UnmatchedRoute; the raw URL path is never logged in its place.
const unmatchedRoute = "unmatched"

// routePattern returns the matched chi route pattern, or unmatchedRoute.
func routePattern(ctx context.Context) string {
	if rctx := chi.RouteContext(ctx); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return unmatchedRoute
}

// withTraceAttrs adds trace_id, span_id and trace_flags of the span in ctx to
// logger, or returns logger unchanged when ctx carries no valid span context.
// The keys and formats match otel.TraceHandler.
func withTraceAttrs(logger *slog.Logger, ctx context.Context) *slog.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return logger
	}
	return logger.With(
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
		slog.String("trace_flags", sc.TraceFlags().String()),
	)
}

// requestIDNullable wraps a possibly-empty request ID.
//
// An empty ID stays Nil rather than becoming a present-but-empty value, so
// ctx.RequestID.HasValue() answers "do we have one?" honestly.
func requestIDNullable(requestID string) Nullable[string] {
	if requestID == "" {
		return Nil[string]()
	}
	return NewNullable(requestID)
}
