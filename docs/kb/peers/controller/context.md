---
type: context
title: "Controller system context and ubiquitous facts"
tags: [context, websocket, ports, auth, container-naming, protocol, secrets]
timestamp: 2026-07-09T11:13:06Z
description: "The controller's two WebSocket surfaces, auth/naming/label contracts, message envelope, and secret handling — stated once for the whole KB"
repo: controller
commit_sha: 4e237d3
evidence:
  - src/main.rs
  - src/config.rs
  - src/orchestrator/websocket_client.rs
  - src/docker/sandbox.rs
  - src/docker/port_holder.rs
  - src/protocol/envelope.rs
  - src/protocol/commands.rs
  - src/protocol/metadata.rs
  - src/websocket/registry.rs
  - docs/dev/decisions/configurable-runtime-container-prefix.md
  - docs/dev/decisions/tenant-isolated-builds.md
---

# System context

These are the **ubiquitous facts** every other concept in this KB relies on. They
are stated **once, here** — capability and interface concepts assume them and do
not repeat them.

## Two WebSocket surfaces (mind the direction)

The controller has exactly two WebSocket surfaces, in opposite directions:

- **UP (client)** — the controller **dials out to the orchestrator** and holds a
  client connection to it. Endpoint comes from `ORCHESTRATOR_WS_URL` (default
  `ws://host.docker.internal:9003/ws`). The orchestrator sends work down this link;
  the controller sends events/replies up it. The controller is the initiator.
- **DOWN (server)** — the controller **runs a WebSocket server** that runtime
  containers connect *into*. It binds `CONTROLLER_WS_PORT` (default `9002`). This
  is where runtimes register and stream their events.

## UP authentication contract

When the controller dials the orchestrator it authenticates via query params:
`?token=<CONTROLLER_TOKEN>&instance_uuid=<uuid>&environment_uuid=<uuid>`
(`environment_uuid` appended only when set; if no token is configured the token
param is omitted). The **orchestrator rejects an unauthenticated connection** — the
`CONTROLLER_TOKEN` is the controller's identity (a workspace-scoped token).

## Container-naming and eviction contract

A managed runtime container is named `{RUNTIME_CONTAINER_PREFIX}-{runtime_name}`
(default prefix `ps-runtime`). **That same prefix is the eviction/reaper
allowlist**: the controller will **never stop a container whose name does not match
`{prefix}-` with a non-empty suffix** — sibling services, databases, and bare
`{prefix}` (no suffix) are all excluded. Parallel dev stacks on one Docker daemon
set **distinct** prefixes (e.g. `ps-runtime-ws0005`) so they never cross-reap each
other's containers. A peer scoping multi-stack work must treat this prefix as the
isolation boundary.

## Docker labels on managed containers

Every managed container carries: `platform-smith.managed=true`,
`platform-smith.type` (e.g. `sandbox`), and `platform-smith.runtime-name=<name>`.
These labels — not the registry — are the durable "is this ours?" marker on the
Docker host.

## Injected container env (product runtimes)

At spawn the controller injects, so the runtime can connect back and identify
itself: `PLATFORM_SMITH_WS_URL` (the controller's DOWN URL), 
`PLATFORM_SMITH_RUNTIME_NAME`, and `PLATFORM_SMITH_INSTANCE_UUID` (echoed back at
registration).

## The 3-tier message envelope

All WebSocket messages share one shape:

- **Tier 1 — MessageEnvelope**: `{version, type, payload}`. `type` is `command` or
  `message`; routing is decided from Tier 1 without parsing deeper.
- **Tier 2 — CommandPayload**: `{command, metadata, data}`. `command` is the action
  name (e.g. `spawn_runtime`); `data` is parsed lazily by the handler.
- **Tier 3 — data**: command-specific payload.

**Identity keys flow in `metadata`**: `runtime_name` (routing — which runtime a
message is to/from) and `instance_uuid` (the specific runtime instance, preserved
when forwarding upstream). `task_id`/`request_id` correlate a reply to its request.

**Unknown metadata keys are forwarded verbatim.** The controller models only the
metadata keys it reads or injects; every other key it does not recognize is
preserved and re-emitted unchanged on each forward hop (a serde catch-all). So a
peer that adds a new metadata field on a runtime-originated command (e.g.
`to_session`) needs **no** controller change for it to reach the orchestrator —
this holds for metadata just as blind-forward holds for the payload `data`.

## Secrets are forwarded verbatim and never logged

Credential material — a runtime's `secret_env_vars`, resolved coding-agent
credential fields, and the Codex ChatGPT-subscription `auth.json` bundle — is
carried **opaquely**: held, forwarded byte-for-byte into the container, and
**never logged, `Debug`-printed, or reformatted**. Treat every credential field as
password-equivalent when reasoning about the controller.

## No database — in-memory registry only

The controller has **no datastore**. Live-runtime state (connections, per-runtime
metadata, pending correlations) is an **in-memory registry lost on restart**;
surviving runtimes reconnect and the orchestrator holds the durable record.
