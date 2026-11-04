package typed

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/handler"
)

// WithLogging creates structured logging middleware for typed handlers.
//
// This middleware logs HTTP requests and responses using the logger from HandlerContext.
// It should be the LAST middleware in the handler.MakeHandler list (since middleware is
// applied in reverse order, this will execute first and last, wrapping all other middleware).
//
// Dependencies: ctx.Logger from HandlerContext
// Context modifications: None
// Use: Apply via MakeHandler(..., RequireAuth, ParseBody, ResponseJSON, WithLogging)
//
// Example:
//
//	handler := MakeHandler(
//	    RouteInfo{Method: "POST", Path: "/api/v1/users"},
//	    myHandler,
//	    RequireAuth,
//	    ParseBody,
//	    ResponseJSON,
//	    WithLogging,  // Last in list = wraps everything
//	)
func WithLogging[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Capture start time
		startTime := time.Now()

		// Log the chi route pattern, never the raw path (unbounded, may carry
		// identifiers). Client address and user agent are not logged.
		route := routePattern(r)

		// Log request
		ctx.Logger.InfoContext(r.Context(), "HTTP Request",
			"method", r.Method,
			"http.route", route,
			"start_time", startTime.Format(time.RFC3339Nano),
		)

		// Call next handler
		response, err := next(ctx, w, r)

		// Capture end time and calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Log response
		if err != nil {
			// The error is described by its code and status, never its
			// text: error text can quote request data. The error response
			// writer logs the cause once, at the point of writing.
			code, status := core.Classify(err)
			level := slog.LevelWarn
			if status >= 500 {
				level = slog.LevelError
			}
			ctx.Logger.Log(r.Context(), level, "HTTP Response Error",
				"method", r.Method,
				"http.route", route,
				"error.type", string(code),
				"http.response.status_code", status,
				"start_time", startTime.Format(time.RFC3339Nano),
				"end_time", endTime.Format(time.RFC3339Nano),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			ctx.Logger.InfoContext(r.Context(), "HTTP Response Success",
				"method", r.Method,
				"http.route", route,
				"start_time", startTime.Format(time.RFC3339Nano),
				"end_time", endTime.Format(time.RFC3339Nano),
				"duration_ms", duration.Milliseconds(),
			)
		}

		return response, err
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
