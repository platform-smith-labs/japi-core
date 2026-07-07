---
type: gotcha
title: "Each migration file is atomic, but the run as a whole is not"
tags: [migrations, transactions, failure-modes, idempotency]
timestamp: 2026-07-07T01:02:42Z
description: "A failure mid-run leaves earlier files committed and recorded; the failed file is not recorded, and the next run resumes from it"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - pkg/migration/migration.go
---
The runner executes each pending file in its own transaction (SQL + the tracking-table
record commit together) and **stops at the first failure**. There is no run-level
transaction and no rollback of previously completed files.

Consequences a peer must expect:
- After a failed run the database sits at an **intermediate schema version**: every file
  before the failure is committed and recorded; the failing file and everything after it
  are not applied.
- The failed file leaves **no tracking record**, so the next run re-selects it and resumes
  from exactly that point — recovery is simply "fix the file (or the data) and re-run".
- Services gated on migration completion (init-container / run-once patterns) will not
  start against the intermediate state, but anything inspecting the database directly may
  observe it.
