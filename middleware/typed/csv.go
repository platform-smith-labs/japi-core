package typed

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/handler"
	"github.com/gocarina/gocsv"
)

// ParseCSV extracts and validates CSV file from multipart form data.
//
// This middleware parses a CSV file uploaded via multipart form data (field name: "file"),
// unmarshals it into the BodyTypeT (which should be a slice type), and validates each row.
//
// Dependencies: gocsv, validator, multipart form parser
// Context modifications: Sets ctx.Body with parsed CSV data
// Use: Apply via MakeHandler(..., ParseCSV, ...)
//
// Example:
//
//	type CSVRow struct {
//	    Name  string `csv:"name" validate:"required"`
//	    Email string `csv:"email" validate:"required,email"`
//	}
//	// BodyTypeT should be []CSVRow
//	handler := MakeHandler(importHandler, ParseCSV, ResponseJSON)
func ParseCSV[ParamTypeT any, BodyTypeT any, ResponseBodyT any](next handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) handler.Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
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

		// Validate file type (CSV)
		if !isCSVFile(fileHeader) {
			return zeroResponse, core.NewProblem(core.CodeUnsupportedMediaType, http.StatusUnsupportedMediaType, "The file must be a CSV file (.csv).")
		}

		// Parse CSV file directly using gocsv
		var csvRows BodyTypeT
		err = gocsv.Unmarshal(file, &csvRows)
		if err != nil {
			// The parser's message can quote the file: log only.
			return zeroResponse, core.BadRequest(core.CodeInvalidCSV, "The file is not valid CSV for this import.").WithCause(err)
		}

		// Use reflection to check length and validate rows
		csvValue := reflect.ValueOf(csvRows)
		if csvValue.Len() == 0 {
			return zeroResponse, core.BadRequest(core.CodeInvalidCSV, "The CSV file has no data rows.")
		}

		// Validate CSV rows using the global validator instance
		for i := 0; i < csvValue.Len(); i++ {
			row := csvValue.Index(i).Interface()
			if err := validate.Struct(row); err != nil {
				// One entry per failed rule, prefixed with the CSV line
				// (header is line 1). CSV has no JSON Pointer.
				errs := fieldErrors(err, false)
				for j := range errs {
					errs[j].Detail = fmt.Sprintf("line %d: %s", i+2, errs[j].Detail)
				}
				return zeroResponse, core.Validation(errs...).WithCause(err)
			}
		}

		// Set parsed CSV data in context
		ctx.Body = handler.NewNullable(csvRows)

		return next(ctx, w, r)
	}
}

// isCSVFile checks if the uploaded file is a CSV file
func isCSVFile(fileHeader *multipart.FileHeader) bool {
	return strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv")
}
