---
type: capability
title: "PID 1 init & supervision"
tags: [pid1, init, supervision, zombie-reaping, shutdown, lifecycle]
timestamp: 2026-07-09T10:42:29Z
description: "Runtime as container ENTRYPOINT: supervises the image's original CMD, reaps only its own children, and owns graceful shutdown"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/main.rs
  - src/signals.rs
  - src/process/manager.rs
  - src/websocket/client.rs
  - src/core/router/handlers.rs
  - docs/dev/decisions/reaper-only-reaps-tracked-pids.md
see_also:
  - {repo: runtime, capability: "Controller WebSocket link", descriptive: false, intent: "registration vs readiness sequencing — a pod is safe to command only after its readiness event"}
  - {repo: runtime, capability: "Shell & process execution", descriptive: false, intent: "the supervised CMD child shares the same managed-process registry as spawned daemons"}
  - {repo: runtime, capability: "In-pod image build (builder mode)", descriptive: false, intent: "what a builder-mode pod does instead of supervising a customer CMD"}
---

# PID 1 init & supervision

**What it does.** The runtime binary is the container ENTRYPOINT and runs as PID 1 in every
Platform Smith pod. It starts the image's original workload as a supervised child, reaps its own
zombie children, and turns termination signals into a graceful shutdown of everything it manages.

**How a peer interacts.** Never invoked directly. Docker appends the image's `CMD` array to the
runtime's argv at pod launch; everything else is driven by WS commands delivered over the
controller link (see the runtime capability "controller-websocket-link"). The one lifecycle command
peers send here is `shutdown`.

**Observable behavior.**
- Non-empty CMD → argv[1..] is spawned as a supervised child (registry id `main`) in its own
  process group; its stdout/stderr go to the container logs. Canonical brownfield path.
- Empty CMD (`CMD []`, greenfield/bootstrap) → no child is spawned; the runtime stays alive as
  PID 1 serving WS commands only.
- Builder mode (`PS_RUNTIME_MODE=builder`) → the CMD spawn is ALWAYS skipped, regardless of what
  the image declares; a builder pod is build-only.
- SIGTERM/SIGINT → SIGTERM to every managed process group, ~5s grace, SIGKILL any stragglers, then
  the process exits 0.
- SIGCHLD → reaps ONLY the PIDs it tracks, never `waitpid(-1)` (see Invariants).
- `shutdown` command → replies `shutdown_ack`, then the runtime process exits 0 — PID 1 terminates
  and the pod goes down.

**Contract.** Configuration is env-only: `PLATFORM_SMITH_WS_URL` and `PLATFORM_SMITH_RUNTIME_NAME`
are required; `PLATFORM_SMITH_INSTANCE_UUID` is preferred when injected, self-minted as a fallback
(legacy path); `PS_RUNTIME_MODE=builder` selects builder mode. `shutdown` in: none needed; out:
`shutdown_ack`.

**Invariants.**
- The reaper waits only on PIDs it spawned and tracks — never a blanket `waitpid(-1)`. The runtime
  also drives async-runtime-owned children (coding-agent sessions, in-pod builds) that have their
  own reaper; a blanket wait would steal their exit statuses and surface as spurious "No child
  process" failures (ADR: reaper-only-reaps-tracked-pids).
- The supervised command is sourced from argv only; the image's original entrypoint script is dead
  and never consulted.
- A builder pod never runs a customer CMD — the guard is enforced in the runtime itself, not just
  by image construction.

**Failure modes.**
- Missing/invalid required config, or failure to bind its local listeners at startup → the process
  exits 1 and the pod dies immediately.
- The CMD child failing to spawn is logged and swallowed — PID 1 stays alive in WS-only mode; a
  peer sees a healthy, commandable runtime with no `main` process running.

**Gotchas.**
- `shutdown` is terminal: the ack is followed by process exit, not a quiescent state. Do not expect
  further responses from that runtime.
- A crashed/exited CMD child does not kill the pod — check the process registry (runtime capability
  "shell-and-process-execution", `list_processes`) rather than pod liveness.
- Registration ≠ readiness: only command a runtime after its readiness event (owned by
  "controller-websocket-link").
