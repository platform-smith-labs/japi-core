package typed

import (
	"net/http"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/platform-smith-labs/japi-core/handler"
)

// ResponseJSON handles writing successful responses as JSON.
//
// This middleware should be the first in the chain (executes last) to handle response formatting.
// It automatically determines the appropriate status code based on the HTTP method
// (201 for POST, 200 for others) and writes the response as JSON.
//
// Dependencies: core.JSON
// Context modifications: None
// Use: Apply via MakeHandler(myHandler, ParseParams, ResponseJSON)
//
// Example:
//
//	handler := MakeHandler(createUserHandler, ParseBody, ResponseJSON)
func ResponseJSON[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
		// Execute the handler
		responseData, err := next(ctx, w, r)
		if err != nil {
			// Don't handle errors here - let the adapter handle them
			return responseData, err
		}

		// Determine appropriate status code based on HTTP method
		var statusCode int
		switch r.Method {
		case "POST":
			statusCode = 201 // Created
		default:
			statusCode = 200 // OK
		}

		// Write successful JSON response
		if err := core.JSON(w, statusCode, responseData); err != nil {
			ctx.Logger.Error("Failed to write JSON response", "error", err.Error(), "path", r.URL.Path)
			return responseData, core.NewAPIError(http.StatusInternalServerError, "Failed to write response")
		}

		return responseData, nil
	}
}

// ResponseJSONFile handles writing successful responses as a downloadable JSON file.
//
// This middleware is similar to ResponseJSON but triggers a file download in the browser.
// It accepts a filename string for the downloaded file.
//
// Dependencies: core.JSONFile
// Context modifications: None
// Use: Apply via MakeHandler(myHandler, ParseParams, ResponseJSONFile("export.json"))
//
// Example:
//
//	handler := MakeHandler(exportHandler, ParseParams, ResponseJSONFile("llm_workflow_export.json"))
func ResponseJSONFile[ParamTypeT any, BodyTypeT any, ResponseBodyT any](filename string) func(next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	return func(next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
		return func(ctx handler.HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error) {
			// Execute the handler
			responseData, err := next(ctx, w, r)
			if err != nil {
				// Don't handle errors here - let the adapter handle them
				return responseData, err
			}

			// Determine appropriate status code based on HTTP method
			var statusCode int
			switch r.Method {
			case "POST":
				statusCode = 201 // Created
			default:
				statusCode = 200 // OK
			}

			// Write successful JSON file response
			if err := core.JSONFile(w, statusCode, responseData, filename); err != nil {
				ctx.Logger.Error("Failed to write JSON file response", "error", err.Error(), "path", r.URL.Path, "filename", filename)
				return responseData, core.NewAPIError(http.StatusInternalServerError, "Failed to write response")
			}

			return responseData, nil
		}
	}
}
