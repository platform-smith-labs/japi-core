---
type: interface
title: "Schema: agent definitions + secrets"
tags: [schema, postgres, agent-definition, secrets]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for agent definition and secret store tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0025_agent_definitions_and_secrets.sql
  - migrations/0001_enums.sql
provides_interfaces:
  - {name: "agent/secret tables", kind: postgres-schema, intent: "agent definitions, their files/secret refs, and the secret store"}
---

# Schema: agent definitions + secrets

### agent_definition

Named configuration templates for coding agents, scoped to company / workspace / project.

| column | type | null | default |
|---|---|---|---|
| agent_definition_id | SERIAL | NOT NULL | — (PK) |
| agent_definition_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NULL | — |
| project_id | INTEGER | NULL | — |
| scope_type | agent_definition_scope_type | NOT NULL | 'project' |
| coding_agent_type | coding_agent_type | NOT NULL | 'claude_code' |
| name | TEXT | NOT NULL | — |
| description | TEXT | NULL | — |
| version | INTEGER | NOT NULL | 1 |
| default_file_policy | file_policy_type | NOT NULL | 'open' |
| is_archived | BOOLEAN | NOT NULL | FALSE |
| is_system_provided | BOOLEAN | NOT NULL | FALSE |
| is_mandatory | BOOLEAN | NOT NULL | FALSE |
| created_by_user_id | INTEGER | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `agent_definition_id`; UNIQUE: `agent_definition_uuid`
- FK: `company_id` → company(company_id)
- FK (composite): `(company_id, workspace_id)` → workspace(company_id, workspace_id)
- FK (composite): `(company_id, project_id)` → project(company_id, project_id)
- UNIQUE: `(company_id, agent_definition_id)` (composite-FK anchor for child tables)
- `workspace_id` is set for workspace and project scopes (NULL otherwise); `project_id` is set for project scope only. `is_mandatory` = sessions in the scope and descendants must use this definition lineage; at-most-one-mandatory-lineage-per-company is enforced at the application write layer, not in SQL.
- `created_by_user_id` has no FK constraint.

**Indexes:**
- `(company_id, project_id)`
- `(company_id, scope_type)`
- `(company_id, workspace_id)` WHERE workspace_id IS NOT NULL
- UNIQUE `(company_id, name)` WHERE scope_type = 'company' AND is_archived = FALSE
- UNIQUE `(company_id, workspace_id, name)` WHERE scope_type = 'workspace' AND is_archived = FALSE
- UNIQUE `(company_id, project_id, name)` WHERE scope_type = 'project' AND is_archived = FALSE (i.e., one active name per scope entity)
- `(company_id)` WHERE is_mandatory = TRUE AND is_archived = FALSE (launch-time mandatory-lineage lookup)

Inbound single-column FKs from other domains: `project.agent_definition_id` and `conversation_participant.agent_definition_id` (both ON DELETE SET NULL), plus composite `(company_id, agent_definition_id)` FKs from Slack channel binding/routing tables. (The workflow-definition tables mirror this table's design but do not reference it.)

### agent_definition_file

Files belonging to an agent definition, materialized into the agent workdir at session start.

| column | type | null | default |
|---|---|---|---|
| agent_definition_file_id | SERIAL | NOT NULL | — (PK) |
| agent_definition_file_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| agent_definition_id | INTEGER | NOT NULL | — |
| concept | TEXT | NOT NULL | 'instructions' |
| file_path | TEXT | NOT NULL | — |
| content | TEXT | NOT NULL | '' |
| merge_strategy | merge_strategy_type | NOT NULL | 'append' |
| file_policy | file_policy_type | NULL | — |
| ordering | INTEGER | NOT NULL | 0 |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `agent_definition_file_id`; UNIQUE: `agent_definition_file_uuid`
- FK: `company_id` → company(company_id)
- FK (composite): `(company_id, agent_definition_id)` → agent_definition(company_id, agent_definition_id)
- FK: `agent_definition_id` → agent_definition(agent_definition_id)
- UNIQUE: `(agent_definition_id, file_path)`
- UNIQUE: `(company_id, agent_definition_file_id)`
- `file_policy` NULL means inherit (directory policy → definition default_file_policy → open). `file_path` is relative within the agent config directory. `merge_strategy` describes how the file combines with same-path files from higher scopes.

**Indexes:**
- `(company_id, agent_definition_id)`

### secret_store

Generic secret store with hierarchical scoping (company/workspace/project), envelope encryption, and a placeholder/contract pattern for deferred provisioning.

| column | type | null | default |
|---|---|---|---|
| secret_store_id | SERIAL | NOT NULL | — (PK) |
| secret_store_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NULL | — |
| project_id | INTEGER | NULL | — |
| environment_id | INTEGER | NULL | — |
| scope_level | secret_scope_level_type | NOT NULL | — |
| name | TEXT | NOT NULL | — |
| description | TEXT | NULL | — |
| secret_type | secret_type | NOT NULL | — |
| provision_status | secret_provision_status_type | NOT NULL | 'provisioned' |
| fulfillment_scope | secret_fulfillment_scope_type | NULL | — |
| placeholder_hint | TEXT | NULL | — |
| key_id | TEXT | NULL | — |
| secret_enc | TEXT | NULL | — |
| secret_policy | secret_policy_type | NOT NULL | 'open' |
| visibility | TEXT | NOT NULL | 'hidden' |
| is_active | BOOLEAN | NOT NULL | TRUE |
| is_system_provided | BOOLEAN | NOT NULL | FALSE |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| revoked_at | TIMESTAMPTZ | NULL | — |

**Constraints:**
- PK: `secret_store_id`; UNIQUE: `secret_store_uuid`
- FK: `company_id` → company(company_id)
- FK (composite): `(company_id, workspace_id)` → workspace(company_id, workspace_id)
- FK (composite): `(company_id, project_id)` → project(company_id, project_id)
- FK (composite): `(company_id, environment_id)` → environment(company_id, environment_id)
- UNIQUE: `(company_id, secret_store_id)`
- CHECK (scope integrity): workspace/project scope requires workspace_id NOT NULL; project scope requires project_id NOT NULL; workspace_id must be NULL unless scope is workspace/project; project_id must be NULL unless scope is project
- CHECK (provisioning integrity): placeholder rows must have secret_enc NULL; provisioned rows must have secret_enc NOT NULL; fulfillment_scope allowed only when provision_status = 'placeholder'; a placeholder may not have secret_policy = 'locked'
- CHECK: visibility IN ('hidden', 'visible')
- `key_id` identifies the KEK used to envelope-encrypt `secret_enc`; `secret_enc` is write-only (never returned by read APIs). `secret_policy` = 'open' lets lower scopes override, 'locked' forbids it.

**Indexes:**
- UNIQUE `(company_id, name)` WHERE scope_level = 'company' AND is_active = TRUE
- UNIQUE `(company_id, workspace_id, name)` WHERE scope_level = 'workspace' AND is_active = TRUE
- UNIQUE `(company_id, workspace_id, project_id, name)` WHERE scope_level = 'project' AND is_active = TRUE
- `(company_id, scope_level)`
- `(company_id, workspace_id)` WHERE workspace_id IS NOT NULL
- `(company_id, project_id)` WHERE project_id IS NOT NULL

### agent_definition_secret_ref

Declares which secrets an agent definition requires, with injection method and launch-blocking config; resolved at runtime by logical name via a top-down scope walk against secret_store (deliberately no FK to secret_store).

| column | type | null | default |
|---|---|---|---|
| agent_definition_secret_ref_id | SERIAL | NOT NULL | — (PK) |
| agent_definition_secret_ref_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| agent_definition_id | INTEGER | NOT NULL | — |
| secret_name | TEXT | NOT NULL | — |
| secret_type | secret_type | NOT NULL | — |
| usage_context | TEXT | NOT NULL | — |
| context_identifier | TEXT | NULL | — |
| injection_method | TEXT | NOT NULL | 'template' |
| inject_as | TEXT | NULL | — |
| file_mode | TEXT | NULL | '0600' |
| required | BOOLEAN | NOT NULL | TRUE |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `agent_definition_secret_ref_id`; UNIQUE: `agent_definition_secret_ref_uuid`
- FK: `company_id` → company(company_id)
- FK (composite): `(company_id, agent_definition_id)` → agent_definition(company_id, agent_definition_id)
- FK: `agent_definition_id` → agent_definition(agent_definition_id)
- UNIQUE: `(company_id, agent_definition_secret_ref_id)`
- CHECK: usage_context IN ('mcp_server', 'skill', 'command', 'hook', 'env', 'template')
- CHECK: injection_method IN ('template', 'env_var', 'file')
- CHECK: injection_method = 'env_var' requires inject_as NOT NULL unless secret_type = 'env_vars'
- `secret_type` is the expected type, validated at binding time against the resolved secret_store row. `required` = TRUE blocks session launch on an unresolved secret; FALSE allows degraded startup. `context_identifier` names the specific MCP server/skill/command/hook (NULL for env/template/general usage).

**Indexes:**
- UNIQUE expression index on `(agent_definition_id, secret_name, usage_context, COALESCE(context_identifier, ''))` — a secret may bind to multiple contexts within one definition
- `(company_id)` WHERE company_id IS NOT NULL
- `(agent_definition_id)`

### Enum types used by these tables

| enum | values |
|---|---|
| agent_definition_scope_type | company, workspace, project |
| coding_agent_type | claude_code, codex_cli, cursor_cli, opencode, gemini_cli |
| merge_strategy_type | append, replace, deep_merge |
| file_policy_type | open, append_only, locked |
| secret_scope_level_type | company, workspace, project |
| secret_type | oauth, bearer_token, api_key, ssh_key, basic_auth, env_vars |
| secret_provision_status_type | placeholder, provisioned, revoked |
| secret_fulfillment_scope_type | workspace, project, environment |
| secret_policy_type | open, locked |
