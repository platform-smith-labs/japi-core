package swagger

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupSwaggerUI registers Swagger documentation routes on the provided router.
// It creates two endpoints:
//   - GET /swagger.json - Returns the OpenAPI specification as JSON
//   - GET /swagger/* - Serves the interactive Swagger UI
//
// Example usage:
//
//	r := chi.NewRouter()
//	swagger.SetupSwaggerUI(r)
func SetupSwaggerUI(r chi.Router) {
	SetupSwaggerUIWithPath(r, "")
}

// SetupSwaggerUIWithPath registers Swagger documentation routes on the provided router
// with a custom base path prefix.
// It creates two endpoints:
//   - GET {basePath}/swagger.json - Returns the OpenAPI specification as JSON
//   - GET {basePath}/swagger/* - Serves the interactive Swagger UI
//
// Example usage:
//
//	r := chi.NewRouter()
//	swagger.SetupSwaggerUIWithPath(r, "/api/docs") // Routes: /api/docs/swagger.json, /api/docs/swagger/*
func SetupSwaggerUIWithPath(r chi.Router, basePath string) {
	// Normalize basePath: remove trailing slash to prevent double slashes
	basePath = strings.TrimSuffix(basePath, "/")

	// Swagger JSON endpoint
	r.Get(basePath+"/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		spec, err := GenerateJSON()
		if err != nil {
			http.Error(w, "Failed to generate API specification", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(spec)
	})

	// Swagger UI
	r.Get(basePath+"/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(basePath+"/swagger.json"), // Point to our custom JSON endpoint
	))
}
