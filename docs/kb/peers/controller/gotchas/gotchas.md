---
type: gotcha
title: "Controller cross-cutting integrator traps"
tags: [controller, thin-bridge, launch-family, container-prefix, port-eviction, readiness, credentials]
timestamp: 2026-07-07T00:00:00Z
description: "Traps a peer repo hits when its task or message transits the controller — payload semantics, launch-family correlation, container-prefix isolation, port visibility, and readiness-vs-credential."
repo: controller
commit_sha: 3412b7d
evidence:
  - src/orchestrator/executor.rs
  - src/orchestrator/websocket_client.rs
  - src/websocket/server.rs
  - src/orchestrator/relay.rs
  - docs/dev/decisions/controller-thin-bridge.md
  - docs/dev/decisions/generic-passthrough-default.md
  - docs/dev/decisions/configurable-runtime-container-prefix.md
  - docs/dev/decisions/host-container-isolation.md
---

# Controller cross-cutting integrator traps

Read before scoping any task that routes a command or message *through* the controller.
These are traps that are not tied to one specific controller capability.

## The controller is a thin bridge — do not attribute payload semantics to it

For most runtime-directed commands the controller reads only `runtime_name` (for
routing) and blind-forwards the rest of the payload byte-identical, in both
directions. It does not interpret, validate, normalize, or default the business
fields inside those payloads — that meaning lives in the orchestrator (producer)
and the runtime (consumer). If a field is silently missing or malformed
end-to-end, the bug is almost never "the controller changed it"; it forwarded
what it received. Exceptions are narrow: content only the controller can author
(devcontainer config, `spawn_error`, `port_mapping`, container IDs). Do not design
a contract that expects the controller to fill in or reinterpret a value.

One sharp edge: the controller only carries metadata keys it names explicitly —
there is no catch-all passthrough for metadata. A metadata key the orchestrator
*requires* on a runtime-originated command must be a field the controller already
knows, or it is dropped on the hop. Adding a new required cross-hop metadata field
is therefore a coordinated three-repo change, not zero-controller-change.

## Launch-family commands have no task_id — do not await a task_response

`spawn_runtime`, `spawn_builder`, and `build_image` are the "launch family": they
carry **no** `task_id`, and the controller **suppresses** the synthetic ACK at the
source. So a caller must NOT block waiting for a correlated `task_response` — none
arrives. The real outcome is delivered later as asynchronous runtime events
(`registration` / `runtime_ready` for a runtime, and `launch_builder_ready` /
`launch_build_started` / `launch_build_complete` / `launch_failed` for a builder),
correlated by runtime/instance identity, not by request id. Treating these as
request/response relays will hang until timeout and report a spurious failure while
the launch actually succeeded.

## Container-prefix isolation — parallel stacks must set distinct prefixes

The controller's eviction and reaper logic only ever touches Docker containers
whose name matches its configured `RUNTIME_CONTAINER_PREFIX`; anything outside that
prefix (and any `platform-smith-*` infra name) is refused. Two controller stacks
sharing one Docker daemon MUST be given distinct prefixes — otherwise each will
consider the other's containers in-scope and cross-reap (auto-stop) them. This is
the single gate protecting a sibling stack's runtimes; default `ps-runtime`
reproduces legacy behavior but gives no isolation between two default stacks.

## Host processes are invisible to port eviction — expect an un-evictable error

The controller can only see and evict Docker **containers** holding a contended
port (it observes solely through the Docker socket, by design — no host namespace
access). If a plain **host process** holds the port, the controller cannot identify
or free it: it reports the holder as `Unknown` with a "free it externally and retry"
hint rather than resolving the conflict. Do not assume a port conflict is always
auto-recoverable; a host-process holder surfaces as an operator-actionable error.

## Readiness ≠ credential provisioned

A runtime reports **ready** when its sandbox is operational — this is decoupled
from coding-agent authentication. The static `auth_token` is optional at
registration, and coding-agent credentials are delivered separately (per-session /
post-registration). So "runtime is ready" does NOT imply "a Claude/Codex credential
was provisioned"; a session can still fail at auth on a fully-ready runtime. Track
readiness and credential state as independent facts.
