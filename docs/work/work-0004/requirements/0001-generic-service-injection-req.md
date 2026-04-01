# Requirements: Generic Service Injection for Handler Registry

**Work Item**: work-0004
**Date**: 2026-04-01
**Status**: Approved
**Validated by**: architect-reviewer, qa-expert

## Overview

Enable japi-core consumers to inject application-defined typed dependencies (HTTP clients, cache clients, message queues, etc.) into handlers through the registry, eliminating the need for global mutable state and init functions.

## Objectives

1. Provide an extension point for application-specific dependencies in the handler pipeline
2. Maintain 100% backward compatibility — zero changes for existing consumers
3. Follow Go idioms and japi-core's functional programming principles
4. Keep the solution generic — japi-core knows nothing about what services contain

## Functional Requirements

### FR-1: Services Field on HandlerContext

- Add a `Services any` field to `HandlerContext[P, B]`
- Default value is `nil` when services are not configured
- Field is accessible to all handlers and middleware via the context
- japi-core never inspects or type-asserts the services value

### FR-2: Functional Options on RegisterWithRouter

- `RegisterWithRouter` accepts variadic `...RegistrationOption` as a final parameter
- Existing calls `RegisterWithRouter(r, db, logger)` continue to work unchanged (variadic defaults to empty)
- `RegistrationOption` is a public type: `type RegistrationOption func(*registrationConfig)`
- `registrationConfig` is an unexported struct holding option values
- `WithServices(s any) RegistrationOption` is the first option function
- Usage: `registry.RegisterWithRouter(r, db, logger, handler.WithServices(appServices))`
- Future options (timeouts, CORS, metrics) added as new `WithX` functions — no API changes needed
- Services are passed to every handler registered in the registry

### FR-3: Services-Aware Handler Adaptation

- Existing `AdaptHandler(db, logger, handler)` signature MUST NOT change (14 direct call sites in tests)
- New `AdaptHandlerWithServices(db, logger, services, handler)` function for services-aware adaptation
- Both delegate to a shared internal `adaptHandler` function to avoid code duplication
- `RegisterWithRouter` internally uses `AdaptHandlerWithServices` when services are configured, `AdaptHandler` otherwise
- `TypedHandler.Adapt()` continues to work without services (calls `AdaptHandler`)

### FR-4: Typed Accessor Pattern (Documentation)

- Document the recommended pattern for consuming services in handler code:
  ```go
  // App defines typed accessor
  func GetServices[P, B any](ctx handler.HandlerContext[P, B]) AppServices {
      return ctx.Services.(AppServices)
  }
  
  // Handler uses typed accessor
  svc := GetServices(ctx)
  svc.OrchClient.Get(...)
  ```
- This is app-level code, not part of japi-core itself

## Non-Functional Requirements

### NFR-1: Zero Breaking Changes

- `RegisterWithRouter(r, db, logger)` MUST remain callable as-is (variadic `...RegistrationOption` preserves this)
- `AdaptableHandler` interface MUST NOT change
- `AdaptHandler` existing 3-parameter signature MUST continue to compile
- `MakeHandler` signature MUST NOT change
- `Handler` and `Middleware` types MUST NOT change
- All existing tests MUST pass without modification

### NFR-2: Performance

- No measurable performance regression for handlers that don't use services
- Services injection adds at most one struct field assignment per request (negligible)

### NFR-3: Concurrency Safety

- Configuration is per-registration-call, not shared mutable state
- Registry mutex behavior unchanged

### NFR-4: Extensibility

- The chosen configuration pattern MUST support future options (timeouts, CORS, metrics) without breaking changes
- Adding a new option should require only adding a new function/field, not modifying existing API

### NFR-5: FP Principles

- No global mutable state introduced
- Configuration is immutable once applied
- Pure function composition preserved

## Acceptance Criteria

### AC-1: Existing Tests Pass
- All existing tests in `handler/context_test.go`, `handler/registry_test.go`, `handler/nullable_test.go`, and `middleware/` pass without modification
- `go test ./...` passes with zero failures

### AC-2: Services Injection Works
- A test handler registered via `RegisterWithRouter(r, db, logger, handler.WithServices(myServices))` receives `myServices` in `ctx.Services`
- Type assertion `ctx.Services.(MyType)` succeeds and returns the correct value

### AC-3: Nil Services When Not Configured
- Handlers registered via plain `RegisterWithRouter(r, db, logger)` have `ctx.Services == nil`
- No panic or unexpected behavior

### AC-4: Mixed Handlers
- A single registry with services configured — all handlers receive services regardless of whether they use them

### AC-5: Nil Services Passed Explicitly
- `WithServices(nil)` does not panic
- Handlers receive `nil` services (same as not configured)

### AC-6: AdaptHandler Backward Compatibility
- Existing direct calls to `AdaptHandler(db, logger, handler)` in tests continue to compile and work
- New `AdaptHandlerWithServices(db, logger, services, handler)` available for services-aware adaptation

### AC-7: Documentation Updated
- README updated with services injection usage example
- Code comments document the services field and configuration API

## Constraints

- Go 1.23+ (current minimum version)
- Changes limited to `handler/` package — no changes to `middleware/`, `db/`, `core/`, or `router/`
- Must work with chi router (current dependency)

## Out of Scope

- Moving `DB`/`Logger` into the services pattern (future consideration)
- Generic `Registry[S any]` approach (rejected — viral generics)
- Service validation/registration at startup (app responsibility)
- Service lifecycle management (app responsibility)
