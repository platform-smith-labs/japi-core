---
type: capability
title: "Git connections & repository import"
tags: [git, github, connections, repo-import, oauth, frontend-consumer]
timestamp: 2026-07-07T06:27:35Z
description: "How the UI connects a GitHub account, scopes it to workspaces, browses repos, and imports them into projects — the backend endpoints it consumes and how git errors surface."
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/git-connections.ts
  - src/api/workspace-repos.ts
  - src/api/projects.ts
  - src/lib/git-error-messages.ts
  - src/components/company-settings/CompanyGitConnections.container.tsx
  - src/components/projects/ImportReposPanel.container.tsx
  - src/components/projects/import-repos.ts
  - src/components/readiness/panels/ConnectGitHubPanel.container.tsx
see_also:
  - {repo: orchestrator, capability: "Git connections & GitHub OAuth", intent: "owns provider registry, OAuth install/callback, connection + assignment persistence, repo listing", descriptive: true}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "forwards /v1/git/* and project endpoints byte-for-byte to orchestrator", descriptive: true}
  - {repo: orchestrator, capability: "Runtime launch / import-and-launch", intent: "owns the import-and-launch endpoint ps-ui exposes as an unused client method", descriptive: true}
---

# Git connections & repository import

**What it does.** Lets a user connect a GitHub account (company-scoped git *connection*), grant/revoke
per-workspace visibility, browse that account's repos, and import one or many repos as Platform Smith
projects with a git link. ps-ui is a pure consumer — all state lives in the backend (ps-api → orchestrator).

**Backend contracts consumed** (all via ps-api at `/v1`; `provider_code` is always `github` today):

- **`GET /v1/git/providers`** → `{ providers[] }`; key fields: `git_provider_uuid`, `type`
  (`github|gitlab|bitbucket|generic`), `display_name`, `host`, `enabled`.
- **`POST /v1/git/connections/install`** — begin OAuth. In: `{ provider_code, redirect_uri }`. Out:
  `{ install_url, state }`. The UI stores `state` in `sessionStorage` and does a full-page redirect to
  `install_url`; the backend later redirects back to `redirect_uri` (see OAuth callback below).
- **`GET /v1/git/connections`** → `{ connections[] }`; key fields per item: `git_connection_uuid`,
  `provider_code`, `display_name`, `github_account_login`, `installation_count`, `status`
  (`active|revoked|error`), `visible_to_all_workspaces`, `created_at`, `last_used_at`.
- **`GET /v1/git/connections/eligible`** → `{ connections[] }` (same item shape) — connections a user
  may assign to a workspace.
- **`PATCH /v1/git/connections/{uuid}`** — In: `{ visible_to_all_workspaces? }`. Out: the updated item.
- **`DELETE /v1/git/connections/{uuid}`** — revoke.
- **`GET|PUT /v1/git/connections/{uuid}/workspaces`** — read/replace the connection's workspace
  assignment set. PUT In: `{ workspace_uuids[] }` (a full replace — clobbers other workspaces).
- **`GET|POST|DELETE /v1/workspaces/{wsUuid}/git/connections[/{connUuid}]`** — per-workspace view +
  idempotent single-pair assign/unassign. Assign POST In: `{ connection_uuids[] }`. **Gotcha:** the
  field is `connection_uuids`, NOT `git_connection_uuids` — the orchestrator validator requires
  `connection_uuids`; ps-api forwards the body byte-for-byte, so a wrong key silently no-ops the assign.
- **`GET /v1/git/connections/{connUuid}/repos`** — browse repos. Query key fields: `page`, `per_page`,
  `q`, `sort` (`pushed_at|name`), `order`. Out `{ items[], page, per_page, total, unfiltered_total,
  has_next, sort_applied, order_applied, total_is_estimate }`; item key fields: `github_repo_id`,
  `owner`, `name`, `full_name`, `default_branch`, `language`, `pushed_at`, `is_private`,
  `already_a_project`, `existing_project` (`{uuid,name,is_archived}|null`).
- **`POST /v1/workspaces/{wsUuid}/projects`** — create a project. In (used here): `{ name }`. Out key
  field: `project_uuid`.
- **`PUT /v1/projects/{projectUuid}/git-link`** — attach the repo. In: `{ git_connection_uuid,
  repo_url, default_branch, github_repo_id, auto_clone_on_launch: true, commit_identity_ref:
  'user_email' }` (`repo_url` is built as `https://github.com/{full_name}`). Out: the git link.
- **`POST /v1/workspaces/{wsUuid}/projects/import-and-launch`** — a one-shot import-and-launch client
  method **defined but NOT currently invoked by any ps-ui view** (available consumption surface, not a
  live-exercised contract; the live import path is the sequential create-project → git-link flow above,
  and launching is the separate launcher capability). Shape, for the record: an exclusive union of a
  **repo source** (`git_connection_uuid`, `repo_url`, `default_branch`, `github_repo_id`,
  `auto_clone_on_launch:true`, `commit_identity_ref:'user_email'`) OR a **sandbox source**
  (`base_image`), plus `{ name, description?, environment_uuid, agent_definition_uuid? }`; backend
  rejects both/neither with 400. Out: `{ project, project_git_link|null, task }`.

**Observable behavior.** Connecting is a browser round-trip, not an API return: install → redirect to
GitHub → backend redirects back to `redirect_uri`. Success comes back as `?connected=1` (the UI then
refetches connections and opens the workspace-assignment dialog for the newest one). Failure comes back
as `?error_code=<code>` (+ optional `?conflicting_account=<login>`) on the query string — there is no
error body to parse. Bulk import runs **sequentially, one repo at a time**, each as create-project →
upsert-git-link; a per-repo failure never aborts the rest (partial success is normal, with per-row retry).

**Failure modes.** Git/OAuth errors reach the UI two ways, both mapped by a single formatter keyed on an
`error_code` string:
- OAuth callback query param `error_code`.
- Repo-browse (`GET .../repos`) errors whose message is shaped `error_code=<code>: …` — the UI extracts
  the leading token; `github_installation_revoked` (or a message containing "revoked"/"Reauthorise")
  switches the browse error UI into a "Reauthorise" call-to-action.

Recognized `error_code` values and their meaning (peers emitting these should keep the codes stable):
`github_account_mismatch` (workspace already bound to a different account — carries `conflicting_account`),
`github_installation_revoked`, `github_rate_limited`, `github_upstream_failure`; any other code falls
through to a generic "install failed (error_code: …)" message. Import-stage HTTP errors: **409** on
create = project-name collision (non-retryable — rename/archive needed); **5xx** = retryable server
error; other 4xx on the link stage are retryable.

**Gotchas.**
- The assign body key is `connection_uuids`, not `git_connection_uuids` (see above) — a real past bug.
- `PUT .../workspaces` is a full-set replace; per-workspace toggles use the single-pair
  POST/DELETE on `/workspaces/{ws}/git/connections` to avoid clobbering other workspaces.
- Git errors are surfaced via `error_code` on a **redirect query string** and via an
  `error_code=<code>:` **prefix inside the browse error message** — not as a structured JSON error body.

**See also / peers.**
- **orchestrator — Git connections & GitHub OAuth**: owns provider registry, the OAuth install/callback,
  connection persistence, workspace assignment validation, and repo listing.
- **ps-api — Gateway request proxy**: forwards these `/v1/git/*` and project endpoints byte-for-byte to
  the orchestrator (the `connection_uuids` passthrough note depends on this no-transform behavior).
- **orchestrator — Runtime launch / import-and-launch**: owns the `import-and-launch` endpoint that
  ps-ui exposes as a client method but does not currently call.
