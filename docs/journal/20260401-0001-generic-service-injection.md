# Session Journal: Generic Service Injection for Handler Registry

**Date**: 2026-04-01
**Session ID**: 20260401-0001
**Type**: Feature Implementation (Full Lifecycle)
**Work Item**: work-0004

## Overview

Complete lifecycle of work-0004: researching, planning, and implementing generic service injection for japi-core's handler registry. This enables consumers to inject application-defined dependencies (HTTP clients, caches, etc.) into handlers without global mutable state.

## Session Flow

### Phase 1: Work Item Creation + Deep Analysis

- Read the source prompt from `platform-smith-api/docs/work/work-0006/japi-core-di-prompt.md`
- Validated all claims against the actual codebase:
  - `HandlerContext` has hardcoded `DB`/`Logger` only (`handler/types.go:21-40`)
  - `AdaptHandler` has 14 direct call sites in `handler/context_test.go`
  - DI pipeline is sealed with no extension point
- Evaluated the prompt's proposed builder pattern vs functional options
- **Decision**: Functional options is more idiomatic Go, less API surface (1 type vs 3), no new interfaces needed
- Created work-0004 with research and requirements documents

### Phase 2: Design Question — Typed Field vs map[string]any

User asked whether `registrationConfig` should use `map[string]any` instead of typed fields to avoid requiring japi-core code changes when adding new injectable things.

**Key insight**: The concern was based on a misconception. `Services any` is already infinitely extensible on the consumer side — consumers add fields to their own struct, zero japi-core changes. The `map[string]any` approach would only save ~1 line per framework option while losing type safety, IDE support, and refactoring safety.

Created research document `0002-typed-field-vs-map-design-research.md` with full tradeoff analysis.

### Phase 3: Implementation Plan

Created single-file master plan with 3 phases:
1. HandlerContext + Adapter Layer (Services field, internal adaptHandler, AdaptHandlerWithServices)
2. Registry + Functional Options (RegistrationOption, WithServices, RegisterWithRouter modification)
3. Tests + Documentation

### Phase 4: Implementation

All 3 phases implemented in sequence with verification after each:

**handler/adapter.go**:
- Extracted shared `adaptHandler` internal function (with `services any` param)
- `AdaptHandler` now delegates to `adaptHandler(db, logger, nil, handler)` — signature unchanged
- Added `AdaptHandlerWithServices` — delegates to `adaptHandler(db, logger, services, handler)`

**handler/types.go**:
- Added `Services any` field to `HandlerContext`
- Added `registrationConfig` struct (unexported), `RegistrationOption` type (public), `WithServices` function
- Added `AdaptWithServices` method on `TypedHandler` (structural typing, not a public interface)
- Modified `RegisterWithRouter` to accept `...RegistrationOption` — backward compatible

**handler/context_test.go**: 3 new tests (services populated, nil services, nil-by-default)

**handler/registry_test.go**: 3 new tests (WithServices injection, backward compat, WithServices(nil))

**README.md**: New "Service Injection" section with setup, handler usage, backward compat note

## Files Changed

| File | Lines | Change |
|------|-------|--------|
| `handler/types.go` | +55/-4 | Services field, RegistrationOption, WithServices, AdaptWithServices, RegisterWithRouter opts |
| `handler/adapter.go` | +30 net | Internal adaptHandler, AdaptHandlerWithServices, AdaptHandler delegates |
| `handler/context_test.go` | +61 | 3 new test functions for service injection |
| `handler/registry_test.go` | +90 | 3 new test functions for registry integration |
| `README.md` | +38 | Service Injection documentation section |
| `docs/work/index.md` | +3/-1 | Work item tracking |
| `docs/work/work-0004/` | new | Manifest, 2 research docs, 1 requirements doc, master plan |

## Key Technical Decisions

1. **Functional options over builder pattern** — canonical Go idiom (Dave Cheney/Rob Pike), 1 new type vs 3, no `ServiceAwareHandler` interface needed
2. **`Services any` over `map[string]any`** — consumer extensibility is infinite with a typed struct, map adds no benefit but loses type safety
3. **New `AdaptHandlerWithServices` over modifying `AdaptHandler` signature** — preserves 14 existing call sites, zero breaking changes
4. **Structural typing for `AdaptWithServices`** — anonymous interface check inside `RegisterWithRouter` avoids adding a public interface, falls back to `Adapt()` for third-party implementations

## Verification

```
go build ./...  — passes
go test ./...   — all existing + 6 new tests pass
go vet ./...    — clean
```

## Artifacts Created

- `docs/work/work-0004/manifest.md`
- `docs/work/work-0004/research/0001-generic-service-injection-research.md`
- `docs/work/work-0004/research/0002-typed-field-vs-map-design-research.md`
- `docs/work/work-0004/requirements/0001-generic-service-injection-req.md`
- `docs/work/work-0004/plans/master.md`

## Next Steps

- Commit the changes
- Update platform-smith-api to use `WithServices` instead of global `var + Init()` pattern
- Consider future `RegistrationOption` additions (timeouts, CORS) as needed

## Session Metrics

- **Commands run**: ~10 (build, test, vet, diff, git status)
- **Files changed**: 6 source files + 5 documentation files
- **Lines added**: ~270 (source) + documentation
- **Tests added**: 6 new test cases
- **Breaking changes**: Zero
