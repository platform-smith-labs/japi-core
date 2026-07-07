---
type: capability
title: "Runtime lifecycle nodes"
tags: [workflow, runtime, conductor-nodes, lifecycle, orchestrator]
timestamp: 2026-07-07T06:49:45Z
description: "The runtime-start / runtime-status / runtime-stop worker nodes a workflow references to provision, read, and release a container runtime"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/nodes/runtime_start.go
  - internal/workers/nodes/runtime_stop.go
  - internal/workers/nodes/reads.go
  - internal/platform/platform.go
  - internal/platform/db_platform.go
see_also:
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "session-start consumes runtime-start's runtime handle (controller_name + runtime_name)"}
  - {repo: orchestrator, capability: "Runtime launch (tasks/spawn) and stop", intent: "provisions the runtime and mints the stable runtime_uuid; owns the graceful-stop route", descriptive: true}
---

# Runtime lifecycle nodes

**What it does.** Three custom Conductor worker nodes let a workflow provision, read, and release a
container runtime for a coding-agent session: `runtime-start` (provision + wait for ready),
`runtime-status` (read a runtime's state), `runtime-stop` (release, trailing cleanup).

**How a peer interacts.** A workflow author references a node by its Conductor **task type**
(`runtime-start` / `runtime-status` / `runtime-stop`) as a node in the workflow definition, and
supplies node inputs under `inputParameters._ps`. That reference IS the invocation — peers never call
these nodes directly.

**Observable behavior.**
- `runtime-start` is a **park-style** node: it kicks off an async launch, holds the Conductor task
  IN_PROGRESS, and completes only when the launch reaches READY. Readiness is polled DB-direct on the
  launched runtime's **newest runtime_instance status** — terminal signal is `status == "ready"`
  (COMPLETED) or `status == "failed"` (FAILED, with `failed_phase`); a wait past the configured
  deadline FAILs. On READY it completes with the runtime identity.
- `runtime-status` is **synchronous**: returns the runtime's current state immediately.
- `runtime-stop` is **synchronous** and idempotent: an already-released / unknown / cross-tenant
  runtime completes as a no-op success. **Gated** — see Failure modes.

**Contract.**
- `runtime-start` in: `_ps.project_uuid` (req), `_ps.workspace_uuid` (req), `_ps.environment_uuid`
  (opt — workspace default), `_ps.agent_definition_uuid` (opt). Out on READY — key fields:
  `runtime_uuid`, `runtime_name`, `controller_name`, `state="ready"`. `runtime_uuid` is the
  **tenant-stable identity of the parent runtime** (minted synchronously by the orchestrator launch
  path); `runtime_name` + `controller_name` are what a session node consumes.
- `runtime-status` in: `_ps.runtime_uuid` (req). Out — key fields: `found`, `status`
  (`active` | `inactive`; active ⇔ ready AND connected), `launch_status` (`requested` … `ready` |
  `failed`), `failed_phase`. **`_ps.runtime_uuid` here is keyed against the runtime *instance* id,
  not the parent runtime id** (see Gotchas).
- `runtime-stop` in: `_ps.runtime_uuid` (req — the parent-runtime identity `runtime-start` emits; the
  node resolves it to the newest runtime_instance internally). Out: `runtime_uuid`, `stopped=true`.

**Invariants.** All three are tenant-scoped, but the unknown/cross-tenant behavior differs by node:
`runtime-status` returns a clean not-found RESULT (`found=false`, never a leak or FAIL) for a
downstream SWITCH; `runtime-stop` treats an unknown/cross-tenant target as an idempotent success
(`stopped=true`); `runtime-start` creates a runtime and has no not-found path. Mutating nodes
(`runtime-start` launch, `runtime-stop` release) call the orchestrator with the originating
**user_uuid + company_uuid** replayed as gateway headers. Missing `company_uuid` FAILs the task
locally; a missing `user_uuid` is **not** checked here — it is forwarded and rejected upstream by the
orchestrator (which re-validates the user against the company). `runtime-stop` release is idempotent
(safe to retry, safe as always-cleanup). `runtime-start` launch
dedupes on `(workflow_id, task_ref)` so a Conductor redelivery re-parks rather than spawning a second
runtime; the underlying orchestrator launch is also stable (same `runtime_uuid` for a given
project/controller), so a residual re-launch is a relaunch of the same runtime, not a duplicate.

**Failure modes.**
- `runtime-start`: launch phase failure → FAILED with `failed_phase`; not READY before deadline →
  FAILED. A missing required `_ps` field → FAILED.
- `runtime-status`: unknown/cross-tenant → COMPLETED with `found=false`; missing `_ps.runtime_uuid`
  → FAILED.
- `runtime-stop`: **NOT_LIVE** whenever `RUNTIME_STOP_LIVE` is false — the orchestrator stop route
  ships per-stack and until it is deployed the node reports an honest terminal NOT_LIVE
  (Conductor FAILED + not_live marker), **never a false success**. Peers on a stack without the route
  deployed will see this. A live release that errors (non-404) → FAILED.

**Gotchas.**
- **`runtime_uuid` names one thing, keys another across these nodes.** `runtime-start` emits a
  `runtime_uuid` that identifies the **parent `runtime`** (stable across relaunches).
  `runtime-stop` consumes that same parent identity. But **`runtime-status` keys its
  `_ps.runtime_uuid` against the `runtime_instance` id** — so feeding `runtime-start`'s output
  `runtime_uuid` straight into `runtime-status` returns `found=false`. Resolve the parent runtime to
  its current instance id before calling `runtime-status`.
- **Readiness lives on the instance, not the parent.** A runtime is "ready" on its newest
  `runtime_instance` status; the parent `runtime` status lags. `runtime-start` already reads the
  instance for its readiness signal — peers reasoning about readiness must do the same.
- Git/PR/commit/test are NOT lifecycle nodes; those happen inside the coding-agent session, not here.

**Business-critical data.** Readiness and status are read DB-direct from `runtime` and
`runtime_instance` (newest instance wins) in the shared `platform_smith` DB; `runtime_instance.status`
(+ `failed_phase`) is the launch state machine's projection and the terminal readiness signal.
(Tenant scoping applies as everywhere — see context.md.)

**See also / peers.** After READY, `runtime_name` + `controller_name` are consumed by ps-workflow's
**Coding-agent session nodes** (session-start) to open a coding-agent session on the runtime. The
orchestrator owns the actual launch (`tasks/spawn`) and stop routes behind the platform seam.
