---
type: gotcha
title: "Cross-cutting integrator traps"
tags: [gateway, auth, cors, proxy, timeouts, errors]
timestamp: 2026-07-07T03:33:49Z
description: "Traps that span ps-api routes: token fallback, 404-for-both, CORS authority, write timeout, proxy flavors, error envelope"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/stream_auth.go
  - cmd/handlers/passthrough.go
  - cmd/handlers/verbatim.go
  - cmd/handlers/sessions.go
  - cmd/handlers/session_sse.go
  - cmd/handlers/environments.go
  - pkg/config/config.go
  - cmd/handlers/port_bindings_test.go
---

# Cross-cutting integrator traps

- **`?token=` JWT query fallback on stream/passthrough routes.** SSE, WebSocket,
  and reverse-proxied passthrough routes accept the JWT either as
  `Authorization: Bearer` or as a `?token=` query parameter — browsers cannot set
  custom headers on EventSource/WebSocket. Browser clients need the query form;
  server-side callers should prefer the header (query tokens can leak into
  logs/URLs). Both forms get the same validation as that route's header path:
  two-layer (signature/expiry plus a DB company-membership check) on SSE and
  passthrough routes; the terminal WebSocket is JWT-only — see the Auth and
  identity gateway capability.

- **404-for-both.** Across resource reads, an unknown resource and a resource
  belonging to another tenant return the same 404 with the same body shape.
  A peer cannot distinguish "doesn't exist" from "not yours" — this is
  deliberate anti-enumeration, not a bug to route around.

- **ps-api's router is the single CORS authority.** Reverse-proxied responses
  have the upstream's `Access-Control-Allow-Origin`,
  `Access-Control-Expose-Headers`, and `Vary` headers stripped before ps-api's
  own CORS middleware adds its set. Adding or changing CORS headers in an
  upstream service has no effect on what the browser sees; without the strip,
  duplicated headers make Chrome reject the response even at status 200.

- **10s server-wide write timeout.** The HTTP server's write timeout defaults to
  10 seconds (`SERVER_WRITE_TIMEOUT`). Any non-SSE response that streams longer
  than that gets cut mid-body. SSE handlers explicitly disable the timeout at
  stream upgrade; a peer expecting a long-lived response on any other route must
  fit the 10s budget.

- **Two proxy flavors shape responses differently.** Typed proxies decode the
  upstream body into a declared response type and **silently drop** unknown
  upstream fields; verbatim relays forward status + body byte-for-byte,
  including non-2xx error bodies. A peer adding a field to an upstream response
  must confirm which flavor carries the route, or the new field never reaches
  the client.

- **Errors are japi-core-shaped.** ps-api-generated errors use the japi-core
  envelope — a JSON `error` object, key fields: `code`, `message`, optional
  `detail`. Exact wire shape: UNKNOWN — TODO: defined in the external japi-core
  module, confirm there. Verbatim-proxied routes forward the upstream error body
  unchanged, so some errors on ps-api's surface are upstream-shaped instead
  (e.g. port-conflict responses where `error` is a string discriminator).
