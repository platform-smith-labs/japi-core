---
type: capability
title: "Secure CORS router"
tags: [router, cors, chi, http, security]
timestamp: 2026-07-07T02:32:18Z
description: "Builds the app's chi router with secure-by-default CORS (deny-all unless origins are explicitly allowed) plus baseline HTTP middleware"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - router/chi.go
  - router/chi_test.go
  - router/export_test.go
see_also:
  - {repo: japi-core, capability: "Route registry & registration", intent: "the returned chi router is passed to route registration to mount handlers", descriptive: false}
  - {repo: japi-core, capability: "Observability (request IDs, logging, Prometheus metrics)", intent: "add a request-id / logging middleware onto the returned router via .Use", descriptive: false}
---

# Secure CORS router

**What it does.** Constructs the chi router the whole HTTP app hangs off, wired with a
**secure-by-default CORS policy** — cross-origin browser requests are denied unless the caller
explicitly allow-lists origins — plus a baseline HTTP middleware stack.

**How a peer interacts.** Call one of three constructors, all returning a `chi.Router`:
- `router.NewChiRouter()` — deny-all CORS (no origin accepted).
- `router.NewChiRouterWithCORS(origins []string)` — allow the given origins, everything else default.
- `router.NewChiRouterWithOptions(opts ...RouterOption)` — full control via functional options:
  `WithAllowedOrigins`, `WithAllowedMethods`, `WithAllowedHeaders`, `WithExposedHeaders`,
  `WithAllowCredentials`, `WithMaxAge`. Unspecified options keep the secure defaults.

The returned value is a standard chi router: the caller mounts handlers on it (routes, sub-routers)
and can attach further middleware with `.Use(...)`.

**Observable behavior.** Every constructor attaches, in order, chi's `RequestID`, `RealIP`, and
`Recoverer` middleware, then the CORS handler — exactly once. With no allowed origins, a browser
preflight/cross-origin request receives no CORS grant headers (deny-all); with matching origins the
preflight is accepted and echoes the requested method/origin. Same-origin and non-browser callers
(server-to-server, curl) are unaffected by CORS regardless of config.

**Contract.** The functional options and their confirmed default values:

| Option | Configures | Default |
|---|---|---|
| `WithAllowedOrigins(origins)` | allowed cross-origin origins | `[]` (deny all) |
| `WithAllowedMethods(methods)` | permitted HTTP methods | `GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS` |
| `WithAllowedHeaders(headers)` | permitted request headers | `Accept, Authorization, Content-Type, X-CSRF-Token` |
| `WithExposedHeaders(headers)` | response headers exposed to the browser | `Link` |
| `WithAllowCredentials(bool)` | allow cookies / HTTP auth on cross-origin | `false` |
| `WithMaxAge(seconds)` | preflight cache lifetime (seconds) | `300` (5 min) |

Out: a `chi.Router`. No error return — construction does not fail.

**Invariants.** Secure default: empty origins list means deny all cross-origin requests — a caller
must opt in explicitly. The middleware stack and CORS handler are always installed, and installed
only once, by whichever constructor is used. Each option replaces (not merges with) its default list.

**Failure modes.** Constructors never error. A misconfiguration is silent: e.g. allowing origins but
narrowing `WithAllowedMethods` to exclude a method causes that method's preflight to be rejected
(the peer's browser sees a CORS failure, not a server error).

**Gotchas.**
- Passing `["*"]` to `WithAllowedOrigins` / `NewChiRouterWithCORS` re-opens the security hole by
  allowing any origin — deliberately not the default. Do not combine `["*"]` with
  `WithAllowCredentials(true)`.
- CORS only constrains **browsers**; it is not an auth mechanism and does not block server-to-server
  or CLI clients.
- `WithAllowedMethods`/`WithAllowedHeaders` **replace** the whole default list rather than adding to
  it — restate the full set you need, or you will silently drop the defaults.
- The returned router is just a chi router; the app is responsible for mounting handlers on it and
  adding any further middleware (auth, request logging) via `.Use`.

**See also.** Route registration (this repo) — the router is handed off to mount typed handlers.
Observability middleware (this repo) — add request-id/logging onto the returned router via `.Use`.
