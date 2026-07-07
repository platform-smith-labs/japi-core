---
type: gotcha
title: "Some migrations are data backfills or repairs, not schema changes"
tags: [migrations, backfill, data-repair]
timestamp: 2026-07-07T01:02:42Z
description: "One-off migrations exist that update or delete rows (backfilling new columns, repairing bad values, dropping constraints with cleanup); the KB schema reference documents final schema only"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0017_runtime_instance_backfill_launch_status.sql
  - migrations/0024_project_source_type_repair.sql
  - migrations/0028b_runtime_drop_all_rows_service_unique.sql
---
Not every migration adds schema. Because applied migrations are additive-only, introducing
a column or tightening semantics on a populated database is done as a pair: an `ALTER` plus
a follow-up **data backfill** that settles existing rows (e.g. deriving a new status column's
value from related state). There are also one-off **repair** migrations that fix bad data,
and **destructive cleanups** that drop a constraint so a later migration can recreate it in
a narrower form.

The trap: reasoning about this repo as "schema history only". Row contents on an existing
database reflect these backfills, and the KB schema reference documents the **final schema
state** — the transitional data-manipulation steps are invisible there. If historical row
values look inconsistent with what current application code would write, a migration-time
backfill is a likely explanation.
