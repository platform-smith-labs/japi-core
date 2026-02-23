package handler

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// HandlerContext contains application dependencies and request-scoped data
// ParamTypeT represents the type of parameters (URL/query params)
// BodyTypeT represents the type of request body
type HandlerContext[ParamTypeT any, BodyTypeT any] struct {
	// Request context (propagated from r.Context())
	// Used for cancellation, timeouts, and trace propagation
	Context context.Context

	// Application dependencies
	DB     *sql.DB
	Logger *slog.Logger

	// Request-scoped data
	Params  Nullable[ParamTypeT] // Optional parameters from URL/query
	Body    Nullable[BodyTypeT]  // Optional request body
	BodyRaw Nullable[[]byte]     // Raw request body bytes
	Headers Nullable[http.Header] // HTTP request headers

	// Authentication data (set by RequireAuth middleware)
	UserUUID    Nullable[uuid.UUID] // Authenticated user UUID from JWT
	CompanyUUID Nullable[uuid.UUID] // Authenticated company UUID from JWT
}

// Handler represents a generic handler function that receives typed context and returns response data
type Handler[ParamTypeT any, BodyTypeT any, ResponseBodyT any] func(ctx HandlerContext[ParamTypeT, BodyTypeT], w http.ResponseWriter, r *http.Request) (ResponseBodyT, error)

// Middleware represents a function that wraps a Handler and can enrich the context
type Middleware[ParamTypeT any, BodyTypeT any, ResponseBodyT any] func(Handler[ParamTypeT, BodyTypeT, ResponseBodyT]) Handler[ParamTypeT, BodyTypeT, ResponseBodyT]

// RouteInfo holds route metadata for automatic registration
type RouteInfo struct {
	Method      string   // HTTP method (GET, POST, PUT, DELETE, etc.)
	Path        string   // Route path pattern
	Summary     string   // Optional: Brief description for Swagger (auto-generated if empty)
	Description string   // Optional: Detailed description for Swagger (auto-generated if empty)
	Tags        []string // Optional: Tags for grouping in Swagger UI
}

// AdaptableHandler interface knows how to create an adapted http.HandlerFunc
type AdaptableHandler interface {
	Adapt(database *sql.DB, logger *slog.Logger) http.HandlerFunc
}

// TypedHandler wraps any Handler type and implements AdaptableHandler
type TypedHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any] struct {
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT]
}

// Adapt converts the typed handler to http.HandlerFunc using AdaptHandler
func (th TypedHandler[ParamTypeT, BodyTypeT, ResponseBodyT]) Adapt(database *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return AdaptHandler(database, logger, th.handler)
}

// PendingRoute stores route information for handlers that need to be registered later
type PendingRoute struct {
	Method          string
	Path            string
	Handler         AdaptableHandler // Interface that knows how to adapt itself
	RouteInfo       RouteInfo        // Complete route metadata for documentation
	MiddlewareNames []string         // Names of middleware functions applied to this route
}

// Global route collection
var (
	globalRoutes = make([]PendingRoute, 0)
	routesMutex  sync.RWMutex
)

// MakeHandler creates a handler with automatic route registration and middleware composition
// Usage: MakeHandler(RouteInfo{Method: "POST", Path: "/api/v1/endpoint"}, baseHandler, middleware...)
// Execution order: last middleware -> ... -> first middleware -> baseHandler
func MakeHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	routeInfo RouteInfo,
	baseHandler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
	middleware ...Middleware[ParamTypeT, BodyTypeT, ResponseBodyT],
) Handler[ParamTypeT, BodyTypeT, ResponseBodyT] {
	handler := baseHandler

	// Extract middleware names for documentation
	middlewareNames := make([]string, len(middleware))
	for i, mw := range middleware {
		middlewareNames[i] = getMiddlewareName(mw)
	}

	// Apply middleware in reverse order so the last one executes first
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	// Wrap the fully composed handler in TypedHandler and register with route information
	routesMutex.Lock()
	globalRoutes = append(globalRoutes, PendingRoute{
		Method:          routeInfo.Method,
		Path:            routeInfo.Path,
		Handler:         TypedHandler[ParamTypeT, BodyTypeT, ResponseBodyT]{handler: handler},
		RouteInfo:       routeInfo,
		MiddlewareNames: middlewareNames,
	})
	routesMutex.Unlock()

	return handler
}

// RegisterCollectedRoutes processes all collected routes and registers them with the chi router
func RegisterCollectedRoutes(r chi.Router, database *sql.DB, logger *slog.Logger) {
	routesMutex.RLock()
	defer routesMutex.RUnlock()

	for _, route := range globalRoutes {
		// Use interface method to adapt handler - no type assertions needed!
		adaptedHandler := route.Handler.Adapt(database, logger)
		registerRoute(r, route.Method, route.Path, adaptedHandler)
	}
}

// GetCollectedRoutes returns a copy of all collected routes for reflection/documentation
func GetCollectedRoutes() []PendingRoute {
	routesMutex.RLock()
	defer routesMutex.RUnlock()

	// Return a copy to prevent external modifications
	routes := make([]PendingRoute, len(globalRoutes))
	copy(routes, globalRoutes)
	return routes
}

// registerRoute helper function to reduce code duplication
func registerRoute(r chi.Router, method, path string, handler http.HandlerFunc) {
	switch method {
	case "GET":
		r.Get(path, handler)
	case "POST":
		r.Post(path, handler)
	case "PUT":
		r.Put(path, handler)
	case "DELETE":
		r.Delete(path, handler)
	case "PATCH":
		r.Patch(path, handler)
	case "HEAD":
		r.Head(path, handler)
	case "OPTIONS":
		r.Options(path, handler)
	}
}

// getMiddlewareName extracts the function name from a middleware function using reflection
func getMiddlewareName[ParamTypeT any, BodyTypeT any, ResponseBodyT any](middleware Middleware[ParamTypeT, BodyTypeT, ResponseBodyT]) string {
	// Get the function value using reflection
	middlewareValue := reflect.ValueOf(middleware)

	// Get the runtime function pointer and its name
	middlewarePtr := middlewareValue.Pointer()
	funcForPC := runtime.FuncForPC(middlewarePtr)
	if funcForPC == nil {
		return "unknown"
	}

	// Get the full function name (e.g., "japi-core/handler.RequireAuth")
	fullName := funcForPC.Name()

	// Extract function name from the full path
	// For generic functions, the format is: package.path.FunctionName[generics...]
	// Use regex to extract the function name before the generic brackets
	re := regexp.MustCompile(`\.([A-Za-z_][A-Za-z0-9_]*)\[`)
	matches := re.FindStringSubmatch(fullName)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback: try standard parsing for non-generic functions
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		lastName := parts[len(parts)-1]
		// Remove any generic type information (e.g., "RequireAuth[...]")
		if bracketIndex := strings.Index(lastName, "["); bracketIndex != -1 {
			lastName = lastName[:bracketIndex]
		}
		if lastName != "" && lastName != "]" {
			return lastName
		}
	}

	return "unknown"
}
