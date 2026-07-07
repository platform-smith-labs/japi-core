---
type: interface
title: "Schema: runtime + launch"
tags: [schema, postgres, runtime, launch]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for runtime, runtime_instance and launch pipeline tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0001_enums.sql
  - migrations/0009_runtime.sql
  - migrations/0010_launch_recipe.sql
  - migrations/0011_runtime_instance.sql
  - migrations/0012_launch_event.sql
  - migrations/0016_runtime_instance_add_launch_status.sql
  - migrations/0018a_idx_runtime_instance_inflight.sql
  - migrations/0018b_idx_runtime_instance_status_updated.sql
  - migrations/0019_launch_event_instance_key.sql
  - migrations/0021_launch_attempt_pr_columns.sql
  - migrations/0022_idx_runtime_company_name.sql
  - migrations/0028a_runtime_add_runtime_kind.sql
  - migrations/0028b_runtime_drop_all_rows_service_unique.sql
  - migrations/0028c_runtime_service_partial_unique.sql
  - migrations/0036_runtime_instance_released_at.sql
  - migrations/0039_launch_attempt_recipe_ref.sql
  - migrations/0042_runtime_instance_requested_by_user_id.sql
  - migrations/0043_runtime_instance_integration_connection.sql
  - migrations/0047_sandbox_autospawn_claim.sql
provides_interfaces:
  - {name: "runtime/launch tables", kind: postgres-schema, intent: "runtimes, their live instances, and the launch attempt/file/event pipeline"}
---

### runtime

A launched instance of a project bound to ONE controller; owns the launch lifecycle. `status` is a derived cache of the launch event HEAD, not an independent source of truth.

| column | type | null | default |
|---|---|---|---|
| runtime_id | SERIAL | NOT NULL | auto (PK) |
| runtime_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NOT NULL | — |
| controller_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NULL | — |
| name | TEXT | NOT NULL | — |
| status | launch_status | NOT NULL | 'requested' |
| failed_phase | launch_status | NULL | — |
| runtime_kind | TEXT | NOT NULL | 'service' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `runtime_id`; UNIQUE `runtime_uuid`.
- FK `company_id` → company(company_id).
- Composite FKs: (company_id, project_id) → project; (company_id, controller_id) → controller; (company_id, workspace_id) → workspace.
- UNIQUE (company_id, runtime_id) — composite-FK target.
- PARTIAL UNIQUE index `runtime_company_project_controller_unique` on (company_id, project_id, controller_id) WHERE runtime_kind = 'service' — service runtimes are singleton per (company, project, controller); sandbox runtimes are unconstrained and may share one.
- CHECK: runtime_kind IN ('service', 'sandbox').

**Indexes:**
- (company_id)
- (company_id, controller_id)
- (company_id, name) — non-unique; name collisions per company are allowed and resolved by the caller.
- the partial unique index above (serves (company_id, project_id) prefix lookups for service runtimes only — sandbox rows are not in the index).

Notes: `controller_id` is the stable controller binding for the runtime's life; the live controller instance is resolved dynamically at dispatch. `workspace_id` is denormalized from the project at launch (immutable by contract). `failed_phase` records which launch state a FAILED launch died in (diagnostic).

### runtime_instance

Each container incarnation of a runtime (one active at a time). Carries the per-incarnation launch-status HEAD; a launch is "ready" on the instance's status, not the parent runtime's.

| column | type | null | default |
|---|---|---|---|
| runtime_instance_id | SERIAL | NOT NULL | auto (PK) |
| runtime_instance_uuid | UUID | NOT NULL | gen_random_uuid() |
| instance_uuid | UUID | NOT NULL | — |
| company_id | INTEGER | NOT NULL | — |
| runtime_id | INTEGER | NOT NULL | — |
| built_from_attempt_id | INTEGER | NULL | — |
| workspace_id | INTEGER | NULL | — |
| version | TEXT | NOT NULL | — |
| platform | TEXT | NOT NULL | — |
| connected | BOOLEAN | NOT NULL | TRUE |
| ready | BOOLEAN | NOT NULL | FALSE |
| port_mapping | JSONB | NULL | — (NULL = controller has not reported yet, distinct from empty) |
| status | launch_status | NOT NULL | 'requested' |
| failed_phase | launch_status | NULL | — |
| requested_by_user_id | INTEGER | NULL | — |
| integration_connection_id | INTEGER | NULL | — |
| is_personal_integration | BOOLEAN | NOT NULL | FALSE |
| first_connected_at | TIMESTAMPTZ | NOT NULL | NOW() |
| last_seen | TIMESTAMPTZ | NOT NULL | NOW() |
| disconnected_at | TIMESTAMPTZ | NULL | — |
| setup_at | TIMESTAMPTZ | NULL | — |
| released_at | TIMESTAMPTZ | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `runtime_instance_id`; UNIQUE `runtime_instance_uuid`.
- FK `company_id` → company(company_id).
- Composite FKs: (company_id, runtime_id) → runtime; (company_id, built_from_attempt_id) → launch_attempt; (company_id, workspace_id) → workspace; (company_id, requested_by_user_id) → users (NULL = system/automated launch); (company_id, integration_connection_id) → integration_connection (NULL = static-key launch).
- NO controller FK — the controller binding lives on runtime.controller_id.
- UNIQUE (company_id, instance_uuid) — instance_uuid is the app-supplied stable pod identity (tracks reconnect vs restart).
- UNIQUE (company_id, runtime_instance_id) — composite-FK target (e.g. for sessions).

**Indexes:**
- (company_id)
- (company_id, runtime_id)
- (company_id, connected) WHERE connected = TRUE
- (company_id, ready) WHERE ready = TRUE
- (company_id, workspace_id, connected) WHERE connected = TRUE — runtime-by-workspace liveness
- (company_id, runtime_id) WHERE status NOT IN ('ready','failed') — in-flight launch admission check
- (status, updated_at) WHERE status NOT IN ('ready','failed') — cross-tenant stuck-launch watchdog (status leads; not company-scoped)

Notes: `built_from_attempt_id` records per-incarnation image provenance. `released_at` is the durable explicit-stop marker (idempotency for the runtime-stop route); distinct from `disconnected_at` (transient heartbeat loss). There is deliberately NO 'stopped'/'released' launch_status value. `integration_connection_id` / `is_personal_integration` freeze the coding-agent credential resolved at launch; sessions inherit them.

### launch_attempt

One author/build try of a launch. `succeeded` = TRUE only if the launch reached READY (not merely build success) — the recipe-cache correctness anchor.

| column | type | null | default |
|---|---|---|---|
| launch_attempt_id | SERIAL | NOT NULL | auto (PK) |
| launch_attempt_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NOT NULL | — |
| runtime_id | INTEGER | NOT NULL | — |
| base_image | TEXT | NULL | — (sandbox cache key; NULL for repo attempts) |
| git_sha | TEXT | NULL | — (repo cache key; dormant) |
| succeeded | BOOLEAN | NOT NULL | FALSE |
| error_message | TEXT | NULL | — |
| pr_url | TEXT | NULL | — |
| pr_raised_at | TIMESTAMPTZ | NULL | — |
| recipe_artifact_version_id | INTEGER | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `launch_attempt_id`; UNIQUE `launch_attempt_uuid`.
- FK `company_id` → company(company_id).
- Composite FKs: (company_id, project_id) → project; (company_id, runtime_id) → runtime; (company_id, recipe_artifact_version_id) → artifact_version (NULL for base-image/legacy attempts).
- UNIQUE (company_id, launch_attempt_id) — composite-FK target.

**Indexes:**
- (company_id)
- (company_id, project_id, base_image, created_at DESC) WHERE succeeded = TRUE — sandbox recipe-cache lookup
- (company_id, project_id, git_sha, created_at DESC) WHERE succeeded = TRUE — repo recipe-cache lookup

### launch_file

Project-scoped, content-addressed recipe files (the launch fileset). `path` is the relative destination materialized under the build context; absolute placement lives in the file content itself.

| column | type | null | default |
|---|---|---|---|
| launch_file_id | SERIAL | NOT NULL | auto (PK) |
| launch_file_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NOT NULL | — |
| path | TEXT | NOT NULL | — |
| content | TEXT | NOT NULL | — |
| content_sha256 | BYTEA | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `launch_file_id`; UNIQUE `launch_file_uuid`.
- FK `company_id` → company(company_id); composite FK (company_id, project_id) → project.
- UNIQUE (company_id, launch_file_id) — join-FK target.
- UNIQUE (company_id, project_id, path, content_sha256) — dedup key (also serves path-prefix lookups).
- CHECK: octet_length(content_sha256) = 32 — SHA-256 of `content` (dedup + integrity).

**Indexes:**
- (company_id)

### launch_attempt_file

Join table: the recipe fileset of a launch attempt.

| column | type | null | default |
|---|---|---|---|
| launch_attempt_file_id | SERIAL | NOT NULL | auto (PK) |
| launch_attempt_file_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| launch_attempt_id | INTEGER | NOT NULL | — |
| launch_file_id | INTEGER | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `launch_attempt_file_id`; UNIQUE `launch_attempt_file_uuid`.
- FK `company_id` → company(company_id); composite FKs (company_id, launch_attempt_id) → launch_attempt and (company_id, launch_file_id) → launch_file.
- UNIQUE (company_id, launch_attempt_file_id).
- UNIQUE (company_id, launch_attempt_id, launch_file_id) — each file linked at most once per attempt.

**Indexes:**
- (company_id)

### launch_event

Append-only launch timeline + SSE source for all launches. `launch_event_id` (BIGSERIAL) is the authoritative SSE cursor/ordering; `created_at DESC` is display-only. No `updated_at` (append-only). runtime.status / runtime_instance.status are derived projections written event-first, non-transactionally.

| column | type | null | default |
|---|---|---|---|
| launch_event_id | BIGSERIAL | NOT NULL | auto (PK) |
| launch_event_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| runtime_id | INTEGER | NOT NULL | — |
| runtime_instance_id | INTEGER | NOT NULL | — |
| launch_attempt_id | INTEGER | NULL | — |
| event_type | TEXT | NOT NULL | — |
| phase | TEXT | NULL | — (free text; NOT constrained to launch_status) |
| severity | TEXT | NOT NULL | 'info' |
| data | JSONB | NOT NULL | '{}' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `launch_event_id`; UNIQUE `launch_event_uuid`.
- FK `company_id` → company(company_id).
- Composite FKs: (company_id, runtime_id) → runtime (denormalized runtime-level rollup key); (company_id, runtime_instance_id) → runtime_instance (primary correlation key — the incarnation); (company_id, launch_attempt_id) → launch_attempt.
- CHECK: severity IN ('info', 'warn', 'error').

**Indexes:**
- (company_id)
- (company_id, runtime_id, created_at DESC) — timeline by runtime
- (company_id, runtime_id, created_at DESC) WHERE severity = 'error' — error filter
- (company_id, runtime_instance_id, launch_event_id DESC) — timeline by incarnation (cursor order)

### sandbox_autospawn_claim

Dedup + cooldown backing for A2A sandbox auto-spawn: one row per in-flight/recent auto-spawn of a target project for a conversation, scoped to (company, workspace, controller, project, conversation). The unique key closes the concurrent-relay race (idempotent INSERT ... ON CONFLICT, no locks). Cleared when the spawned pod connects; `last_failed_at` restamped on launch failure to gate the re-spawn cooldown.

| column | type | null | default |
|---|---|---|---|
| sandbox_autospawn_claim_id | SERIAL | NOT NULL | auto (PK) |
| sandbox_autospawn_claim_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NOT NULL | — |
| controller_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NOT NULL | — |
| conversation_id | INTEGER | NOT NULL | — |
| runtime_id | INTEGER | NULL | — (back-filled after spawn) |
| last_failed_at | TIMESTAMPTZ | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK `sandbox_autospawn_claim_id`; UNIQUE `sandbox_autospawn_claim_uuid`.
- FK `company_id` → company(company_id).
- Composite FKs: (company_id, workspace_id) → workspace; (company_id, controller_id) → controller; (company_id, project_id) → project; (company_id, conversation_id) → conversation.
- Single-column FK runtime_id → runtime(runtime_id) ON DELETE SET NULL — intentionally NOT composite: SET NULL cannot null the NOT NULL company_id, and runtime_id is globally unique (SERIAL PK).
- UNIQUE (company_id, workspace_id, controller_id, project_id, conversation_id) — the dedup key.

**Indexes:**
- (company_id)
- (runtime_id)

Notes: `controller_id` doubles as the environment identity (a controller binds a (workspace, environment) pair); auto-spawn co-locates the target on the sender's controller, so dedup is per-environment.

### Enum types used

**launch_status** (complete value set, never extended after creation; order is a cross-repo contract):
`requested`, `resolving_recipe`, `builder_starting`, `cloning`, `authoring`, `building`, `starting_runtime`, `setting_up`, `ready`, `failed`

Used by: runtime.status, runtime.failed_phase, runtime_instance.status, runtime_instance.failed_phase. There is intentionally no 'stopped'/'released' value — explicit release is recorded via runtime_instance.released_at.

`runtime.runtime_kind` is NOT an enum type — it is TEXT constrained by CHECK to `service` | `sandbox`.
