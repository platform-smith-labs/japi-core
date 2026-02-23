package swagger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/swaggo/swag"
	"github.com/platform-smith-labs/japi-core/handler"
)

// SwaggerInfo holds the general Swagger information
var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{"http", "https"},
	Title:            "Junix API",
	Description:      "A high-performance Go API with functional programming patterns and JWT authentication",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

// docTemplate is the base OpenAPI template
const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {},
    "definitions": {},
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header",
            "description": "JWT token with 'Bearer ' prefix"
        }
    }
}`

// GenerateSpec creates an OpenAPI spec from collected routes using reflection
func GenerateSpec() *spec.Swagger {
	swagger := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: "2.0",
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "Junix API",
					Description: "A high-performance Go API with functional programming patterns and JWT authentication",
					Version:     "1.0.0",
				},
			},
			Host:        "localhost:8080",
			BasePath:    "/",
			Schemes:     []string{"http", "https"},
			Paths:       &spec.Paths{Paths: make(map[string]spec.PathItem)},
			Definitions: make(map[string]spec.Schema),
			SecurityDefinitions: map[string]*spec.SecurityScheme{
				"BearerAuth": {
					SecuritySchemeProps: spec.SecuritySchemeProps{
						Type:        "apiKey",
						Name:        "Authorization",
						In:          "header",
						Description: "JWT token with 'Bearer ' prefix",
					},
				},
			},
		},
	}

	// Process collected routes from handler package
	routes := handler.GetCollectedRoutes()

	// Group routes by path to handle multiple HTTP methods for the same path
	routesByPath := make(map[string][]handler.PendingRoute)
	for _, route := range routes {
		routesByPath[route.Path] = append(routesByPath[route.Path], route)
	}

	// Generate PathItems for each unique path, combining all HTTP methods
	for path, pathRoutes := range routesByPath {
		pathItem := generatePathItemFromRoutes(pathRoutes, swagger)
		if pathItem != nil {
			swagger.Paths.Paths[path] = *pathItem
		}
	}

	return swagger
}

// generatePathItemFromRoutes creates a PathItem from multiple routes with the same path
func generatePathItemFromRoutes(routes []handler.PendingRoute, swagger *spec.Swagger) *spec.PathItem {
	pathItem := &spec.PathItem{}

	// Process each route and add its operation to the appropriate HTTP method
	for _, route := range routes {
		operation := generateOperation(route, swagger)
		if operation == nil {
			continue
		}

		// Add operation to appropriate method
		switch strings.ToUpper(route.Method) {
		case "GET":
			pathItem.Get = operation
		case "POST":
			pathItem.Post = operation
		case "PUT":
			pathItem.Put = operation
		case "DELETE":
			pathItem.Delete = operation
		case "PATCH":
			pathItem.Patch = operation
		case "HEAD":
			pathItem.Head = operation
		case "OPTIONS":
			pathItem.Options = operation
		}
	}

	// Return nil if no operations were added
	if pathItem.Get == nil && pathItem.Post == nil && pathItem.Put == nil &&
		pathItem.Delete == nil && pathItem.Patch == nil && pathItem.Head == nil && pathItem.Options == nil {
		return nil
	}

	return pathItem
}

// generatePathItem creates a PathItem from a single route using reflection (legacy function for backward compatibility)
func generatePathItem(route handler.PendingRoute, swagger *spec.Swagger) *spec.PathItem {
	return generatePathItemFromRoutes([]handler.PendingRoute{route}, swagger)
}

// generateOperation creates an Operation from a route using reflection
func generateOperation(route handler.PendingRoute, swagger *spec.Swagger) *spec.Operation {
	operation := &spec.Operation{
		OperationProps: spec.OperationProps{
			Summary:     generateSummary(route),
			Description: generateDescription(route),
			Tags:        generateTags(route),
			Consumes:    []string{"application/json"},
			Produces:    []string{"application/json"},
			Parameters:  []spec.Parameter{},
			Responses:   &spec.Responses{ResponsesProps: spec.ResponsesProps{StatusCodeResponses: make(map[int]spec.Response)}},
		},
	}

	// Extract type information from handler using reflection
	handlerType := reflect.TypeOf(route.Handler)
	if handlerType == nil {
		return operation
	}

	// Try to get type parameters from TypedHandler
	if handlerType.Kind() == reflect.Struct {
		for i := 0; i < handlerType.NumField(); i++ {
			field := handlerType.Field(i)
			if field.Name == "handler" && field.Type.Kind() == reflect.Func {
				funcType := field.Type
				if funcType.NumIn() > 0 {
					// First parameter should be HandlerContext
					contextType := funcType.In(0)
					if contextType.Kind() == reflect.Struct {
						// Extract ParamTypeT and BodyTypeT from HandlerContext
						addParametersFromContext(operation, contextType, swagger)
						addRequestBodyFromContext(operation, contextType, swagger)
					}
				}
				// Extract ResponseBodyT from handler function signature
				addResponseBodyFromHandler(operation, handlerType, swagger)
				break
			}
		}
	}

	// Check for authentication requirement based on middleware
	if requiresAuth(route) {
		operation.Security = []map[string][]string{
			{"BearerAuth": []string{}},
		}
	}

	// Add standard responses
	addStandardResponses(operation, swagger)

	return operation
}

// addParametersFromContext extracts parameters from HandlerContext type
func addParametersFromContext(operation *spec.Operation, contextType reflect.Type, swagger *spec.Swagger) {
	for i := 0; i < contextType.NumField(); i++ {
		field := contextType.Field(i)
		if field.Name == "Params" && field.Type.Kind() == reflect.Struct {
			// Extract from Nullable[ParamTypeT]
			if field.Type.NumField() > 0 {
				paramField := field.Type.Field(0)
				if paramField.Type.Kind() == reflect.Struct && paramField.Type != reflect.TypeOf(struct{}{}) {
					addParametersFromStruct(operation, paramField.Type)
				}
			}
		}
	}
}

// addRequestBodyFromContext extracts request body from HandlerContext type
func addRequestBodyFromContext(operation *spec.Operation, contextType reflect.Type, swagger *spec.Swagger) {
	for i := 0; i < contextType.NumField(); i++ {
		field := contextType.Field(i)
		if field.Name == "Body" && field.Type.Kind() == reflect.Struct {
			// Extract from Nullable[BodyTypeT]
			if field.Type.NumField() > 0 {
				bodyField := field.Type.Field(0)
				if bodyField.Type.Kind() == reflect.Struct && bodyField.Type != reflect.TypeOf(struct{}{}) {
					addRequestBodyFromStruct(operation, bodyField.Type, swagger)
				}
			}
		}
	}
}

// addResponseBodyFromHandler extracts response body type from handler function signature
func addResponseBodyFromHandler(operation *spec.Operation, handlerType reflect.Type, swagger *spec.Swagger) {
	if handlerType.Kind() != reflect.Struct {
		return
	}

	// Look for the handler field in TypedHandler
	for i := 0; i < handlerType.NumField(); i++ {
		field := handlerType.Field(i)
		if field.Name == "handler" && field.Type.Kind() == reflect.Func {
			funcType := field.Type
			// Check if function has return values
			if funcType.NumOut() >= 2 {
				// First return value should be ResponseBodyT, second is error
				responseType := funcType.Out(0)

				// Handle struct types
				if responseType.Kind() == reflect.Struct && responseType != reflect.TypeOf(struct{}{}) {
					addResponseBodyFromStruct(operation, responseType, swagger)
				} else if responseType.Kind() == reflect.Slice || responseType.Kind() == reflect.Array {
					// Handle slice/array types (e.g., []models.User)
					addResponseBodyFromSlice(operation, responseType, swagger)
				}
			}
			break
		}
	}
}

// addParametersFromStruct creates parameters from struct fields with param/query tags
func addParametersFromStruct(operation *spec.Operation, structType reflect.Type) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Handle embedded structs for parameter promotion
		// This ensures param/query tags in embedded structs are promoted to the parent
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			jsonTag := field.Tag.Get("json")
			// Only recurse if there's no json tag (true embedded behavior)
			if jsonTag == "" || jsonTag == "-" {
				addParametersFromStruct(operation, field.Type)
				continue
			}
		}

		// Check for param tag (path parameters)
		if paramTag := field.Tag.Get("param"); paramTag != "" {
			param := spec.Parameter{
				ParamProps: spec.ParamProps{
					Name:        paramTag,
					In:          "path",
					Required:    isRequired(field),
					Description: generateFieldDescription(field),
				},
				SimpleSchema: spec.SimpleSchema{
					Type:   getSwaggerType(field.Type),
					Format: getSwaggerFormat(field.Type),
				},
			}
			operation.Parameters = append(operation.Parameters, param)
		}

		// Check for query tag (query parameters)
		if queryTag := field.Tag.Get("query"); queryTag != "" {
			param := spec.Parameter{
				ParamProps: spec.ParamProps{
					Name:        queryTag,
					In:          "query",
					Required:    isRequired(field),
					Description: generateFieldDescription(field),
				},
				SimpleSchema: spec.SimpleSchema{
					Type:   getSwaggerType(field.Type),
					Format: getSwaggerFormat(field.Type),
				},
			}
			operation.Parameters = append(operation.Parameters, param)
		}
	}
}

// addRequestBodyFromStruct creates request body schema from struct
func addRequestBodyFromStruct(operation *spec.Operation, structType reflect.Type, swagger *spec.Swagger) {
	schemaName := structType.Name()

	// Generate schema definition with nested struct support
	schema := generateSchemaFromStructWithDefinitions(structType, swagger.Definitions)
	swagger.Definitions[schemaName] = *schema

	// Add request body parameter
	param := spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:        "body",
			In:          "body",
			Required:    true,
			Description: fmt.Sprintf("%s request body", schemaName),
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", schemaName)),
				},
			},
		},
	}
	operation.Parameters = append(operation.Parameters, param)
}

// addResponseBodyFromStruct creates response body schema from struct
func addResponseBodyFromStruct(operation *spec.Operation, structType reflect.Type, swagger *spec.Swagger) {
	schemaName := structType.Name()

	// Generate schema definition with nested struct support
	schema := generateSchemaFromStructWithDefinitions(structType, swagger.Definitions)
	swagger.Definitions[schemaName] = *schema

	// Update 200 response with the actual schema
	if operation.Responses == nil {
		operation.Responses = &spec.Responses{ResponsesProps: spec.ResponsesProps{StatusCodeResponses: make(map[int]spec.Response)}}
	}

	operation.Responses.StatusCodeResponses[200] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Success",
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", schemaName)),
				},
			},
		},
	}
}

// addResponseBodyFromSlice creates response body schema for slice/array types
func addResponseBodyFromSlice(operation *spec.Operation, sliceType reflect.Type, swagger *spec.Swagger) {
	// Get the element type from the slice/array
	elementType := sliceType.Elem()

	// Initialize responses if needed
	if operation.Responses == nil {
		operation.Responses = &spec.Responses{ResponsesProps: spec.ResponsesProps{StatusCodeResponses: make(map[int]spec.Response)}}
	}

	// Handle different element types
	var itemSchema *spec.Schema

	if elementType.Kind() == reflect.Struct && elementType != reflect.TypeOf(struct{}{}) {
		// For struct elements (e.g., []models.User), create a reference to the definition
		schemaName := elementType.Name()

		// Generate schema definition for the element type
		schema := generateSchemaFromStructWithDefinitions(elementType, swagger.Definitions)
		swagger.Definitions[schemaName] = *schema

		// Create reference schema for the array items
		itemSchema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", schemaName)),
			},
		}
	} else {
		// For primitive types (e.g., []string, []int), create inline schema
		itemSchema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:   []string{getSwaggerType(elementType)},
				Format: getSwaggerFormat(elementType),
			},
		}
	}

	// Create the array response schema
	operation.Responses.StatusCodeResponses[200] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Success",
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: itemSchema,
					},
				},
			},
		},
	}
}

// generateSchemaFromStruct creates a Swagger schema from a Go struct
func generateSchemaFromStruct(structType reflect.Type) *spec.Schema {
	return generateSchemaFromStructWithDefinitions(structType, nil)
}

// generateSchemaFromStructWithDefinitions creates a Swagger schema from a Go struct with nested definitions support
func generateSchemaFromStructWithDefinitions(structType reflect.Type, definitions map[string]spec.Schema) *spec.Schema {
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
			Required:   []string{},
		},
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Handle embedded/anonymous struct fields (field promotion)
		// This matches Go's JSON marshaling behavior where embedded struct fields are promoted
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			jsonTag := field.Tag.Get("json")

			// Only flatten if there's NO json tag (true embedded behavior)
			// If there's a json tag, treat as a nested object (regular field)
			if jsonTag == "" || jsonTag == "-" {
				// Recursively get schema for the embedded struct
				embeddedSchema := generateSchemaFromStructWithDefinitions(field.Type, definitions)

				// Promote properties to parent schema (parent fields take precedence - shadowing)
				for propName, propSchema := range embeddedSchema.Properties {
					if _, exists := schema.Properties[propName]; !exists {
						schema.Properties[propName] = propSchema
					}
				}

				// Promote required fields from embedded struct
				for _, requiredField := range embeddedSchema.Required {
					alreadyRequired := false
					for _, existing := range schema.Required {
						if existing == requiredField {
							alreadyRequired = true
							break
						}
					}
					if !alreadyRequired {
						schema.Required = append(schema.Required, requiredField)
					}
				}

				// Skip regular field processing for embedded structs
				continue
			}
		}

		// Regular field processing
		jsonTag := field.Tag.Get("json")

		// Skip fields without json tags or with json:"-"
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag to get field name
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = strings.ToLower(field.Name)
		}

		// Create property schema
		propSchema := createPropertySchema(field, definitions)

		schema.Properties[jsonName] = propSchema

		// Add to required if field is required
		if isRequired(field) {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

// createPropertySchema creates a schema for a struct field, handling nested structs
func createPropertySchema(field reflect.StructField, definitions map[string]spec.Schema) spec.Schema {
	fieldType := field.Type

	// Handle special types first (before checking for array, since uuid.UUID is [16]byte)
	if fieldType.String() == "time.Time" || fieldType.String() == "uuid.UUID" {
		propSchema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:        []string{getSwaggerType(fieldType)},
				Format:      getSwaggerFormat(fieldType),
				Description: generateFieldDescription(field),
			},
		}
		addValidationConstraints(&propSchema, field)
		addExampleValue(&propSchema, fieldType)
		return propSchema
	}

	// Handle array/slice types
	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		elementType := fieldType.Elem()

		// Create a temporary field for the element type to get validation constraints
		elementField := reflect.StructField{
			Name: field.Name + "Element",
			Type: elementType,
			Tag:  field.Tag,
		}

		// Create schema for the array items
		var itemSchema spec.Schema

		// Handle struct element types
		if elementType.Kind() == reflect.Struct &&
			elementType.String() != "time.Time" &&
			elementType.String() != "uuid.UUID" {

			schemaName := elementType.Name()

			// Add to definitions if we have a definitions map and the type has a name
			if definitions != nil && schemaName != "" {
				if _, exists := definitions[schemaName]; !exists {
					nestedSchema := generateSchemaFromStructWithDefinitions(elementType, definitions)
					definitions[schemaName] = *nestedSchema
				}

				// Items reference the definition
				itemSchema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", schemaName)),
					},
				}
			} else {
				// Inline the schema
				itemSchema = *generateSchemaFromStructWithDefinitions(elementType, nil)
			}
		} else {
			// Handle primitive element types
			itemSchema = spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type:   []string{getSwaggerType(elementType)},
					Format: getSwaggerFormat(elementType),
				},
			}

			// Add validation constraints to items (e.g., min/max for string length)
			addValidationConstraints(&itemSchema, elementField)
		}

		// Return array schema with items
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:        []string{"array"},
				Description: generateFieldDescription(field),
				Items: &spec.SchemaOrArray{
					Schema: &itemSchema,
				},
			},
		}
	}

	// Handle nested structs (but not special types like time.Time or uuid.UUID)
	if fieldType.Kind() == reflect.Struct &&
		fieldType.String() != "time.Time" &&
		fieldType.String() != "uuid.UUID" {

		schemaName := fieldType.Name()

		// Only add to definitions if we have a definitions map and the type has a name
		if definitions != nil && schemaName != "" {
			// Generate schema for nested struct if not already defined
			if _, exists := definitions[schemaName]; !exists {
				nestedSchema := generateSchemaFromStructWithDefinitions(fieldType, definitions)
				definitions[schemaName] = *nestedSchema
			}

			// Return a reference to the definition
			return spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", schemaName)),
				},
			}
		}

		// Fallback: inline the schema (for when definitions map is not available)
		return *generateSchemaFromStructWithDefinitions(fieldType, nil)
	}

	// Handle primitive types and special structs (time.Time, uuid.UUID)
	propSchema := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:        []string{getSwaggerType(fieldType)},
			Format:      getSwaggerFormat(fieldType),
			Description: generateFieldDescription(field),
		},
	}

	// Add validation constraints from tags
	addValidationConstraints(&propSchema, field)

	// Add example values for better documentation
	addExampleValue(&propSchema, fieldType)

	return propSchema
}

// addValidationConstraints adds validation constraints from struct tags
func addValidationConstraints(schema *spec.Schema, field reflect.StructField) {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return
	}

	// Parse validation rules
	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		if strings.HasPrefix(rule, "min=") {
			if min, err := strconv.ParseInt(strings.TrimPrefix(rule, "min="), 10, 64); err == nil {
				if getSwaggerType(field.Type) == "string" {
					schema.MinLength = &min
				} else {
					minFloat := float64(min)
					schema.Minimum = &minFloat
				}
			}
		}

		if strings.HasPrefix(rule, "max=") {
			if max, err := strconv.ParseInt(strings.TrimPrefix(rule, "max="), 10, 64); err == nil {
				if getSwaggerType(field.Type) == "string" {
					schema.MaxLength = &max
				} else {
					maxFloat := float64(max)
					schema.Maximum = &maxFloat
				}
			}
		}

		if rule == "email" {
			schema.Format = "email"
		}
	}
}

// addExampleValue adds example values to schema properties for better documentation
func addExampleValue(schema *spec.Schema, fieldType reflect.Type) {
	// Only add examples for primitive types and special structs
	switch fieldType.Kind() {
	case reflect.String:
		if fieldType.String() == "time.Time" {
			schema.Example = "2023-12-01T15:30:00Z"
		} else if fieldType.String() == "uuid.UUID" {
			schema.Example = "123e4567-e89b-12d3-a456-426614174000"
		} else if schema.Format == "email" {
			schema.Example = "user@example.com"
		} else {
			schema.Example = "string"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Example = 42
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Example = 42
	case reflect.Float32, reflect.Float64:
		schema.Example = 3.14
	case reflect.Bool:
		schema.Example = true
	case reflect.Struct:
		// Handle special struct types
		if fieldType.String() == "time.Time" {
			schema.Example = "2023-12-01T15:30:00Z"
		} else if fieldType.String() == "uuid.UUID" {
			schema.Example = "123e4567-e89b-12d3-a456-426614174000"
		}
	}
}

// Helper functions

func generateSummary(route handler.PendingRoute) string {
	// Use custom summary if provided
	if route.RouteInfo.Summary != "" {
		return route.RouteInfo.Summary
	}

	// Generate summary from path and method
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	if len(pathParts) > 0 {
		lastPart := pathParts[len(pathParts)-1]
		// Remove parameter placeholders
		re := regexp.MustCompile(`\{[^}]+\}`)
		lastPart = re.ReplaceAllString(lastPart, "")

		verb := getVerbFromMethod(route.Method)
		return fmt.Sprintf("%s %s", verb, strings.Title(lastPart))
	}
	return fmt.Sprintf("%s %s", route.Method, route.Path)
}

func generateDescription(route handler.PendingRoute) string {
	// Use custom description if provided
	if route.RouteInfo.Description != "" {
		return route.RouteInfo.Description
	}
	return fmt.Sprintf("%s endpoint for %s", route.Method, route.Path)
}

func generateTags(route handler.PendingRoute) []string {
	// Use custom tags if provided
	if len(route.RouteInfo.Tags) > 0 {
		return route.RouteInfo.Tags
	}

	// Auto-generate tags from path
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	for _, part := range pathParts {
		// Skip parameters and common prefixes
		if !strings.HasPrefix(part, "{") && part != "api" && part != "v1" {
			return []string{strings.Title(part)}
		}
	}

	return []string{"API"}
}

func generateFieldDescription(field reflect.StructField) string {
	// Try to get description from tag
	if desc := field.Tag.Get("description"); desc != "" {
		return desc
	}
	// Generate from field name
	return fmt.Sprintf("%s field", field.Name)
}

func getVerbFromMethod(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "Get"
	case "POST":
		return "Create"
	case "PUT":
		return "Update"
	case "DELETE":
		return "Delete"
	case "PATCH":
		return "Patch"
	default:
		return strings.Title(strings.ToLower(method))
	}
}

func isRequired(field reflect.StructField) bool {
	validateTag := field.Tag.Get("validate")
	return strings.Contains(validateTag, "required")
}

func getSwaggerType(t reflect.Type) string {
	// Check for special types first (before checking Kind, since uuid.UUID is [16]byte)
	if t.String() == "time.Time" || t.String() == "uuid.UUID" {
		return "string"
	}

	// Handle other struct types
	if t.Kind() == reflect.Struct {
		// All other structs are objects
		return "object"
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Array, reflect.Slice:
		return "array"
	default:
		return "string"
	}
}

func getSwaggerFormat(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	default:
		// Check for time.Time
		if t.String() == "time.Time" {
			return "date-time"
		}
		// Check for uuid.UUID
		if t.String() == "uuid.UUID" {
			return "uuid"
		}
		return ""
	}
}

func requiresAuth(route handler.PendingRoute) bool {
	// Check if RequireAuth middleware is present in the middleware chain
	for _, middlewareName := range route.MiddlewareNames {
		if middlewareName == "RequireAuth" {
			return true
		}
	}
	return false
}

func addStandardResponses(operation *spec.Operation, swagger *spec.Swagger) {
	if operation.Responses == nil {
		operation.Responses = &spec.Responses{ResponsesProps: spec.ResponsesProps{StatusCodeResponses: make(map[int]spec.Response)}}
	}

	// Add generic success response only if no specific 200 response was set
	if _, exists := operation.Responses.StatusCodeResponses[200]; !exists {
		operation.Responses.StatusCodeResponses[200] = spec.Response{
			ResponseProps: spec.ResponseProps{
				Description: "Success",
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
					},
				},
			},
		}
	}

	// Add error responses
	operation.Responses.StatusCodeResponses[400] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Bad Request - Validation Error",
		},
	}

	operation.Responses.StatusCodeResponses[401] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Unauthorized - Invalid or Missing JWT",
		},
	}

	operation.Responses.StatusCodeResponses[403] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Forbidden - User or Company Not Found",
		},
	}

	operation.Responses.StatusCodeResponses[500] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Internal Server Error",
		},
	}
}

// GenerateJSON returns the OpenAPI spec as JSON
func GenerateJSON() ([]byte, error) {
	spec := GenerateSpec()
	return json.MarshalIndent(spec, "", "  ")
}
