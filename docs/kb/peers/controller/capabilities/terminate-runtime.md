---
type: capability
title: "Terminate runtime"
tags: [runtime-lifecycle, teardown, docker, websocket, idempotent]
timestamp: 2026-07-07T00:00:00Z
description: "Stop and remove a named runtime container by request, with a correlated task response"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/orchestrator/executor.rs
  - src/docker/sandbox.rs
  - src/protocol/orchestrator.rs
---

# Terminate runtime

**What it does.** Tears down one named runtime container — stops it and removes it. This is the
teardown counterpart to launching a runtime (e.g. reaping a staging/bootstrap pod once it is no
longer needed, or an operator-requested teardown).

**How a peer interacts.** The orchestrator sends the `terminate_runtime` task over its WebSocket
connection to the controller, naming the target runtime. Unlike the launch family (which correlates
by instance UUID and returns no task response), `terminate_runtime` carries a real `task_id`, so the
controller **does** emit a correlated task response the caller can await.

**Observable behavior.** The controller unregisters the runtime from its registry first, then stops
the container (graceful SIGTERM with a ~10-second grace, then SIGKILL) and removes it. On success the
caller gets a success response with a human-readable note. Terminating a container that is already
gone is a no-op that still returns success.

**Contract.** In: target `runtime_name` plus an optional informational `reason` string (known values
include `bootstrap_complete`, `operator_requested`; extensible). Out: the standard task response —
`success` boolean plus a `response`/`error` message. `reason` does **not** affect behavior — the
controller terminates regardless of what (or whether) a reason is given.

**Invariants.**
- **Idempotent.** Terminating an absent/already-removed container returns `success: true` (no-op),
  not an error.
- **Unregister-before-remove.** The runtime is removed from the registry before the container is
  destroyed, so any in-flight relay directed at it resolves as a runtime-disconnected error rather
  than hanging on a soon-to-vanish target.
- **Scoped teardown.** Only containers under the controller's runtime-container naming prefix are
  targeted (see context.md) — sacred/infrastructure containers are never stopped.

**Failure modes.**
- **Already gone** → success no-op (see idempotency).
- **Docker stop/remove error** (daemon unreachable, unexpected Docker failure) → `success: false`
  with an error message naming the runtime; the container may be left in an indeterminate state.

**Gotchas.**
- The success response confirms the container was stopped+removed (or was already absent) — it does
  **not** distinguish a graceful shutdown from a SIGKILL, and it does not wait for the runtime's own
  cleanup logic beyond the ~10s stop grace.
- `reason` is telemetry only; do not expect it to gate or alter teardown.
