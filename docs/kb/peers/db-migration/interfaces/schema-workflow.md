---
type: interface
title: "Schema: workflows"
tags: [schema, postgres, workflow]
timestamp: 2026-07-09T10:37:36Z
description: "Final-state reference for workflow definition/approval/notification tables"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - migrations/0029_workflow_definitions_and_secret_refs.sql
  - migrations/0031_workflow_approval.sql
  - migrations/0032_workflow_notification.sql
  - migrations/0047_workflow_definition_ui_columns.sql
  - migrations/0048_workflow_run_context.sql
  - migrations/0025_agent_definitions_and_secrets.sql
provides_interfaces:
  - {name: "workflow tables", kind: postgres-schema, intent: "workflow definitions, secret refs, approvals, notifications and per-run context"}
---

# Schema: workflows

### workflow_definition

Scoped canonical record of a workflow: runnable Conductor definition JSON plus PS sidecar annotations and scoping/governance; the Conductor engine itself stays tenant-blind.

| column | type | null | default |
|---|---|---|---|
| workflow_definition_id | SERIAL | no | (PK) |
| workflow_definition_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | yes | — |
| project_id | INTEGER | yes | — |
| cloned_from_id | INTEGER | yes | — |
| created_by_user_id | INTEGER | yes | — |
| scope_type | workflow_definition_scope_type | no | 'project' |
| name | TEXT | no | — |
| description | TEXT | yes | — |
| version | INTEGER | no | 1 |
| template_version | INTEGER | yes | — |
| conductor_json | JSONB | no | — |
| ps_metadata | JSONB | no | '{}' |
| is_archived | BOOLEAN | no | FALSE |
| is_system_provided | BOOLEAN | no | FALSE |
| is_mandatory | BOOLEAN | no | FALSE |
| execution_context | workflow_execution_context | yes | — |
| category | TEXT | yes | — |
| labels | TEXT[] | no | '{}' |
| published_at | TIMESTAMPTZ | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK: `workflow_definition_id`; UNIQUE: `workflow_definition_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, workspace_id) → workspace(company_id, workspace_id)
- Composite FK (company_id, project_id) → project(company_id, project_id)
- Composite self-FK (company_id, cloned_from_id) → workflow_definition(company_id, workflow_definition_id) — clone-to-override provenance, same-company only
- UNIQUE (company_id, workflow_definition_id)
- Note: `created_by_user_id` has no FK constraint.

**Indexes:**
- `idx_workflow_definition_company_project` (company_id, project_id)
- `idx_workflow_definition_company_scope` (company_id, scope_type)
- `idx_workflow_definition_company_workspace` (company_id, workspace_id) WHERE workspace_id IS NOT NULL
- UNIQUE `idx_workflow_definition_unique_name_company` (company_id, name) WHERE scope_type = 'company' AND is_archived = FALSE
- UNIQUE `idx_workflow_definition_unique_name_workspace` (company_id, workspace_id, name) WHERE scope_type = 'workspace' AND is_archived = FALSE
- UNIQUE `idx_workflow_definition_unique_name_project` (company_id, project_id, name) WHERE scope_type = 'project' AND is_archived = FALSE
- `idx_workflow_definition_mandatory` (company_id) WHERE is_mandatory = TRUE AND is_archived = FALSE

Semantics: `workspace_id` set for workspace and project scopes (NULL for company); `project_id` set for project scope only. There is no 'system' scope — platform templates are `is_system_provided` clones per company. At most one mandatory lineage per company is enforced at the write layer, not by the schema. `conductor_json` is the runnable Conductor WorkflowDef; PS per-task concepts ride in task `inputParameters._ps`. `template_version` pins the clone-source version at clone time ("update available" signal); NULL if not a clone.

Workflow Management UI classifiers (all additive, nullable/defaulted, no backfill — NULL/empty on legacy rows): `execution_context` is the run context a definition needs (`workspace` | `project`), a **separate axis** from `scope_type` (the definition level — note `company` is a valid scope_type but never an execution_context); NULL on legacy rows, set by ps-workflow on create/edit. `category` is a single coarse free-form classifier for list grouping (NULL = uncategorized). `labels` is a free-form `TEXT[]` of tags for filter/search (empty array, never NULL). `published_at` is set to NOW() on a successful Publish (draft→live signal for the badge + run-gate); NULL = never published (still a draft).

### workflow_definition_secret_ref

Declares which secrets a workflow definition requires, by logical name; resolved at runtime via a scope walk against the secret store (deliberately no FK to it — the same logical name resolves to different rows per execution scope).

| column | type | null | default |
|---|---|---|---|
| workflow_definition_secret_ref_id | SERIAL | no | (PK) |
| workflow_definition_secret_ref_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workflow_definition_id | INTEGER | no | — |
| secret_name | TEXT | no | — |
| secret_type | secret_type | no | — |
| usage_context | TEXT | no | — |
| context_identifier | TEXT | yes | — |
| injection_method | TEXT | no | 'template' |
| inject_as | TEXT | yes | — |
| file_mode | TEXT | yes | '0600' |
| required | BOOLEAN | no | TRUE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK: `workflow_definition_secret_ref_id`; UNIQUE: `workflow_definition_secret_ref_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, workflow_definition_id) → workflow_definition(company_id, workflow_definition_id)
- FK `workflow_definition_id` → workflow_definition(workflow_definition_id)
- UNIQUE (company_id, workflow_definition_secret_ref_id)
- CHECK `usage_context IN ('node', 'workflow_input', 'template', 'env')`
- CHECK `injection_method IN ('template', 'env_var', 'file')`
- CHECK `injection_method != 'env_var' OR secret_type = 'env_vars' OR inject_as IS NOT NULL`

**Indexes:**
- UNIQUE `idx_workflow_def_secret_ref_definition_name_context_unique` (workflow_definition_id, secret_name, usage_context, COALESCE(context_identifier, '')) — one binding per (definition, name, context); COALESCE handles NULL context
- `idx_workflow_def_secret_ref_company_id` (company_id)
- `idx_workflow_def_secret_ref_definition` (workflow_definition_id)

Semantics: `context_identifier` is the Conductor taskReferenceName for node-scoped secrets, NULL otherwise. `required = TRUE` means an unresolved secret blocks execution start.

### workflow_approval

Durable tenant-scoped correlation of a parked Conductor HUMAN/approval task (workflow_id, task_ref_name) to an approval decision. Decision identity is (company_id, workflow_id, task_ref_name) — tenant is part of the key. The pending→terminal status flip (single atomic UPDATE WHERE status = 'pending') is the exactly-once "first decision wins" primitive; a second decision updates zero rows and is a no-op.

| column | type | null | default |
|---|---|---|---|
| workflow_approval_id | SERIAL | no | (PK) |
| workflow_approval_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workflow_id | TEXT | no | — |
| task_ref_name | TEXT | no | — |
| status | workflow_approval_status | no | 'pending' |
| approvers | JSONB | no | '[]' |
| decided_by_user_id | INTEGER | yes | — |
| decision_reason | TEXT | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| decided_at | TIMESTAMPTZ | yes | — |

**Constraints:**
- PK: `workflow_approval_id`; UNIQUE: `workflow_approval_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, decided_by_user_id) → users(company_id, user_id) — decider must be same-company; MATCH SIMPLE skips the check while decided_by_user_id is NULL
- UNIQUE (company_id, workflow_id, task_ref_name) — also serves the hot lookup/decide path via its implicit index
- CHECK `(status = 'pending') = (decided_at IS NULL)`
- CHECK `(status = 'pending') = (decided_by_user_id IS NULL)`

**Indexes:**
- `idx_workflow_approval_pending` (company_id, created_at) WHERE status = 'pending' — pending worklist / cleanup sweep

Semantics: `workflow_id` is the Conductor workflow instance id (string); `approvers` is a JSONB array of allowed deciders sourced from `inputParameters._ps.approvers` — app-enforced, not an FK set.

### workflow_notification

Durable tenant-scoped in-app notification rows emitted by the send-notification workflow node. Company-scoped; `user_id` NULL means company-wide.

| column | type | null | default |
|---|---|---|---|
| workflow_notification_id | SERIAL | no | (PK) |
| workflow_notification_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| user_id | INTEGER | yes | — |
| workflow_id | TEXT | no | — |
| task_ref_name | TEXT | no | — |
| channel | workflow_notification_channel | no | 'in-app' |
| title | TEXT | no | — |
| body | TEXT | no | — |
| data | JSONB | no | '{}' |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK: `workflow_notification_id`; UNIQUE: `workflow_notification_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, user_id) → users(company_id, user_id) — recipient must be same-company; MATCH SIMPLE skips the check when user_id is NULL

**Indexes:**
- `idx_workflow_notification_company_user` (company_id, user_id) — recipient inbox
- `idx_workflow_notification_company_workflow` (company_id, workflow_id) — per-workflow lookup

Semantics: `data` is a schema-less payload (links, ids). No UNIQUE (company_id, workflow_notification_id) composite; no check constraints.

### workflow_run_context

Per-execution run-context sidecar: one row per Conductor workflow instance, written at start-execution by ps-workflow. The single source of truth for the workspace/project/execution_context a run targeted plus its owning definition (for friendly-name resolution in list-executions). Joined by `(company_id, workflow_id)` for the inbox roll-up and list-executions; the approval/notification tables carry no run-context. (Table name deliberately differs from the `workflow_execution_context` enum — Postgres types and tables share one namespace.)

| column | type | null | default |
|---|---|---|---|
| workflow_run_context_id | SERIAL | no | (PK) |
| workflow_run_context_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workflow_definition_id | INTEGER | yes | — |
| workspace_id | INTEGER | yes | — |
| project_id | INTEGER | yes | — |
| started_by_user_id | INTEGER | yes | — |
| workflow_id | TEXT | no | — |
| execution_context | workflow_execution_context | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK: `workflow_run_context_id`; UNIQUE: `workflow_run_context_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, workflow_definition_id) → workflow_definition(company_id, workflow_definition_id) — no ON DELETE CASCADE
- Composite FK (company_id, workspace_id) → workspace(company_id, workspace_id)
- Composite FK (company_id, project_id) → project(company_id, project_id)
- Composite FK (company_id, started_by_user_id) → users(company_id, user_id)
- UNIQUE (company_id, workflow_id) — one context row per Conductor instance; the natural upsert key
- UNIQUE (company_id, workflow_run_context_id) (composite-FK anchor)

**Indexes (all partial):**
- `idx_workflow_run_context_company_workspace` (company_id, workspace_id) WHERE workspace_id IS NOT NULL
- `idx_workflow_run_context_company_project` (company_id, project_id) WHERE project_id IS NOT NULL
- `idx_workflow_run_context_company_definition` (company_id, workflow_definition_id) WHERE workflow_definition_id IS NOT NULL

Semantics: `workflow_id` is the Conductor workflow instance id (string) — the join key to `workflow_approval`/`workflow_notification`. `workflow_definition_id` is the owning definition for name resolution (NULL if not resolvable). `execution_context` is a snapshot of the definition's `execution_context` at start (NULL if unknown). `workspace_id` is set for project-context runs too; `project_id` is NULL for workspace-context runs.

## Enum types

- **workflow_definition_scope_type**: `company`, `workspace`, `project`
- **workflow_execution_context**: `workspace`, `project` — the run context a definition/execution targets; distinct from `scope_type` (extend later via `ALTER TYPE ... ADD VALUE`, which needs the no-transaction path)
- **workflow_approval_status**: `pending`, `approved`, `rejected` — pending is the only non-terminal state
- **workflow_notification_channel**: `in-app` (single value; room to grow)
- **secret_type** (shared type, defined elsewhere in the schema and reused by workflow_definition_secret_ref): `oauth`, `bearer_token`, `api_key`, `ssh_key`, `basic_auth`, `env_vars`
