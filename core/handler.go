package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// HandlerFunc is an http handler that returns an error.
//
// A returned error is written with WriteError: a *Problem as constructed,
// anything else as the generic 500 internal_error problem (its text is logged,
// never sent). A context.Canceled error (the client went away) writes nothing.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP calls h and writes its error, if any.
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		_ = WriteError(w, r, err)
	}
}

// WrapHandler converts a regular http.HandlerFunc to our HandlerFunc
func WrapHandler(h http.HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}

// APIError is the error type of the old {"error":{...}} envelope.
//
// Deprecated: use *Problem (NewProblem, BadRequest, NotFound, Internal, ...).
// Pipelines still write an APIError in the old envelope, with a 5xx reduced to
// its status text, until the type is removed in a later major version.
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

// NewAPIError creates a new API error.
//
// Deprecated: use NewProblem or a status constructor (BadRequest, NotFound,
// Internal, ...). Never pass err.Error() as a message or detail.
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

// NewValidationError creates a new validation error with field details.
//
// Deprecated: use Validation with FieldError values (the "errors" member).
func NewValidationError(message string) *APIError {
	return &APIError{
		Code:    http.StatusBadRequest,
		Message: message,
		Fields:  make(map[string]string),
	}
}

// AddField adds a field error to the APIError and returns the error for chaining.
//
// Deprecated: use Problem.WithErrors.
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

// Common API errors.
//
// Deprecated: use the Problem constructors.
var (
	ErrBadRequest   = &APIError{Code: http.StatusBadRequest, Message: "Bad Request"}
	ErrUnauthorized = &APIError{Code: http.StatusUnauthorized, Message: "Unauthorized"}
	ErrForbidden    = &APIError{Code: http.StatusForbidden, Message: "Forbidden"}
	ErrNotFound     = &APIError{Code: http.StatusNotFound, Message: "Not Found"}
	ErrInternal     = &APIError{Code: http.StatusInternalServerError, Message: "Internal Server Error"}
)
