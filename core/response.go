package core

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

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
