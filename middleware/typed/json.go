package typed

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/platform-smith-labs/japi-core/handler"
)

// ParseJSON extracts and validates JSON file from multipart form data.
//
// This middleware parses a JSON file uploaded via multipart form data (field name: "file"),
// unmarshals it into the BodyTypeT, and validates the parsed data.
//
// Dependencies: encoding/json, validator, multipart form parser
// Context modifications: Sets ctx.Body with parsed JSON data
// Use: Apply via MakeHandler(..., ParseJSON, ...)
//
// Example:
//
//	type ImportData struct {
//	    Name  string `json:"name" validate:"required"`
//	    Email string `json:"email" validate:"required,email"`
//	}
//	// BodyTypeT should be ImportData or []ImportData
//	handler := MakeHandler(importHandler, ParseJSON, ResponseJSON)
func ParseJSON[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		var zeroResponse ResponseBodyT

		// Parse multipart form data (32MB max)
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Failed to parse multipart form", err.Error())
		}

		// Get the uploaded file
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Missing or invalid 'file' field in form data")
		}
		defer file.Close()

		// Validate file type (JSON)
		if !isJSONFile(fileHeader) {
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "File must be a JSON file (.json)")
		}

		// Parse JSON file
		var jsonData BodyTypeT
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&jsonData)
		if err != nil {
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "Failed to parse JSON file", err.Error())
		}

		// Validate parsed JSON data using the global validator instance
		if err := validate.Struct(jsonData); err != nil {
			return zeroResponse, core.NewAPIError(http.StatusBadRequest, "JSON validation failed", err.Error())
		}

		// Set parsed JSON data in context
		ctx.Body = handler.NewNullable(jsonData)

		return next(ctx, w, r)
	}
}

// isJSONFile checks if the uploaded file is a JSON file
func isJSONFile(fileHeader *multipart.FileHeader) bool {
	return strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".json")
}
