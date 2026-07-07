---
type: capability
title: "Route registry & registration"
tags: [routing, registry, chi, service-injection, framework]
timestamp: 2026-07-07T02:32:18Z
description: "How handlers self-register into a Registry and bind onto a chi router with shared DB, logger, and optional injected services"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - handler/types.go
  - handler/adapter.go
  - handler/registry_test.go
  - README.md
see_also:
  - {repo: japi-core, capability: "Typed handler framework", intent: "the typed handler/middleware model whose values self-register here"}
  - {repo: japi-core, capability: "Swagger generation", intent: "reads the same Registry to emit OpenAPI docs"}
---

# Route registry & registration

**What it does.** Collects route registrations into a `Registry`, then binds them all onto a chi
router in one call, injecting the shared `*sql.DB` and logger (plus an optional application-defined
services value) into every handler. One `Registry` per server or package lets an app run multiple
independent servers with isolated route sets.

**How a peer interacts.** A consuming Go service:
1. Creates a registry: `registry := handler.NewRegistry()` (conventionally a package-level `var
   Server = handler.NewRegistry()`).
2. Declares handlers with `handler.MakeHandler(registry, RouteInfo{...}, baseHandler, middleware...)`
   — each call appends its route to that registry as the value is initialised.
3. Binds them: `registry.RegisterWithRouter(r, db, logger)`, optionally with
   `handler.WithServices(appServices)` to inject application dependencies.

**Observable behavior.** Registration happens as `MakeHandler` values are initialised — i.e. at
package-init time, so a handler only registers if its package is actually imported/initialised.
`RegisterWithRouter` walks the collected routes and maps each `RouteInfo.Method`/`Path` onto the
corresponding chi verb (GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS). When `WithServices` is supplied,
the injected value is exposed on each request's `HandlerContext.Services` (typed `any`); handlers
recover their concrete type with a single type assertion at the boundary. Without the option,
`ctx.Services` is `nil` — fully backward-compatible.

**Contract.** `NewRegistry()` → `*Registry`. `MakeHandler(reg, routeInfo, baseHandler,
middleware...)` returns the composed handler and records a pending route on `reg`.
`RegisterWithRouter(r chi.Router, db *sql.DB, logger *slog.Logger, opts ...RegistrationOption)` binds
all pending routes. The only option today is `WithServices(any)`. Registration is concurrency-safe
(the registry guards its route list). Unsupported HTTP methods are silently skipped (no route bound).

**Invariants.** Each registry is independent — routes in one never leak into another. `db`, `logger`,
and any injected services are shared by reference across every handler bound in that call. Routes are
bound at `RegisterWithRouter` time; the registry itself does no HTTP serving.

**Failure modes.** A handler whose package is never imported never registers → its route silently
404s. A `RouteInfo.Method` outside the supported verb set is ignored (no binding, no error). Duplicate
or conflicting paths are the caller's responsibility — resolution is chi's, not the registry's.

**Gotchas.**
- Handlers only register if their package is imported/initialised — a peer that keeps handlers in a
  separate package must blank-import it (`import _ "your-app/handlers"`) so init runs.
- `go run` compiles registrations at build time; adding a new handler needs a rebuild/restart, not
  just a re-request, before the route appears.
- `WithServices` takes an app-defined value opaque to japi-core; the typed accessor (a type assertion)
  lives in the consuming app, and a wrong assertion panics at request time.
- `db` may be `nil` at registration (tests do this); handlers must not assume it is non-nil.

**See also / peers.** The values registered here come from japi-core's **Handler framework**
(typed handlers + middleware composition). japi-core's **Swagger generation** consumes the same
`Registry` (its collected routes and `RouteInfo` metadata) to emit API documentation, so both the
live router and the docs derive from one source of truth.
