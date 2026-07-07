---
type: capability
title: "Git integration gateway"
tags: [git, github-app, oauth, connections, project-git-link, gateway]
timestamp: 2026-07-07T03:33:49Z
description: "GitHub App connection lifecycle (install, callback, grants) and project↔repo linking, fronted by the gateway"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/git.go
  - cmd/handlers/project_git_link.go
  - cmd/handlers/passthrough.go
  - cmd/server/main.go
see_also:
  - {repo: orchestrator, capability: "Git connection management", intent: "owns OAuth state validation, connection CRUD truth, and GitHub App installation-token minting; the App private key never leaves orchestrator", descriptive: true}
---

# Git integration gateway

**What it does.** Fronts the GitHub App integration for the UI: install a git connection via an OAuth-style flow, manage which workspaces may use a connection, and link a project to a specific repository. Most routes authenticate and forward to orchestrator, which owns the truth; a few reads are answered from the shared DB directly.

**How a peer interacts.** JWT-authenticated REST under `/api/v1/git/*`, `/api/v1/workspaces/{workspace_uuid}/git/*`, and `/api/v1/projects/{project_uuid}/git-link`:
- `GET /git/providers` — global provider catalog (DB-direct).
- `POST /git/connections/install` — begin the install flow (proxied).
- `GET /git/connections` and `GET /git/connections/{id}/repos` — raw reverse-proxy passthroughs to orchestrator (see Gotchas).
- `GET /git/connections/eligible` — connections usable for workspace creation (proxied, query string forwarded).
- `PATCH` / `DELETE /git/connections/{id}` — update properties (e.g. visibility to all workspaces) / revoke (proxied).
- `GET` / `PUT /git/connections/{id}/workspaces` — list (DB-direct) / bulk-set (proxied) workspace grants.
- `GET` / `POST /workspaces/{ws}/git/connections`, `DELETE …/{id}` — per-workspace grant list/assign/unassign (proxied).
- `GET /git/callback` — unauthenticated OAuth callback GitHub drives the browser to (raw forward; a legacy workspace-scoped callback path also survives for in-flight flows).
- `GET` / `PUT` / `DELETE /projects/{project_uuid}/git-link` — read (DB-direct) / create-or-replace / remove a project's repo link (writes proxied verbatim; orchestrator owns validation).

**Observable behavior.** Proxied routes return orchestrator's response body; passthrough and git-link write routes return it byte-for-byte including status. The callback forwards orchestrator's 303 + Location back to the browser rather than following it — the redirect targets the user's machine. Repo enumeration (`…/repos`) reflects what the GitHub App installation can see at request time.

**Contract.** Inputs on proxied writes are forwarded verbatim — orchestrator's wire IS the contract; ask orchestrator's KB for shapes. Gateway-owned shapes: provider entries (key fields: `git_provider_uuid`, `type`, `display_name`, `host`, `enabled`); git-link (key fields: `project_git_link_uuid`, `git_connection_uuid`, `git_installation_uuid`, `repo_url`, `default_branch`, `auto_clone_on_launch`, `clone_depth`, `github_repo_id`). Identifier bridge: a project's git-link carries both the human `repo_url` and the provider-native `github_repo_id` for the same repository, plus the `git_connection_uuid` that authorizes access to it.

**Invariants.**
- All routes require a valid JWT except the two OAuth callback routes (GitHub arrives with no token; orchestrator validates the OAuth `state` instead).
- Connection, grant, and git-link data are company-scoped to the caller; the provider catalog is deliberately global (same list for every tenant).
- The GitHub App private key and installation-token minting live only in orchestrator; this gateway never sees or stores git credentials.
- A project without a git-link is a legal "local-only" state, not an error condition of the system.

**Failure modes.**
- Orchestrator unreachable → 502 on the raw callback/passthrough routes; typed proxied routes
  map it to 503 (network/retries exhausted) or 504 (timeout).
- `GET …/git-link` → 404 both for unknown project and for a project with no link — callers cannot distinguish the two from status alone.
- Malformed UUIDs in paths → 400 at the gateway, before any upstream call.
- Passthrough routes → 401 at the gateway if the JWT or user/company validation fails.

**Gotchas.**
- `GET /git/connections` (list) and `GET /git/connections/{id}/repos` are raw streaming passthroughs, not typed gateway routes: no response reshaping, and orchestrator's CORS headers are stripped so the gateway's own CORS is authoritative. Their wire shape is owned entirely by orchestrator.
- The callback proxy intentionally does NOT follow redirects; a client (or test) that expects a 200 from the callback will instead see the 303 meant for the browser.
- Empty upstream bodies on deletes are normalized to `{}` (JSON), not 204-with-empty-body, on the typed delete routes; the git-link delete is verbatim and does return orchestrator's 204.
- A workspace-scoped legacy callback path still exists for pending GitHub App callback-URL configurations; new flows use the company-scoped `/git/callback`.

**Business-critical data.** `git_provider` — global catalog (no tenant column; querying it per-company would return nothing). Workspace-grant rows link `workspace_uuid` ↔ `git_connection_uuid` (read directly for the grants list). `project_git_link` — one row per linked project keyed by `company_uuid` + `project_uuid` (sufficient for tenant isolation on reads); its `git_connection_uuid` / `git_installation_uuid` / `github_repo_id` are what downstream clone/launch machinery consumes.

**See also.** orchestrator — "Git connection management" (connection truth, OAuth state, installation-token minting, repo enumeration).
