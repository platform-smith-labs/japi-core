package core

import (
	"encoding/json"
	"log/slog"
	"net/http"
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

// Error sends an error response with logging
func Error(w http.ResponseWriter, r *http.Request, status int, message string) error {
	apiErr := NewAPIError(status, message)
	return WriteAPIError(w, r, *apiErr)
}

// WriteAPIError sends an error response for APIError types with comprehensive logging
func WriteAPIError(w http.ResponseWriter, r *http.Request, apiErr APIError) error {
	// Build log fields
	logFields := []any{
		"status", apiErr.Code,
		"message", apiErr.Message,
	}

	if apiErr.Detail != "" {
		logFields = append(logFields, "detail", apiErr.Detail)
	}

	if len(apiErr.Fields) > 0 {
		logFields = append(logFields, "validation_field_count", len(apiErr.Fields))
		logFields = append(logFields, "validation_fields", apiErr.Fields)
	}

	// Add request context
	logFields = append(logFields, extractRequestContext(r)...)

	// Log based on status code
	if apiErr.Code >= 500 {
		slog.Error("API error response", logFields...)
	} else if apiErr.Code >= 400 {
		slog.Warn("API error response", logFields...)
	} else {
		slog.Info("API error response", logFields...)
	}

	// Unified response structure
	response := map[string]any{
		"error": apiErr,
	}
	return JSON(w, apiErr.Code, response)
}

// extractRequestContext extracts useful request context for logging
func extractRequestContext(r *http.Request) []any {
	logFields := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
	}

	if r.URL.RawQuery != "" {
		logFields = append(logFields, "query", r.URL.RawQuery)
	}

	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		logFields = append(logFields, "request_id", requestID)
	}

	return logFields
}
