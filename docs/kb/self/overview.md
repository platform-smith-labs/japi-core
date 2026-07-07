---
type: overview
title: "japi-core — type-safe Go REST/PostgreSQL framework"
tags: [japi-core, go-framework, rest-api, postgresql, library]
timestamp: 2026-07-07T02:32:18Z
description: "What japi-core is: the compile-time Go framework a consuming service builds its HTTP + DB layer on"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - README.md
  - CLAUDE.md
  - go.mod
  - CHANGELOG.md
  - docs/kb/self/extract/structure.md
---

# japi-core — overview

japi-core is a **Go framework library**, not a running service — it has no port, no
`main`, no deployable artifact. Consuming Go services import it
(`go get github.com/platform-smith-labs/japi-core/v3`, module major **v3**) and build their
REST API + PostgreSQL data layer on top of it. Within Platform Smith its consumers ("peers")
are the **orchestrator** and **ps-api** Go services; a change here ripples to both, so it is
usually bumped in lockstep with them.

## What a peer gets from it

A consuming service uses japi-core to define HTTP handlers as strongly-typed generic functions,
self-register them into a route registry, compose behavior (auth, parsing, validation, logging,
metrics) as functional middleware, run type-safe PostgreSQL queries with request-context
propagation, and expose auto-generated Swagger docs and Prometheus metrics — without writing the
plumbing itself. The headline guarantee is **compile-time type safety**: params, request body,
and response shapes are Go generic type parameters, so mismatches fail to compile rather than at
runtime.

## Package architecture (layered)

- **Layer 0 — foundation**: `core/` (API error and response types), `db/` (PostgreSQL connection
  pool, type-safe `QueryOne`/`QueryMany`/`Exec`, transaction wrapper with automatic rollback,
  health checks), `jwt/` (token generation/validation).
- **Layer 1**: `handler/` (the generic `Handler`/`HandlerContext` framework + route registry that
  handlers self-register into) and `router/` (Chi router setup).
- **Layer 2**: `middleware/` (standard HTTP + generic "typed" middleware: parsing, JWT auth,
  validation, request-ID, logging) and `swagger/` (OpenAPI docs generated from route metadata).
- `metrics/` supplies opt-in Prometheus HTTP instrumentation.

## Headline value

- **Generics-based compile-time type safety** — handlers are parameterized over param/body/response
  types; DB helpers scan rows into typed structs.
- **Self-registering routes** — a handler registers itself (with metadata) into a per-server
  registry; multiple independent registries/servers can coexist in one process.
- **Functional middleware composition** — middleware are higher-order functions chained in order.
- **Secure-by-default** — the router denies all cross-origin requests unless origins are explicitly
  allowed; prepared statements guard against SQL injection.
- **Context propagation** — the HTTP request context threads through handlers into DB queries and
  transactions, giving automatic cancellation, timeouts, and rollback on client disconnect.
- **Auto Swagger + Prometheus** — OpenAPI docs from route metadata; opt-in request-count / latency /
  in-flight metrics with automatic path normalization.
- **Nullable monad** — `Nullable[T]` for type-safe optional request-scoped values instead of raw
  pointers.

For specific interaction contracts and behaviors, see the KB `capabilities/` and `interfaces/`
concepts.

## Notes

- Primary DB interface is **pgx/v5**; consumers registering the `lib/pq` driver in tests now need
  **PostgreSQL 14+**.
- Module versions v3.0.0 and v3.1.0 are **retracted** (published without the required `/v3` path
  suffix); consumers must use a later v3 tag.
- License: UNKNOWN — the README leaves the license unspecified.
