---
type: capability
title: "Typed handler framework"
tags: [handler, generics, http, framework, type-safety, go]
timestamp: 2026-07-07T02:32:18Z
description: "Compile-time-typed generic HTTP handler system: define a handler as a function over param/body/response types and let the framework parse, validate, and adapt it to net/http"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - handler/types.go
  - handler/adapter.go
  - handler/context_test.go
  - core/handler.go
  - middleware/typed/request.go
see_also:
  - {repo: japi-core, capability: "Route registry & registration", intent: "collects composed handlers and mounts them on the chi router", descriptive: false}
  - {repo: japi-core, capability: "Typed middleware pipeline", intent: "populates the handler context (params, body, auth, response) in composed order", descriptive: false}
  - {repo: japi-core, capability: "Nullable optional type", intent: "the wrapper type every request-scoped context field arrives in", descriptive: false}
  - {repo: japi-core, capability: "Error & response model", intent: "the APIError type a handler returns to control HTTP status/body", descriptive: false}
---

# Typed handler framework

**What it does.** Lets a consumer service define an HTTP handler as a single strongly-typed
function over three type parameters — URL/query params `P`, request body `B`, response `R` — while
the framework handles parsing, validation, dependency wiring, and adaptation to standard
`net/http`. Type mismatches between a handler, its middleware, and its response are caught at
compile time rather than at runtime.

**How a peer interacts.** Call `handler.MakeHandler(registry, handler.RouteInfo{Method, Path, ...},
handlerFn, middleware...)`. `handlerFn` has the shape
`func(ctx handler.HandlerContext[P, B], w http.ResponseWriter, r *http.Request) (R, error)`.
`MakeHandler` composes the middleware around the handler and records the route in the registry;
the registry later adapts each route to an `http.HandlerFunc` and mounts it on a chi router
(optionally injecting application services via `handler.WithServices`).

**Observable behavior.** Middleware wrap the base handler and execute in composed order to enrich
`HandlerContext` before the handler body runs (params parsed, body parsed, auth resolved). The
handler reads its inputs off the context and returns `(R, error)`. On a nil error the response is
**not** written by the framework — writing the success response is delegated to a response
middleware in the chain (e.g. the JSON responder). On a non-nil error the adapter writes the error
response: a returned `*core.APIError` maps to its own status/body; `context.Canceled` writes
nothing (client gone); `context.DeadlineExceeded` yields 504; any other error yields 500.

**Contract.** In: a `RouteInfo` (key fields: `Method`, `Path`, plus optional `Summary`,
`Description`, `Tags` for docs), the typed handler function, and a variadic middleware chain (all
sharing the same `[P, B, R]`). The handler reads request-scoped data from `HandlerContext[P, B]`
(key fields: `Context`, `DB`, `Logger`, `Services`, `Params`, `Body`, `BodyRaw`, `Headers`,
`RequestID`, `UserUUID`, `CompanyUUID`). Out: the typed response `R` (written by a response
middleware) or an `error` — return `*core.APIError` to control HTTP status and message.

**Invariants.** Handler, middleware, and response types are unified at compile time — a middleware
can only be applied to a handler with matching type parameters. `ctx.Context` is always non-nil and
is the request's own context (cancellation/timeout/trace values propagate). Middleware execute in
list order: the **first** middleware passed runs outermost/first, the last passed sits innermost
(runs just before the base handler), and the base handler runs last.

**Failure modes.** A handler that reads `ctx.Body` or `ctx.Params` without the corresponding parse
middleware in its chain sees an absent (Nil) value, not parsed data. Parse/validation failures
surface as `*core.APIError` (e.g. 400 for a missing required body or invalid JSON) returned before
the handler body runs. A returned plain error (non-APIError) is logged and collapsed to a generic
500 — detail is not exposed to the client.

**Gotchas.** Every request-scoped context field is a `Nullable[T]`, not a bare value — accessing it
requires the Nullable accessors and handling the absent case; a field is only populated if a
middleware set it (body absent until the body-parse middleware runs; `UserUUID`/`CompanyUUID` absent
until the auth middleware runs). Success-response writing lives in middleware, so a handler chain
with no response middleware returns 200 with an empty body. `struct{}` as `P` or `B` is the
framework's signal for "no params / no body" — the parse middleware short-circuit on it.

**See also.** Route registry (mounting), Typed middleware (context population + response writing),
Nullable optional (context field wrapper), Error response model (`APIError`) — all in japi-core.
