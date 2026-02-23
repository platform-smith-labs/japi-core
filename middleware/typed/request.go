// Package typed provides generic typed middleware that works with Handler[ParamTypeT, BodyTypeT, ResponseBodyT].
// These middleware provide type-safe request handling with automatic validation.
// Use: Apply via MakeHandler composition
package typed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/platform-smith-labs/japi-core/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ParseParams extracts and validates URL path parameters and query parameters.
//
// This middleware extracts parameters from the URL path (via chi.URLParam) and query string
// based on struct tags (`param:"name"` for path params, `query:"name"` for query params).
// It performs type conversion and validation using the validator package.
//
// Dependencies: chi.URLParam, validator
// Context modifications: Sets ctx.Params
// Use: Apply via MakeHandler(..., ParseParams, ...)
//
// Example:
//
//	type UserParams struct {
//	    ID   uuid.UUID `param:"id" validate:"required"`
//	    Sort string    `query:"sort"`
//	}
//	handler := MakeHandler(myHandler, ParseParams, ResponseJSON)
func ParseParams[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Check if this handler expects parameters (ParamTypeT is not empty struct{})
		var zero ParamTypeT
		zeroType := reflect.TypeOf(zero)
		expectsParams := zeroType.Kind() != reflect.Struct || zeroType.NumField() > 0

		// If no parameters are expected, set Nil and continue
		if !expectsParams {
			ctx.Params = handler.Nil[ParamTypeT]()
			return next(ctx, w, r)
		}

		var params ParamTypeT
		val := reflect.ValueOf(&params).Elem()
		typ := val.Type()

		// Extract URL path parameters and query parameters based on struct tags
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := typ.Field(i)

			paramTag := fieldType.Tag.Get("param")
			queryTag := fieldType.Tag.Get("query")

			var paramValue string
			var paramName string

			// Handle path parameters
			if paramTag != "" {
				paramValue = chi.URLParam(r, paramTag)
				paramName = paramTag
			} else if queryTag != "" {
				// Handle query parameters
				paramValue = r.URL.Query().Get(queryTag)
				paramName = queryTag
			} else {
				// Skip fields without param or query tags
				continue
			}

			// Check if required parameter is missing
			if paramValue == "" && isRequired(fieldType) {
				var zeroResponse ResponseBodyT
				paramType := "parameter"
				if queryTag != "" {
					paramType = "query parameter"
				}
				return zeroResponse, core.NewAPIError(http.StatusBadRequest,
					"Required "+paramType+" '"+paramName+"' is missing")
			}

			// Convert string parameter to appropriate type
			if err := setFieldValue(field, paramValue); err != nil {
				var zeroResponse ResponseBodyT
				paramType := "parameter"
				if queryTag != "" {
					paramType = "query parameter"
				}
				return zeroResponse, core.NewAPIError(http.StatusBadRequest,
					"Invalid "+paramType+" '"+paramName+"': "+err.Error())
			}
		}

		// Validate the populated struct
		if err := validate.Struct(params); err != nil {
			var zeroResponse ResponseBodyT
			fieldErrors := parseValidationErrors(err)
			validationErr := core.NewValidationError("Parameter validation failed")
			for field, errors := range fieldErrors {
				validationErr.AddField(field, strings.Join(errors, " || "))
			}
			return zeroResponse, validationErr
		}

		// Set validated parameters in context
		ctx.Params = handler.NewNullable(params)
		return next(ctx, w, r)
	}
}

// ParseBody extracts and validates JSON request body.
//
// This middleware decodes the JSON request body and validates it using the validator package.
// It enforces body requirements based on type BodyTypeT - fails fast if body expected but missing.
//
// Dependencies: json decoder, validator
// Context modifications: Sets ctx.Body
// Use: Apply via MakeHandler(..., ParseBody, ...)
//
// Example:
//
//	type CreateUserBody struct {
//	    Email    string `json:"email" validate:"required,email"`
//	    Password string `json:"password" validate:"required,min=8"`
//	}
//	handler := MakeHandler(myHandler, ParseBody, ResponseJSON)
func ParseBody[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Read raw body first if present (before checking if handler expects it)
		var rawBody []byte
		if r.ContentLength > 0 {
			var err error
			rawBody, err = io.ReadAll(r.Body)
			if err != nil {
				var zeroResponse ResponseBodyT
				return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Failed to read request body: "+err.Error())
			}
			// Store raw body in context
			ctx.BodyRaw = handler.NewNullable(rawBody)
		} else {
			// No body provided
			ctx.BodyRaw = handler.Nil[[]byte]()
		}

		// Check if this handler expects a body (BodyTypeT is not empty struct{})
		var zero BodyTypeT
		zeroType := reflect.TypeOf(zero)
		expectsBody := zeroType.Kind() != reflect.Struct || zeroType.NumField() > 0

		// If no body is expected, set Nil and continue
		if !expectsBody {
			ctx.Body = handler.Nil[BodyTypeT]()
			return next(ctx, w, r)
		}

		// Body is expected - ensure it's provided
		if r.ContentLength == 0 {
			var zeroResponse ResponseBodyT
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Request body is required")
		}

		// Parse JSON body from the raw bytes
		var body BodyTypeT
		if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&body); err != nil {
			var zeroResponse ResponseBodyT
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Invalid JSON format: "+err.Error())
		}

		// Validate body structure
		if err := validate.Struct(body); err != nil {
			var zeroResponse ResponseBodyT
			fieldErrors := parseValidationErrors(err)
			validationErr := core.NewValidationError("Validation failed")
			for field, errors := range fieldErrors {
				validationErr.AddField(field, strings.Join(errors, " || "))
			}
			return zeroResponse, validationErr
		}

		// Set validated body in context
		ctx.Body = handler.NewNullable(body)
		return next(ctx, w, r)
	}
}

// ParseHeaders captures all HTTP request headers.
//
// This middleware extracts all headers from the HTTP request and stores them in the context.
// Headers are stored as http.Header type, which supports multiple values per header key.
//
// Dependencies: None
// Context modifications: Sets ctx.Headers
// Use: Apply via MakeHandler(..., ParseHeaders, ...)
//
// Example:
//
//	handler := MakeHandler(myHandler, ParseHeaders, ResponseJSON)
//	// In handler:
//	contentType := ctx.Headers.Value().Get("Content-Type")
//	authHeader := ctx.Headers.Value().Get("Authorization")
func ParseHeaders[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Store all request headers in context
		ctx.Headers = handler.NewNullable(r.Header)
		return next(ctx, w, r)
	}
}

// Helper functions

// isRequired checks if a field is marked as required in validation tags
func isRequired(field reflect.StructField) bool {
	validateTag := field.Tag.Get("validate")
	return strings.Contains(validateTag, "required")
}

// setFieldValue sets the field value from string parameter
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			field.SetInt(0)
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer: %s", value)
			}
			field.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			field.SetUint(0)
		} else {
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer: %s", value)
			}
			field.SetUint(uintVal)
		}
	case reflect.Float32, reflect.Float64:
		if value == "" {
			field.SetFloat(0)
		} else {
			floatVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid float: %s", value)
			}
			field.SetFloat(floatVal)
		}
	case reflect.Bool:
		if value == "" {
			field.SetBool(false)
		} else {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid boolean: %s", value)
			}
			field.SetBool(boolVal)
		}
	case reflect.Array:
		// Handle uuid.UUID which is internally [16]byte
		if field.Type().String() == "uuid.UUID" {
			if value == "" {
				// Set zero UUID for empty values
				var zeroUUID uuid.UUID
				field.Set(reflect.ValueOf(zeroUUID))
			} else {
				parsedUUID, err := uuid.Parse(value)
				if err != nil {
					return fmt.Errorf("invalid UUID format: %s", value)
				}
				field.Set(reflect.ValueOf(parsedUUID))
			}
		} else {
			return fmt.Errorf("unsupported array type: %s", field.Type())
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// parseValidationErrors converts validator.ValidationErrors to a structured field map
func parseValidationErrors(err error) map[string][]string {
	fieldErrors := make(map[string][]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			// Convert field name to lowercase for consistent JSON keys
			fieldName := strings.ToLower(fieldError.Field())

			// Remove struct name prefix if present (e.g., "CreateUserRequest.Password" -> "password")
			if dotIndex := strings.LastIndex(fieldName, "."); dotIndex != -1 {
				fieldName = fieldName[dotIndex+1:]
			}

			// Generate user-friendly error message
			message := generateFieldErrorMessage(fieldError)

			// Append error message to field (supports multiple errors per field)
			fieldErrors[fieldName] = append(fieldErrors[fieldName], message)
		}
	}

	return fieldErrors
}

// generateFieldErrorMessage converts validator field error to user-friendly message
func generateFieldErrorMessage(fieldError validator.FieldError) string {
	fieldName := fieldError.Field()
	tag := fieldError.Tag()
	param := fieldError.Param()

	// Remove struct name prefix for display
	if dotIndex := strings.LastIndex(fieldName, "."); dotIndex != -1 {
		fieldName = fieldName[dotIndex+1:]
	}

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "min":
		if fieldError.Kind().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters", fieldName, param)
		}
		return fmt.Sprintf("%s must be at least %s", fieldName, param)
	case "max":
		if fieldError.Kind().String() == "string" {
			return fmt.Sprintf("%s must be at most %s characters", fieldName, param)
		}
		return fmt.Sprintf("%s must be at most %s", fieldName, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldName)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fieldName)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fieldName)
	case "eqfield":
		return fmt.Sprintf("%s must match %s", fieldName, param)
	default:
		// Fallback for unknown tags
		return fmt.Sprintf("%s validation failed on '%s' tag", fieldName, tag)
	}
}
