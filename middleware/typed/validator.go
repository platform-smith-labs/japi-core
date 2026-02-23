// Package typed validator configuration
// This file contains the global validator instance and its configuration
package typed

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Global validator instance for middleware
var validate = validator.New()

func init() {
	// Register a function to use JSON tag names in validation errors
	// This ensures field names in error messages match the JSON API contract
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		jsonTag := fld.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// Extract field name from json tag (before comma)
			name := strings.Split(jsonTag, ",")[0]
			if name != "" {
				return name
			}
		}
		// Fallback to snake_case conversion of field name
		return toSnakeCase(fld.Name)
	})
}

// toSnakeCase converts PascalCase/camelCase to snake_case
func toSnakeCase(str string) string {
	// Insert underscore before uppercase letters that follow lowercase/digits
	reg := regexp.MustCompile("([a-z0-9])([A-Z])")
	str = reg.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(str)
}
