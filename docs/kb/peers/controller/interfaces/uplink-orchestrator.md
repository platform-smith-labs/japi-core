---
type: interface
title: "Uplink: controller ⇄ orchestrator (WS 9003)"
tags: [websocket, orchestrator, uplink, command-catalog, launch-family]
timestamp: 2026-07-09T11:13:06Z
description: "Inbound action catalog the orchestrator sends the controller, and what the controller sends back upstream"
repo: controller
commit_sha: 4e237d3
evidence:
  - src/protocol/commands.rs
  - src/orchestrator/executor.rs
  - src/orchestrator/websocket_client.rs
  - src/protocol/orchestrator.rs
provides_interfaces:
  - name: spawn_runtime
    kind: ws-command
    peer: orchestrator
    intent: "Create+start one product runtime container from a prebuilt image"
  - name: terminate_runtime
    kind: ws-command
    peer: orchestrator
    intent: "Stop+remove a named runtime container"
  - name: execute_command
    kind: ws-command
    peer: orchestrator
    intent: "Run a shell command in a runtime and get its correlated result"
consumes_interfaces:
  - name: registration
    kind: ws-event
    peer: runtime
    intent: "Runtime handshake, enriched and forwarded upstream"
---

# Uplink: controller ⇄ orchestrator (WS 9003)

The controller connects **out** to the orchestrator as a client (port 9003,
`ORCHESTRATOR_WS_URL`), authenticating with `CONTROLLER_TOKEN`. It receives
task commands and forwards runtime events/results upstream. All traffic uses the
shared 3-tier envelope (see context.md); this file is the direction-specific
action catalog. The controller is a **thin bridge** — for runtime-directed
commands it reads only `runtime_name` for routing and blind-forwards the rest
verbatim; payload *meaning* lives in the orchestrator or runtime, not here.

## Inbound (orchestrator → controller)

Named by wire command. "Correlated" = the orchestrator gets a `task_response`
keyed by `task_id`. "Fire-and-forget" = synthetic ACK, no runtime reply awaited.
"Launch-family" = **no `task_id`**, the ACK is **suppressed**, and the
orchestrator instead awaits later runtime events (see below).

### Self-contained handlers (controller acts directly)
- **spawn_runtime** — create+start a product runtime container from a prebuilt
  image, publish ports, inject secret env. *Launch-family* (ACK suppressed;
  success correlates by `instance_uuid` via the runtime's later `registration` /
  `runtime_ready`; **failure is delivered as a controller-origin
  `launch_failed`** — see Outbound). See capability `spawn-runtime`.
- **spawn_builder** — bring up the fixed-`ubuntu:22.04` builder pod that later
  runs the in-pod image build. *Launch-family* (ACK suppressed; correlate by
  `instance_uuid` via later `launch_*` events). See capability `build-pipeline`.
- **terminate_runtime** — stop+remove the named runtime container. Correlated
  (returns a `task_response`). Terminates regardless of the informational
  `reason`. See capability `terminate-runtime`.
- **send_message** *(legacy)* — deliver a text message to a runtime by name.

### Relayed (correlated request → runtime → `task_response`)
Forwarded to the target runtime; the runtime's reply is turned into a
`task_response` keyed by the request's `task_id`. A disconnecting runtime
resolves the pending relay as a failure (`RuntimeDisconnected`), never a hang;
default reply timeout is `CONTROLLER_COMMAND_TIMEOUT_SECS` (300s).
- **execute_command** — run a shell command; reply carries stdout/stderr/exit_code.
- **execute_claude** — run a one-shot Claude turn; reply carries stdout/stderr/exit_code.
- **spawn_claude_session** — start an interactive coding-agent session; reply is
  the session-started (or session-failed) event.
- **setup_claude_credentials** — provision coding-agent credentials in the
  runtime; reply reports the configured credential type.
- **check_claude_installation** — probe the runtime's Claude CLI; reply reports
  installed / path / version.

### Fire-and-forget (forwarded down, synthetic ACK)
- **claude_session_input** — send input to a running session. Correlated by
  `task_id` (synthetic success ACK; no runtime reply awaited).
- **close_claude_session** — close a running session. Correlated (synthetic ACK).
- **build_image** — hand the `.platform-smith/` recipe fileset to the builder
  pod to run the in-pod `docker build`. *Launch-family* (blind passthrough, ACK
  suppressed; outcome arrives as `launch_*` events). See capability `build-pipeline`.

### Forwarded down to a runtime (routed by `runtime_name`)
- **git_mint_token_response** — the orchestrator's answer to a runtime's
  `git_mint_token_request`; delivered as a `message` to the named runtime.

## Outbound (controller → orchestrator)

- **task_response** — result of a **correlated** task (`success`,
  `response`/`error`, and for command/claude tasks `stdout`/`stderr`/`exit_code`).
  For launch-family commands **no `task_response` is sent at all** (suppressed
  at source). The `TaskResponseData` shape retains a `spawn_error` and a
  `port_mapping` field for task_id-carrying consumers; the launch-family spawn's
  `port_mapping` readback was retired (never delivered; work-2607070349).
  Correlated commands always emit theirs.
- **launch_failed (controller-origin)** — emitted when the controller's OWN
  `spawn_runtime` handling fails (work-2607070349): `data` carries
  `instance_uuid` (correlation), `phase: "starting_runtime"`, and an
  `error_message` composed from the structured spawn error as
  `"{class}: {raw} — {hint}"`. Emitted **only** on failure with a present
  `instance_uuid`; success emits nothing. Same wire shape as a forwarded
  builder-pod `launch_failed`.
- **heartbeat** — every 30s: controller `instance_uuid`, version, uptime, and
  the connected-runtime census (name + `runtime_instance_uuid`). Orchestrator
  treats >90s silence as stale. See capability `heartbeat-liveness`.
- **forwarded runtime events** — every runtime→controller message (see
  `downlink-runtime.md`) is enriched with `runtime_name` + `instance_uuid` in
  metadata and forwarded up. Unrecognized runtime messages are forwarded too
  (generic passthrough), never dropped; unknown metadata keys ride through
  verbatim as well (see context.md), so a peer adding a cross-hop metadata field
  needs no controller change.

## Gotchas
- **Launch-family is event-driven, not request/response.** A peer that waits for
  a `task_response` to `spawn_runtime` / `spawn_builder` / `build_image` will
  wait forever — await `registration` / `runtime_ready` / `launch_*` instead.
- On an unparseable inbound envelope the controller tries to recover the
  `task_id` and synthesize a FAILED `task_response` so the task doesn't hang in
  SENT; if `task_id` is unrecoverable the message is dropped.
- The controller **forwards** runtime-originated `launch_*` events as a dumb
  pipe (no synthesis, dedup, or reorder) — with ONE origination exception: it
  emits `launch_failed` for its **own** `spawn_runtime` failures
  (work-2607070349). Builder-pod death is still the orchestrator's
  BUILDING-phase timeout, never controller-synthesized.
- A `spawn_builder` / `build_image` failure inside the controller is still NOT
  delivered (only `spawn_runtime` gained the failure event) — those remain
  timeout-covered.
- Secrets (`secret_env_vars`, credential `fields`, codex `auth.json`) are
  forwarded verbatim and never logged.
