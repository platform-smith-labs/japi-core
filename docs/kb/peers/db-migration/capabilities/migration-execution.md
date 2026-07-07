---
type: capability
title: "Migration execution"
tags: [migrations, postgres, schema, lifecycle]
timestamp: 2026-07-07T01:02:42Z
description: "How the platform database schema gets applied — ordering, idempotency, failure semantics"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - pkg/migration/migration.go
  - cmd/migrator/main.go
  - pkg/config/config.go
  - migrations/README.md
  - Dockerfile
  - build.sh
  - README.md
---

# Migration execution

**What it does.** Applies the Platform Smith PostgreSQL schema by running the repo's ordered SQL
migration files against the platform database, exactly once each. It is the single owner of schema
evolution: every other service assumes the schema is whatever the latest migration set defines.

**How a peer interacts.** Not an API — a run-to-completion container. Peers sequence themselves
*after* it: as a Kubernetes init container/Job, a compose service with a completed-successfully
dependency, or a plain `docker run` before starting services. A developer in another repo gets a
schema change applied by adding a new numbered SQL file to this repo's `migrations/` directory
(numeric prefix + snake_case name, e.g. `0100_add_thing.sql`) and re-running the container.

**Observable behavior.**
- On start it ensures the `script_log` tracking table exists, discovers all `.sql` files, sorts them
  by filename (lexicographic — the numeric prefix is the ordering contract), and computes the
  unexecuted set as files minus `script_log` entries (a single SQL EXCEPT query).
- Each unexecuted file runs inside one transaction together with its `script_log` insert, so a file
  either fully applies and is recorded, or rolls back entirely. Exception: a file carrying an
  `@no-transaction` marker in its first 10 lines (required for `ALTER TYPE … ADD VALUE`) runs
  outside a transaction; its `script_log` record is written afterwards in a separate transaction.
- Completion signal: process exit code 0 and a log line that all migrations completed; it also
  writes a JSON status report to `/tmp/migration-status.json` inside the container (success flag,
  timings, error text).
- Exit behavior is mode-dependent (`EXECUTION_MODE`): `kubernetes` (default) exits immediately
  after the run — exit 0 on success, 1 on failure; `standalone` keeps the process alive after the
  run (resources released) until SIGTERM/SIGINT, then exits 0/1. SIGTERM during a run exits 130.

**Contract.** Inputs: env vars only — `DB_HOST`, `DB_USER`, `DB_NAME` (required), `DB_PORT`,
`DB_PASSWORD` (required when `ENVIRONMENT=production`), `DB_SSL_MODE` (must not be `disable` in
production), pool tuning (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_MAX_LIFETIME`,
`DB_MAX_IDLE_TIME`), `LOG_LEVEL`, `LOG_FORMAT`, `ENVIRONMENT`, `EXECUTION_MODE`. Invalid or missing
config fails validation and the process exits 1 before touching the database. Output: mutated
schema + `script_log` rows; exit code is the success signal. Migration files themselves must carry
a header comment (migration name + purpose) and be idempotent (`IF NOT EXISTS` / `IF EXISTS`).

**Invariants.**
- Exactly-once per filename: an executed file is recorded in `script_log` and skipped on every
  re-run — re-running the container against an up-to-date database is a safe no-op.
- Strict ascending filename order; a failing file halts the run, so no later file ever applies
  before an earlier one.
- Transactional atomicity per file (except `@no-transaction` files).

**Failure modes.**
- A failing migration stops the run at that file: everything before it is committed and recorded
  (partial application by file, never within a file); the failing file is rolled back and NOT
  recorded, so the next run retries it first. Peer observes exit code 1 (init container/Job
  failure blocks dependent services from starting).
- Database unreachable → exit 1 without running anything.
- A failing `@no-transaction` file can leave partial statement effects committed AND unrecorded —
  the whole file re-runs next time, which is why such files should contain a single idempotent
  statement.

**Gotchas.**
- Tracking is by **filename only — no checksum**. Editing an already-applied migration is silently
  skipped forever on populated databases; all post-cutover schema changes must be new additive
  files that `ALTER` + backfill.
- Discovery recurses into subdirectories of `migrations/` but records only the basename — never
  organize migrations into subfolders (they would collide/re-run).
- In `standalone` mode the process does not exit on its own, even after failure — orchestration
  that waits for exit will hang unless it sends SIGTERM. Use `kubernetes` mode for any
  "wait for completion" dependency.
- Lockstep ordering across repos: for a cross-repo feature the migration here must be applied
  before the consuming service version that needs the new schema is deployed.

**Business-critical data.** `script_log` (`script_name` unique, timestamps) — the execution ledger;
deleting a row causes that file to re-run (safe only if the file is truly idempotent). All other
tables are *products* of the migrations, not dependencies.

**See also / peers.** UNKNOWN — consuming services (orchestrator, ps-api, controller) depend on the
resulting schema, but no specific peer capability is verifiable from this repo.
