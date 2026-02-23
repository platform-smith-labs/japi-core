package typed

import (
	"net/http"
	"time"

	"github.com/platform-smith-labs/japi-core/handler"
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

		// Log request
		ctx.Logger.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"start_time", startTime.Format(time.RFC3339Nano),
		)

		// Call next handler
		response, err := next(ctx, w, r)

		// Capture end time and calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Log response
		if err != nil {
			ctx.Logger.Error("HTTP Response Error",
				"method", r.Method,
				"path", r.URL.Path,
				"error", err.Error(),
				"start_time", startTime.Format(time.RFC3339Nano),
				"end_time", endTime.Format(time.RFC3339Nano),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			ctx.Logger.Info("HTTP Response Success",
				"method", r.Method,
				"path", r.URL.Path,
				"start_time", startTime.Format(time.RFC3339Nano),
				"end_time", endTime.Format(time.RFC3339Nano),
				"duration_ms", duration.Milliseconds(),
			)
		}

		return response, err
	}
}
