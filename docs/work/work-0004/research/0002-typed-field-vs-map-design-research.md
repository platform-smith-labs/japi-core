# Typed Field vs map[string]any for Service Injection — Design Analysis

**Document ID**: 0002
**Work Item**: work-0004
**Created**: 2026-04-01
**Type**: Research
**Status**: Complete

## Overview

Should `registrationConfig` and `HandlerContext` use typed fields (`services any`) or a generic map (`map[string]any`) to avoid requiring japi-core code changes when consumers need to inject additional things?

## The Two Approaches

### Approach A: Typed Field (Current Plan)

```go
// registrationConfig (internal)
type registrationConfig struct {
    services any
    // Future: requestTimeout time.Duration ← requires japi-core code change
    // Future: corsConfig *CORSConfig       ← requires japi-core code change
}

// HandlerContext (public)
type HandlerContext[P, B any] struct {
    // ...existing fields...
    Services any  // Consumer puts ONE struct containing everything
}

// Consumer side — add fields to YOUR struct, zero japi-core changes
type AppServices struct {
    OrchClient   *httpclient.Client
    CacheClient  *redis.Client       // ← add anytime, no japi-core change
    EmailService *email.Service      // ← add anytime, no japi-core change
}
```

### Approach B: map[string]any (User's Proposal)

```go
// registrationConfig (internal)
type registrationConfig struct {
    extras map[string]any
}

// HandlerContext (public)
type HandlerContext[P, B any] struct {
    // ...existing fields...
    Extras map[string]any  // Multiple independent things by key
}

// Generic add function
func WithExtra(key string, value any) RegistrationOption {
    return func(cfg *registrationConfig) {
        if cfg.extras == nil {
            cfg.extras = make(map[string]any)
        }
        cfg.extras[key] = value
    }
}

// Consumer usage
registry.RegisterWithRouter(r, db, logger,
    handler.WithExtra("orchClient", orchClient),
    handler.WithExtra("cacheClient", cacheClient),
    handler.WithExtra("emailService", emailSvc),
)

// In handler — type assert every access
client := ctx.Extras["orchClient"].(*httpclient.Client)
cache := ctx.Extras["cacheClient"].(*redis.Client)
```

## Deep Analysis

### The Key Insight: What Actually Needs Extensibility?

There are **two separate extensibility concerns** that must be analyzed independently:

#### Concern 1: Consumer-defined dependencies (HTTP clients, caches, queues, etc.)

These are things **the consumer defines and only the consumer uses**. japi-core doesn't know or care what they are — it just carries them through.

**With typed field (`Services any`)**: The consumer defines their own struct and adds fields freely. **Zero japi-core changes** for adding a 2nd, 3rd, or 100th dependency:

```go
// Consumer adds fields anytime — no PR to japi-core needed
type AppServices struct {
    OrchClient   *httpclient.Client  // day 1
    CacheClient  *redis.Client       // added week 2
    EmailService *email.Service      // added month 3
    FeatureFlags *flipt.Client       // added later
}
```

**With `map[string]any`**: Same result — no japi-core changes. But now every access requires a string key + type assertion:

```go
client := ctx.Extras["orchClient"].(*httpclient.Client)  // string typo = nil panic
```

**Verdict for Concern 1**: `Services any` is **already infinitely extensible** on the consumer side. The map adds no extensibility benefit here — only risk.

#### Concern 2: Framework-level options (timeouts, CORS, metrics, etc.)

These are things **japi-core needs to understand and act on**. A timeout option changes how `RegisterWithRouter` wraps handlers. A CORS option adds middleware. The framework must interpret these.

**With typed field**: Yes, adding a new framework option requires a japi-core code change:
```go
type registrationConfig struct {
    services       any
    requestTimeout time.Duration  // ← new japi-core code
}
func WithRequestTimeout(d time.Duration) RegistrationOption { ... }  // ← new japi-core code
```

**With `map[string]any`**: The field can be added without changing the struct, but **`RegisterWithRouter` must still be changed** to read and act on it:
```go
// Still needs japi-core code to DO something with the timeout
if t, ok := cfg.extras["requestTimeout"].(time.Duration); ok {
    // wrap handlers with timeout
}
```

**Verdict for Concern 2**: The map avoids adding a struct field, but the framework still needs code to interpret and act on each option. The "no code change" benefit is illusory — you save 1 line (the field declaration) but still write the same logic.

### Detailed Tradeoff Analysis

| Dimension | Typed Field (`Services any`) | `map[string]any` |
|-----------|------------------------------|-------------------|
| **Consumer extensibility** | Infinite — consumer adds fields to their own struct | Infinite — consumer adds keys to map |
| **Framework extensibility** | Needs code change (field + option function + logic) | Needs code change (option function + logic) — saves only 1 line |
| **Type safety** | One assertion at boundary, then fully typed | Every access requires string key + type assertion |
| **IDE support** | Full autocomplete on consumer's struct | No autocomplete — string keys |
| **Typo safety** | Compile error if field name wrong | Runtime panic if key name wrong |
| **Discoverability** | `ctx.Services.(AppServices).` shows all fields | `ctx.Extras["???"]` — must check docs/source |
| **Key collisions** | N/A — one struct, one namespace | Risk if multiple packages use same key name |
| **Nil safety** | One nil check (`ctx.Services != nil`) | Must nil-check map AND each value |
| **Testing** | Construct struct literal with all fields | Build map with string keys, assert each |
| **FP principles** | Clean — one immutable value flows through | Mutable map, string-keyed lookups |
| **Performance** | Single struct assignment, one type assertion | Map allocation, N lookups, N type assertions |
| **Go community precedent** | `context.Context` uses typed keys for good reason | `gin.Context.Keys` uses map — widely criticized |

### What Go Community Best Practice Says

**`context.Context`** chose typed keys over `map[string]any` deliberately:

```go
// Go stdlib — typed keys, not string keys
type contextKey struct{ name string }
var userKey = contextKey{"user"}
ctx = context.WithValue(ctx, userKey, user)
user := ctx.Value(userKey).(User)
```

Why? Because string keys cause collisions, have no compile-time checking, and are a maintenance burden. The Go team explicitly designed `context.WithValue` to discourage string keys.

**`gin.Context.Keys map[string]any`** is the counter-example — and it's widely cited as one of Gin's design weaknesses. Handler code becomes littered with string constants and type assertions.

**`echo.Context.Set/Get`** also uses string keys — and the Echo team recommends typed middleware context instead for production code.

### The "No Code Change" Argument Examined

The appeal of `map[string]any` is "no japi-core code changes to add a new thing." Let's test this claim:

**Scenario: Consumer wants to inject a 2nd dependency (Redis client)**

| Step | Typed Field | map[string]any |
|------|------------|----------------|
| japi-core changes | **None** — consumer adds field to their `AppServices` struct | **None** — consumer adds a new `WithExtra("redis", client)` call |
| Consumer changes | Add `CacheClient *redis.Client` to `AppServices` | Add `WithExtra("redis", client)` to registration |
| Handler access | `svc.CacheClient.Get(...)` (typed, autocomplete) | `ctx.Extras["redis"].(*redis.Client).Get(...)` (string + assertion) |

**Result**: Both require zero japi-core changes. But the typed field gives compile-time safety.

**Scenario: japi-core wants to add a framework option (request timeout)**

| Step | Typed Field | map[string]any |
|------|------------|----------------|
| Add config field | Add `requestTimeout time.Duration` to `registrationConfig` | Skip (use map) |
| Add option function | `func WithRequestTimeout(d time.Duration) RegistrationOption` | `func WithRequestTimeout(d time.Duration) RegistrationOption` |
| Add logic | Read `cfg.requestTimeout` in `RegisterWithRouter` | Read `cfg.extras["requestTimeout"].(time.Duration)` in `RegisterWithRouter` |
| Total lines saved | 0 | 1 (the field declaration) |

**Result**: The map saves exactly 1 line of code per new framework option, at the cost of type safety.

### Risk Analysis of map[string]any

1. **Silent failures**: `ctx.Extras["orchClient"]` returns `nil` if key is misspelled — no compile error, runtime panic on type assertion
2. **No refactoring support**: Rename a key? Find-and-replace strings across the codebase. Miss one? Runtime crash.
3. **Documentation burden**: Every key-value pair must be documented separately. With a struct, the fields ARE the documentation.
4. **Concurrency risk**: Maps are not concurrency-safe in Go. The struct is a value copy per request — inherently safe.
5. **Testing friction**: Test setup requires building maps with magic strings instead of constructing typed structs.

## Honest Recommendation

**The typed field approach (`Services any`) is clearly better.**

The `map[string]any` approach solves a problem that doesn't exist:
- Consumer extensibility is already infinite with `Services any` — consumers add fields to their own struct
- Framework extensibility saves at most 1 line per option while losing type safety

The map approach introduces real problems:
- String-keyed access is error-prone and unrefactorable
- Every handler pays the cost of string lookups + type assertions on every access
- It violates the FP principles this codebase is built on (typed values > untyped bags)
- Go community consensus (context.WithValue design, criticism of gin.Keys) favors typed approaches

**The one legitimate concern** — "what if japi-core needs many framework options?" — is already solved by the functional options pattern. Each `WithX` function is one function + one field. This is the canonical Go approach and scales cleanly.

### When map[string]any WOULD Be Appropriate

- Plugin systems where the framework truly cannot predict what keys exist
- Configuration files parsed from YAML/JSON at runtime
- Middleware that needs to pass arbitrary metadata between unrelated packages

None of these apply to japi-core's service injection use case, where:
- The consumer knows exactly what services they have (it's their own code)
- The framework knows exactly what options it supports (it's the framework's API)

## Summary

| Question | Answer |
|----------|--------|
| Does `Services any` require japi-core changes to inject more dependencies? | **No** — consumer adds fields to their own struct |
| Does `map[string]any` provide more extensibility? | **No** — same extensibility, less safety |
| Does the map save framework code? | **~1 line per option** — not meaningful |
| Which approach is more idiomatic Go? | **Typed field** — per context.WithValue design precedent |
| Which approach is safer? | **Typed field** — compile-time field access vs runtime string lookup |
| Which aligns with japi-core's FP principles? | **Typed field** — typed values over untyped bags |

**Recommendation**: Keep the current plan's `Services any` field unchanged. It is the correct design.
