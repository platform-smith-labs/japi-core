---
type: context
title: "japi-core system context"
tags: [japi-core, go-library, framework, context, postgres, chi]
timestamp: 2026-07-07T02:32:18Z
description: "Who consumes japi-core and the ubiquitous facts that hold across every capability"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - README.md
  - CLAUDE.md
  - go.mod
  - db/connection.go
see_also:
  - {repo: orchestrator, capability: "HTTP API framework usage", intent: "imports japi-core to build its REST API + DB layer", descriptive: true}
  - {repo: ps-api, capability: "Gateway HTTP framework usage", intent: "imports japi-core to build its REST API + DB layer", descriptive: true}
---

# japi-core system context

japi-core is a **compile-time Go framework library**, not a running service. It has **no
process and no network port**. Consumers pull it in as a Go module —
`go get github.com/platform-smith-labs/japi-core/v3` — and import subpackages under
`github.com/platform-smith-labs/japi-core/v3/...`. All interaction is by linking against it
and calling its exported types/functions at build time; there is nothing to reach over the
wire. The module is at **major version v3** (import path MUST carry the `/v3` suffix — the
`v3.0.0`/`v3.1.0` tags are retracted for omitting it).

## Who consumes it

Its "clients" are the consuming service's own code, in two roles:

- **The service `main()`** — wires the pieces together at startup: opens the DB pool
  (`db.Connect`), constructs a chi router, and registers a handler registry against that
  router (`RegisterWithRouter`), optionally injecting an app-specific services struct.
- **The service's handler packages** — declare typed handlers (`MakeHandler`) that
  self-register into a registry, and run business logic inside a per-request
  `HandlerContext` (DB handle, logger, parsed params/body, auth identity, request ID).

Within Platform Smith the consumers are **orchestrator** and **ps-api** (both Go services).
Because those two services build their REST APIs on top of this framework, a change here
ripples to both, and they should be bumped/verified in lockstep.

## Ubiquitous facts (stated once here; capabilities assume them)

- **Go generics throughout.** Handlers, queries, and the optional wrapper are generic, so
  type checking happens at compile time. Requires the Go toolchain declared in `go.mod`.
- **HTTP layer is the chi router** (`go-chi/chi/v5`). All routing, middleware chaining, and
  path-pattern matching go through chi.
- **DB layer wraps Go's standard `database/sql`** over PostgreSQL, using the `pgx` stdlib
  driver, with a configurable connection pool (sensible defaults: 25 open / 25 idle /
  5-minute lifetime / 5-minute idle). Row scanning uses `scany`.
- **japi-core owns NO database schema and NO tables.** The consuming service owns all
  tables; the framework only provides query/transaction helpers that run the service's own
  SQL. **Therefore capability concepts here have no data section** — there is no
  business-critical table for the framework itself to name.
- **`context.Context` is threaded through every handler and every DB call.** The HTTP
  request context is set on `HandlerContext.Context` and passed as the first argument to all
  DB operations, giving automatic cancellation, timeout propagation, and transaction
  rollback on client disconnect.
- **Secure-by-default posture.** The default router denies all cross-origin requests (CORS
  allow-list is empty until the consumer opts specific origins in); the DB helpers use
  parameterized/prepared statements, so callers pass SQL args positionally rather than
  interpolating.

UNKNOWN — the exact orchestrator/ps-api capability names that consume this framework are not
grounded in this repo's own evidence (the peer names in `see_also` are best-guess
placeholders for kb-sync to reconcile).
