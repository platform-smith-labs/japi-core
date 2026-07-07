---
type: interface
title: "Schema: artifacts, MCP tools, audit"
tags: [schema, postgres, artifact, mcp, audit]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for artifact plane, MCP tool grant and audit event tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0038_artifact_plane.sql
  - migrations/0037_mcp_tool_seam.sql
  - migrations/0015_audit_event.sql
  - migrations/0001_enums.sql
  - migrations/0039_launch_attempt_recipe_ref.sql
provides_interfaces:
  - {name: "artifact/mcp/audit tables", kind: postgres-schema, intent: "artifact storage plane, MCP tool grants and the audit event log"}
---

# Schema: artifacts, MCP tools, audit

### artifact_blob

Scope-agnostic, content-addressed blob store for artifact content; dedup key is (company_id, content_sha256). Distinct from the project-scoped recipe store.

| column | type | null | default |
|---|---|---|---|
| artifact_blob_id | SERIAL (PK) | no | auto |
| artifact_blob_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| content_sha256 | BYTEA | no | — |
| content | TEXT | no | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `artifact_blob_id`; `artifact_blob_uuid` UNIQUE. FK `company_id` → company(company_id). UNIQUE (company_id, content_sha256); UNIQUE (company_id, artifact_blob_id). CHECK: `octet_length(content_sha256) = 32` (true SHA-256 digest).

**Indexes:** `idx_artifact_blob_company_id` on (company_id).

### artifact

One logical, named artifact per (scope, name). Polymorphic scope: exactly one of session/project/workspace, discriminated by `scope_type`.

| column | type | null | default |
|---|---|---|---|
| artifact_id | SERIAL (PK) | no | auto |
| artifact_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | yes | — |
| project_id | INTEGER | yes | — |
| session_id | INTEGER | yes | — |
| name | TEXT | no | — |
| kind | TEXT | no | — |
| scope_type | artifact_scope_type | no | — |
| current_version_id | INTEGER | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `artifact_id`; `artifact_uuid` UNIQUE. FKs: `company_id` → company; composite (company_id, workspace_id) → workspace; composite (company_id, project_id) → project; composite (company_id, session_id) → session; composite (company_id, current_version_id) → artifact_version(company_id, artifact_version_id) — circular FK, nullable until first publish. UNIQUE (company_id, artifact_id). CHECKs: `kind IN ('recipe','result','note','discovery','design')`; scope-consistency — the FK column matching `scope_type` must be NOT NULL and the other two NULL.

**Indexes:** `idx_artifact_company_id` on (company_id). Three partial UNIQUE indexes enforce per-scope name uniqueness: (company_id, session_id, name) WHERE scope_type = 'session'; (company_id, project_id, name) WHERE scope_type = 'project'; (company_id, workspace_id, name) WHERE scope_type = 'workspace'.

### artifact_version

Immutable artifact version; each save/edit is a new row.

| column | type | null | default |
|---|---|---|---|
| artifact_version_id | SERIAL (PK) | no | auto |
| artifact_version_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| artifact_id | INTEGER | no | — |
| content_ref | TEXT | no | — |
| content_type | TEXT | no | — |
| status | TEXT | no | — |
| provenance | JSONB | no | '{}' |
| applies_to_ref | TEXT | no | 'any' |
| approval_uuid | UUID | yes | — |
| approved_by | UUID | yes | — |
| approved_at | TIMESTAMPTZ | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `artifact_version_id`; `artifact_version_uuid` UNIQUE. FKs: `company_id` → company; composite (company_id, artifact_id) → artifact(company_id, artifact_id). UNIQUE (company_id, artifact_version_id). CHECK: `status IN ('draft','proposed','published','superseded','rejected')`.

**Indexes:** `idx_artifact_version_company_id` on (company_id); `idx_artifact_version_company_artifact` on (company_id, artifact_id).

Gotchas: `content_ref` is a sha256 hex string — a soft reference (no FK) resolved to artifact_blob by (company_id, decoded digest); the writer must persist the blob before the version. `approval_uuid` is a present-but-unused approval linkage (no FK). Inbound reference: launch_attempt carries a nullable composite FK (company_id, recipe_artifact_version_id) → artifact_version, linking a launch to the recipe version it used.

### ps_mcp_tool

Global platform MCP tool registry/catalog — NOT tenant-scoped (no company_id). `name` is the globally-unique MCP tool name surfaced in tools/list.

| column | type | null | default |
|---|---|---|---|
| ps_mcp_tool_id | SERIAL (PK) | no | auto |
| ps_mcp_tool_uuid | UUID | no | gen_random_uuid() |
| name | TEXT | no | — |
| description | TEXT | no | — |
| input_schema | JSONB | no | — |
| handler_key | TEXT | no | — |
| min_role | TEXT | yes | — |
| enabled | BOOLEAN | no | TRUE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `ps_mcp_tool_id`; `ps_mcp_tool_uuid` UNIQUE; UNIQUE (name). No FKs.

**Indexes:** none beyond PK/unique implicit indexes.

`input_schema` is the JSON Schema returned in tools/list; `handler_key` is a dispatch key resolved to a handler registered at orchestrator boot; `min_role` is an optional coarse policy hint.

### ps_mcp_tool_grant

Per-session, tenant-scoped grant of a platform MCP tool; created before the runtime instance spawns and retained for audit (soft-revoke).

| column | type | null | default |
|---|---|---|---|
| ps_mcp_tool_grant_id | SERIAL (PK) | no | auto |
| ps_mcp_tool_grant_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| session_id | INTEGER | no | — |
| ps_mcp_tool_id | INTEGER | no | — |
| constraints | JSONB | no | '{}' |
| granted_at | TIMESTAMPTZ | no | NOW() |
| revoked_at | TIMESTAMPTZ | yes | — |

**Constraints:** PK `ps_mcp_tool_grant_id`; `ps_mcp_tool_grant_uuid` UNIQUE. FKs: `company_id` → company; composite (company_id, session_id) → session(company_id, session_id); `ps_mcp_tool_id` → ps_mcp_tool (single-column — the tool registry is global, not tenant-scoped). UNIQUEs: (company_id, ps_mcp_tool_grant_id); (company_id, session_id, ps_mcp_tool_id).

**Indexes:** `idx_ps_mcp_tool_grant_company_id` on (company_id); partial `idx_ps_mcp_tool_grant_active_lookup` on (company_id, session_id) WHERE revoked_at IS NULL.

Gotchas: NULL `revoked_at` = active grant; rows are never deleted. Re-granting after revoke must be an UPDATE (clear `revoked_at`) — the (company_id, session_id, ps_mcp_tool_id) UNIQUE blocks a second INSERT. No `created_at`/`updated_at`; `granted_at` plays the creation-timestamp role.

### audit_event

Immutable audit log for workspace-scoped actions. Rows are never updated or deleted; there is deliberately no `updated_at`.

| column | type | null | default |
|---|---|---|---|
| audit_event_id | BIGSERIAL (PK) | no | auto |
| audit_event_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | no | — |
| workspace_uuid | UUID | no | — |
| actor_user_uuid | UUID | yes | — |
| action_type | audit_action_type | no | — |
| target_entity_type | audit_entity_type | no | — |
| target_entity_uuid | UUID | no | — |
| metadata | JSONB | no | '{}' |
| created_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `audit_event_id` (BIGSERIAL — high-volume log); `audit_event_uuid` UNIQUE. FKs: `company_id` → company; composite (company_id, workspace_id) → workspace(company_id, workspace_id). No UNIQUE (company_id, audit_event_id) — nothing references audit_event.

**Indexes:** `idx_audit_event_company_id` on (company_id); `idx_audit_event_workspace_time` on (company_id, workspace_uuid, created_at DESC); partial `idx_audit_event_actor_time` on (company_id, actor_user_uuid, created_at DESC) WHERE actor_user_uuid IS NOT NULL.

Gotchas: `workspace_uuid` is denormalized from workspace to avoid a join in list queries. `actor_user_uuid` NULL means system-initiated. `metadata` is PII-scrubbed structured context and must never contain credential values.

## Enum types used

- **artifact_scope_type**: `session`, `project`, `workspace` (used by artifact.scope_type).
- **audit_action_type**: `workspace_settings_changed`, `environment_created`, `environment_updated`, `environment_archived`, `runtime_spawned`, `runtime_killed`, `workspace_token_minted`, `workspace_token_revoked`, `git_connection_installed`, `git_connection_revoked`, `git_connection_reinstalled` (used by audit_event.action_type).
- **audit_entity_type**: `workspace`, `environment`, `runtime`, `workspace_token`, `git_connection` (used by audit_event.target_entity_type).
