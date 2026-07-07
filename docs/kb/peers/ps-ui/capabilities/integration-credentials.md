---
type: capability
title: "Integration credentials (coding-agent auth)"
tags: [integrations, credentials, coding-agent, claude-code, codex, consumer]
timestamp: 2026-07-07T06:27:35Z
description: "The ps-api integration-credential endpoints ps-ui consumes to manage coding-agent (Claude Code / Codex) auth used at runtime launch"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/integrations.ts
  - src/api/integrations.test.ts
  - src/components/integrations/integration-schema.ts
  - src/components/integrations/IntegrationSchemaForm.tsx
  - src/components/account-settings/MyIntegrations.container.tsx
  - src/components/company-settings/CompanyIntegrations.container.tsx
see_also:
  - {repo: ps-api, capability: "Integration credentials gateway", intent: "owns the /v1/integrations endpoints this repo calls; enforces the write-only credential boundary + server-side encryption", descriptive: true}
  - {repo: orchestrator, capability: "Runtime credential resolution", intent: "resolves and freezes the coding-agent credential into a runtime at launch (personal-first)", descriptive: true}
---

# Integration credentials (coding-agent auth)

**What it does.** Lets a user register, scope, and manage the credentials a coding agent
(Claude Code / Codex) authenticates with — API keys or OAuth setup tokens — plus assign a
company-shared credential to specific workspaces. ps-ui is a **pure consumer**: it renders these
screens and calls ps-api; it stores and encrypts nothing itself. The credential a launched runtime
actually uses is resolved server-side at launch time (see peers).

**How a peer interacts.** This repo owns no wire API. It calls ps-api (`:9004`) endpoints, all under
`/v1/integrations/...` (the api-client base URL already ends in `/api`):
- `GET  /v1/integrations/providers` — provider catalog.
- `GET  /v1/integrations/providers/{provider_uuid}/auth-types` — the auth types for a provider, each
  declaring its input fields (see Contract).
- `GET  /v1/integrations/connections` — the caller's connections (shared + own personal).
- `POST /v1/integrations/connections` — create a connection (the only place a secret leaves the client).
- `PATCH /v1/integrations/connections/{uuid}` — update non-secret fields only (never the credential).
- `DELETE /v1/integrations/connections/{uuid}` — **revoke** (destructive: deletes the stored secret).
- `POST /v1/integrations/connections/{uuid}/disable` · `/enable` — non-destructive pause / restore.
- `GET  /v1/integrations/connections/eligible` — connections eligible for use.
- `GET  /v1/workspaces/{ws}/integrations` — connections assigned to a workspace.
- `POST /v1/workspaces/{ws}/integrations` — assign **one** connection · `DELETE …/{connUuid}` — unassign.

**Contract.** Auth types are **data-driven**: each declares `field_schema` (secret fields → encrypted
server-side) and `config_schema` (non-secret cleartext). ps-ui renders a form straight from those and
never hardcodes provider fields; `auth_type` is an **opaque string** to the UI (it drives server-side
runtime dispatch, not UI branching).
- Create body `key fields:` `{provider_code, auth_type, fields{}, config{}, display_name?, labels?,
  visible_to_all_workspaces?, personal?}`. The secret goes in `fields` (wire key literally `fields`).
- Read shape `key fields:` `{integration_connection_uuid, provider_code, auth_type, has_credential,
  personal_only, owned_by_me, visible_to_all_workspaces, config{}, labels{}, status, created_at,
  last_used_at}`.
- **Coding-agent credential types (platform fact, owned by ps-api / orchestrator, not enforced here):**
  a Claude OAuth setup token (`sk-ant-oat01-…`) uses `auth_type = claude_oauth_setup_token` with secret
  field `oauth_token`; a Claude API key (`sk-ant-api03-…`) uses `auth_type = claude_api_key` with secret
  field `api_key`. ps-ui presents whichever fields the server's auth-type declares — the specific
  auth_type↔field mapping is defined server-side, not in this repo.
- Errors surface as the api-client `ApiClientError` (`status` + message); write failures become a
  destructive toast carrying the server's message.

**Observable behavior.** `has_credential: boolean` is the **only** credential signal on any read — read
payloads NEVER carry the ciphertext or key id. Secrets are **write-once**: created via POST, never
returned, never editable via PATCH. `status` is one of `active | disabled | revoked | error`. Assignment
is scoped per workspace; the connection-side "which workspaces?" read is unimplemented server-side
(404), so allocation is derived by fanning out over each workspace's assigned list.

**Failure modes.** A credential write that fails closed server-side (e.g. no KMS available, so the
secret can't be encrypted) returns an error from ps-api; ps-ui does not special-case it — it surfaces
the server's message as a destructive toast and the connection is not created. On `disable`/`enable`,
HTTP 404 (gone) and 409 (already in target state) are treated as **soft** — swallowed, followed by a
refetch to reconcile with server truth — rather than shown as errors.

**Gotchas.**
- **Secret rotation = revoke + re-add.** There is no "change the key" path; PATCH intentionally omits any
  credential field. Editing only touches display name / config / visibility.
- **Disable ≠ revoke.** Disable is a reversible pause (secret retained, credential not resolvable);
  revoke permanently deletes the secret. Sessions/workspaces relying on a disabled/revoked credential
  fall back to another available credential or the platform default.
- **Credential is frozen into the runtime at launch** (server-side): a credential change does not affect
  an already-running runtime — a fresh runtime is needed to pick it up.
- **Personal-first resolution** (server-side): for the owning user, a personal credential is preferred
  over a shared one at launch; alert / service-principal sessions need a **shared** credential.
- **Scope split.** `owned_by_me` credentials are the "My Integrations" (personal) screen; the rest are
  the org-wide "Company Settings" screen. `personal: true` on create makes even a non-`personal_only`
  method (e.g. a user's own API key) personal and slotless; a `personal_only` auth type is always personal.
- **Assign takes a single `integration_connection_uuid`**, not an array — distinct from the Git-connection
  assign endpoint, which takes a `connection_uuids` array. Callers loop to map one connection to many
  workspaces.

**See also / peers.** ps-api — *Integration credentials gateway* (owns these endpoints, the write-only
credential boundary, and server-side encryption). orchestrator — *Runtime credential resolution*
(resolves + freezes the coding-agent credential into a runtime at launch, personal-first).
