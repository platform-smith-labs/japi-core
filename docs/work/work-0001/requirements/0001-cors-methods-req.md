# Requirements: CORS AllowedMethods Fix and Override API

**Work Item**: work-0001
**Created**: 2026-03-30
**Updated**: 2026-03-30 — override mechanism changed to functional options pattern
**Status**: Draft

---

## 1. Background

`router/chi.go` hardcodes the CORS `AllowedMethods` list as `["GET", "POST", "PUT", "DELETE", "OPTIONS"]` in both `NewChiRouter()` (line 41) and `NewChiRouterWithCORS()` (line 83). PATCH and HEAD are absent despite being fully supported by chi (`handler/types.go:154-171`), the swagger generator, and all typed middleware.

The consequence is a CORS preflight failure for any PATCH request originating from a browser. No application-level change can work around this without forking the library.

A secondary issue is that both constructors duplicate the same CORS initialisation block. The functional options pattern eliminates this duplication while providing the override mechanism.

---

## 2. Goals

1. Fix the default `AllowedMethods` list to include PATCH and HEAD — in both existing constructors — without changing their signatures.
2. Introduce a functional options pattern (`RouterOption`, `WithAllowedOrigins`, `WithAllowedMethods`, …) as the underlying configuration mechanism.
3. Expose a new `NewChiRouterWithOptions(opts ...RouterOption)` constructor for consuming apps that need full control.
4. Refactor `NewChiRouter()` and `NewChiRouterWithCORS()` to delegate to the new internal constructor — eliminating code duplication.
5. Close the test-coverage gap with tests in `router/chi_test.go`.
6. Correct the broken `AllowedMethods` examples in README.md and add usage examples for the new API.

---

## 3. Functional Requirements

### FR-1: Default AllowedMethods includes PATCH and HEAD

The default configuration applied when no option overrides it must include:

```
GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
```

This matches the complete set of methods handled by `registerRoute()` in `handler/types.go`.

### FR-2: RouterOption type and unexported config struct

Add an exported `RouterOption` function type and an unexported `routerConfig` struct:

```go
// RouterOption is a functional option that configures the Chi router.
type RouterOption func(*routerConfig)

type routerConfig struct {
    allowedOrigins   []string
    allowedMethods   []string
    allowedHeaders   []string
    exposedHeaders   []string
    allowCredentials bool
    maxAge           int
}
```

`routerConfig` is unexported. Callers interact only via `RouterOption` functions.

### FR-3: Option constructor functions

The following exported functions must be added, each returning a `RouterOption`:

| Function | Sets |
|----------|------|
| `WithAllowedOrigins(origins []string) RouterOption` | `allowedOrigins` |
| `WithAllowedMethods(methods []string) RouterOption` | `allowedMethods` |
| `WithAllowedHeaders(headers []string) RouterOption` | `allowedHeaders` |
| `WithExposedHeaders(headers []string) RouterOption` | `exposedHeaders` |
| `WithAllowCredentials(allow bool) RouterOption` | `allowCredentials` |
| `WithMaxAge(seconds int) RouterOption` | `maxAge` |

Each function returns a closure that mutates the corresponding field of `*routerConfig`.

### FR-4: Internal newChiRouter constructor

An unexported `newChiRouter(opts ...RouterOption) chi.Router` function must:

1. Start from `defaultRouterConfig()` (the corrected defaults from FR-1)
2. Apply each option in order
3. Build a chi router with `middleware.RequestID`, `middleware.RealIP`, `middleware.Recoverer`
4. Apply `cors.Handler` using the resulting config
5. Return the router

This is the single location where the chi middleware stack and CORS handler are constructed. No other function may duplicate this logic.

### FR-5: defaultRouterConfig()

An unexported `defaultRouterConfig() routerConfig` function returns the baseline config:

```go
routerConfig{
    allowedOrigins:   []string{},
    allowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
    allowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    exposedHeaders:   []string{"Link"},
    allowCredentials: false,
    maxAge:           300,
}
```

Must return a new value on every call with independent slice backing arrays.

### FR-6: Existing constructors become thin wrappers — signatures unchanged

```go
func NewChiRouter() chi.Router {
    return newChiRouter()
}

func NewChiRouterWithCORS(allowedOrigins []string) chi.Router {
    return newChiRouter(WithAllowedOrigins(allowedOrigins))
}
```

No other logic inside these functions. Signatures are frozen.

### FR-7: New public constructor NewChiRouterWithOptions

```go
// NewChiRouterWithOptions creates a Chi router with fully customisable CORS configuration.
// Use the With* option functions to override specific defaults.
//
// Example:
//
//  r := router.NewChiRouterWithOptions(
//      router.WithAllowedOrigins([]string{"https://app.example.com"}),
//      router.WithAllowedMethods([]string{"GET", "POST", "PATCH"}),
//  )
func NewChiRouterWithOptions(opts ...RouterOption) chi.Router {
    return newChiRouter(opts...)
}
```

### FR-8: README.md corrected and updated

- All existing `AllowedMethods` examples updated to include PATCH and HEAD
- Inline example at line 27 in `NewChiRouter()` doc comment updated
- New section added in README.md CORS configuration area demonstrating `NewChiRouterWithOptions` with `With*` functions

---

## 4. Non-Functional Requirements

### NFR-1: Backward compatibility — zero breaking changes

`NewChiRouter()` and `NewChiRouterWithCORS(allowedOrigins []string)` signatures are frozen. No existing call site requires modification.

### NFR-2: No new module dependencies

No new entries in `go.mod`. `github.com/go-chi/cors v1.2.2` is sufficient.

### NFR-3: No duplication of router initialisation logic

After the refactor, the chi middleware stack (`RequestID`, `RealIP`, `Recoverer`) and `cors.Handler(...)` call must appear exactly once — inside `newChiRouter`. The existing two-function duplication is eliminated.

### NFR-4: Test coverage for CORS method completeness

New tests in `router/chi_test.go` using `net/http/httptest` must cover:

- `NewChiRouter()` preflight response contains all 7 default methods
- `NewChiRouterWithCORS()` preflight response contains all 7 default methods
- `NewChiRouterWithOptions(WithAllowedMethods(...))` preflight response contains only the supplied methods
- `defaultRouterConfig().allowedMethods` is a superset of every method in `registerRoute()`

### NFR-5: No changes outside router/ and README.md

All code changes confined to `router/chi.go`, `router/chi_test.go`, and `README.md`.

### NFR-6: No global or package-level mutable variables

Consistent with FP principles in CLAUDE.md.

### NFR-7: Go 1.24 compatibility

All code compiles and passes tests under Go 1.24.0.

---

## 5. Acceptance Criteria

**AC-1**: A browser-initiated `PATCH` request to a server using `NewChiRouterWithCORS([]string{"https://app.example.com"})` succeeds — the `OPTIONS` preflight response includes `PATCH` in `Access-Control-Allow-Methods`.

**AC-2**: `NewChiRouter()` continues to compile, returns a router that denies all cross-origin requests, and its call sites require zero modification.

**AC-3**: `NewChiRouterWithCORS([]string{"https://x.com"})` continues to compile, returns a router allowing that origin, and its call sites require zero modification.

**AC-4**: `NewChiRouterWithOptions(WithAllowedMethods([]string{"GET", "POST"}))` emits only `GET, POST` in `Access-Control-Allow-Methods` on preflight.

**AC-5**: `NewChiRouterWithOptions(WithAllowedOrigins([]string{"https://a.com"}), WithAllowedMethods([]string{"GET", "PATCH"}))` emits `https://a.com` in `Access-Control-Allow-Origin` and `GET, PATCH` in `Access-Control-Allow-Methods`.

**AC-6**: `go test ./router/...` passes covering all four test cases in NFR-4.

**AC-7**: `go build ./...` and `go vet ./...` produce no errors or warnings.

**AC-8**: No `AllowedMethods` example in README.md omits PATCH or HEAD.

**AC-9**: README.md CORS section includes an example using `NewChiRouterWithOptions` with at least `WithAllowedOrigins` and `WithAllowedMethods`.

**AC-10**: `router/chi.go` contains exactly one location where `cors.Handler(...)` is called (inside `newChiRouter`).

---

## 6. Constraints

**C-1**: Signatures `NewChiRouter() chi.Router` and `NewChiRouterWithCORS(allowedOrigins []string) chi.Router` are frozen.

**C-2**: `routerConfig` must remain unexported. Callers must not be able to construct it directly — only via `RouterOption` functions.

**C-3**: Each `With*` function must set exactly one field. No combined options (e.g. no `WithCORS(origins, methods []string)`).

**C-4**: Option functions must be safe to call with nil slices. `WithAllowedMethods(nil)` is valid and sets the field to nil/empty — the caller's intent is respected.

**C-5**: No global or package-level mutable variables introduced.

---

## 7. Out of Scope

- **CORSConfig struct** — superseded by the functional options approach
- **Runtime CORS reconfiguration** — options applied once at construction
- **Wildcard origin or credentials behaviour changes**
- **Deprecation of `NewChiRouter()` or `NewChiRouterWithCORS()`**
- **Changes to handler/, middleware/, swagger/, db/, metrics/ packages**

---

## 8. Relevant Files

| File | Change |
|------|--------|
| `router/chi.go` | Refactor: add `RouterOption`, `With*` funcs, `newChiRouter`, `NewChiRouterWithOptions`; make existing constructors thin wrappers; fix default methods |
| `router/chi_test.go` | New: CORS method coverage tests |
| `README.md` | Correct line 1168 + doc comments; add `NewChiRouterWithOptions` example |
| `handler/types.go:154-171` | Reference only: canonical method list for `registerRoute()` |

---

*Derived from research document `./research/0001-cors-methods-research.md`.*
