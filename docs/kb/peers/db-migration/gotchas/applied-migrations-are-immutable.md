---
type: gotcha
title: "Editing an applied migration is a silent no-op"
tags: [migrations, tracking, idempotency, additive-only]
timestamp: 2026-07-07T01:02:42Z
description: "Migrations are tracked by filename only (no checksum), so editing an already-applied file never re-executes on populated databases; all schema change is via new additive migrations"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/README.md
  - pkg/migration/migration.go
  - docs/dev/decisions/additive-migrations-after-cutover.md
---
The runner tracks completed migrations by **filename only — there is no content checksum**.
Once a file has run against a database, editing it in place is silently skipped there
forever: the change takes effect only on fresh databases, quietly forking schema between
new and existing environments.

House rule: applied migrations are **immutable**. Any schema change ships as a **new
additive migration** (a fresh, higher-sorting file that `ALTER`s and backfills existing
rows), never as an edit to an existing file.

Related trap: an archived pre-cutover migration history exists **outside** the active
migrations directory. Never move archived files back in — the runner recurses into
subdirectories and would re-execute them by basename on a fresh database.
