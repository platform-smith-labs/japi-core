---
type: capability
title: "Controller WebSocket link"
tags: [websocket, registration, readiness, reconnect, handshake, backpressure]
timestamp: 2026-07-06T23:40:38Z
description: "The runtime's single northbound WS to the controller: registration → readiness sequencing, infinite-backoff reconnect, sole-writer metadata injection, drop-on-saturation backpressure"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/websocket/client.rs
  - src/utils/retry.rs
  - src/output.rs
  - src/config.rs
see_also:
  - {repo: controller, capability: "runtime-registration-bridge", intent: "the controller-side peer that binds the connection on registration and forwards/filter-forwards these events upstream"}
---

# Controller WebSocket link

**What it does.** Maintains the runtime's single outbound WebSocket to the controller (the URL
injected as `PLATFORM_SMITH_WS_URL`). Everything the runtime says or hears — commands in, events and
session output out — travels on this one connection. It never accepts inbound connections.

**How a peer interacts.** The controller accepts the connection, then receives exactly two startup
events in a fixed order: `registration` first, then one readiness event. Commands sent down the same
socket are dispatched to the runtime's command router.

**Observable behavior.**
- **Handshake sequencing (the load-bearing contract):**
  1. `registration` is sent **before** the command router exists — it lets the controller bind the
     connection to a runtime identity, nothing more. Commands sent in this window are not
     dispatched until after the readiness event — the read loop starts only then.
  2. The router is built.
  3. Exactly **one** readiness event is emitted, chosen by launch path:
     `launch_ready{instance_uuid}` (unified product — the orchestrator-injected UUID was present;
     legacy `runtime_ready` is **suppressed** on this path) | `launch_builder_ready` (builder mode;
     echoes the injected `instance_uuid` when present, omits the key otherwise — the controller
     injects correlation on forward) | `runtime_ready{runtime_name}` (legacy self-minted-UUID path).
  A runtime is safe to command only **after** its readiness event.
- **Reconnect:** connection loss (or server Close) re-enters an infinite exponential-backoff retry
  loop — 1s initial delay, ×2 per attempt, capped at 60s; it never gives up. Every reconnect re-runs
  the **full** handshake (fresh registration + fresh readiness event), so the controller will see
  repeat registrations from the same runtime over its lifetime.
- **Keepalive:** WS Ping from the server is answered with Pong on a control channel.

**Contract.**
- `registration` — key fields: `name` (runtime name), `version`, `platform`, `instance_uuid`, `role`
  (mode wire string, e.g. `builder`; non-builder is product).
- Readiness events carry `instance_uuid` + `role` (launch events) or `runtime_name`
  (legacy `runtime_ready`). Startup/launch events carry no `request_id` — correlation is by UUID.
- Optional inbound `registration_ack{status, runtime_instance_id, message?}` — on success the
  runtime stores `runtime_instance_id`; a failure ack is only logged, never fatal, and no reply is
  sent. The controller is not required to ack.
- Every outbound payload (all commands' responses and events, not just handshake) has
  `runtime_name` + `instance_uuid` injected into `payload.metadata` by the single forwarder task
  that owns the socket's write half; metadata fields already set by the producer (e.g.
  `originating_session_id`) are preserved untouched.

**Invariants.**
- **instance_uuid echo rules:** an orchestrator-minted UUID (injected env) is echoed verbatim; when
  absent the runtime self-mints a fallback that is **never** echoed for correlation — correlation
  fields carry the empty string (or omit the key on typed launch payloads) instead. The
  `registration` payload's `instance_uuid` key must always be **present**, even when empty: the
  controller's registration struct has no default, and an absent key drops the registration.
- One readiness event per connection, always after registration, never before the router exists.
- Sole-writer: the forwarder task is the only writer to the socket — no interleaved partial frames.

**Failure modes.**
- **Backpressure:** the outbound channel is bounded (1000 messages) and sends are non-blocking — on
  saturation the message is **dropped** (producer sees a send-failure), never queued or blocked. A
  flooded link loses events silently from the peer's perspective.
- Dropped WS: in-flight correlated requests (credential mints, bridged tool calls) fail immediately
  or time out; in-memory correlation state survives within the process but nothing survives a
  process restart. After reconnect the controller must treat the runtime as freshly registered.
- Metadata injection failure on a message is non-fatal: the original message is forwarded without
  the injected fields.

**Gotchas.**
- Registration must **never** be treated as readiness — the window between them has no command
  router. Gate commands on the readiness event.
- On the unified-product path do not wait for `runtime_ready`; it is deliberately suppressed in
  favor of `launch_ready`.
- Reconnects replay registration + readiness; peers must be idempotent to repeat handshakes from a
  live runtime.
- Delivery is best-effort under load (drop-on-saturation); the link gives no delivery receipt.

**See also / peers.** controller — *runtime-registration-bridge*: binds the connection on
`registration`, filters builder registrations locally, and forwards readiness/launch events
upstream to the orchestrator.
