---
type: overview
title: "db-migration — owner of the platform_smith database schema"
tags: [db-migration, schema, postgresql, migrations]
timestamp: 2026-07-09T10:37:36Z
description: "The Go migration runner that defines and applies the platform_smith PostgreSQL schema every Platform Smith service depends on"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - README.md
  - CLAUDE.md
  - migrations/README.md
  - pkg/migration
  - cmd/migrator
---

# Overview

db-migration is the Go service that **owns the `platform_smith` PostgreSQL schema**. Every table,
enum, constraint, and index that Platform Smith services read or write is defined here — and only
here. No peer service creates or alters schema; they connect to a database whose shape this repo
has already established.

## What it does

- Applies ordered SQL migrations to the shared PostgreSQL database, executed in ascending
  filename order, each file wrapped in a transaction (rollback on failure) — except files
  carrying the no-transaction marker (see the enum gotcha), which run unwrapped.
- Tracks applied migrations idempotently in a `script_log` table — re-runs are safe and skip
  already-executed files.
- Runs **before** the services in the platform dependency chain, in one of two modes:
  - `kubernetes` (default): exits immediately after migrations — for init containers and Jobs.
  - `standalone`: waits for SIGTERM after migrations — for local Docker debugging.

## Role for peers

If your repo queries the `platform_smith` database, the authoritative definition of what exists
there lives in this repo's KB. Schema changes land here first (lockstep): a cross-repo feature
that adds tables or columns ships its migration in this repo before the consuming service code.
Applied migrations are immutable — all schema evolution is additive (new migrations that ALTER
and backfill), never edits to applied files.

## What this KB covers

- **This overview** — the runner and its place in the platform.
- [/self/context.md](/self/context.md) — who touches the schema, plus the ubiquitous data
  conventions (tenancy, dual keys, timestamps, naming) stated once so no other concept repeats them.
- [/self/glossary.md](/self/glossary.md) — domain vocabulary for reading the schema.
- Final-state schema reference under `/self/interfaces/` — the tables/enums as they exist after
  all migrations, grouped by domain.

## Freshness — when this KB was last synced

**This KB is a snapshot, not a live view.** The schema evolves rapidly; every concept describes
the database only up to the commit recorded in its `commit_sha` frontmatter, and the newest entry
in [/log.md](/log.md) is the bundle's authoritative **last-synced marker** (date + commit).
Migrations merged after that commit — including entirely new tables — are **not reflected here**.
Before relying on this reference for a schema-sensitive task, check the last-synced marker; if the
repo has migrations newer than it, treat the KB as stale for those areas and ask for a
regeneration. See the staleness gotcha in [/self/gotchas/](/self/gotchas/index.md).
