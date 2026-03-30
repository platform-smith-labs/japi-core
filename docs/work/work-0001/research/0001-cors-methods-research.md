# Research: CORS AllowedMethods Fix & Override API

**Work Item**: work-0001
**Created**: 2026-03-30
**Updated**: 2026-03-30 — override mechanism changed to functional options pattern

---

## Executive Summary

PATCH requests are blocked by CORS because `router/chi.go` hardcodes `AllowedMethods` without "PATCH" (or "HEAD") in both router constructors. The router, handler, and swagger layers already fully support PATCH and HEAD — the gap is exclusively in the CORS middleware configuration. A functional options pattern will be used to allow consuming apps to customize CORS behaviour, while `NewChiRouter()` and `NewChiRouterWithCORS()` become thin wrappers over the new unified constructor.

---

## 1. Root Cause

**File**: `router/chi.go:41` and `router/chi.go:83`

Both `NewChiRouter()` and `NewChiRouterWithCORS()` hardcode:
```go
AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
```
PATCH and HEAD are absent.

**CORS preflight flow**:
1. Browser sends `OPTIONS` preflight before `PATCH`
2. CORS middleware responds: `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
3. Browser sees PATCH is not listed → blocks the actual request with `net::ERR_FAILED`
4. The application never receives the request

---

## 2. Method Support Matrix

| Method  | chi Router | CORS Default | `registerRoute()` | Swagger | Gap      |
|---------|-----------|--------------|-------------------|---------|----------|
| GET     | ✅        | ✅           | ✅                | ✅      | —        |
| POST    | ✅        | ✅           | ✅                | ✅      | —        |
| PUT     | ✅        | ✅           | ✅                | ✅      | —        |
| DELETE  | ✅        | ✅           | ✅                | ✅      | —        |
| PATCH   | ✅        | ❌ MISSING   | ✅                | ✅      | **CORS** |
| HEAD    | ✅        | ❌ MISSING   | ✅                | ✅      | **CORS** |
| OPTIONS | ✅        | ✅           | ✅                | ✅      | —        |

Routing (`handler/types.go:154-171`), swagger (`swagger/generator.go:115-130`), and chi all support PATCH and HEAD. Only the CORS middleware is missing them.

---

## 3. Affected Locations

| File | Lines | Issue |
|------|-------|-------|
| `router/chi.go` | 41 | `NewChiRouter()` — missing PATCH, HEAD |
| `router/chi.go` | 83 | `NewChiRouterWithCORS()` — missing PATCH, HEAD |
| `README.md` | 1168 | Example also shows the broken list (doc bug) |

---

## 4. Current Public API Surface

```go
// router/chi.go
func NewChiRouter() chi.Router
func NewChiRouterWithCORS(allowedOrigins []string) chi.Router
func AdaptErrorHandler(handler core.HandlerFunc) http.HandlerFunc
```

Both constructors are part of the public API used by consuming applications. Their signatures must not change.

The two router constructors are structurally identical except for the `AllowedOrigins` value. This duplication is the motivation for the functional options refactor.

---

## 5. Chosen Design: Functional Options Pattern

### Rationale

The `CORSConfig` struct approach (Option D in the original analysis) would work but leaves code duplication in place and exposes the full CORS surface as a flat struct. The **functional options pattern** is the better long-term choice because:

- Single source of defaults (`defaultCORSOptions`)
- `NewChiRouter()` and `NewChiRouterWithCORS()` become one-line wrappers — no duplication
- Callers compose only what they need
- New options (e.g. `WithMaxAge`, `WithAllowedHeaders`) can be added without changing any existing signature
- Consistent with the codebase's existing higher-order function style (middleware stack, `MakeHandler`)

### API Design

```go
// RouterOption is a functional option for configuring the Chi router.
type RouterOption func(*routerConfig)

// routerConfig holds internal CORS configuration; not exported.
type routerConfig struct {
    allowedOrigins   []string
    allowedMethods   []string
    allowedHeaders   []string
    exposedHeaders   []string
    allowCredentials bool
    maxAge           int
}

// defaultRouterConfig returns the secure defaults applied when no options override them.
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

// WithAllowedOrigins sets the CORS allowed origins.
func WithAllowedOrigins(origins []string) RouterOption

// WithAllowedMethods replaces the default allowed methods list.
func WithAllowedMethods(methods []string) RouterOption

// WithAllowedHeaders replaces the default allowed headers list.
func WithAllowedHeaders(headers []string) RouterOption

// WithExposedHeaders replaces the default exposed headers list.
func WithExposedHeaders(headers []string) RouterOption

// WithAllowCredentials enables or disables credential support.
func WithAllowCredentials(allow bool) RouterOption

// WithMaxAge sets the preflight cache duration in seconds.
func WithMaxAge(seconds int) RouterOption

// newChiRouter is the single internal constructor all public functions delegate to.
func newChiRouter(opts ...RouterOption) chi.Router

// Public API — signatures unchanged:
func NewChiRouter() chi.Router {
    return newChiRouter() // uses all defaults
}

func NewChiRouterWithCORS(allowedOrigins []string) chi.Router {
    return newChiRouter(WithAllowedOrigins(allowedOrigins))
}

// New public constructor for full customisation:
func NewChiRouterWithOptions(opts ...RouterOption) chi.Router {
    return newChiRouter(opts...)
}
```

### Usage Examples

```go
// Existing usage — unchanged, zero migration cost:
r := router.NewChiRouter()
r := router.NewChiRouterWithCORS([]string{"https://app.example.com"})

// New: custom methods only
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
    router.WithAllowedMethods([]string{"GET", "POST", "PATCH"}),
)

// New: start from defaults, add one origin
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
)
```

### Internal structure

```
newChiRouter(opts ...RouterOption) chi.Router
    cfg := defaultRouterConfig()
    for _, opt := range opts { opt(&cfg) }
    r := chi.NewRouter()
    r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
    r.Use(cors.Handler(cors.Options{ ...cfg fields... }))
    return r
```

`NewChiRouter()` and `NewChiRouterWithCORS()` are one-line delegating wrappers. No logic duplication.

---

## 6. go-chi/cors Capability

Version: `v1.2.2` (go.mod:8). `cors.Options.AllowedMethods` is a plain `[]string` — any HTTP method name is valid. No upgrade needed.

---

## 7. Missing Test Coverage

No tests in `router/` validate that CORS `AllowedMethods` matches the methods registered via `registerRoute()`. This gap allowed the bug to exist undetected. New tests in `router/chi_test.go` must cover preflight responses for all three constructors plus the superset assertion.
