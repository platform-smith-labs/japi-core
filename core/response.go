package core

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/platform-smith-labs/japi-core/v4/internal/spanstatus"
)

// Simple response helper functions

// JSON sends a JSON response with the given status and data
func JSON[T any](w http.ResponseWriter, status int, data T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// JSONFile sends a JSON response as a downloadable file attachment.
// The filename parameter specifies the name of the downloaded file.
func JSONFile[T any](w http.ResponseWriter, status int, data T, filename string) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// Success sends a 200 OK JSON response
func Success[T any](w http.ResponseWriter, data T) error {
	return JSON(w, http.StatusOK, data)
}

// Created sends a 201 Created JSON response
func Created[T any](w http.ResponseWriter, data T) error {
	return JSON(w, http.StatusCreated, data)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// List sends a JSON response with data wrapped in a list structure
func List[T any](w http.ResponseWriter, data []T) error {
	response := map[string]any{
		"data":  data,
		"count": len(data),
	}
	return JSON(w, http.StatusOK, response)
}

// Health sends a health check response
func Health(w http.ResponseWriter, status string, checks map[string]bool) error {
	response := map[string]any{
		"status": status,
		"checks": checks,
	}
	return JSON(w, http.StatusOK, response)
}

// Error sends an error response in the deprecated envelope.
//
// Deprecated: return or write a *Problem (WriteProblem, WriteError). Error
// and the envelope are removed in a later major version.
func Error(w http.ResponseWriter, r *http.Request, status int, message string) error {
	apiErr := NewAPIError(status, message)
	return WriteAPIError(w, r, *apiErr)
}

// WriteAPIError writes apiErr in the deprecated {"error":{...}} envelope.
//
// For a 5xx the message is replaced by the HTTP status text and detail and
// fields are dropped: they may carry internal error text. The originals go to
// the log only.
//
// Deprecated: return or write a *Problem (WriteProblem, WriteError).
func WriteAPIError(w http.ResponseWriter, r *http.Request, apiErr APIError) error {
	return writeLegacy(w, r, apiErr)
}

// writeLegacy writes the deprecated envelope.
func writeLegacy(w http.ResponseWriter, r *http.Request, apiErr APIError, opts ...WriteOption) error {
	cfg := writeConfig{logger: slog.Default()}
	for _, o := range opts {
		o(&cfg)
	}
	ctx := r.Context()

	attrs := []slog.Attr{
		slog.Int("http.response.status_code", apiErr.Code),
		slog.String("http.request.method", r.Method),
		slog.String("http.route", requestRoute(r)),
	}
	// Message and detail are logged only for a server error: a 4xx detail
	// built from err.Error() can quote the request.
	if apiErr.Code >= 500 || apiErr.Code < 400 {
		attrs = append(attrs, slog.String("message", apiErr.Message))
		if apiErr.Detail != "" {
			attrs = append(attrs, slog.String("detail", apiErr.Detail))
		}
	}
	if len(apiErr.Fields) > 0 {
		attrs = append(attrs, slog.Int("validation_field_count", len(apiErr.Fields)))
	}
	attrs = append(attrs, cfg.logAttrs...)

	status := apiErr.Code
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	if !cfg.recorded {
		level := slog.LevelWarn
		if status >= 500 {
			level = slog.LevelError
			spanstatus.Pin(ctx, strconv.Itoa(status), strconv.Itoa(status))
		}
		cfg.logger.LogAttrs(ctx, level, "API error response", attrs...)
	}

	out := apiErr
	out.Code = status
	if status >= 500 {
		out = APIError{Code: status, Message: http.StatusText(status)}
	}
	return JSON(w, status, map[string]any{"error": out})
}

// unmatchedRoute is logged as http.route when no chi route pattern matched.
// It equals otel.UnmatchedRoute.
const unmatchedRoute = "unmatched"

// requestRoute returns the matched chi route pattern, or unmatchedRoute. The
// raw URL path is never used in its place.
func requestRoute(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return unmatchedRoute
}
