package typed

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/handler"
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
			return zeroResponse, multipartProblem(err)
		}

		// Get the uploaded file
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			return zeroResponse, core.BadRequest(core.CodeBadRequest, "The form must carry the upload in a 'file' field.").WithCause(err)
		}
		defer file.Close()

		// Validate file type (JSON)
		if !isJSONFile(fileHeader) {
			return zeroResponse, core.NewProblem(core.CodeUnsupportedMediaType, http.StatusUnsupportedMediaType, "The file must be a JSON file (.json).")
		}

		// Parse JSON file
		var jsonData BodyTypeT
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&jsonData)
		if err != nil {
			// The decoder's message can quote the file: log only.
			return zeroResponse, core.BadRequest(core.CodeInvalidJSON, "The file is not valid JSON.").WithCause(err)
		}

		// Validate parsed JSON data using the global validator instance
		if err := validate.Struct(jsonData); err != nil {
			return zeroResponse, core.Validation(fieldErrors(err, true)...).WithCause(err)
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

// multipartProblem classifies a ParseMultipartForm failure with a fixed
// detail; the parser's own text goes to the log only.
func multipartProblem(err error) *core.Problem {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || errors.Is(err, multipart.ErrMessageTooLarge) {
		return core.NewProblem(core.CodeFileTooLarge, http.StatusRequestEntityTooLarge,
			"The upload is too large.").WithCause(err)
	}
	if errors.Is(err, http.ErrNotMultipart) {
		return core.NewProblem(core.CodeUnsupportedMediaType, http.StatusUnsupportedMediaType,
			"The request must be multipart/form-data.").WithCause(err)
	}
	return core.BadRequest(core.CodeBadRequest, "The request is not valid multipart form data.").WithCause(err)
}
