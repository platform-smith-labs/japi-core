---
type: glossary
title: "japi-core framework glossary"
tags: [japi-core, glossary, go, framework, handlers, middleware]
timestamp: 2026-07-07T02:32:18Z
description: "One-line definitions of japi-core's core framework terms, for reading the rest of the KB"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - handler/types.go
  - handler/nullable.go
  - core/handler.go
  - middleware/typed/request.go
  - middleware/typed/auth.go
  - middleware/http/request_id.go
  - middleware/http/logging.go
  - middleware/http/contenttype.go
---

# japi-core framework glossary

Lookup for the framework's domain terms. Each entry names the concept and states, in a clause or two, what it is. japi-core is a compile-time Go generics library (v3) for building type-safe REST handlers on top of chi.

- **HandlerContext[Param, Body]** — the per-request struct passed to every handler; carries app dependencies (`DB`, `Logger`, `Services`) plus request-scoped data (typed `Params`, `Body`, headers, request ID) and auth identity (`UserUUID`, `CompanyUUID`), each held as a `Nullable` so a handler can tell "not populated" from "empty".

- **Handler[Param, Body, Response]** — the generic handler function signature: given a `HandlerContext[Param, Body]` plus the raw `http.ResponseWriter`/`*http.Request`, it returns a typed `Response` value and an `error`. Business handlers return data, not HTTP; error/serialization is handled by the framework.

- **Middleware[Param, Body, Response]** — a higher-order function `Handler -> Handler` of the same three type params; wraps a handler to enrich the context or short-circuit, composed by `MakeHandler` in reverse listing order so the last-listed middleware wraps the base handler most closely.

- **RouteInfo** — route metadata (HTTP method, path pattern, and optional Swagger summary/description/tags) supplied at registration time and stored for automatic route registration and API-doc generation.

- **Registry** — a thread-safe collection of pending routes for one server instance; handlers register into it, then `RegisterWithRouter` binds them all onto a chi router (optionally with injected services).

- **MakeHandler** — the composition entry point: takes the registry, a `RouteInfo`, a base handler, and a variadic list of middleware; composes the middleware chain (the first-listed middleware is outermost and runs first on the request path; the last-listed sits closest to the base handler and runs just before it), records the route in the registry, and returns the composed handler.

- **typed middleware** — generics-aware middleware in package `typed` that operates on `Handler[Param, Body, Response]` and can read/populate the strongly-typed `HandlerContext` (e.g. `ParseParams`, `ParseBody`, `RequireAuth`); applied via `MakeHandler`.

- **http middleware** — standard `func(http.Handler) http.Handler` middleware in package `http` (e.g. `WithRequestID`, `WithLogging`, `WithContentType`) that operates on the raw HTTP layer with no knowledge of the typed context; applied on the chi router, not through `MakeHandler`.

- **Nullable[T]** — a type-safe optional (value + presence flag) used for `HandlerContext` request-scoped fields; accessed via `Value() (T, error)`, `TryValue() (T, bool)`, `ValueOr(default)`, `ValueOrDefault()`, `HasValue()`. Preferred over `*T` in framework context fields; not used inside API model structs.

- **APIError** — the framework's structured error type (`Code`, `Message`, optional `Detail`, and per-field `Fields` for validation errors); when a handler returns one it is written as the HTTP response with that status code. Constructed via `NewAPIError` / `NewValidationError`; common presets include `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrInternal`.

- **Parse* family (typed middleware)** — request-parsing typed middleware that reads the raw request and populates the typed context, validating via struct tags (`validator`): `ParseParams` (URL path `param:"…"` + query `query:"…"` → `ctx.Params`), `ParseBody` (JSON body → `ctx.Body`), and `ParseHeaders` (→ `ctx.Headers`). Related response/format middleware in the same package: `ParseJSON`, `ParseCSV`.

- **RequireAuth** — typed auth middleware: validates the `Bearer` JWT from the `Authorization` header against a supplied secret, runs a caller-provided callback to confirm the user/company still exist, and on success sets `ctx.UserUUID` and `ctx.CompanyUUID`; returns 401 for missing/invalid tokens, while the failure status of the user/company check is whatever the caller's callback returns (consumer-defined).

- **WithServices** — a registration option (`WithServices(services any)`) passed to `RegisterWithRouter` to inject an app-defined dependency struct into every handler; the value is opaque to japi-core and surfaces on `HandlerContext.Services` for handlers to type-assert.
