package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/platform-smith-labs/japi-core/v3/core"
)

// RouterOption is a functional option that configures the Chi router's CORS settings.
// Use the With* functions to construct options; pass them to NewChiRouterWithOptions.
type RouterOption func(*routerConfig)

// routerConfig holds CORS configuration used during router construction.
// It is unexported — callers interact only via RouterOption functions.
type routerConfig struct {
	allowedOrigins   []string
	allowedMethods   []string
	allowedHeaders   []string
	exposedHeaders   []string
	allowCredentials bool
	maxAge           int
}

// defaultRouterConfig returns the secure baseline CORS configuration.
// AllowedOrigins is empty (deny-all) by default — callers must explicitly
// set origins via WithAllowedOrigins.
func defaultRouterConfig() routerConfig {
	return routerConfig{
		allowedOrigins:   []string{},
		allowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		allowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		exposedHeaders:   []string{"Link"},
		allowCredentials: false,
		maxAge:           300,
	}
}

// WithAllowedOrigins sets the list of origins permitted to make cross-origin requests.
// Pass an empty slice to deny all origins (the secure default).
func WithAllowedOrigins(origins []string) RouterOption {
	return func(cfg *routerConfig) { cfg.allowedOrigins = origins }
}

// WithAllowedMethods replaces the default list of HTTP methods permitted in CORS requests.
// The default list is: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
func WithAllowedMethods(methods []string) RouterOption {
	return func(cfg *routerConfig) { cfg.allowedMethods = methods }
}

// WithAllowedHeaders replaces the default list of request headers permitted in CORS requests.
// The default list is: Accept, Authorization, Content-Type, X-CSRF-Token.
func WithAllowedHeaders(headers []string) RouterOption {
	return func(cfg *routerConfig) { cfg.allowedHeaders = headers }
}

// WithExposedHeaders replaces the default list of response headers exposed to the browser.
// The default list is: Link.
func WithExposedHeaders(headers []string) RouterOption {
	return func(cfg *routerConfig) { cfg.exposedHeaders = headers }
}

// WithAllowCredentials controls whether the browser may send credentials
// (cookies, HTTP authentication) with cross-origin requests.
// Defaults to false. Do not set to true when AllowedOrigins contains "*".
func WithAllowCredentials(allow bool) RouterOption {
	return func(cfg *routerConfig) { cfg.allowCredentials = allow }
}

// WithMaxAge sets the duration (in seconds) the browser may cache preflight results.
// Defaults to 300 (5 minutes).
func WithMaxAge(seconds int) RouterOption {
	return func(cfg *routerConfig) { cfg.maxAge = seconds }
}

// newChiRouter is the single internal constructor all public constructors delegate to.
// It applies defaults then each option in order, constructs the chi router, and attaches
// the standard middleware stack and CORS handler exactly once.
func newChiRouter(opts ...RouterOption) chi.Router {
	cfg := defaultRouterConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.allowedOrigins,
		AllowedMethods:   cfg.allowedMethods,
		AllowedHeaders:   cfg.allowedHeaders,
		ExposedHeaders:   cfg.exposedHeaders,
		AllowCredentials: cfg.allowCredentials,
		MaxAge:           cfg.maxAge,
	}))
	return r
}

// NewChiRouter creates a Chi router with the secure default CORS configuration.
//
// SECURITY WARNING: AllowedOrigins defaults to empty (deny all cross-origin requests).
// This is a secure default. To accept cross-origin requests use NewChiRouterWithCORS
// or NewChiRouterWithOptions.
func NewChiRouter() chi.Router {
	return newChiRouter()
}

// NewChiRouterWithCORS creates a Chi router that permits cross-origin requests
// from the specified origins. All other CORS settings use secure defaults,
// including AllowedMethods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
//
// Example:
//
//	r := router.NewChiRouterWithCORS([]string{"https://app.example.com"})
//
// WARNING: Never use []string{"*"} in production — it allows any origin.
func NewChiRouterWithCORS(allowedOrigins []string) chi.Router {
	return newChiRouter(WithAllowedOrigins(allowedOrigins))
}

// NewChiRouterWithOptions creates a Chi router with fully customisable CORS configuration.
// Apply any combination of With* options; unspecified settings retain secure defaults.
//
// Example — restrict to specific origins and a reduced method set:
//
//	r := router.NewChiRouterWithOptions(
//	    router.WithAllowedOrigins([]string{"https://app.example.com"}),
//	    router.WithAllowedMethods([]string{"GET", "POST", "PATCH"}),
//	)
//
// Example — custom request headers:
//
//	r := router.NewChiRouterWithOptions(
//	    router.WithAllowedOrigins([]string{"https://app.example.com"}),
//	    router.WithAllowedHeaders([]string{"Authorization", "Content-Type", "X-Request-ID"}),
//	)
func NewChiRouterWithOptions(opts ...RouterOption) chi.Router {
	return newChiRouter(opts...)
}

// AdaptErrorHandler adapts a core.HandlerFunc to work with Chi.
func AdaptErrorHandler(handler core.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			if apiErr, ok := err.(*core.APIError); ok {
				core.WriteAPIError(w, r, *apiErr)
			} else {
				core.Error(w, r, http.StatusInternalServerError, "Internal Server Error")
			}
		}
	}
}
