---
type: interface
title: "Schema: foundation (company, users, workspace)"
tags: [schema, postgres, tenancy, foundation]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for the tenancy foundation tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0002_foundation.sql
  - migrations/0051_users_service_principal.sql
  - migrations/0001_enums.sql
provides_interfaces:
  - {name: "foundation tables", kind: postgres-schema, intent: "tenancy root tables every service row hangs off"}
---

# Foundation schema domain

Tenancy root tables: `company` (tenant root), `users` (humans + synthetic service principals), `workspace` (resource grouping within a tenant), `workspace_token` (controller self-registration tokens), and the system table `script_log` (migration tracking; exempt from dual-key/company_id conventions).

### company

Tenant root — every tenant-scoped row in the database references it.

| column | type | null | default |
|---|---|---|---|
| company_id | SERIAL (PK) | NOT NULL | auto |
| company_uuid | UUID | NOT NULL | gen_random_uuid() |
| name | TEXT | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `company_id`; `company_uuid` UNIQUE; `company_name_unique` UNIQUE (name) — intentionally global (tenant root).

**Indexes:** none beyond implicit PK/UNIQUE indexes.

### users

Platform user account (Argon2id-hashed password). A row may also be a synthetic, non-human **workspace service principal** (system actor, e.g. Slack alerts) flagged by `is_service_principal`; human-facing surfaces filter these out with `WHERE is_service_principal = FALSE`.

| column | type | null | default |
|---|---|---|---|
| user_id | SERIAL (PK) | NOT NULL | auto |
| user_uuid | UUID | NOT NULL | gen_random_uuid() |
| name | TEXT | NOT NULL | — |
| email | TEXT | NOT NULL | — |
| company_id | INTEGER | NOT NULL | — |
| password | TEXT | NOT NULL | — |
| is_service_principal | BOOLEAN | NOT NULL | FALSE |
| service_principal_workspace_id | INTEGER | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `user_id`; `user_uuid` UNIQUE; FK `users_company_id_fk` (company_id) → company(company_id); composite FK `users_company_service_principal_workspace_fk` (company_id, service_principal_workspace_id) → workspace(company_id, workspace_id) — NULL workspace bypasses the check (human rows); UNIQUE `users_company_user_id_unique` (company_id, user_id); UNIQUE `users_company_name_unique` (company_id, name); UNIQUE `users_email_unique` (email) — intentionally global; CHECK `users_service_principal_workspace_required_check` (NOT is_service_principal OR service_principal_workspace_id IS NOT NULL) — a service principal must be workspace-scoped.

**Indexes:** partial UNIQUE index `users_service_principal_workspace_unique` on (company_id, service_principal_workspace_id) WHERE is_service_principal — one service principal per (company, workspace).

### workspace

Named workspace grouping resources within a tenant company.

| column | type | null | default |
|---|---|---|---|
| workspace_id | SERIAL (PK) | NOT NULL | auto |
| workspace_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| owner_user_id | INTEGER | NOT NULL | — |
| name | TEXT | NOT NULL | — |
| description | TEXT | NULL | — |
| is_archived | BOOLEAN | NOT NULL | FALSE |
| settings | JSONB | NOT NULL | '{}' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| archived_at | TIMESTAMPTZ | NULL | — |

**Constraints:** PK `workspace_id`; `workspace_uuid` UNIQUE (external API identifier — `workspace_id` is never exposed); FK `workspace_company_id_fk` (company_id) → company(company_id); composite FK `workspace_company_owner_user_id_fk` (company_id, owner_user_id) → users(company_id, user_id); UNIQUE `workspace_company_name_unique` (company_id, name); UNIQUE `workspace_company_id_unique` (company_id, workspace_id) — the composite-FK anchor for child tables. `archived_at` is application-set when `is_archived` flips TRUE.

**Indexes:** `idx_workspace_company_id` (company_id); `idx_workspace_company_archived` (company_id, is_archived).

### workspace_token

Token store allowing apps/controllers to self-register against a workspace; multiple active tokens per workspace are supported. Only a SHA-256 hash of the raw token is stored.

| column | type | null | default |
|---|---|---|---|
| workspace_token_id | SERIAL (PK) | NOT NULL | auto |
| workspace_token_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NOT NULL | — |
| name | TEXT | NOT NULL | — |
| token_hash | TEXT | NOT NULL | — |
| revoked | BOOLEAN | NOT NULL | FALSE |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| revoked_at | TIMESTAMPTZ | NULL | — |

**Constraints:** PK `workspace_token_id`; `workspace_token_uuid` UNIQUE (external identifier); FK `workspace_token_company_id_fk` (company_id) → company(company_id); composite FK `workspace_token_company_workspace_id_fk` (company_id, workspace_id) → workspace(company_id, workspace_id); UNIQUE `workspace_token_company_workspace_name_unique` (company_id, workspace_id, name); UNIQUE `workspace_token_company_id_unique` (company_id, workspace_token_id); UNIQUE `workspace_token_hash_unique` (token_hash) — global, serving WebSocket-auth lookup by hash. `revoked = TRUE` means the token is rejected on use; `revoked_at` is application-set on revocation.

**Indexes:** `idx_workspace_token_company_id` (company_id); `idx_workspace_token_company_workspace` (company_id, workspace_id); partial `idx_workspace_token_active` (company_id, workspace_id, revoked) WHERE revoked = FALSE.

### script_log

Migration-runner tracking table (which migration files have executed). System table: no uuid, no company_id.

| column | type | null | default |
|---|---|---|---|
| script_id | SERIAL (PK) | NOT NULL | auto |
| script_name | TEXT | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `script_id`; `script_name` UNIQUE.

**Indexes:** none beyond implicit PK/UNIQUE indexes.

## Enum types

None — no foundation table uses a PostgreSQL ENUM type (states are booleans: `is_archived`, `revoked`, `is_service_principal`).
