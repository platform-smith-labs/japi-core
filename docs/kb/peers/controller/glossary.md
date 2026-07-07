---
type: glossary
title: "Controller glossary"
tags: [glossary, runtime, builder, registry, relay, terminology]
timestamp: 2026-07-07T00:00:00Z
description: "Domain terms a peer repo needs to reason about the controller"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/websocket/registry.rs
  - src/protocol/commands.rs
  - src/orchestrator/executor.rs
  - src/docker/sandbox.rs
  - src/docker/port_holder.rs
  - docs/dev/decisions/relay-pipeline-pattern.md
  - docs/dev/decisions/controller-thin-bridge.md
---

# Glossary

- **runtime (product runtime)** — a per-user customer container the controller
  spawns; runs the platform-smith runtime binary as PID 1 and connects back to the
  controller's DOWN server. Role `Product` (the default).
- **builder pod** — an internal, short-lived container (fixed `ubuntu:22.04`) that
  runs an in-pod `docker build` to produce a customer image. Role `Builder`;
  observability-only distinction, not a routing gate.
- **sandbox container** — a managed runtime container carrying the
  `platform-smith.type=sandbox` label; the Docker-host object the controller
  creates, starts, and reaps.
- **runtime_name** — the routing key for a runtime: which container a message is
  to/from, and the suffix in the `{prefix}-{runtime_name}` container name.
- **instance_uuid** — the unique id of a *specific* runtime instance; injected at
  spawn, echoed at registration, preserved when forwarding upstream. Distinct from
  `runtime_name` (a name can be re-used across instances).
- **RuntimeRegistry** — the controller's in-memory map of connected runtimes,
  per-runtime metadata, and pending correlations. Lost on restart.
- **relay** — the correlated request/response path: send a command to a runtime,
  await its matching reply via a oneshot keyed on `task_id`, with a timeout.
- **fire-and-forget** — a runtime-directed send that gets a synthetic ACK
  immediately; no correlated reply is awaited.
- **event-forwarding** — an async stream of runtime-originated events forwarded
  upstream verbatim, not correlated to any pending request.
- **launch-family command** — an inbound command that carries **no** `task_id` and
  whose synthetic ACK is suppressed (`spawn_runtime`, `spawn_builder`,
  `build_image`); its real outcome arrives later as asynchronous runtime/builder
  events, correlated by runtime/instance identity, not by request id.
- **launch_\* events** — builder-pod launch progress (`launch_builder_ready`,
  `launch_build_started`, `launch_build_complete`, `launch_failed`) forwarded
  verbatim from a builder pod up to the orchestrator; the controller does not
  originate them.
- **thin bridge** — the design rule that the controller routes and manages
  lifecycle but owns no payload semantics; meaning lives in orchestrator/runtime.
- **eviction (reaper)** — stopping a stale/conflicting container, gated by the
  `is_safe_to_stop` allowlist: only names matching `{prefix}-<non-empty>` qualify.
- **devcontainer setup** — controller-generated dev-container configuration
  (OS-specific package managers) pushed to a runtime to prepare its environment.
