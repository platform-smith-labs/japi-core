# Generic Service Injection for Handler Registry — Implementation Plan

**Work Item**: work-0004
**Created**: 2026-04-01
**Type**: Implementation Plan
**Status**: ✅ Completed

## Overview

Add a `Services any` field to `HandlerContext` and use the functional options pattern on `RegisterWithRouter` so that japi-core consumers can inject application-defined dependencies into handlers without global mutable state.

## Documentation Chain

- **Work Item**: [work-0004](../manifest.md)
- **Research**: [0001-generic-service-injection-research.md](../research/0001-generic-service-injection-research.md)
- **Requirements**: [0001-generic-service-injection-req.md](../requirements/0001-generic-service-injection-req.md)

## Architecture Review

**Reviewed By**: architect-reviewer, research-analyst
**Status**: Approved

**Key Decisions**:
- Functional options over builder pattern (canonical Go idiom, less API surface)
- `Services any` field on `HandlerContext` (single type assertion boundary, idiomatic)
- New `AdaptHandlerWithServices` function (preserves 14 existing `AdaptHandler` call sites)
- No new interfaces (no `ServiceAwareHandler`)

## Current State

- `HandlerContext` has hardcoded `DB *sql.DB` and `Logger *slog.Logger` — no extension point (`handler/types.go:21-40`)
- `AdaptHandler` creates `HandlerContext` with fixed fields (`handler/adapter.go:47-53`)
- `RegisterWithRouter` calls `route.Handler.Adapt(database, logger)` in a loop (`handler/types.go:131-140`)
- `AdaptHandler` is public API with 14 direct call sites in `handler/context_test.go`
- Consumers (platform-smith-api) use global `var + Init()` pattern as workaround

## Desired End State

```go
// Consumer main.go — single injection point, no globals
appServices := services.Services{OrchClient: orchClient}
handlers.Server.RegisterWithRouter(r, db, logger, handler.WithServices(appServices))

// Consumer handler — typed access, no init functions
svc := services.FromContext(ctx)
resp, err := svc.OrchClient.Get(r.Context(), "/api/v1/tasks", reqCtx)
```

- `go test ./...` passes (all existing + new tests)
- `go build ./...` succeeds
- Existing consumers compile and behave identically without any code changes

## What We're NOT Doing

- Moving `DB`/`Logger` into the services struct (future consideration)
- Generic `Registry[S any]` (rejected — viral generics)
- Service validation/lifecycle at startup (app responsibility)
- Changing `AdaptableHandler` interface, `Handler` type, `Middleware` type, or `MakeHandler`

## Recommended Tools for Implementation

**Skills**:
- `/commit` — Conventional commits after each phase
- `/learn` — Capture any discoveries during implementation

**Agents by Phase**:
- **Phase 1** (Adapter): **backend-developer** for Go generics implementation, **code-reviewer** before completion
- **Phase 2** (Registry): **backend-developer** for functional options pattern, **code-reviewer** before completion
- **Phase 3** (Tests + Docs): **qa-expert** for test strategy validation, **code-reviewer** for final review

---

## Phase 1: HandlerContext + Adapter Layer

### Goal

Add `Services any` to `HandlerContext`. Refactor `AdaptHandler` to delegate to a shared internal function. Add new `AdaptHandlerWithServices`.

### 1.1 Add `Services` field to `HandlerContext`

**File**: `handler/types.go:21-40`

Add one field after `Logger`:

```go
type HandlerContext[ParamTypeT any, BodyTypeT any] struct {
	Context context.Context
	DB      *sql.DB
	Logger  *slog.Logger
	Services any // Application-defined dependencies (set via WithServices option)

	Params      Nullable[ParamTypeT]
	Body        Nullable[BodyTypeT]
	BodyRaw     Nullable[[]byte]
	Headers     Nullable[http.Header]
	RequestID   Nullable[string]
	UserUUID    Nullable[uuid.UUID]
	CompanyUUID Nullable[uuid.UUID]
}
```

Zero-value is `nil` — no impact on existing code.

### 1.2 Create internal `adaptHandler` function

**File**: `handler/adapter.go`

Extract the core logic from the current `AdaptHandler` into an unexported `adaptHandler` that accepts a `services any` parameter:

```go
// adaptHandler is the shared implementation for both AdaptHandler and AdaptHandlerWithServices.
func adaptHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	services any,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestCtx := r.Context()
		logger.Debug("AdaptHandler creating context",
			"db_nil", db == nil,
			"path", r.URL.Path,
		)
		ctx := HandlerContext[ParamTypeT, BodyTypeT]{
			Context:     requestCtx,
			DB:          db,
			Logger:      logger,
			Services:    services,
			UserUUID:    Nil[uuid.UUID](),
			CompanyUUID: Nil[uuid.UUID](),
		}
		_, err := handler(ctx, w, r)
		if err != nil {
			// ... existing error handling unchanged ...
		}
	}
}
```

### 1.3 Refactor public `AdaptHandler` to delegate

**File**: `handler/adapter.go`

Replace the current `AdaptHandler` body with a one-line delegation. Signature unchanged:

```go
func AdaptHandler[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return adaptHandler(db, logger, nil, handler)
}
```

All 14 existing call sites in `handler/context_test.go` continue to compile and work identically — `services` is `nil`, same as before.

### 1.4 Add public `AdaptHandlerWithServices`

**File**: `handler/adapter.go`

```go
// AdaptHandlerWithServices converts a typed Handler to http.HandlerFunc with service injection.
// Like AdaptHandler, but populates ctx.Services with the provided value.
func AdaptHandlerWithServices[ParamTypeT any, BodyTypeT any, ResponseBodyT any](
	db *sql.DB,
	logger *slog.Logger,
	services any,
	handler Handler[ParamTypeT, BodyTypeT, ResponseBodyT],
) http.HandlerFunc {
	return adaptHandler(db, logger, services, handler)
}
```

### Success Criteria — Phase 1

#### Automated Verification:
- [x] `go build ./...` succeeds
- [x] `go test ./handler/...` passes (all 14 existing `AdaptHandler` tests unchanged)
- [x] `go vet ./...` clean

#### Manual Verification:
- [x] `HandlerContext.Services` is `nil` in all existing test paths (no behavior change)

---

## Phase 2: Functional Options on RegisterWithRouter

### Goal

Add `RegistrationOption` type, `WithServices` option function, and modify `RegisterWithRouter` to accept variadic options and wire services through to handler adaptation.

### 2.1 Add `RegistrationOption` type and `registrationConfig`

**File**: `handler/types.go` (after the `Registry` type, before `NewRegistry`)

```go
// registrationConfig holds optional configuration applied during route registration.
type registrationConfig struct {
	services any
}

// RegistrationOption configures how routes are registered with the router.
// Pass options to RegisterWithRouter to customize registration behavior.
type RegistrationOption func(*registrationConfig)

// WithServices configures application-defined dependencies to inject into all handlers.
// The services value is opaque to japi-core — each app defines its own typed struct.
//
// Usage:
//
//	registry.RegisterWithRouter(r, db, logger, handler.WithServices(appServices))
func WithServices(services any) RegistrationOption {
	return func(cfg *registrationConfig) {
		cfg.services = services
	}
}
```

### 2.2 Modify `RegisterWithRouter` to accept options

**File**: `handler/types.go:131-140`

Add variadic `...RegistrationOption`, apply options, and use `AdaptHandlerWithServices` when services are configured:

```go
func (reg *Registry) RegisterWithRouter(r chi.Router, database *sql.DB, logger *slog.Logger, opts ...RegistrationOption) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	cfg := &registrationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	for _, route := range reg.routes {
		var adaptedHandler http.HandlerFunc
		if cfg.services != nil {
			if sa, ok := route.Handler.(interface {
				AdaptWithServices(*sql.DB, *slog.Logger, any) http.HandlerFunc
			}); ok {
				adaptedHandler = sa.AdaptWithServices(database, logger, cfg.services)
			} else {
				adaptedHandler = route.Handler.Adapt(database, logger)
			}
		} else {
			adaptedHandler = route.Handler.Adapt(database, logger)
		}
		registerRoute(r, route.Method, route.Path, adaptedHandler)
	}
}
```

### 2.3 Add `AdaptWithServices` to `TypedHandler`

**File**: `handler/types.go` (after the existing `Adapt` method)

```go
// AdaptWithServices converts the typed handler to http.HandlerFunc with service injection.
func (th TypedHandler[ParamTypeT, BodyTypeT, ResponseBodyT]) AdaptWithServices(database *sql.DB, logger *slog.Logger, services any) http.HandlerFunc {
	return AdaptHandlerWithServices(database, logger, services, th.handler)
}
```

This is not a public interface — it's checked via structural typing (anonymous interface) inside `RegisterWithRouter`. The existing `AdaptableHandler` interface is **unchanged**.

### Success Criteria — Phase 2

#### Automated Verification:
- [x] `go build ./...` succeeds
- [x] `go test ./...` passes (all existing tests)
- [x] `go vet ./...` clean

#### Manual Verification:
- [x] `RegisterWithRouter(r, db, logger)` still works (no options)
- [x] `RegisterWithRouter(r, db, logger, WithServices(s))` compiles

---

## Phase 3: Tests and Documentation

### Goal

Add comprehensive tests for service injection. Update README with usage examples.

### 3.1 Add adapter tests

**File**: `handler/context_test.go`

Add tests after existing test cases:

```go
func TestAdaptHandlerWithServices(t *testing.T) {
	t.Run("services populated in context", func(t *testing.T) {
		type TestServices struct{ Name string }
		svc := TestServices{Name: "test-service"}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		var capturedServices any
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			capturedServices = ctx.Services
			return struct{}{}, nil
		}

		adapted := AdaptHandlerWithServices[struct{}, struct{}, struct{}](nil, logger, svc, handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		adapted.ServeHTTP(w, req)

		if capturedServices == nil {
			t.Fatal("expected services to be non-nil")
		}
		typed, ok := capturedServices.(TestServices)
		if !ok {
			t.Fatalf("expected TestServices, got %T", capturedServices)
		}
		if typed.Name != "test-service" {
			t.Errorf("expected Name='test-service', got %q", typed.Name)
		}
	})

	t.Run("nil services does not panic", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
			if ctx.Services != nil {
				t.Error("expected nil services")
			}
			return struct{}{}, nil
		}
		adapted := AdaptHandlerWithServices[struct{}, struct{}, struct{}](nil, logger, nil, handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		adapted.ServeHTTP(w, req)
	})
}

func TestAdaptHandlerServicesNilByDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
		if ctx.Services != nil {
			t.Error("expected nil services for AdaptHandler (no services)")
		}
		return struct{}{}, nil
	}
	adapted := AdaptHandler[struct{}, struct{}, struct{}](nil, logger, handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	adapted.ServeHTTP(w, req)
}
```

### 3.2 Add registry integration tests

**File**: `handler/registry_test.go`

```go
func TestRegisterWithRouterWithServices(t *testing.T) {
	type TestServices struct{ Value string }
	svc := TestServices{Value: "injected"}

	t.Run("services injected into handler via WithServices option", func(t *testing.T) {
		reg := NewRegistry()
		var capturedServices any

		MakeHandler(reg, RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				capturedServices = ctx.Services
				w.WriteHeader(http.StatusOK)
				return struct{}{}, nil
			},
		)

		r := chi.NewRouter()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		reg.RegisterWithRouter(r, nil, logger, WithServices(svc))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		typed, ok := capturedServices.(TestServices)
		if !ok {
			t.Fatalf("expected TestServices, got %T", capturedServices)
		}
		if typed.Value != "injected" {
			t.Errorf("expected 'injected', got %q", typed.Value)
		}
	})

	t.Run("no services — backward compatible", func(t *testing.T) {
		reg := NewRegistry()
		var capturedServices any

		MakeHandler(reg, RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				capturedServices = ctx.Services
				w.WriteHeader(http.StatusOK)
				return struct{}{}, nil
			},
		)

		r := chi.NewRouter()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		reg.RegisterWithRouter(r, nil, logger) // no options

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if capturedServices != nil {
			t.Errorf("expected nil services, got %v", capturedServices)
		}
	})

	t.Run("WithServices nil does not panic", func(t *testing.T) {
		reg := NewRegistry()
		MakeHandler(reg, RouteInfo{Method: "GET", Path: "/test"},
			func(ctx HandlerContext[struct{}, struct{}], w http.ResponseWriter, r *http.Request) (struct{}, error) {
				w.WriteHeader(http.StatusOK)
				return struct{}{}, nil
			},
		)

		r := chi.NewRouter()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		reg.RegisterWithRouter(r, nil, logger, WithServices(nil))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}
```

### 3.3 Update README

**File**: `README.md`

Add a "Service Injection" section after the existing "Getting Started" or "Usage" section. Include:
- Problem statement (1-2 sentences)
- `WithServices` usage example
- Typed accessor pattern example
- Note about backward compatibility

### Success Criteria — Phase 3

#### Automated Verification:
- [x] `go test ./...` passes (all existing + all new tests)
- [x] `go build ./...` succeeds
- [x] `go vet ./...` clean

#### Manual Verification:
- [x] README service injection section is clear and accurate
- [x] New test names are descriptive and follow existing conventions

---

## Risk Management

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `AdaptHandler` refactor changes error behavior | Low | High | Internal `adaptHandler` preserves all error handling verbatim; existing 14 tests verify |
| Structural type assertion in `RegisterWithRouter` fails | Low | Medium | `TypedHandler` always implements `AdaptWithServices`; fallback to `Adapt()` for any third-party `AdaptableHandler` implementations |
| `WithServices(nil)` treated differently than no options | Low | Low | Explicit test for `nil` case; `registrationConfig.services` zero value is already `nil` |

## Progress Tracking

| Phase | Status | Files Changed |
|-------|--------|---------------|
| Phase 1: HandlerContext + Adapter | ✅ Complete | `handler/types.go`, `handler/adapter.go` |
| Phase 2: Registry + Functional Options | ✅ Complete | `handler/types.go` |
| Phase 3: Tests + Documentation | ✅ Complete | `handler/context_test.go`, `handler/registry_test.go`, `README.md` |

## Next Steps

Use `/implement_plan docs/work/work-0004/plans/master.md` to begin implementation.
