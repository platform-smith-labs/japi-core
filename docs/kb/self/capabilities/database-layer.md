---
type: capability
title: "Type-safe database layer"
tags: [database, postgres, connection-pool, generics, transactions, context]
timestamp: 2026-07-07T02:32:18Z
description: "Pooled PostgreSQL connection plus generic, struct-scanning query and transaction helpers with context propagation"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - db/connection.go
  - db/query.go
  - db/query_context_test.go
see_also:
  - {repo: japi-core, capability: "Typed handler framework", intent: "handlers reach these helpers via ctx.DB and ctx.Context"}
  - {repo: japi-core, capability: "Error & response model", intent: "classifies DB errors, e.g. IsUniqueConstraintError, into HTTP responses"}
---

# Type-safe database layer

**What it does.** Wraps Go's `database/sql` for PostgreSQL: opens a validated, pooled connection
(via the pgx stdlib driver) and offers generic, struct-scanning query and transaction helpers. This
is a mechanism, not a schema — japi-core owns **no tables of its own**; the consuming service owns
all schema and supplies every SQL string.

**How a peer interacts.** Open once at startup with `db.Connect(db.Config{...})`, which returns a
`*sql.DB`. Then call the package-level helpers — the query/transaction helpers take `ctx` as the
FIRST argument:
- `db.QueryOne[T](ctx, querier, sql, args...)` → one row scanned into `T`
- `db.QueryMany[T](ctx, querier, sql, args...)` → `[]T`
- `db.Exec(ctx, querier, sql, args...)` → `sql.Result` (no rows)
- `db.WithTx(ctx, db, func(txCtx, tx) (T, error) {...})` → run work in a transaction
- `db.HealthCheck(db)` → liveness ping (no `ctx` arg; uses its own short timeout)

The `querier` param is a `Querier` interface satisfied by **both** `*sql.DB` and `*sql.Tx`, so the
same query helpers run inside or outside a transaction. Result rows scan into a struct `T` by its
`db:"column"` field tags.

**Observable behavior.** Every operation honors its context for cancellation and timeout: a client
disconnect or a deadline cancels the in-flight query and returns a context error. `WithTx` begins a
transaction, runs the callback, then **commits on a nil error** and **rolls back on any error or on
context cancellation**; a panic inside the callback triggers rollback and is re-raised. `Connect`
eagerly pings the database before returning, so a bad DSN or unreachable host fails fast at startup.

**Contract.**
- `Config` fields (complete): `Host`, `Port`, `User`, `Password`, `Database`, `SSLMode`,
  `MaxOpenConns`, `MaxIdleConns`, `MaxLifetime` (`time.Duration`), `MaxIdleTime` (`time.Duration`).
- Type parameter `T` is the caller's row struct; columns bind via `db:` tags. `T` may be a pointer
  or value type.
- Outputs: the scanned value(s) or `sql.Result`; a non-nil error on failure.
- Pool defaults applied when a field is zero: `MaxOpenConns=25`, `MaxIdleConns=25`,
  `MaxLifetime=5m`, `MaxIdleTime=5m`.

**Invariants.** All SQL runs as parameterized statements (positional `$1, $2, …` args) → no string
interpolation, SQL-injection-safe. A `WithTx` body is atomic: all-or-nothing commit. `MaxIdleConns`
must not exceed `MaxOpenConns` — violating it makes `Connect` return a validation error.

**Failure modes.** `QueryOne` on no rows returns `sql.ErrNoRows` (check with `errors.Is`). A
cancelled or timed-out context surfaces `context.Canceled` / `context.DeadlineExceeded`. `Connect`
returns an error for an invalid config (idle > open) or a failed open/ping. `WithTx` returns the
callback's error after rolling back; if the rollback itself fails, both errors are reported.

**Gotchas.** Context is a REQUIRED first argument on every operation (since v2) — a v1-style call
without `ctx` will not compile. Pick the `querier`: pass the `*sql.DB` for standalone queries, or the
`*sql.Tx` handed to your `WithTx` callback to keep work inside the transaction. Constraint violations
(e.g. unique) arrive as driver errors from `Exec`/`QueryOne`; classify them via the error response
model rather than string-matching.

(No "Business-critical data" section: japi-core owns no tables — the consuming service owns all
schema.)

**See also / peers.** japi-core **Handler framework** (handlers obtain a querier and context through
`ctx.DB` / `ctx.Context`). japi-core **Error response model** (maps DB/constraint errors, such as
`IsUniqueConstraintError`, into HTTP responses).
