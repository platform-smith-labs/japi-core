---
type: capability
title: "Observability (request IDs, logging, Prometheus metrics)"
tags: [observability, request-id, logging, metrics, prometheus, tracing, middleware]
timestamp: 2026-07-07T02:32:18Z
description: "Request-ID correlation, structured per-request logging, and Prometheus HTTP metrics as opt-in middleware"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - middleware/http/request_id.go
  - middleware/http/logging.go
  - middleware/http/contenttype.go
  - middleware/typed/request_id.go
  - middleware/typed/logging.go
  - metrics/prometheus.go
see_also:
  - {repo: japi-core, capability: "Typed middleware pipeline", intent: "typed.* observability middleware compose inside MakeHandler"}
  - {repo: japi-core, capability: "CORS and router", intent: "http.* middleware register on the chi router"}
---

# Observability (request IDs, logging, Prometheus metrics)

**What it does.** Provides three opt-in facilities a consumer service wires onto its chi router: a
request-ID for cross-service correlation, structured per-request logging with timing, and Prometheus
HTTP metrics. All three are library helpers — nothing is on by default.

**How a peer interacts.** Two layers, distinguished by where they attach:
- `http.*` middleware attach to the chi router via `.Use(...)` and operate on the raw
  `http.Handler` chain: `http.WithRequestID()`, `http.WithLogging(logger)`.
- `typed.*` middleware compose inside `MakeHandler(...)` and enrich the typed `HandlerContext`:
  `typed.WithRequestID`, `typed.WithLogging`. Their type parameters are inferred from `MakeHandler`
  — pass them bare (no `()`, no explicit type args).
- Metrics is enabled once on the router: `metrics.EnablePrometheusMetrics(r, "/metrics")`, or
  `metrics.EnablePrometheusMetricsWithOptions(r, path, MetricsOptions{...})` to tune it.

**Observable behavior.**
- *Request IDs.* `http.WithRequestID()` reads the `X-Request-ID` request header; if absent it
  generates a new UUID. It stores the value in the request context and echoes it back on the
  response as `X-Request-ID`. `typed.WithRequestID` then lifts that value into `HandlerContext.RequestID`
  and adds a `request_id` field to `ctx.Logger`. The header is externally observable — a peer reads
  it off the response and propagates it on downstream calls to correlate a request across services.
- *Logging.* `http.WithLogging(logger)` emits a structured log at request start and one at response
  end (with captured status). `typed.WithLogging` logs start and end using `ctx.Logger` (so it
  carries the `request_id` when request-ID middleware ran first), including duration and, on handler
  error, an error-level line. Logs are `log/slog` structured records.
- *Metrics.* The metrics middleware records, per request: total count, latency, and a live in-flight
  gauge. Values are exposed in Prometheus text format at the configured path (default `/metrics`),
  scraped by a Prometheus server.

**Contract.**
- Header: `X-Request-ID` (request in, response out).
- Metrics (with the default `http` namespace, empty subsystem):
  - `http_requests_total{method,path,status}` — counter.
  - `http_request_duration_seconds{method,path}` — histogram; default buckets
    `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10]` seconds.
  - `http_requests_in_flight` — gauge (no labels).
  - The `path` label is the **chi route pattern** (`/users/{id}`), not the raw request path
    (`/users/123`), to bound cardinality; if no route pattern is available it falls back to the raw
    path.
  - `MetricsOptions` fields: `DurationBuckets`, `Namespace` (default `http`), `Subsystem` (default
    empty). Namespace/subsystem prefix all three metric names.
- Endpoint: the metrics text-format exposition served at the path passed to
  `EnablePrometheusMetrics` (peers commonly use `/metrics`).

**Invariants.**
- A request always carries an `X-Request-ID` on the response once `http.WithRequestID()` is applied —
  provided by the caller or minted here.
- Metric label cardinality is bounded by the router's registered route patterns, not by raw URLs.
- Metrics register on `prometheus.DefaultRegisterer` (production path); enabling metrics twice in one
  process double-registers and panics.

**Failure modes.**
- `typed.WithRequestID` with no upstream `http.WithRequestID()` finds no request ID and leaves
  `ctx.RequestID` empty and the logger un-enriched (no error) — correlation is silently lost.
- Re-registering the same metric names on the default registerer panics at startup (duplicate
  collector registration).

**Gotchas.**
- All three facilities are **opt-in** — a peer that doesn't wire them gets no request IDs, no request
  logs, and no `/metrics` endpoint.
- Apply request-ID middleware **early** in the chain (before logging) so the request ID is present
  when logs and the typed context are built.
- The `/metrics` endpoint is unauthenticated as provided — the consumer must place it behind network
  policy or auth; do not expose it publicly.
- Structured request logs use the raw `r.URL.Path`; metrics use the normalized route pattern — the two
  differ intentionally (logs are per-request, metrics must stay low-cardinality).
- Avoid adding high-cardinality metric labels (user IDs, raw paths) — the built-in label sets are
  deliberately narrow.

**See also / peers.**
- japi-core — *Typed middleware*: how `typed.WithRequestID` / `typed.WithLogging` compose inside
  `MakeHandler` with type inference.
- japi-core — *CORS and router*: the chi router that `http.*` middleware and metrics register onto.
