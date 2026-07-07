---
type: capability
title: "Workspace and tenant administration"
tags: [workspaces, users, environments, controllers, tokens, admin]
timestamp: 2026-07-07T03:33:49Z
description: "Company-scoped admin surface: workspaces, users, workspace/controller tokens, environments, controller inventory, audit log"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/workspaces.go
  - cmd/handlers/users.go
  - cmd/handlers/slack_links.go
  - cmd/handlers/workspace_tokens.go
  - cmd/handlers/environments.go
  - cmd/handlers/workspace_devops.go
  - cmd/handlers/controllers.go
  - cmd/handlers/controller_tokens.go
  - cmd/db/workspace_operations.go
  - cmd/db/workspace_token_operations.go
  - cmd/db/auth_operations.go
  - cmd/db/controller_operations.go
see_also:
  - {repo: ps-api, capability: "Auth and identity gateway", intent: "issues the JWT whose company claim scopes every admin call"}
  - {repo: orchestrator, capability: "Environment lifecycle", intent: "owns environment write validation (single-default invariant, live-controller check) behind the verbatim proxy", descriptive: true}
  - {repo: orchestrator, capability: "Controller token registry", intent: "mints and validates the controller tokens ps-api proxies; a new controller instance authenticates with this token", descriptive: true}
  - {repo: controller, capability: "Controller identity and bootstrap", intent: "consumes a minted controller token as its identity when connecting", descriptive: true}
---

# Workspace and tenant administration

**What it does.** The admin surface for a company's tenancy: manage workspaces,
company users, workspace tokens, environments, controller inventory, and the
workspace audit log. Mixed implementation: most reads/CRUD are served directly
from the shared database; environment writes and controller-token minting are
proxied to orchestrator, which owns their validation.

**How a peer interacts.** JWT-authenticated REST under `/api/v1`:
`workspaces` (CRUD), `users` (create + company roster list),
`workspaces/{id}/tokens` (mint/list/revoke), `workspaces/{id}/environments`
(+ `environments/{uuid}` detail), `workspaces/{id}/audit`, `controllers`
(+ per-name detail), `controller-tokens` (mint/list). A controllers health
summary route under a workspace is proxied raw to orchestrator.

**Observable behavior.**
- Workspace update is a **full replace** of the mutable fields (name,
  description, settings) — not a field-wise patch. Delete = soft archive;
  archived workspaces disappear from lists and reject token minting.
- Workspace token mint returns the raw token (`pst_…`) **exactly once**; only
  its SHA-256 hash is stored, so it can never be retrieved again. List returns
  active and revoked tokens, never raw values. Revoke is soft (row retained
  for audit); revoking an already-revoked token returns 404.
- Controller-token mint is proxied to orchestrator; the raw token value is
  likewise returned only in the creation response. This token is how a new
  controller instance gets its identity.
- Environment list/get are DB-direct reads (list is live-only by default;
  `?include_torn_down=true` includes tombstones). Environment create/update/
  archive are forwarded verbatim to orchestrator — its status codes and error
  bodies pass through unchanged. Archive accepts `?force=true` to archive an
  environment that still has live runtimes.
- Controller inventory aggregates per-controller health: connected instance
  count, last-seen/connected-at, total vs active runtime counts.
- The workspace audit log is keyset-paginated (opaque `cursor` from the
  previous page's `next_cursor`; filters: `actor`, `action_type`, `since`,
  `until`, `limit` 1–200, default 50).

**Contract.** Entities are addressed by UUID except **controllers, which are
keyed by name**. Workspace key fields: `workspace_uuid, name, description,
settings (JSON), is_archived`. User create takes `name, email, password` and
returns the user without the password. Duplicate names/emails within scope
return 409 (workspace name per company, token name per workspace, user email).

**Invariants.** Every route is scoped to the caller's JWT company; a resource
outside that company behaves exactly like a missing one (404-for-both — see
context.md for the tenancy rule). Raw token values are never logged or listed.

**Failure modes.** Invalid UUID path params → 400. Not-found / cross-tenant /
archived-target → 404. Scope-unique conflicts → 409. Orchestrator unreachable
on proxied routes → gateway-mapped upstream error.

**Gotchas.**
- The workspace PATCH is semantically a PUT: omitting a field clears it.
- Losing a token creation response means re-minting — there is no recovery.
- User listing excludes service principals server-side; the roster is not a
  complete `users`-table dump.
- Environment write errors originate in orchestrator (e.g. its single-default
  and live-controller rules), so their shape differs from DB-direct routes.

**Business-critical data.** `workspace` (name unique per company, soft-archive
flag), `workspace_token` (stores token hash only, soft-revoke), `users` (email
unique), `controller` + `controller_instance` + `runtime` (health/count
aggregation source), audit events table for the workspace audit log.
