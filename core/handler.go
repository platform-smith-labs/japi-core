package core

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// HandlerFunc represents a handler that can return an error for cleaner composition
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP converts our custom HandlerFunc to standard http.Handler
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		handleError(w, r, err)
	}
}

// APIError represents a structured API error
type APIError struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e APIError) Error() string {
	msg := fmt.Sprintf("API Error %d: %s", e.Code, e.Message)

	if e.Detail != "" {
		msg += fmt.Sprintf(" - %s", e.Detail)
	}

	if len(e.Fields) > 0 {
		var fields []string
		for k, v := range e.Fields {
			fields = append(fields, fmt.Sprintf("%s=%s", k, v))
		}
		msg += fmt.Sprintf(" [fields: %s]", strings.Join(fields, ", "))
	}

	return msg
}

// NewAPIError creates a new API error
func NewAPIError(code int, message string, detail ...string) *APIError {
	err := &APIError{
		Code:    code,
		Message: message,
	}
	if len(detail) > 0 {
		err.Detail = detail[0]
	}
	return err
}

// NewValidationError creates a new validation error with field details
func NewValidationError(message string) *APIError {
	return &APIError{
		Code:    http.StatusBadRequest,
		Message: message,
		Fields:  make(map[string]string),
	}
}

// AddField adds a field error to the APIError and returns the error for chaining
func (e *APIError) AddField(fieldName, fieldError string) *APIError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}

	// If field already has an error, join with " || " separator
	if existing, exists := e.Fields[fieldName]; exists && strings.TrimSpace(existing) != "" {
		e.Fields[fieldName] = existing + " || " + fieldError
	} else {
		e.Fields[fieldName] = fieldError
	}

	return e
}

// Common API errors
var (
	ErrBadRequest   = &APIError{Code: http.StatusBadRequest, Message: "Bad Request"}
	ErrUnauthorized = &APIError{Code: http.StatusUnauthorized, Message: "Unauthorized"}
	ErrForbidden    = &APIError{Code: http.StatusForbidden, Message: "Forbidden"}
	ErrNotFound     = &APIError{Code: http.StatusNotFound, Message: "Not Found"}
	ErrInternal     = &APIError{Code: http.StatusInternalServerError, Message: "Internal Server Error"}
)

// handleError handles errors in a centralized way
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	// Handle APIError types directly
	if apiErr, ok := err.(*APIError); ok {
		WriteAPIError(w, r, *apiErr)
		return
	}

	// Handle unknown errors - convert to APIError
	slog.Error("Unexpected error in handler",
		"original_error", err.Error(),
		"method", r.Method,
		"path", r.URL.Path,
	)

	// Convert to APIError and send response
	apiErr := NewAPIError(http.StatusInternalServerError, "Internal Server Error", err.Error())
	WriteAPIError(w, r, *apiErr)
}

// WrapHandler converts a regular http.HandlerFunc to our HandlerFunc
func WrapHandler(h http.HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}
