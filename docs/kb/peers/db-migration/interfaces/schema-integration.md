---
type: interface
title: "Schema: integrations"
tags: [schema, postgres, integration]
timestamp: 2026-07-09T10:37:36Z
description: "Final-state reference for integration provider/connection tables"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - migrations/0025_agent_definitions_and_secrets.sql
  - migrations/0037_integration_enums.sql
  - migrations/0038_integration_provider.sql
  - migrations/0039_integration_connection.sql
  - migrations/0040_workspace_integration_connection.sql
  - migrations/0044_codex_subscription_personal_only.sql
  - migrations/0045_integration_status_disabled.sql
  - migrations/0046_personal_workspace_scope.sql
  - migrations/0048_slack_auth_type_enum.sql
  - migrations/0049_slack_provider_seed.sql
  - migrations/0059_llm_provider_claude.sql
provides_interfaces:
  - {name: "integration tables", kind: postgres-schema, intent: "integration providers, auth types, credential connections and workspace links"}
---

# Schema: integrations

Third-party integration substrate: a global provider catalog with declarative per-auth-type form
schemas, company-scoped encrypted credential connections, and workspace assignment rows.
Cardinality ("one effective connection per provider") is enforced entirely in the database via
denormalized `enforce_single` + partial unique indexes — no application guard.

**Gotcha (wire contract):** the three partial unique indexes named WITHOUT the `idx_` prefix
(`integration_connection_company_shared_name_unique`, `integration_connection_company_user_name_unique`,
`integration_connection_global_slot_unique`, plus `workspace_integration_connection_provider_slot_unique`)
are contract strings — on a unique violation Postgres reports the index relname verbatim and ps-api maps
those exact names to 409 responses. They must never be renamed.

### integration_provider
Deployment-wide (global, no company_id) catalog of third-party integration providers; non-secret catalog data only.

| column | type | null | default |
|---|---|---|---|
| integration_provider_id | SERIAL | NOT NULL | — (PK) |
| integration_provider_uuid | UUID | NOT NULL | gen_random_uuid() |
| code | TEXT | NOT NULL | — |
| display_name | TEXT | NOT NULL | — |
| category | TEXT | NOT NULL | — |
| coding_agent_type | coding_agent_type | NULL | — |
| cardinality | TEXT | NOT NULL | 'single' |
| capabilities | JSONB | NOT NULL | '{}' |
| labels | JSONB | NOT NULL | '{}' |
| enabled | BOOLEAN | NOT NULL | TRUE |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `integration_provider_id`; UNIQUE: `integration_provider_uuid`
- UNIQUE: `code` (globally unique stable provider code, e.g. `claude_code`, `codex`)
- CHECK `integration_provider_category_check`: category IN ('coding_agent','issue_tracker','docs','ci','cloud','sandbox','vcs','observability','notification','registry','communication','llm')
- CHECK `integration_provider_cardinality_check`: cardinality IN ('single','multiple')

**Indexes:**
- `idx_integration_provider_enabled` on (enabled) WHERE enabled = TRUE (partial)

Notes: `coding_agent_type` is set only for coding-agent providers (launch-resolver key); NULL otherwise.
`cardinality` is immutable once any connection exists; `'single'` denormalizes into `enforce_single` on
connections at INSERT. The `llm` category (e.g. the `anthropic` "LLM Provider - Claude" row) is
distinct from `coding_agent`: an `llm` provider makes runtime-less structured Messages-API calls with a
plain API key (`coding_agent_type` = NULL, auth type `claude_api_key`) and must not be conflated with
the `claude_code` coding-agent provider whose OAuth setup token launches Claude Code sessions.

### integration_provider_auth_type
Per-provider auth mechanisms with declarative form schemas: `field_schema` describes SECRET fields (encrypted into a connection's `credential_enc`), `config_schema` describes non-secret config (stored cleartext in a connection's `config`).

| column | type | null | default |
|---|---|---|---|
| integration_provider_auth_type_id | SERIAL | NOT NULL | — (PK) |
| integration_provider_auth_type_uuid | UUID | NOT NULL | gen_random_uuid() |
| integration_provider_id | INTEGER | NOT NULL | — |
| auth_type | integration_auth_type | NOT NULL | — |
| display_name | TEXT | NOT NULL | — |
| personal_only | BOOLEAN | NOT NULL | FALSE |
| field_schema | JSONB | NOT NULL | '[]' |
| config_schema | JSONB | NOT NULL | '[]' |
| sort_order | INTEGER | NOT NULL | 0 |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `integration_provider_auth_type_id`; UNIQUE: `integration_provider_auth_type_uuid`
- FK: (integration_provider_id) → integration_provider(integration_provider_id)
- UNIQUE: (integration_provider_id, auth_type)

**Indexes:**
- `idx_integration_provider_auth_type_provider` on (integration_provider_id)

Notes: `personal_only = TRUE` means connections of this auth type are user-owned and cannot be shared
(ps-api forces the connection's user_id to the requester at create time). Field NAMES inside
`field_schema`/`config_schema` are a wire contract with ps-api validation and connector decryption —
never renamed.

### integration_connection
Company-scoped credential instance for a provider. `credential_enc` is encrypted by ps-api and decrypted only by the orchestrator launch resolver; it is never selected onto the wire.

| column | type | null | default |
|---|---|---|---|
| integration_connection_id | SERIAL | NOT NULL | — (PK) |
| integration_connection_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| user_id | INTEGER | NULL | — |
| integration_provider_id | INTEGER | NOT NULL | — |
| auth_type | integration_auth_type | NOT NULL | — |
| display_name | TEXT | NOT NULL | — |
| credential_enc | TEXT | NULL | — |
| key_id | TEXT | NULL | — |
| config | JSONB | NOT NULL | '{}' |
| labels | JSONB | NOT NULL | '{}' |
| status | integration_status | NOT NULL | 'active' |
| visible_to_all_workspaces | BOOLEAN | NOT NULL | TRUE |
| enforce_single | BOOLEAN | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| last_used_at | TIMESTAMPTZ | NULL | — |
| expires_at | TIMESTAMPTZ | NULL | — |
| revoked_at | TIMESTAMPTZ | NULL | — |

**Constraints:**
- PK: `integration_connection_id`; UNIQUE: `integration_connection_uuid`
- FK: (company_id) → company(company_id)
- FK composite: (company_id, user_id) → users(company_id, user_id)
- FK: (integration_provider_id) → integration_provider(integration_provider_id)
- UNIQUE: (company_id, integration_connection_id)
- UNIQUE: (company_id, integration_connection_id, integration_provider_id) — composite FK target proving provider match on assignment rows

**Indexes:**
- `idx_integration_connection_company_id` on (company_id)
- `integration_connection_company_shared_name_unique` UNIQUE on (company_id, integration_provider_id, display_name) WHERE user_id IS NULL AND status <> 'revoked'
- `integration_connection_company_user_name_unique` UNIQUE on (company_id, integration_provider_id, display_name, user_id) WHERE user_id IS NOT NULL AND status <> 'revoked'
- `integration_connection_global_slot_unique` UNIQUE on (company_id, COALESCE(user_id, 0), integration_provider_id) WHERE enforce_single AND visible_to_all_workspaces = TRUE AND status = 'active'
- `idx_integration_connection_labels_gin` GIN on (labels jsonb_path_ops)
- `idx_integration_connection_resolve` on (company_id, integration_provider_id, created_at DESC) WHERE status = 'active'

Notes: `user_id` NULL = company-shared, set = personal (owner-only). The global-slot index buckets by
COALESCE(user_id, 0): shared rows share one org-global slot; each user gets one personal global slot
(user_id SERIAL starts at 1, so 0 is a safe shared sentinel). A revoked connection's display_name is
reusable; a `disabled` connection keeps credential_enc/key_id intact (unlike `revoked`) and is excluded
from launch resolution because the resolver filters status = 'active'. `enforce_single` is denormalized
from provider cardinality = 'single' at INSERT and immutable.

### workspace_integration_connection
Assignment row: which workspaces can use which company integration connections. Denormalizes provider, enforce_single, and owner user_id so cardinality slots are DB-enforced atomically.

| column | type | null | default |
|---|---|---|---|
| workspace_integration_connection_id | SERIAL | NOT NULL | — (PK) |
| workspace_integration_connection_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NOT NULL | — |
| integration_connection_id | INTEGER | NOT NULL | — |
| integration_provider_id | INTEGER | NOT NULL | — |
| enforce_single | BOOLEAN | NOT NULL | — |
| assigned_by_user_id | INTEGER | NULL | — |
| user_id | INTEGER | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| assigned_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `workspace_integration_connection_id`; UNIQUE: `workspace_integration_connection_uuid`
- FK: (company_id) → company(company_id)
- FK composite: (company_id, workspace_id) → workspace(company_id, workspace_id)
- FK composite: (company_id, integration_connection_id, integration_provider_id) → integration_connection(company_id, integration_connection_id, integration_provider_id) — guarantees the denormalized provider matches the connection's real provider
- FK composite: (company_id, assigned_by_user_id) → users(company_id, user_id)
- FK composite: (company_id, user_id) → users(company_id, user_id)
- UNIQUE: (company_id, workspace_id, integration_connection_id)
- UNIQUE: (company_id, workspace_integration_connection_id)

**Indexes:**
- `idx_workspace_integration_connection_company_id` on (company_id)
- `idx_workspace_integration_connection_company_workspace` on (company_id, workspace_id)
- `workspace_integration_connection_provider_slot_unique` UNIQUE on (company_id, workspace_id, COALESCE(user_id, 0), integration_provider_id) WHERE enforce_single

Notes: `user_id` is denormalized from the connection's owner at assign time (NULL = shared/org
assignment; set = personal assignment owned by that user) — it partitions the per-(workspace, provider)
slot so one user's personal assignment never collides with a teammate's or the org's.
`assigned_by_user_id` is NULL for system-initiated assignments. `enforce_single` is denormalized from
the connection at INSERT.

## Enum types used

- **integration_auth_type** (provider-prefixed auth mechanism; launch resolver + runtime dispatcher branch on it): `claude_api_key`, `claude_bedrock`, `claude_vertex`, `claude_oauth_setup_token`, `codex_api_key`, `codex_chatgpt_subscription`, `linear_api_key`, `jira_api_token`, `notion_api_key`, `github_actions_pat`, `aws_iam_keys`, `aws_assume_role`, `azure_service_principal`, `gcp_service_account`, `cloudflare_api_token`, `e2b_api_key`, `modal_token`, `daytona_api_key`, `slack_bot_token`
- **integration_status** (connection lifecycle): `active`, `revoked`, `expired`, `error`, `disabled`
- **coding_agent_type** (shared enum, referenced by integration_provider.coding_agent_type): `claude_code`, `codex_cli`, `cursor_cli`, `opencode`, `gemini_cli`
