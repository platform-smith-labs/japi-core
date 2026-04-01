# Research: Generic Service Injection for Handler Registry

**Work Item**: work-0004
**Date**: 2026-04-01
**Researcher**: Claude Code (codebase-analyzer, research-analyst, architect-reviewer)

## 1. Problem Validation

### Codebase Verification

All claims from the source prompt have been validated against the current codebase:

| Claim | Status | Evidence |
|-------|--------|----------|
| `HandlerContext` has hardcoded `DB`/`Logger` only | **Confirmed** | `handler/types.go:21-40` — no extension point for app-specific dependencies |
| DI pipeline is sealed | **Confirmed** | `RegisterWithRouter` → `Adapt()` → `AdaptHandler()` → fixed `HandlerContext{}` at `handler/adapter.go:47-53` |
| `AdaptableHandler` interface is fixed | **Confirmed** | `handler/types.go:58-60` — `Adapt(database *sql.DB, logger *slog.Logger)` |
| `AdaptHandler` is public API with direct callers | **Confirmed** | 14 direct call sites in `handler/context_test.go` — signature change breaks them |
| `TypedHandler.Adapt` delegates to `AdaptHandler` | **Confirmed** | `handler/types.go:69` |
| Consumer forced into global mutable state | **Confirmed** | platform-smith-api uses `var orchestratorClient` + `InitXHandlers()` pattern |

### Root Cause

The dependency injection pipeline has no extension point between `Registry.RegisterWithRouter()` and `HandlerContext` construction. The only way to get data into a handler is through the two hardcoded parameters (`*sql.DB`, `*slog.Logger`) or through closure over package-level variables.

## 2. Solutions Analysis

### Solutions Correctly Rejected by Prompt

| Solution | Why Rejected | Agree? |
|----------|-------------|--------|
| Add `ServicesT` generic param to `HandlerContext` | Cascades to every type (`Handler`, `Middleware`, `MakeHandler`, all middleware) — massive breaking change | **Yes** — viral generics is the top anti-pattern in Go DI |
| `map[string]any` on `HandlerContext` | No type safety, runtime key typos, worse than closures | **Yes** |
| `context.Context` value bag | Mixes DI with request-scoped data, no type safety, semantically wrong | **Yes** — services are app-scoped singletons |
| Change `RegisterWithRouter` signature | Breaks all consumers | **Partially** — variadic options preserve backward compat (see below) |
| Service locator / container | Anti-pattern, hides deps, runtime panics | **Yes** |

### Builder Pattern (Evaluated and Rejected)

The source prompt proposes: `Services any` field + `RegistrationBuilder` with `WithServices()` + `ServiceAwareHandler` interface.

**Strengths:**
1. Zero breaking changes to existing public API
2. Clean builder chain: `registry.WithServices(s).RegisterWithRouter(r, db, logger)`
3. Extensible for future options (`WithRequestTimeout`, `WithCORS`)
4. Opt-in — apps that don't need services change nothing
5. Single `any` type assertion at the boundary, fully typed after

**Weaknesses (reasons for rejection):**
1. **`ServiceAwareHandler` adds interface complexity** — a second optional interface checked via type assertion at runtime, when the framework itself controls all `TypedHandler` creation
2. **Dual `WithServices` methods** — one on `*Registry` (entry point) and one on `*RegistrationBuilder` (chaining) creates API surface duplication (4 new methods total)
3. **`AdaptHandler` signature change** — breaks 14 test call sites in `context_test.go`
4. **Less idiomatic** — builder pattern is valid Go but functional options is the canonical pattern for optional configuration (Dave Cheney, Rob Pike)

### Chosen Solution: Functional Options on RegisterWithRouter

```go
type RegistrationOption func(*registrationConfig)

func WithServices(s any) RegistrationOption {
    return func(c *registrationConfig) { c.services = s }
}

// Backward compatible — existing calls still work
func (reg *Registry) RegisterWithRouter(
    r chi.Router, db *sql.DB, logger *slog.Logger, opts ...RegistrationOption,
) {
    cfg := &registrationConfig{}
    for _, opt := range opts { opt(cfg) }
    // ... register routes, passing cfg.services to AdaptHandler
}
```

**Usage:**
```go
// Existing (unchanged)
registry.RegisterWithRouter(r, db, logger)

// New — with services
registry.RegisterWithRouter(r, db, logger, handler.WithServices(appServices))

// Future — more options
registry.RegisterWithRouter(r, db, logger,
    handler.WithServices(appServices),
    handler.WithRequestTimeout(30 * time.Second),
)
```

**Why this wins over the builder pattern:**

| Aspect | Builder (Rejected) | Functional Options (Chosen) |
|--------|-----------------|-------------------|
| Idiomatic Go | Yes (less common) | **Yes (canonical — Dave Cheney/Rob Pike pattern)** |
| New types introduced | `RegistrationBuilder`, `ServiceAwareHandler` | **`RegistrationOption`, `registrationConfig` (unexported)** |
| API surface | 4 new methods | **1 modified method + N option functions** |
| Breaking changes | None | **None (variadic is backward compatible)** |
| Extensibility | Add field to builder + 2 methods | **Add 1 option function** |
| Concurrency | Builder is per-call (safe) | Config is per-call (safe) |
| Discoverability | IDE shows `WithServices` on Registry | IDE shows `WithServices` as package function |
| `ServiceAwareHandler` needed? | Yes — builder needs to distinguish | **No — `RegisterWithRouter` directly passes services** |

### Design Decision

**Functional Options on `RegisterWithRouter`** combined with **`Services any` field on `HandlerContext`** and **new `AdaptHandlerWithServices` function** (preserving existing `AdaptHandler`).

This eliminates:
- `RegistrationBuilder` type entirely
- `ServiceAwareHandler` interface entirely
- Dual `WithServices` methods

While preserving:
- Full backward compatibility
- Clean extensibility via option functions
- The `Services any` field design (which is sound)
- Single type assertion boundary pattern

### AdaptHandler Backward Compatibility

`AdaptHandler` is public API with 14 direct test call sites. Two approaches:

**Option A: Variadic services parameter**
```go
func AdaptHandler[P, B, R any](
    db *sql.DB, logger *slog.Logger,
    handler Handler[P, B, R],
    services ...any,  // optional, backward compatible
) http.HandlerFunc
```
Problem: `services ...any` after `handler` is confusing; handler is already the "last" parameter.

**Option B: Separate function (recommended)**
```go
// Existing — UNCHANGED
func AdaptHandler[P, B, R any](db, logger, handler) http.HandlerFunc { ... }

// New — with services
func AdaptHandlerWithServices[P, B, R any](db, logger, services any, handler) http.HandlerFunc { ... }
```
Both delegate to a shared internal `adaptHandler` function. Zero breaking changes.

## 3. `any` Type Safety Assessment

Using `any` for the services field is **acceptable and idiomatic**:

- `context.Context.Value()` returns `any` — used throughout Go stdlib
- `echo.Context.Set/Get` uses `any`
- `fx.Provide` accepts `any`
- `chi` middleware configuration uses `any` values

**Best practice**: Provide a typed accessor so consumers never touch `any` directly:

```go
// In the consuming app (e.g., platform-smith-api)
func GetServices[P, B any](ctx handler.HandlerContext[P, B]) Services {
    return ctx.Services.(Services)
}
```

One type assertion at the app boundary, fully typed after that. The framework itself never inspects or cares about the type.

## 4. Files Requiring Changes

| File | Change | Breaking? |
|------|--------|-----------|
| `handler/types.go` | Add `Services any` field to `HandlerContext` | No — zero value is `nil` |
| `handler/types.go` | Add `RegistrationOption` type and `WithServices` function | No — new API |
| `handler/types.go` | Modify `RegisterWithRouter` to accept `...RegistrationOption` | No — variadic is backward compatible |
| `handler/adapter.go` | Add `AdaptHandlerWithServices` function | No — new API |
| `handler/adapter.go` | Refactor `AdaptHandler` to delegate to internal `adaptHandler` | No — same behavior |
| `handler/context_test.go` | Add tests for services injection | No — existing tests unchanged |
| `handler/registry_test.go` | Add tests for `WithServices` option | No — existing tests unchanged |

**NOT changed**: All middleware, `MakeHandler`, `Handler` type, `Middleware` type, `AdaptableHandler` interface, `TypedHandler.Adapt`.

## 5. Final Design Summary

**Chosen approach: Functional Options**

| Aspect | Design |
|--------|--------|
| Configuration API | Functional options: `RegisterWithRouter(r, db, logger, WithServices(s))` |
| New interfaces | None |
| New public types | `RegistrationOption` |
| New private types | `registrationConfig` |
| New public functions | `WithServices(any) RegistrationOption`, `AdaptHandlerWithServices(...)` |
| Modified functions | `RegisterWithRouter` (adds variadic `...RegistrationOption`) |
| `HandlerContext` change | Add `Services any` field |
| Unchanged | `AdaptHandler`, `AdaptableHandler`, `Handler`, `Middleware`, `MakeHandler`, `TypedHandler.Adapt`, all middleware |

**Key design properties:**
- Zero breaking changes — variadic options are backward compatible
- Canonical Go idiom — functional options pattern
- Minimal API surface — 1 new type, 2 new functions, 1 modified signature
- Future options (timeouts, CORS, metrics) each require only 1 new `WithX` function
- No new interfaces or intermediate builder types
