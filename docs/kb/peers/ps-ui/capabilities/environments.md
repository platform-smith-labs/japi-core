---
type: capability
title: "Environments"
tags: [environments, controller, topology, workspace]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui lists, creates, edits and archives workspace environments and reads controller health"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/environments.ts
  - src/routes/_auth/workspaces/$workspaceUuid/environments/index.tsx
  - src/components/readiness/panels/connect-commands.ts
see_also:
  - {repo: ps-ui, capability: "Runtime readiness polling", intent: "an environment must have a live controller before a runtime can launch"}
  - {repo: orchestrator, capability: "Controller registration", intent: "owns the controller identity an environment binds to", descriptive: true}
---

# Environments

**What it does.** An environment is a workspace-scoped execution target with a `topology`
(observed value: `docker_pods`) that is bound 1:1 to a controller. One environment per workspace is
the default and carries the controller that runtimes launch into. ps-ui lists, creates, edits, and
archives environments and shows each controller's online/offline health.

**Backend contracts consumed** (ps-api :9004, all under `/v1/workspaces/{workspaceUuid}`):
- `GET /environments` → `{ environments[], total }`.
- `POST /environments` → an environment. Body key fields: `name`, `controller_uuid` (nullable),
  `is_default`, `topology` (sent as `docker_pods`).
- `PATCH /environments/{environmentUuid}` → the updated environment. Body key fields (all optional):
  `name`, `controller_uuid`, `is_default`.
- `DELETE /environments/{environmentUuid}` — archive; accepts `?force=true` to force.

Environment read key fields: `environment_uuid`, `name`, `topology`, `controller_uuid` (nullable —
the stable logical controller), `last_seen_at` (nullable heartbeat), `torn_down_at` (nullable),
`is_default`, `created_at`, `updated_at`. There is **no** `project_uuid` on the environment read
(environments are 1:1 with controllers, not linked to a project).

**Observable behavior.** Controller health is **derived client-side**, not returned by the backend:
a controller is `online` when `last_seen_at` is within the last 60 seconds, otherwise `offline`
(including when `last_seen_at` is null). The 60s window is a ps-ui constant, not a backend contract.

**Failure modes.** UNKNOWN — the error shapes for create/update/archive (e.g. duplicate default,
archiving an in-use environment, the `force` semantics) are not asserted client-side and are owned by
ps-api/orchestrator.

**Gotchas.**
- `topology` is sent as the literal `docker_pods` on create; no other value is emitted by ps-ui.
- `controller_uuid` is the stable logical controller id, distinct from any rotating process/instance
  id; it may be null for an environment with no controller yet bound.
- A minted workspace token + this environment's `environment_uuid` are what a self-hosted controller
  uses to bind (see Workspace tokens) — the UI surfaces `environment_uuid` into the controller's
  `ENVIRONMENT_UUID`.

**See also.** ps-ui — Runtime readiness (a live controller on the environment gates runtime launch);
Workspace tokens (the identity a controller binds with); orchestrator — Controller registration
(owner of the controller side of the binding).
