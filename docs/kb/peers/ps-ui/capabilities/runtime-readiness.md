---
type: capability
title: "Runtime readiness polling"
tags: [readiness, polling, launch-preflight, runtime, ps-ui]
timestamp: 2026-07-07T06:27:35Z
description: "How the ps-ui advisory widget polls the readiness resolver and what terminal signal it waits for"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/readiness.ts
  - src/api/readiness.test.ts
  - src/components/readiness/poll.ts
  - src/components/readiness/poll.test.ts
  - src/components/readiness/ReadinessWidget.container.tsx
see_also:
  - {repo: orchestrator, capability: "Launch readiness resolver", intent: "hosts the readiness chain ps-ui polls; reuses spawn validators so a READY verdict can't drift from a launch 422", descriptive: true}
  - {repo: ps-api, capability: "Readiness passthrough proxy", intent: "passthrough-proxies GET /v1/readiness to the orchestrator", descriptive: true}
  - {repo: controller, capability: "Runtime controller bind", intent: "when the runner binds the environment the resolver's controller_connected flips and the chain advances toward READY", descriptive: true}
---

# Runtime readiness polling

**What it does.** Surfaces, to the user, whether a workspace (or project) has satisfied every
pre-flight step required before a coding runtime can be launched — coding-agent credential, git
connection, environment, and a connected runner. It is an **advisory** widget: it walks the user
through the unsatisfied steps and disappears once everything is ready. It never redirects or blocks.

**How a peer interacts.** ps-ui is a pure consumer here. It reads the readiness verdict from
`GET /v1/readiness` with optional query params `scope` (`workspace` | `project`), `workspace`, and
`project`. The backend owns the resolver; ps-ui only polls it and renders the result.

**Backend contract consumed.** Response is a discriminated descriptor keyed by `state`:
- `state` — the FIRST unsatisfied step, one of (dependency order):
  `NEEDS_CODING_AGENT_CREDENTIAL` → `NEEDS_GIT_CONNECTION` → `NEEDS_ENVIRONMENT` → `NEEDS_RUNNER` →
  `READY`. **`READY` is the terminal value** ps-ui waits for.
- `chain[]` — every step with a `satisfied` boolean (drives the stepper / "N left" count).
- `context` — pre-fills the active step; its shape is narrowed by `state`. key fields per state:
  - `NEEDS_RUNNER`: `{ workspace_uuid, environment_uuid, environment_name?, controller_connected }`.
    `controller_connected` flips true when the runner binds the environment — this is the
    READY-drift guard the poll watches on the runner step.
  - `NEEDS_GIT_CONNECTION`: `{ workspace_uuid, connections[] }` (workspace-usable git connections).
  - `NEEDS_ENVIRONMENT` / `NEEDS_CODING_AGENT_CREDENTIAL`: `{ workspace_uuid }`.
  - `READY`: `{ workspace_uuid, environment_uuid }`.

**Observable behavior.**
- **Cadence:** `GET /v1/readiness` is re-polled every **2500 ms** while the chain is being walked.
- **Stop conditions:** polling stops entirely once `state === 'READY'`. The widget also only polls
  when it is actually being looked at — while the inline section is **expanded**, OR when
  `state === 'NEEDS_RUNNER'` (waiting for the runner to connect) even while collapsed. Otherwise it
  refetches on mount/focus only, not on the interval.
- **What the UI shows per state:** a stepper of the four steps with satisfied/unsatisfied marks and a
  panel for the first unsatisfied step. On the runner step the panel auto-advances off the poll when
  `controller_connected` flips. On an in-session transition to `READY` it shows a terminal
  "You're all set" card; a user who is already `READY` on first load sees nothing at all.

**Failure modes.** A failed readiness fetch surfaces an error state with a Retry action (re-fetches);
it does not block the user. OAuth return params (`?connected=1`, `?error_code`, `?setup=1`) are
handled on mount: connected → refetch + success toast; error_code → error toast; setup → re-expand.

**Gotchas.**
- This readiness poll is a **launch pre-flight**, NOT a per-runtime-instance readiness poll. It keys
  on `workspace` / `project` / `environment_uuid` and reads `controller_connected` — it does **not**
  read a `runtime_instance.status` or `runtime.status` field. The platform rule that "a runtime is
  ready on `runtime_instance.status`, not the parent `runtime.status`" applies to runtime-instance
  readiness elsewhere; whether ps-ui polls a runtime-instance status directly is UNKNOWN for this
  capability (out of its scope).
- The id used to poll is the **workspace UUID** (and optionally a project UUID) via query params —
  there is no runtime/instance id in the request.
- A `READY` verdict is advisory and reuses the same validators as the launch path, so it should not
  diverge from a subsequent launch — but ps-ui treats it as guidance, not a guarantee.
