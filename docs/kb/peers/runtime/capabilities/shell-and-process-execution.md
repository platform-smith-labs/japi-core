---
type: capability
title: "Shell & process execution"
tags: [shell, exec, daemon, process-management, commands]
timestamp: 2026-07-06T23:40:38Z
description: "One-shot shell commands with captured output, plus daemon spawn/kill/list lifecycle inside a pod"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/core/router/handlers.rs
  - src/process/manager.rs
  - src/core/protocol/payload.rs
see_also:
  - {repo: runtime, capability: "controller-websocket-link", intent: "the transport these commands and replies travel over; readiness gating applies"}
  - {repo: runtime, capability: "pid1-init-supervision", intent: "the supervised image CMD lives in this same registry (id `main`) and dies with shutdown"}
---

# Shell & process execution

**What it does.** Executes arbitrary commands inside the pod on behalf of the platform: one-shot
shell commands returning captured output, and long-running daemon processes with a
spawn/kill/list lifecycle.

**How a peer interacts.** Four WS commands: `execute_command`, `spawn_process`, `kill_process`,
`list_processes`.

**Observable behavior.**
- `execute_command` runs `/bin/sh -c <command>` (optional working dir, extra env vars), waits for
  it to finish, then replies `command_response` carrying the full captured stdout, stderr, and
  exit code. One reply on completion — nothing is streamed while it runs.
- `spawn_process` starts a detached daemon in its own process group (setsid) and replies
  immediately: `process_started` `{id, pid}` on success, `process_failed` `{id, error}` otherwise.
  The daemon's stdin is closed; its stdout/stderr inherit the container's — output lands in pod
  logs and is NOT sent back over the wire.
- `kill_process` signals the entire process group of the daemon named by its caller-assigned `id`;
  success → `kill_process_response` `{success: true}`.
- `list_processes` → `list_processes_response` listing every tracked process — including already
  exited ones — with `id`, `pid`, `command`, `status`. Polling this list is the only way to observe
  a daemon's exit and exit code; there is no exit event.

**Contract.**
- `execute_command` in — key fields: `command` (required), `working_dir?`, `env` (key/value list).
  Out: `command_response` `{stdout, stderr, exit_code}`.
- `spawn_process` in — key fields: `id` (caller-assigned, used for all later correlation),
  `command`, `args`, `env`. Out: `process_started` | `process_failed`.
- `kill_process` in: `{id, signal?}` — `signal` is a numeric Unix signal, default 15 (SIGTERM);
  an unrecognized number silently falls back to SIGTERM. Out: `kill_process_response` or
  `error_response`.
- Errors on all four surface as `error_response` with a message string.

**Invariants.**
- Every daemon runs in its own process group, so `kill_process` reaches the whole child tree, not
  just the top PID.
- Daemon exit statuses are reaped and recorded by the runtime's PID 1 reaper; entries persist in
  the registry after exit.
- Spawned daemons never inherit `PLATFORM_SMITH_*` platform-internal env vars (stripped at spawn;
  see context.md).

**Failure modes.**
- Daemon binary missing/unspawnable → `process_failed` with the error; the pod is unaffected.
- `kill_process` on an unknown `id` → `error_response` "Process not found: <id>".
- Shell launch failure on `execute_command` → `error_response` instead of `command_response`.

**Gotchas.**
- Daemon output is NOT streamed back — a peer that spawns a process and waits for its output over
  the wire will wait forever. Capture output some other way (redirect to a file, or use
  `execute_command`).
- `execute_command` has no runtime-side timeout: a hung command means no reply ever. Callers own
  their own timeout.
- Unlike daemons, one-shot `execute_command` children inherit the runtime's full environment,
  including platform-internal vars.
- The supervised image CMD appears in this registry as id `main` — killing it is possible and
  kills the customer workload.
