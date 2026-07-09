---
type: capability
title: "Heartbeat & liveness census"
tags: [heartbeat, liveness, websocket, runtime-census, reconnect]
timestamp: 2026-07-09T11:13:06Z
description: "How the controller reports its liveness and connected-runtime census to the orchestrator, and how its upstream connection reconnects."
repo: controller
commit_sha: 4e237d3
evidence:
  - src/orchestrator/websocket_client.rs
  - src/protocol/orchestrator.rs
---

# Heartbeat & liveness census

**What it does.** On its upstream connection to the orchestrator, the controller emits a periodic `heartbeat` that both proves the controller is alive and reports which runtimes are currently connected to it — the orchestrator's source of truth for controller liveness and its runtime census.

**How a peer interacts.** A peer consumes the inbound `heartbeat` message (controller → orchestrator) on the controller's upstream socket. There is no request — it is unsolicited and periodic.

**Observable behavior.** A `heartbeat` is sent every 30 seconds. The first heartbeat fires ~30s after the connection is established (the immediate tick on connect is skipped — do not expect one at t=0). Each heartbeat carries: the controller instance UUID, the list of currently connected runtimes (each with its name and runtime-instance UUID), the runtime count, the controller version, and seconds of uptime on the current connection.

**Contract.** The payload is the `heartbeat` command carrying `HeartbeatData` (controller instance UUID, connected-runtime list, count, version, uptime seconds). Referenced by name — ask for the current shape rather than assuming fields.

**Invariants.** The 30s cadence is a hard contract: the orchestrator marks the controller stale after ~90s (3× the interval), so a gap beyond ~90s reads as a dead controller. `uptime_seconds` resets on every reconnect (it measures the current connection, not process lifetime). The instance UUID is generated per controller process and is stable across reconnects.

**Failure modes.** If heartbeat delivery fails, the controller tears down and reconnects rather than silently continuing. The upstream connection reconnects with exponential backoff (~1s doubling to a 60s cap). Across a reconnect, task responses already produced (including delayed responses from long-running spawn builds) are buffered and drained onto the new connection — a reconnect does not lose in-flight results.

**Gotchas.** A long inline operation on the upstream socket that starves the heartbeat tick for >90s will get the controller declared stale and dropped even though it is working; heavy work is expected to run off the socket's event loop. The runtime census reflects only runtimes currently connected to *this* controller instance — after a controller restart the census rebuilds as runtimes reconnect, so a transiently low count right after reconnect is normal, not a loss of runtimes.
