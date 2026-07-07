---
type: capability
title: "Workspaces & projects management"
tags: [workspaces, projects, tenancy, active-workspace, membership-guard]
timestamp: 2026-07-07T06:27:35Z
description: "How the ps-ui frontend lists/creates/selects workspaces and manages projects within them, and the backend endpoints + fields it consumes"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/workspaces.ts
  - src/api/projects.ts
  - src/stores/active-workspace-store.ts
  - src/stores/auth.ts
  - src/routes/_auth/workspaces/$workspaceUuid/route.tsx
  - src/routes/_auth/index.tsx
  - src/components/workspaces/WorkspaceList.container.tsx
  - src/components/projects/ProjectList.container.tsx
see_also:
  - {repo: ps-api, capability: "Workspaces & projects CRUD API", intent: "serves the /v1/workspaces and /v1/projects endpoints this UI consumes", descriptive: true}
  - {repo: ps-api, capability: "Workspace membership authorization", intent: "server-side 403 on foreign-workspace data that the UI guard mirrors", descriptive: true}
---

# Workspaces & projects management

**What it does.** Workspaces are the top-level tenant container a user belongs to; projects are
code/repo units that live inside one workspace. The UI lets a user list their workspaces, create /
rename / archive them, **select** one as the persisted "active workspace," and within a workspace
list / import / rename / archive projects. Everything below the selected workspace (dashboard,
projects, settings, environments, sandboxes) is scoped to that workspace's UUID in the route.

**Backend contracts consumed.** All under the ps-api gateway.

Workspaces:
- `GET /v1/workspaces` — list. Response `key fields:` `{ workspaces: [...], total }`; each workspace
  `key fields:` `workspace_uuid`, `name`, `description?`, `is_archived`, `settings?`, `created_at`,
  `updated_at`, `archived_at?`.
- `GET /v1/workspaces/{workspace_uuid}` — single workspace (same shape).
- `POST /v1/workspaces` — create. Body `key fields:` `{ name, description?, settings? }`; returns the
  created workspace.
- `PATCH /v1/workspaces/{workspace_uuid}` — update `{ name?, description?, settings? }`.
- `DELETE /v1/workspaces/{workspace_uuid}` — archive (soft). Returns `{ message, workspace_uuid }`.

Projects (scoped by workspace on list/create; by project UUID thereafter):
- `GET /v1/workspaces/{workspace_uuid}/projects` — list. Response `key fields:` `{ projects, total }`;
  each project `key fields:` `project_uuid`, `name`, `description`, `is_archived`, `default_workdir`,
  `base_image`, `devcontainer_ref`, `created_at`, `updated_at`.
- `GET /v1/projects/{project_uuid}` — single project; its response additionally carries an **inlined**
  `port_bindings?` array (may be absent on legacy responses → treated as empty).
- `POST /v1/workspaces/{workspace_uuid}/projects` — create `{ name, description? }`.
- `PATCH /v1/projects/{project_uuid}` — update `{ name?, description? }`.
- `DELETE /v1/projects/{project_uuid}` — archive; returns `{ message, project_uuid }`.

**Observable behavior.**
- **Active workspace** is client-side UI state persisted to browser storage (key `ps:active-workspace:v1`),
  storing only `{ uuid, name }` — it is **not** a server preference. Selecting a workspace sets this
  store and navigates to that workspace's dashboard.
- **Cache keys.** The workspace list is cached under the app-wide React Query key `['workspaces']`
  (shared by the list, switcher, landing, and the membership guard); the project list under
  `['projects', workspace_uuid]`. Create/rename/archive mutations invalidate their respective key so
  the list refetches. Archive is soft (workspace/project hidden from the list, reversible by an admin).
- **Landing resolution.** At `/`, the UI picks a destination: a stored-and-still-valid workspace →
  its dashboard; else the first workspace in the list; else the bare workspace list. A stored
  workspace no longer present in the list is treated as stale and cleared.

**Membership guard (UI layer over server authz).** The `/_auth/workspaces/{workspace_uuid}` route
subtree runs a `beforeLoad` guard: it reads the `['workspaces']` cache (via `ensureQueryData`, so it
usually costs no extra fetch) and, if `workspace_uuid` is not among the user's workspaces, redirects
to `/`. This is a convenience layer only — **the server 403s foreign-workspace data regardless**; the
guard just keeps a user off an all-403 broken page (e.g. a shared link, a stale bookmark, or a
stored UUID from a different account). If the list can't be fetched (transient error) the guard
**fails open** and lets the page load so its own queries surface the real error.

**Failure modes.**
- Foreign / non-member workspace UUID in the URL → UI redirect to `/` (guard), or a server **403** if
  the guard failed open or was bypassed.
- Archiving an already-gone project tolerates a **404** as success (treats it as already archived).
- On login/logout the entire query cache is cleared and the stored active workspace is dropped —
  otherwise a prior account's cached `['workspaces']` would defeat the membership guard and leak
  cross-account data.

**Gotchas.**
- The active workspace lives **only** in browser storage, not on the server; a different browser /
  cleared storage starts with no selection and falls back to landing resolution.
- The membership guard is **advisory**, not the security boundary — a backend change must not rely on
  the UI to prevent foreign-workspace access; server-side authz is the real gate.
- Workspace and project identifiers are UUIDs used only in routes/requests; the UI displays `name`,
  never the raw UUID, to users.

**Business-critical data.** UI-side only: the persisted active workspace `{ uuid, name }` in browser
storage. Workspace/project rows themselves are owned by ps-api (see_also). Tenant scoping applies as
everywhere — see context.md.
