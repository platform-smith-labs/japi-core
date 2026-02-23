package router

import (
	"net/http"

	"github.com/platform-smith-labs/japi-core/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewChiRouter creates a new Chi router with standard middleware.
//
// SECURITY WARNING: CORS is configured to DENY ALL origins by default.
// This is a secure default. You MUST explicitly configure allowed origins
// for your application to accept cross-origin requests.
//
// To configure CORS, use NewChiRouterWithCORS() or manually add CORS middleware:
//
//	r := router.NewChiRouterWithCORS([]string{"https://yourdomain.com"})
//
// Or after router creation:
//
//	r := router.NewChiRouter()
//	r.Use(cors.Handler(cors.Options{
//	    AllowedOrigins: []string{"https://yourdomain.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
//	}))
func NewChiRouter() chi.Router {
	r := chi.NewRouter()

	// Chi built-in middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// CORS middleware - SECURE DEFAULT: Deny all origins
	// Applications MUST explicitly configure allowed origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{}, // Empty = deny all (secure default)
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Custom middleware can be added using middleware/http package
	// r.Use(httpMiddleware.WithLogging(logger))
	// r.Use(httpMiddleware.WithContentType("application/json"))

	return r
}

// NewChiRouterWithCORS creates a new Chi router with custom CORS configuration.
//
// This is a convenience function that creates a router with explicitly allowed origins.
// Use this when you need to accept cross-origin requests from specific domains.
//
// Example:
//
//	r := router.NewChiRouterWithCORS([]string{
//	    "https://yourdomain.com",
//	    "https://app.yourdomain.com",
//	})
//
// For local development, you might use:
//
//	r := router.NewChiRouterWithCORS([]string{"http://localhost:3000"})
//
// WARNING: Never use []string{"*"} in production as it allows any origin.
func NewChiRouterWithCORS(allowedOrigins []string) chi.Router {
	r := chi.NewRouter()

	// Chi built-in middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// CORS middleware with custom allowed origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	return r
}

// AdaptErrorHandler adapts a core.HandlerFunc to work with Chi
func AdaptErrorHandler(handler core.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			// Handle the error appropriately
			if apiErr, ok := err.(*core.APIError); ok {
				core.WriteAPIError(w, r, *apiErr)
			} else {
				core.Error(w, r, http.StatusInternalServerError, "Internal Server Error")
			}
		}
	}
}
