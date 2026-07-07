---
type: capability
title: "Port-conflict eviction on spawn"
tags: [spawn, port-conflict, eviction, runtime-lifecycle, docker]
timestamp: 2026-07-07T00:00:00Z
description: "When a runtime spawn fails on a pinned host port, the controller auto-evicts an evictable peer runtime holding that port and retries once"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/orchestrator/executor.rs
  - src/protocol/spawn_error.rs
---

# Port-conflict eviction on spawn

**What it does.** When a runtime spawn fails because an explicitly-pinned host
port is already bound, the controller tries to self-heal: it identifies what is
holding the port and, if the holder is an evictable peer runtime, stops that
peer, waits for the port to free, and retries the spawn once. This lets a caller
re-pin a port that a stale peer is still occupying without manual cleanup.

**How a peer interacts.** There is **no direct trigger** — this is a side effect
of the `spawn_runtime` task. A peer never asks for eviction; it only pins a host
port on a spawn and observes the outcome. What eviction adds is the possibility
that a first-attempt `PortInUse` failure turns into a success, plus richer
diagnostics when it cannot.

**Observable behavior.**
- Eviction is attempted **only** when the failing port is an explicitly-pinned
  host port AND the port named in Docker's bind error matches that pinned port.
  Auto-assigned/auto-resolved ports never trigger eviction.
- If the port holder is an evictable peer runtime, the controller stops+removes
  it, waits up to **5 seconds** for the port to free, then retries the spawn
  **exactly once**. Success looks like a normal spawn success to the caller.
- If eviction is not possible or the retry fails, the original spawn failure is
  returned with a structured, operator-facing diagnostic (below).

**Contract.** No new input. On failure the controller constructs a structured
`spawn_error` — a `PortInUse` variant carrying: `host_port`, a raw Docker/daemon
message, an optional `holder` describing what holds the port, and an optional
operator `hint`. `holder` is one of: `container` (a third-party or peer-runtime
container), `platform_smith_infra` (a Platform Smith infrastructure container —
never evictable), or `unknown` (e.g. a host process). Reference the
`SpawnErrorData` / `Holder` contracts by name; do not assume field order.

**Delivery.** A terminal `PortInUse` from a launch-family spawn is delivered to
the orchestrator as a controller-origin **`launch_failed`** event
(work-2607070349): `data.error_message` is composed as
`"port_in_use: {raw message} — {hint}"`, so the failure class and the
holder/eviction hint arrive in string form within seconds. (The structured
`SpawnErrorData` object itself rides `TaskResponseData` only for task_id-carrying
commands — a launch-family spawn's `task_response` remains suppressed.) The
full structured detail, including the `holder`, also remains in the controller's
audit logs (below). A peer's launch timeout is now a backstop, not the primary
failure signal.

**Invariants.**
- Eviction targets **only** peer runtime containers (names matching the
  configured runtime container prefix — see context.md's naming contract).
  Infrastructure containers, including the controller itself, are always refused.
- At most **one** retry per spawn. No eviction loops.
- Before evicting, the controller re-verifies the port match against Docker's
  actual bind-error string, so an auto-assigned-port failure never causes it to
  evict the wrong holder.

**Failure modes (what the peer/operator observes).**
- **Holder is a host process** (not a Docker container): the controller sees no
  container holding the port, reports `holder: unknown`, and refuses to evict —
  the hint directs the operator to free the port externally. Host processes are
  unidentifiable by design (namespace isolation).
- **Holder is Platform Smith infra**: refused; hint says to choose a different
  `host_port`.
- **Port stuck in TIME_WAIT**: if the OS has not released the port within the
  5-second window (typical after stopping the peer), the controller does **not**
  loop — it refuses with a "retry in ~30 seconds" hint.
- **Race after eviction**: if another container grabs the freed port before the
  retry, the retry fails and the hint names the new holder to stop.

**Gotchas.**
- Do not rely on eviction as a guaranteed outcome — it is best-effort and
  bounded to a single retry. A pinned-port spawn can still return `PortInUse`.
- Eviction is silent to the runtime being evicted from the caller's side, but it
  is a real teardown: a peer runtime can be stopped+removed out from under its
  own session to free a port for a newer pinned spawn.
- Correlate the two audit log lines `port_eviction_attempted` (emitted before the
  stop) and `port_eviction_completed` (emitted only after the port is verified
  free) to distinguish "we evicted X" from "we tried but the stop failed."
