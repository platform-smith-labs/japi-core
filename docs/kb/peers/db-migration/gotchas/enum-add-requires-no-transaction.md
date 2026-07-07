---
type: gotcha
title: "Adding an enum value cannot run inside a transaction"
tags: [migrations, enums, transactions, postgres]
timestamp: 2026-07-07T01:13:56Z
description: "ALTER TYPE ... ADD VALUE is disallowed in a transaction block; such migrations carry a no-transaction marker the runner honors, and should hold exactly one statement"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - pkg/migration/migration.go
  - migrations/0045_integration_status_disabled.sql
  - migrations/0048_slack_auth_type_enum.sql
  - docs/dev/decisions/one-statement-per-no-transaction-migration.md
  - CLAUDE.md
---
By default every migration file runs wrapped in a transaction. PostgreSQL forbids
`ALTER TYPE ... ADD VALUE` inside a transaction block, so a migration that adds an enum
value must carry the literal marker comment `-- @no-transaction` within its first 10 lines
(the runner matches the `@no-transaction` substring); the
runner then executes the file's SQL outside a transaction and records completion in a
separate small transaction afterwards.

Two traps follow:
- A no-transaction file gets **no rollback** — a mid-file failure leaves earlier statements
  applied but the file unrecorded, so the re-run repeats them. House rule: exactly one
  statement per no-transaction migration, written idempotently (`IF NOT EXISTS`).
- **Renaming or removing** an enum value is different: it requires the recreate-type approach
  (create new type, migrate the column, drop the old type) in a normal transactional migration.
