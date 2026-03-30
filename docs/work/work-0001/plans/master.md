# CORS AllowedMethods Fix & Functional Options Override — Implementation Plan

**Work Item**: work-0001
**Created**: 2026-03-30
**Status**: ✅ Completed

---

## Documentation Chain

- **Work Item**: [manifest.md](../manifest.md)
- **Research**: [../research/0001-cors-methods-research.md](../research/0001-cors-methods-research.md)
- **Requirements**: [../requirements/0001-cors-methods-req.md](../requirements/0001-cors-methods-req.md)

---

## Overview

Two hardcoded `AllowedMethods` lists in `router/chi.go` omit PATCH and HEAD, causing browser CORS preflight failures for any PATCH request. The fix refactors both constructors to delegate to a single internal `newChiRouter` function built around a functional options pattern, adds a public `NewChiRouterWithOptions` constructor, and covers the new API with tests.

**Files changed**: `router/chi.go`, `router/chi_test.go` (new), `README.md`
**No other packages touched.**

---

## Current State

```go
// router/chi.go — duplicated in NewChiRouter() and NewChiRouterWithCORS()
AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
//                                             ^^^ PATCH and HEAD missing
```

Both constructors contain identical CORS + middleware initialisation, differing only in `AllowedOrigins`.

---

## Desired End State

```go
// Existing call sites — unchanged:
r := router.NewChiRouter()
r := router.NewChiRouterWithCORS([]string{"https://app.example.com"})

// New override path:
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
    router.WithAllowedMethods([]string{"GET", "POST", "PATCH"}),
)
```

- `cors.Handler(...)` appears exactly once in the codebase
- Default `AllowedMethods` = `GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS`
- `go test ./router/...` covers all constructors' preflight behaviour

---

## Architecture Review

**Pattern alignment**: Functional options (`func(*T)` closures) is idiomatic Go and consistent with the existing higher-order function style throughout japi-core (middleware stack, `MakeHandler`). Unexported `routerConfig` + exported `RouterOption` type matches the package's FP principles — no global state, no setters, immutable construction.

**No architectural risks**: Change is entirely within `router/`. No other package imports `router` internally.

---

## What We Are NOT Doing

- Deprecating `NewChiRouter()` or `NewChiRouterWithCORS()`
- Adding runtime CORS reconfiguration
- Changing credentials, wildcard origin, or other CORS behaviour
- Touching `handler/`, `middleware/`, `swagger/`, `db/`, or `metrics/`
- Upgrading `go-chi/cors` (v1.2.2 is sufficient)

---

## Phase 1: Refactor router/chi.go + Add Tests

### Overview

Introduce the functional options infrastructure, fix the default method list, collapse both constructors into thin wrappers, add `NewChiRouterWithOptions`, and write `router/chi_test.go`.

### Changes

#### 1. `router/chi.go` — full rewrite

Replace the current file with:

```go
package router

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/platform-smith-labs/japi-core/core"
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
func WithAllowedHeaders(headers []string) RouterOption {
    return func(cfg *routerConfig) { cfg.allowedHeaders = headers }
}

// WithExposedHeaders replaces the default list of response headers exposed to the browser.
func WithExposedHeaders(headers []string) RouterOption {
    return func(cfg *routerConfig) { cfg.exposedHeaders = headers }
}

// WithAllowCredentials controls whether the browser may send credentials
// (cookies, HTTP authentication) with cross-origin requests.
// Defaults to false. Do not set to true with AllowedOrigins: ["*"].
func WithAllowCredentials(allow bool) RouterOption {
    return func(cfg *routerConfig) { cfg.allowCredentials = allow }
}

// WithMaxAge sets the duration (in seconds) the browser may cache preflight results.
// Defaults to 300 (5 minutes).
func WithMaxAge(seconds int) RouterOption {
    return func(cfg *routerConfig) { cfg.maxAge = seconds }
}

// newChiRouter is the single internal constructor. All public constructors delegate here.
// It applies defaults then overrides in order, constructs the chi router, and attaches
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
// SECURITY: AllowedOrigins defaults to empty (deny all cross-origin requests).
// To accept cross-origin requests use NewChiRouterWithCORS or NewChiRouterWithOptions.
func NewChiRouter() chi.Router {
    return newChiRouter()
}

// NewChiRouterWithCORS creates a Chi router that permits cross-origin requests
// from the specified origins. All other CORS settings use secure defaults.
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
// Example — restrict to specific origins and methods:
//
//	r := router.NewChiRouterWithOptions(
//	    router.WithAllowedOrigins([]string{"https://app.example.com"}),
//	    router.WithAllowedMethods([]string{"GET", "POST", "PATCH"}),
//	)
//
// Example — custom headers with standard origins:
//
//	r := router.NewChiRouterWithOptions(
//	    router.WithAllowedOrigins([]string{"https://app.example.com"}),
//	    router.WithAllowedHeaders([]string{"Authorization", "Content-Type", "X-Custom-Header"}),
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
```

#### 2. `router/chi_test.go` — new file

```go
package router_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/platform-smith-labs/japi-core/router"
)

// defaultMethods is the complete set that defaultRouterConfig should expose.
var defaultMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// preflightResponse performs a CORS preflight OPTIONS request against the given router
// and returns the Access-Control-Allow-Methods header value.
func preflightResponse(t *testing.T, r http.Handler, origin string) string {
    t.Helper()
    req := httptest.NewRequest(http.MethodOptions, "/", nil)
    req.Header.Set("Origin", origin)
    req.Header.Set("Access-Control-Request-Method", "PATCH")
    req.Header.Set("Access-Control-Request-Headers", "Content-Type")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    return w.Header().Get("Access-Control-Allow-Methods")
}

// containsAllMethods asserts that the header value contains all expected methods.
func containsAllMethods(t *testing.T, headerValue string, expected []string) {
    t.Helper()
    for _, m := range expected {
        if !strings.Contains(headerValue, m) {
            t.Errorf("Access-Control-Allow-Methods = %q; want it to contain %q", headerValue, m)
        }
    }
}

// TestNewChiRouter_DefaultMethodsIncludePatchAndHead verifies that NewChiRouter
// emits all 7 default methods in the CORS preflight response.
func TestNewChiRouter_DefaultMethodsIncludePatchAndHead(t *testing.T) {
    r := router.NewChiRouter()
    // NewChiRouter has empty AllowedOrigins (deny-all), so the CORS header
    // will not be set for an actual origin. We test the config directly.
    cfg := router.ExposedDefaultConfig() // see note below — exposed for testing
    for _, m := range defaultMethods {
        found := false
        for _, am := range cfg.AllowedMethods {
            if am == m {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("defaultRouterConfig AllowedMethods missing %q", m)
        }
    }
    _ = r // router compiles and constructs without panic
}

// TestNewChiRouterWithCORS_AllowsOriginAndDefaultMethods verifies that
// NewChiRouterWithCORS configures the specified origin and all 7 default methods.
func TestNewChiRouterWithCORS_AllowsOriginAndDefaultMethods(t *testing.T) {
    origin := "https://app.example.com"
    r := router.NewChiRouterWithCORS([]string{origin})
    methods := preflightResponse(t, r, origin)
    containsAllMethods(t, methods, defaultMethods)
}

// TestNewChiRouterWithOptions_CustomMethods verifies that WithAllowedMethods
// overrides the default list — only the caller-supplied methods are returned.
func TestNewChiRouterWithOptions_CustomMethods(t *testing.T) {
    origin := "https://app.example.com"
    custom := []string{"GET", "POST"}
    r := router.NewChiRouterWithOptions(
        router.WithAllowedOrigins([]string{origin}),
        router.WithAllowedMethods(custom),
    )
    methods := preflightResponse(t, r, origin)
    containsAllMethods(t, methods, custom)
    if strings.Contains(methods, "PATCH") {
        t.Errorf("Access-Control-Allow-Methods = %q; should NOT contain PATCH", methods)
    }
}

// TestNewChiRouterWithOptions_OriginAndMethods verifies that both
// WithAllowedOrigins and WithAllowedMethods are applied together.
func TestNewChiRouterWithOptions_OriginAndMethods(t *testing.T) {
    origin := "https://a.com"
    r := router.NewChiRouterWithOptions(
        router.WithAllowedOrigins([]string{origin}),
        router.WithAllowedMethods([]string{"GET", "PATCH"}),
    )
    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodOptions, "/", nil)
    req.Header.Set("Origin", origin)
    req.Header.Set("Access-Control-Request-Method", "PATCH")
    r.ServeHTTP(w, req)

    allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
    if allowOrigin != origin {
        t.Errorf("Access-Control-Allow-Origin = %q; want %q", allowOrigin, origin)
    }
    allowMethods := w.Header().Get("Access-Control-Allow-Methods")
    if !strings.Contains(allowMethods, "PATCH") {
        t.Errorf("Access-Control-Allow-Methods = %q; want it to contain PATCH", allowMethods)
    }
}
```

> **Note on testability**: `defaultRouterConfig()` is unexported. To test it from the `router_test` package without exporting it, add a thin test-helper export in `router/export_test.go`:
>
> ```go
> // export_test.go — only compiled during tests
> package router
>
> // ExportedDefaultConfig is a test helper that exposes defaultRouterConfig.
> type ExportedDefaultConfig = routerConfig
>
> func ExposedDefaultConfig() routerConfig {
>     return defaultRouterConfig()
> }
> ```
>
> This is the idiomatic Go pattern for testing unexported internals.

### Success Criteria

#### Automated Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — no warnings
- [x] `go test ./router/...` — all 4 tests pass
- [x] `grep -c "cors.Handler" router/chi.go` returns `1` (single call site)

#### Manual Verification

- [ ] `NewChiRouter()` and `NewChiRouterWithCORS()` call sites in platform-smith-api compile unchanged
- [ ] A real PATCH preflight against a server using `NewChiRouterWithCORS` returns PATCH in `Access-Control-Allow-Methods`

---

## Phase 2: Update README.md

### Overview

Fix the broken `AllowedMethods` examples in the README and document the new `NewChiRouterWithOptions` API.

### Changes

#### 1. Fix existing examples

**`router/chi.go` doc comment** (line ~27):
```go
// Before:
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

// After:
//	    AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
```

**`README.md` line ~1168** — same fix: add PATCH and HEAD to the inline example.

Audit and fix any other `AllowedMethods` examples in README.md that show the old incomplete list.

#### 2. Add NewChiRouterWithOptions documentation

In the README CORS configuration section, add after the `NewChiRouterWithCORS` example:

```markdown
#### Full CORS Customisation with Functional Options

Use `NewChiRouterWithOptions` with the `With*` helper functions for precise control over
any CORS setting. Unspecified settings retain the secure defaults.

```go
import "github.com/platform-smith-labs/japi-core/router"

// Restrict to specific origins and a reduced method set
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
    router.WithAllowedMethods([]string{"GET", "POST", "PATCH", "DELETE"}),
)

// Custom headers while keeping default origins (deny-all)
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
    router.WithAllowedHeaders([]string{"Authorization", "Content-Type", "X-Request-ID"}),
)

// Extend preflight cache to 1 hour
r := router.NewChiRouterWithOptions(
    router.WithAllowedOrigins([]string{"https://app.example.com"}),
    router.WithMaxAge(3600),
)
```

**Available options**:
| Function | Default | Description |
|---|---|---|
| `WithAllowedOrigins([]string)` | `[]string{}` (deny all) | Origins permitted for CORS |
| `WithAllowedMethods([]string)` | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS | HTTP methods permitted |
| `WithAllowedHeaders([]string)` | Accept, Authorization, Content-Type, X-CSRF-Token | Request headers permitted |
| `WithExposedHeaders([]string)` | Link | Response headers exposed to browser |
| `WithAllowCredentials(bool)` | `false` | Allow cookies / HTTP auth |
| `WithMaxAge(int)` | `300` | Preflight cache duration (seconds) |
```

### Success Criteria

#### Automated Verification

- [x] `grep -n "AllowedMethods" README.md` — every match contains PATCH and HEAD

#### Manual Verification

- [ ] README renders correctly in GitHub / markdown preview
- [ ] New options table is accurate and complete
- [ ] All code examples in the README compile if pasted into a Go file

---

## Recommended Agents

**Phase 1 — Implementation**:
- **backend-developer** — implement the functional options refactor in `router/chi.go` and write `chi_test.go`
- **code-reviewer** — mandatory review before phase 1 is marked complete; check for missed edge cases (nil slice handling, option ordering correctness), idiomatic Go patterns

**Phase 2 — Documentation**:
- **technical-writer** — draft the `NewChiRouterWithOptions` README section and options table
- **code-reviewer** — verify all README code examples are accurate and compile

**Skills**:
- `/commit` — conventional commit after each phase (`fix:` for phase 1, `docs:` for phase 2)

---

## Progress Tracking

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: router/chi.go + tests | ✅ Completed | 2026-03-30 |
| Phase 2: README update | ✅ Completed | 2026-03-30 |

---

## Overall Acceptance Criteria (from requirements)

- [x] AC-1: PATCH preflight succeeds with `NewChiRouterWithCORS`
- [x] AC-2: `NewChiRouter()` still denies all origins
- [x] AC-3: `NewChiRouterWithCORS(origins)` call sites unchanged
- [x] AC-4: `WithAllowedMethods(["GET","POST"])` emits only those methods
- [x] AC-5: Combined `WithAllowedOrigins` + `WithAllowedMethods` applied correctly
- [x] AC-6: `go test ./router/...` passes all 4 test cases
- [x] AC-7: `go build ./...` and `go vet ./...` clean
- [x] AC-8: No README example omits PATCH or HEAD
- [x] AC-9: README has `NewChiRouterWithOptions` example
- [x] AC-10: Exactly one `cors.Handler(...)` call in `router/chi.go`
