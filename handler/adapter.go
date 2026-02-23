package handler

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/google/uuid"
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
	return func(w http.ResponseWriter, r *http.Request) {
		// Log database connection status for debugging
		logger.Debug("AdaptHandler creating context",
			"db_nil", db == nil,
			"path", r.URL.Path,
		)

		// Create handler context with application dependencies
		ctx := HandlerContext[ParamTypeT, BodyTypeT]{
			DB:          db,
			Logger:      logger,
			UserUUID:    Nil[uuid.UUID](), // No auth by default
			CompanyUUID: Nil[uuid.UUID](), // No auth by default
		}

		// Execute the handler and handle response/errors
		_, err := handler(ctx, w, r)
		if err != nil {
			// Log the error for debugging
			logger.Error("Handler error", "error", err.Error(), "path", r.URL.Path)

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
