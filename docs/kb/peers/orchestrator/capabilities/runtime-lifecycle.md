---
type: capability
title: "Runtime container lifecycle"
tags: [runtime, lifecycle, spawn, launch, stop, rest-api, multi-tenant]
timestamp: 2026-07-07T00:00:00Z
description: "How a peer spawns, launches, lists, and stops runtime containers via the orchestrator REST API"
repo: orchestrator
commit_sha: 6843154
evidence:
  - REST_API.md
  - cmd/handlers/launch.go
  - cmd/handlers/tasks.go
  - cmd/handlers/runtimes.go
  - cmd/handlers/middleware.go
  - cmd/models/runtime_instance.go
---

# Runtime container lifecycle

**What it does.** Lets a caller bring a runtime container up (spawn/launch), stop it, and read its
state. A "runtime" is one containerized workspace pod that a controller runs; the orchestrator is the
central authority that requests launches and teardowns and tracks each runtime's launch timeline.

**How a peer interacts.** Four company-scoped REST routes on the orchestrator (internal, behind the
ps-api gateway):
- `POST /api/v1/launch` — the consolidated launch endpoint. Body carries `kind` (`service` |
  `sandbox`), `project_uuid` (required — targets an existing project), and optional
  `environment_uuid`, `agent_definition_uuid`, `branch`/`base_branch`, and `initial_prompt`/`model`
  (seeds an eager coding-agent session). `service` builds via the recipe path; `sandbox` spawns from
  the prebuilt sandbox image.
- `POST /api/v1/tasks/spawn` — the older spawn surface. Two modes: project-aware (`project_uuid` +
  `workspace_uuid` → resolves environment/controller and generates the runtime name) and
  workspace-agent (`workspace_uuid` + `workspace_agent=true`). The project-less base-image mode is
  retired.
- `POST /api/v1/runtimes/{id}/stop` — explicit teardown; `{id}` is the runtime_instance_uuid.
- `GET /api/v1/runtimes` and `GET /api/v1/runtimes/{id}` — list all runtime instances for the company,
  or fetch one by runtime_instance_uuid.

**Observable behavior.** Launch/spawn are accepted asynchronously: the response confirms the launch
was *requested*, not that the container is running. `/api/v1/launch` returns `runtime_uuid` (the
tenant-stable launch identity), `instance_uuid` (keys the launch-timeline event stream),
`status` = `requested`, and — when an `initial_prompt` was supplied — `session_uuid` + `session_name`
for the eagerly-created conversation. `/api/v1/tasks/spawn` returns a task carrying the runtime name,
controller name, and (project-aware path) `runtime_uuid`, with a pending/requested status. The caller
polls the runtime **instance** record for readiness — the container is usable only when the instance's
`ready` flag is true (equivalently its launch `status` reaches `ready`), not when the request returns.
Stop is release-first: it marks the instance released, then dispatches teardown to the owning
controller.

**Contract.** Read paths (`GET /api/v1/runtimes[/id]`, task reads) are **deprecated on the
orchestrator** — ps-api now serves these DB-direct; a peer should read runtime/task state from ps-api,
not here. Stop returns a runtime-action response (a status message + `runtime_uuid`, plus `task_uuid`
when a teardown was dispatched); the consumer depends only on the HTTP status, not the body. Identity:
`runtime_uuid` is the stable per-launch identity; `runtime_instance_uuid` (the `{id}` in the runtime
routes) is the specific instance and the correct key for stop and instance reads.

**Getting the names needed to run a task (identity bridge).** `POST /api/v1/launch` returns only
UUIDs (`runtime_uuid`, `instance_uuid`), but the task-execution submits (`/api/v1/tasks/command|message|claude`)
require the `runtime_name` + `controller_name` strings. Bridge them by reading the runtime **instance**
record — it carries `runtime_name`, `controller_name`, `ready`, and `status` in one shot, so it
answers both "what do I call it" and "is it ready". That read is served DB-direct by **ps-api** (the
orchestrator's own instance-read route is deprecated — see below). Alternatively,
`POST /api/v1/tasks/spawn` returns the runtime name + controller name inline in its Task response,
avoiding the extra read (a reason to prefer it for a spawn-then-run flow, despite `/launch` being the
newer surface).

**Invariants.** Every route is company-scoped via trusted gateway identity headers (`X-Company-UUID`,
`X-User-UUID`) that ps-api injects — the orchestrator does no login and refuses requests without valid
headers. All reads/writes are isolated to the caller's company. Only the `docker_pods` environment
topology is supported this release. Stop is idempotent: an already-released instance returns 200 with
no re-dispatch, so it is safe to retry and to run as trailing cleanup. Launch/spawn dispatch is
durable — if the owning controller is disconnected the teardown/task stays pending and is re-delivered
on reconnect.

**Failure modes.** Unknown or cross-tenant `{id}` on stop → 404 for both cases (no existence leak).
Missing/invalid gateway headers → rejected (unauthenticated). Non-`docker_pods` topology, or an
environment with no controller registered → 422. Launch with no `project_uuid` → 400; a project-less
sandbox (`workspace_uuid` only) → 422 (not supported yet). `permission_mode` on launch → 422 (not
wired yet). Legacy project-less base-image spawn → 400 directing the caller to the import-and-launch
project route. Not-found handling is inconsistent across the two surfaces: on `/launch` an unknown
environment → 404 but an unknown project → 400; on `/tasks/spawn` an unknown project → 404. A peer
should treat either a 400 or 404 as "target not found" and read the error message rather than key on
the status code.

**Gotchas.** A 2xx from launch/spawn is "accepted," not "running" — do not treat it as a readiness
signal; watch the launch timeline via `instance_uuid`. Two overlapping spawn surfaces exist: prefer
`POST /api/v1/launch` for new integrations; `/api/v1/tasks/spawn` is the older path. Do not call the
orchestrator's runtime/task **read** endpoints from new code — they are deprecated in favor of
ps-api's DB-direct reads. Stop keys on `runtime_instance_uuid`, not `runtime_uuid`. Teardown succeeds
(200) even when the owning controller is down, because release is recorded before dispatch.

**Business-critical data.** A runtime's launch identity is `runtime_uuid`; each concrete instance is a
`runtime_instance` row whose readiness is read from the **instance** status (the parent runtime status
lags). Stop records `runtime_instance.released_at` as the idempotent release marker. Teardown and
spawn are dispatched to the owning controller by runtime name. (Company/tenant scoping applies on
every query — see context.md.)
