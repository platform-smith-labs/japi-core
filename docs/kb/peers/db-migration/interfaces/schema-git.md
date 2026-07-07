---
type: interface
title: "Schema: git integration"
tags: [schema, postgres, git]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for the git provider/connection tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0004_git.sql
  - migrations/0001_enums.sql
  - migrations/0006_project_git_link.sql
provides_interfaces:
  - {name: "git tables", kind: postgres-schema, intent: "git provider, connection, installation and OAuth state"}
---

# Schema: git integration

Five tables covering git-hosting integration: a global provider catalog, company-scoped auth connections, GitHub App installations, single-use OAuth CSRF nonces, and a workspace-access assignment table.

### git_provider

Deployment-wide catalog of git hosting providers (global — no `company_id`). Non-secret catalog data only; platform secrets live in environment variables. Provider rows (e.g. github.com) are seeded at migration time.

| column | type | null | default |
|---|---|---|---|
| git_provider_id | SERIAL (PK) | no | auto |
| git_provider_uuid | UUID | no | gen_random_uuid() |
| type | TEXT | no | — |
| display_name | TEXT | no | — |
| host | TEXT | no | — |
| api_base_url | TEXT | no | — |
| auth_url | TEXT | no | — |
| token_url | TEXT | no | — |
| oauth_client_id | TEXT | yes | — |
| github_app_id | TEXT | yes | — |
| github_app_slug | TEXT | yes | — |
| scopes_default | TEXT | yes | — |
| capabilities | JSONB | no | '{}' |
| enabled | BOOLEAN | no | TRUE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `git_provider_id`; UNIQUE `git_provider_uuid`; UNIQUE `(host)`; CHECK `type IN ('github','gitlab','bitbucket','generic')`.

**Indexes:** `idx_git_provider_enabled` on `(enabled)` WHERE `enabled = TRUE` (partial).

### git_connection

Company-scoped binding to a git_provider, holding auth material (encrypted tokens / SSH keys). `user_id` NULL = company-shared; NOT NULL = user-owned. Workspace access is granted via workspace_git_connection.

| column | type | null | default |
|---|---|---|---|
| git_connection_id | SERIAL (PK) | no | auto |
| git_connection_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| user_id | INTEGER | yes | — |
| git_provider_id | INTEGER | no | — |
| kind | TEXT | no | — |
| key_id | TEXT | yes | — |
| access_token_enc | TEXT | yes | — |
| refresh_token_enc | TEXT | yes | — |
| expires_at | TIMESTAMPTZ | yes | — |
| ssh_public_key | TEXT | yes | — |
| ssh_private_key_enc | TEXT | yes | — |
| scopes_granted | TEXT | yes | — |
| display_name | TEXT | yes | — |
| status | TEXT | no | 'active' |
| visible_to_all_workspaces | BOOLEAN | no | TRUE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| last_used_at | TIMESTAMPTZ | yes | — |
| revoked_at | TIMESTAMPTZ | yes | — |

**Constraints:** PK `git_connection_id`; UNIQUE `git_connection_uuid`; FK `company_id → company(company_id)`; composite FK `(company_id, user_id) → users(company_id, user_id)`; FK `git_provider_id → git_provider(git_provider_id)`; UNIQUE `(company_id, git_connection_id)`; CHECK `kind IN ('oauth','app_install','pat','ssh')`; CHECK `status IN ('active','revoked','expired','error')`.

**Indexes:**
- `idx_git_connection_company_id` on `(company_id)`
- `idx_git_connection_company_shared_unique` UNIQUE on `(company_id, git_provider_id)` WHERE `user_id IS NULL AND status <> 'revoked'` (partial — at most one non-revoked company-shared connection per provider)
- `idx_git_connection_company_user_owned_unique` UNIQUE on `(company_id, git_provider_id, user_id)` WHERE `user_id IS NOT NULL AND status <> 'revoked'` (partial — at most one non-revoked user-owned connection per provider per user)
- `idx_git_connection_company_active` on `(company_id, status)` WHERE `status = 'active'` (partial)

`revoked_at` is a soft-revocation timestamp; NULL means never revoked.

### git_installation

GitHub App installation binding; installation tokens are minted on demand from this row. `provider_installation_id` is the provider's numeric installation id (TEXT for forward-compat) and is never exposed externally — API responses use `git_installation_uuid`. `suspended_at` set = suspended/revoked; NULL = active.

| column | type | null | default |
|---|---|---|---|
| git_installation_id | SERIAL (PK) | no | auto |
| git_installation_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| git_connection_id | INTEGER | no | — |
| installed_by_user_id | INTEGER | yes | — |
| provider_installation_id | TEXT | no | — |
| account_login | TEXT | yes | — |
| account_type | TEXT | yes | — |
| repository_selection | TEXT | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| suspended_at | TIMESTAMPTZ | yes | — |

**Constraints:** PK `git_installation_id`; UNIQUE `git_installation_uuid`; FK `company_id → company(company_id)`; composite FK `(company_id, git_connection_id) → git_connection(company_id, git_connection_id)`; composite FK `(company_id, installed_by_user_id) → users(company_id, user_id)`; UNIQUE `(company_id, git_installation_id)`; CHECK `account_type IN ('User','Organization')`; CHECK `repository_selection IN ('all','selected')`.

**Indexes:**
- `idx_git_installation_company_id` on `(company_id)`
- `idx_git_installation_company_connection_provider_active` UNIQUE on `(company_id, git_connection_id, provider_installation_id)` WHERE `suspended_at IS NULL` (partial — one active row per installation)

### git_oauth_state

Single-use CSRF nonce for GitHub App OAuth install/callback. Only the SHA-256 hex of the state value is stored; plaintext is never persisted. `user_id` NULL for system-initiated flows.

| column | type | null | default |
|---|---|---|---|
| git_oauth_state_id | SERIAL (PK) | no | auto |
| git_oauth_state_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| user_id | INTEGER | yes | — |
| git_provider_id | INTEGER | no | — |
| state_hash | TEXT | no | — |
| redirect_uri | TEXT | no | — |
| expires_at | TIMESTAMPTZ | no | — |
| consumed_at | TIMESTAMPTZ | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `git_oauth_state_id`; UNIQUE `git_oauth_state_uuid`; FK `company_id → company(company_id)`; composite FK `(company_id, user_id) → users(company_id, user_id)`; FK `git_provider_id → git_provider(git_provider_id)`; UNIQUE `(company_id, git_oauth_state_id)`.

**Indexes:**
- `idx_git_oauth_state_unconsumed_unique` UNIQUE on `(state_hash)` WHERE `consumed_at IS NULL` (partial; deliberately NOT company-leading — the nonce is cryptographically unique and enforced globally single-use; expiry sweeping is cross-tenant)
- `idx_git_oauth_state_expires_unconsumed` on `(expires_at)` WHERE `consumed_at IS NULL` (partial)

### workspace_git_connection

Assignment table: which workspaces can access which company-scoped git connections. `assigned_by_user_id` NULL for system-initiated assignments (backfill, auto-assign).

| column | type | null | default |
|---|---|---|---|
| workspace_git_connection_id | SERIAL (PK) | no | auto |
| workspace_git_connection_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workspace_id | INTEGER | no | — |
| git_connection_id | INTEGER | no | — |
| assigned_by_user_id | INTEGER | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| assigned_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `workspace_git_connection_id`; UNIQUE `workspace_git_connection_uuid`; FK `company_id → company(company_id)`; composite FK `(company_id, workspace_id) → workspace(company_id, workspace_id)`; composite FK `(company_id, git_connection_id) → git_connection(company_id, git_connection_id)`; composite FK `(company_id, assigned_by_user_id) → users(company_id, user_id)`; UNIQUE `(company_id, workspace_id, git_connection_id)`; UNIQUE `(company_id, workspace_git_connection_id)`.

**Indexes:**
- `idx_workspace_git_connection_company_id` on `(company_id)`
- `idx_workspace_git_connection_company_workspace` on `(company_id, workspace_id)`
- `idx_workspace_git_connection_company_connection` on `(company_id, git_connection_id)`

### Inbound references

The `project_git_link` table (project↔repo binding, documented separately) holds composite FKs into `git_connection(company_id, git_connection_id)` and `git_installation(company_id, git_installation_id)`.

### ENUM types

None. These tables use no PostgreSQL ENUM types; all constrained value sets are TEXT columns with CHECK constraints (value sets listed per table above).
