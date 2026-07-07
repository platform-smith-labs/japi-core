---
type: interface
title: "Schema: project + environment"
tags: [schema, postgres, project, environment]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for project, git link, port binding and environment tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0001_enums.sql
  - migrations/0005_project.sql
  - migrations/0006_project_git_link.sql
  - migrations/0007_environment.sql
  - migrations/0008_project_port_binding.sql
  - migrations/0024_project_source_type_repair.sql
  - migrations/0025_agent_definitions_and_secrets.sql
provides_interfaces:
  - {name: "project/environment tables", kind: postgres-schema, intent: "projects, their git links, port bindings and deployment environments"}
---

# Schema: project + environment

### project

Workspace-scoped unit of work; covers both base-image sandboxes and git-repo projects via `source_type` (read once at launch to pick the launch path; sandbox→repo promotion is a `source_type` flip). Optional repo binding lives in `project_git_link`.

| column | type | null | default |
|---|---|---|---|
| project_id | SERIAL (PK) | no | auto |
| project_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | no | — |
| created_by_user_id | INTEGER | no | — |
| name | TEXT | no | — |
| description | TEXT | yes | — |
| default_workdir | TEXT | yes | — |
| source_type | TEXT | no | 'base_image' |
| base_image | TEXT | yes | — |
| devcontainer_ref | TEXT | yes | — |
| is_archived | BOOLEAN | no | FALSE |
| agent_definition_id | INTEGER | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK `project_id`; UNIQUE `project_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, workspace_id) → workspace(company_id, workspace_id)
- Composite FK (company_id, created_by_user_id) → users(company_id, user_id)
- FK `agent_definition_id` → agent_definition(agent_definition_id) ON DELETE SET NULL (single-column, nullable; same-company integrity enforced in application layer)
- UNIQUE (company_id, project_id)
- CHECK `source_type IN ('base_image', 'git_repo')`

**Indexes:**
- `idx_project_company_id` (company_id)
- `idx_project_company_workspace` (company_id, workspace_id)
- `idx_project_active_name_unique` UNIQUE (company_id, workspace_id, name) WHERE is_archived = FALSE — live projects in a workspace cannot share a name; archived rows are tombstones
- `idx_project_workspace_live` (company_id, workspace_id) WHERE is_archived = FALSE
- `idx_project_agent_definition` (agent_definition_id) WHERE agent_definition_id IS NOT NULL

Notes: `base_image` is the customer image when source_type=base_image, NULL for repo projects. `agent_definition_id` = agent definition persisted at most recent launch; NULL = none pinned.

### project_git_link

Optional 1:1 project↔repo binding. NULL `git_connection_id` = resolve via workspace default connection for the repo's provider.

| column | type | null | default |
|---|---|---|---|
| project_git_link_id | SERIAL (PK) | no | auto |
| project_git_link_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| project_id | INTEGER | no | — |
| git_connection_id | INTEGER | yes | — |
| git_installation_id | INTEGER | yes | — |
| repo_url | TEXT | no | — |
| default_branch | TEXT | yes | — |
| auto_clone_on_launch | BOOLEAN | no | TRUE |
| clone_depth | INTEGER | yes | — |
| commit_identity_ref | TEXT | no | 'user_email' |
| github_repo_id | BIGINT | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK `project_git_link_id`; UNIQUE `project_git_link_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, project_id) → project(company_id, project_id)
- Composite FK (company_id, git_connection_id) → git_connection(company_id, git_connection_id)
- Composite FK (company_id, git_installation_id) → git_installation(company_id, git_installation_id)
- UNIQUE (company_id, project_git_link_id)
- UNIQUE (company_id, project_id) — enforces 1:1 with project; UPSERT target
- CHECK `commit_identity_ref IN ('user_email', 'bot', 'signed_ssh')`
- CHECK `clone_depth IS NULL OR clone_depth > 0`

**Indexes:**
- `idx_project_git_link_company_id` (company_id)
- `idx_project_git_link_company_project` (company_id, project_id)
- `idx_project_git_link_connection` (company_id, git_connection_id) WHERE git_connection_id IS NOT NULL
- `idx_project_git_link_company_github_repo` (company_id, github_repo_id) WHERE github_repo_id IS NOT NULL

Notes: `github_repo_id` is GitHub's stable numeric repo id (NULL for non-GitHub rows) — marks a repo as "already a project" without URL matching. `git_installation_id` non-NULL only when pinning to a specific GitHub App installation. Projects holding a git link row are expected to have `source_type='git_repo'` (an application-maintained invariant, not DB-enforced).

### project_port_binding

Per-project port mappings. `source=dockerfile` rows are auto-managed by the orchestrator on clone completion; `source=operator` rows are user-managed and preserved across re-clones (an operator edit flips a dockerfile row to operator, one-way until reset).

| column | type | null | default |
|---|---|---|---|
| project_port_binding_id | SERIAL (PK) | no | auto |
| project_port_binding_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| project_id | INTEGER | no | — |
| container_port | INTEGER | no | — |
| protocol | port_protocol | no | 'tcp' |
| host_port | INTEGER | yes | — |
| source | port_binding_source | no | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK `project_port_binding_id`; UNIQUE `project_port_binding_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, project_id) → project(company_id, project_id) — no ON DELETE CASCADE
- UNIQUE (company_id, project_port_binding_id)
- UNIQUE (company_id, project_id, container_port, protocol)
- CHECK `container_port BETWEEN 1 AND 65535`
- CHECK `host_port IS NULL OR host_port BETWEEN 1 AND 65535`

**Indexes:**
- `idx_project_port_binding_company_id` (company_id)
- `idx_project_port_binding_company_project` (company_id, project_id)

Notes: `host_port` NULL = Docker assigns a random port; a specific value pins the host port (resolver refuses with 409 port_conflict when held by a non-evictable holder).

### environment

Deployment environment, 1:1 with its controller via the stable `controller_id` binding (set once on first registration; NULL during the pre-connect window). A runtime's environment is derived through its controller — no environment_id is stored on runtime rows.

| column | type | null | default |
|---|---|---|---|
| environment_id | SERIAL (PK) | no | auto |
| environment_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | no | — |
| controller_id | INTEGER | yes | — |
| name | TEXT | no | — |
| topology | environment_topology | no | 'docker_pods' |
| provisioner_state | JSONB | no | '{}' |
| bootstrap_token_hash | TEXT | yes | — |
| long_lived_credential_ref | TEXT | yes | — |
| is_default | BOOLEAN | no | FALSE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| last_seen_at | TIMESTAMPTZ | yes | — |
| torn_down_at | TIMESTAMPTZ | yes | — |

**Constraints:**
- PK `environment_id`; UNIQUE `environment_uuid`
- FK `company_id` → company(company_id)
- Composite FK (company_id, workspace_id) → workspace(company_id, workspace_id)
- Composite FK (company_id, controller_id) → controller(company_id, controller_id)
- UNIQUE (company_id, environment_id)
- UNIQUE (company_id, controller_id) — enforces 1:1 environment↔controller

**Indexes:**
- `idx_environment_company_id` (company_id)
- `idx_environment_company_workspace` (company_id, workspace_id)
- `idx_environment_live_name_unique` UNIQUE (company_id, workspace_id, name) WHERE torn_down_at IS NULL — live environment names unique per workspace
- `idx_environment_single_default_per_workspace` UNIQUE (company_id, workspace_id) WHERE is_default = TRUE AND torn_down_at IS NULL — at most one live default per workspace

Notes: `torn_down_at` is the tombstone timestamp (NULL = live). `bootstrap_token_hash` and `long_lived_credential_ref` are reserved, not populated in V1.

## ENUM types used by these tables

Complete value sets (no later additions exist for these types):

- **environment_topology**: `docker_pods`, `docker_compose`, `k8s_helm` — MVP writes docker_pods only; the others are reserved.
- **port_protocol**: `tcp`, `udp`
- **port_binding_source**: `dockerfile`, `operator`

Note: `project.source_type` is TEXT with a CHECK constraint (`base_image` | `git_repo`), not a Postgres ENUM.
