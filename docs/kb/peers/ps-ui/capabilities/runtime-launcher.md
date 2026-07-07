---
type: capability
title: "Runtime / sandbox launcher"
tags: [launcher, sandbox, runtime, launch, git-branch, readiness]
timestamp: 2026-07-07T06:27:35Z
description: "How the ps-ui frontend launches a sandbox runtime (repo/branch/agent) and which launch-response identifiers it keys the runtime by afterward"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/launch.ts
  - src/api/launches.ts
  - src/api/runtimes.ts
  - src/lib/launch-events.ts
  - src/components/launcher/Launcher.container.tsx
  - src/components/launcher/launch-context.ts
  - src/api/readiness.ts
  - src/components/sandboxes/use-runtime.ts
see_also:
  - {repo: orchestrator, capability: "Sandbox launch orchestration", intent: "executes POST /v1/launch — recipe resolution, branch/clone resolution, runtime spawn; owns the launch lifecycle status and readiness verdict", descriptive: true}
  - {repo: controller, capability: "Runtime container lifecycle", intent: "owns the sandbox container the launch spawns; readiness/liveness ultimately reflects controller state", descriptive: true}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "proxies /v1/launch, /v1/launches, /v1/sessions, /v1/readiness, /v1/runtimes verbatim to the orchestrator", descriptive: true}
---

# Runtime / sandbox launcher

**What it does.** The launcher is the ps-ui surface a user drives to start a new
sandbox runtime — pick a project (repo), optionally an environment, a coding
agent, and a git branch, type an opening prompt, and launch. It also drives the
alternate "attach a new session to an already-running sandbox" path (no new
container).

**Backend contracts consumed (what ps-ui calls).**

- **New sandbox launch** — `POST /v1/launch`. Request key fields ps-ui sends:
  `kind: "sandbox"`, `project_uuid` (REQUIRED — a project-less sandbox is
  rejected 422), `environment_uuid?` (omitted → server defaults to the project's
  workspace-default environment), `agent_definition_uuid?`, `model?` (omitted →
  agent default), `initial_prompt` (always sent so an eager session is seeded), and
  optional git-branch controls `branch?` / `base_branch?` (both OMITTED when blank,
  never `""`). ps-ui never sends `permission_mode` — the server 422s on it.
- **Attach a session to a running sandbox** — `POST /v1/sessions`, keyed by
  `runtime_name` (NOT a uuid), with `session_id`, `initial_prompt`,
  `agent_definition_uuid?`. `controller_name` is omitted (server resolves it).
- **Pre-flight readiness gate** — `GET /v1/readiness?scope=&workspace=&project=`.
  Advisory: returns the first-unsatisfied step in the chain
  `NEEDS_CODING_AGENT_CREDENTIAL → NEEDS_GIT_CONNECTION → NEEDS_ENVIRONMENT →
  NEEDS_RUNNER → READY`. ps-ui blocks the launch button until `READY` (skippable).
  Server reuses the spawn validators so a READY verdict can't drift from a launch 422.
- **Launch reads** (async progress) — `GET /v1/launches/{instance_uuid}` (detail),
  `GET /v1/launches` (list), `GET /v1/launches/{instance_uuid}/attempts`, and the
  timeline SSE stream `GET /v1/launches/{instance_uuid}/events/stream`.
- **Runtime roster** — `GET /v1/runtimes`. NOTE: this read is COMPANY-scoped (no
  workspace filter); ps-ui fetches all rows and filters client-side by
  `workspace_uuid`. Rows carry `runtime_uuid` + `runtime_name`, a derived
  liveness `status` (`active`/`inactive`), and a separate `launch_status`.

**Launch response — the identifiers ps-ui reads.** `POST /v1/launch` returns
(key fields): `instance_uuid`, `runtime_uuid`, `session_uuid`, `session_name`,
`status`. `session_uuid`/`session_name` are present only because
`initial_prompt` was supplied.

**CRITICAL identity seam — one launched runtime, three keys.** The launch does
NOT hand back a single addressable id; the follow-up read decides which form is
required, and they are NOT interchangeable:

- `instance_uuid` (the `runtime_instance_uuid`, "stable wire key") → keys the
  launch-detail read `GET /v1/launches/{instance_uuid}` and the launch-timeline
  SSE. Returned in the form the read needs — no bridge.
- `runtime_uuid` (tenant-unique launch identity) → keys the Sandboxes roster rows
  (`GET /v1/runtimes`) and the sandbox route param
  (`/workspaces/{ws}/sandboxes/{runtime_uuid}/...`). Also returned directly, but
  ps-ui often re-derives it (see bridge below).
- `runtime_name` → the key to ATTACH further sessions (`POST /v1/sessions`
  takes the NAME, not a uuid). Not returned by the runtimes list as a lookup
  input — it is a row field.
- `session_name` → the session route param + transcript SSE
  (`GET /v1/sessions/{session_name}/events/stream`). Opaque; embeds a uuid — never
  parsed or rendered.

**The bridge ps-ui actually performs.** For the attach path, the scope carries
only the runtime NAME. To navigate to that sandbox's sessions list ps-ui needs a
`runtime_uuid`, so it BRIDGES name → uuid by fetching `GET /v1/runtimes` and
matching `runtime_name === scope.runtimeName` → `runtime_uuid`. So a peer that
hands ps-ui a runtime by name (and expects it addressable in the UI) is relying on
that runtime being present in the company-scoped `/v1/runtimes` roster.

**Observable behavior.**

- Launch is ASYNC. `status` on the 201 is the launch HEAD in a 10-value lifecycle
  (`requested → resolving_recipe → builder_starting → cloning → authoring →
  building → starting_runtime → setting_up → ready | failed`). `ready` is the sole
  success terminal, `failed` the sole failure terminal. A peer/UI observes
  progress by polling `GET /v1/launches/{instance_uuid}` (`status`, plus `ready`
  and `connected` booleans, and `failed_phase` when failed) or by consuming the
  timeline SSE `phase_changed` frames until `ready`/`failed`.
- **Created-from-base surface.** When `branch`/`base_branch` are supplied and the
  target branch didn't exist, the orchestrator reports the outcome as fields on
  the `product_clone_complete` timeline event: `resolved_branch` (branch actually
  checked out), `created_from_base` (bool), `created_from` (source branch, present
  only when created_from_base is true). ps-ui renders a "created `<branch>` from
  `<base>`" note ONLY for the create-from-base case; a plain checkout or any
  missing field renders nothing. Branch precedence (launch > conversation >
  project default) is resolved server-side by the orchestrator.
- On launch success ps-ui optimistically shows the prompt, opens the new
  `session_name` as the active tab, and navigates to the sessions list scoped to
  where the launch was triggered (project / env / sandbox / global).

**Failure modes.** Missing `project_uuid` → 422. Sending `permission_mode` → 422.
Attaching to a torn-down sandbox → the orchestrator returns a "not connected / no
connected runtime" error, surfaced to the user as "this sandbox isn't running
anymore." Readiness never hard-blocks — it is advisory and skippable.

**Gotchas.**

- **Identity handoff is the trap** — do not assume the launch's `runtime_uuid`,
  `instance_uuid`, and `runtime_name` are one id. Launch reads want
  `instance_uuid`; the sandbox roster/route wants `runtime_uuid`; attaching a
  session wants `runtime_name`.
- The launch `status` (launch HEAD) is NOT the same as a runtime's derived
  liveness `status` on `/v1/runtimes` — never conflate them.
- `/v1/runtimes` is company-scoped and returns inactive rows; the workspace filter
  is client-side. `runtime_kind` (sandbox vs service) and environment name are NOT
  on that roster shape today.
- Timeline event `data` is contractually free-form — a peer must not depend on its
  inner keys except the three documented `product_clone_complete` fields above.

**See also / peers.** The launch is executed by the orchestrator (recipe/clone/branch
resolution, runtime spawn); the runtime container lifecycle is owned by the controller.
