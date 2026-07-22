---
type: context
title: "System context and ubiquitous schema conventions"
tags: [db-migration, multi-tenant, conventions, schema]
timestamp: 2026-07-09T10:39:10Z
description: "Who interacts with the platform_smith database, and the universal data conventions stated once for the whole schema reference"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - migrations/0002_foundation.sql
  - migrations/0010_launch_recipe.sql
  - migrations/0013_task.sql
  - migrations/0042_conversation_message.sql
  - migrations/0037_mcp_tool_seam.sql
  - CLAUDE.md
---

# System context

Peer services (orchestrator, ps-api, and any service holding database credentials) connect
directly to the shared `platform_smith` PostgreSQL database. This repo runs **first** in the
platform dependency chain — postgres becomes healthy, migrations apply and the runner exits,
then the services start against the finished schema. Peers never run DDL; they only read/write
data in tables defined here.

The conventions below apply schema-wide. Other KB concepts do **not** repeat them — assume them
for every table unless a concept explicitly notes an exception.

## Multi-tenant pattern

`company` is the tenant root. Every tenant-scoped table carries `company_id` and:

- a **composite FK** to any tenant-scoped parent: `(company_id, parent_id)` referencing
  `parent(company_id, parent_id)` — single-column FKs to tenant-scoped tables are forbidden
  because they would permit cross-tenant references;
- a plain `company_id` FK to `company` as a separate constraint;
- a **composite unique** `(company_id, {table}_id)` so it can itself be a composite-FK target;
- an index on `company_id`, and composite indexes always company-leading.

**Exceptions** (deliberately not tenant-scoped): `script_log` (system table),
deployment-wide catalogs such as the git provider, integration provider, and platform MCP tool
registries, and a handful of intentional global uniques (e.g. user email, token hashes,
channel event ids) documented on their tables.

## Dual-key convention

Every application table has both `{table}_id SERIAL PRIMARY KEY` (BIGSERIAL for high-volume
append-only event tables) and
`{table}_uuid UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE`. The integer id is **internal** —
it is what all FKs reference. The uuid is the **externally visible ID** used in API responses;
the integer id is never exposed outside the database tier. (`script_log` is exempt: id only.)

## Timestamps

Every table carries `created_at` and `updated_at`, both
`TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()`. Near-universal, with one exception class:
append-only event/log tables — the launch event log, the session event log, the
conversation-message log, and the audit event log — carry `created_at` only (rows are
never updated; the message log's BIGSERIAL sequence, not `created_at`, is its authoritative
ordering cursor). The MCP tool grant table carries neither: `granted_at`/`revoked_at` play
the timestamp role there. Optional event timestamps (`started_at`, `completed_at`,
`revoked_at`, …) are nullable with no default.

## Naming

- **Tables**: snake_case, singular (`task`, `session`, `controller_instance`). Exception: `users`.
- **Constraints**: `{table}_{columns}_{fk|unique|check}` — e.g. `workspace_company_owner_user_id_fk`,
  `users_company_user_id_unique`, `task_status_check`.
- **Indexes**: `idx_{table}_{columns}` — e.g. `idx_workspace_company_archived`. Partial indexes
  (WHERE on a status/boolean) are common for "active rows only" queries.
- **Enums**: PostgreSQL ENUM types for 3+ value states; booleans for binary states.
