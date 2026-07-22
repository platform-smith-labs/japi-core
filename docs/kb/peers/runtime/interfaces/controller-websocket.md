---
type: interface
title: "Controller WebSocket protocol"
tags: [websocket, protocol, commands, events, controller]
timestamp: 2026-07-09T10:42:29Z
description: "The runtime's single wire interface: 3-tier JSON envelopes, the closed inbound command set, and every outbound event a peer must handle"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/protocol/envelope.rs
  - src/core/protocol/command.rs
  - src/core/protocol/payload.rs
  - src/core/router/mod.rs
  - src/core/router/handlers.rs
  - src/core/router/seen_deliveries.rs
  - src/websocket/client.rs
  - src/session/io.rs
  - src/session/monitor.rs
  - src/mcp_server/tools.rs
  - src/cred_server/mod.rs
provides_interfaces:
  - {name: "runtime-command-dispatch", kind: websocket-command, peer: controller, intent: "controller (relaying the orchestrator) drives the runtime with type:command envelopes"}
  - {name: "runtime-event-stream", kind: websocket-message, peer: controller, intent: "runtime emits lifecycle, reply, session-stream, builder, and bridge events as type:message envelopes"}
consumes_interfaces:
  - {name: "controller-ws-endpoint", kind: websocket, peer: controller, intent: "runtime dials out to the controller's WS URL at boot; infinite reconnect with backoff"}
  - {name: "correlated-message-replies", kind: websocket-message, peer: controller, intent: "four controller→runtime type:message inputs: git_mint_token_response, agent_tool_result, a2a_result, a2a_deliver"}
see_also:
  - {repo: controller, capability: "Runtime WebSocket bridge", intent: "the peer that terminates this connection and relays to/from the orchestrator"}
---

# Controller WebSocket protocol

The runtime's **only** wire interface. The runtime dials the controller's WS endpoint at boot and
reconnects forever with backoff; everything else (commands, events, tool bridging, A2A) rides this
one connection as JSON text frames.

## Envelope (3 tiers)

`{version, type, payload}` where `payload = {command, metadata, data}`. `version` is `"1.0"`.
`type` is one of:

- **`command`** — controller → runtime. Dispatched to a handler by `payload.command`.
- **`message`** — runtime → controller events, **plus** exactly four controller → runtime
  correlated replies handled on the message path: `git_mint_token_response`, `agent_tool_result`,
  `a2a_result`, `a2a_deliver`.

Correlation identifiers on inbound controller→runtime replies ride in `metadata` (`request_id`,
and for a2a delivery `session_id`, `from_project`). Exception: `git_mint_token_response` carries
its correlation fields (`request_id`, `token`, `expires_at`, `error`) inside `data`. Session-stream
events carry `session_id`/`sequence` in `data`. The runtime's sole-writer
forwarder **auto-injects `runtime_name` + `instance_uuid` into every outbound message's
`metadata`**, preserving any producer-set fields. Lifecycle/launch events carry no `request_id` —
peers correlate them by `instance_uuid`.

Two session identifiers coexist and must never be conflated: `session_id` = the orchestrator
session NAME (registry key, `session_output` correlation); `session_uuid` = the DB UUID (MCP
header, tool-seed key, `originating_session_id`).

## Inbound commands (CLOSED list — the whole dispatch table)

| Command | Reply posture |
|---|---|
| `execute_command` | correlated `command_response` (stdout/stderr/exit_code) or `error_response` |
| `spawn_process` | `process_started {id, pid}` or `process_failed {id, error}` |
| `execute_claude` | correlated one-shot `claude_response` or `error_response` |
| `kill_process` | `kill_process_response {success}`; unknown id / kill failure → `error_response` |
| `list_processes` | `list_processes_response {processes[]}` |
| `shutdown` | `shutdown_ack {processes_terminated}` then the **process exits (exit 0)** — the pod's PID 1 terminates |
| `setup_devcontainer` | `setup_complete {success, message, version?}`; static `auth_token` is optional — readiness = sandbox operational, not credentialed |
| `setup_codex_credentials` | fire-and-forget, **no reply**; write failure is non-fatal and surfaces later as an agent auth error |
| `setup_coding_agent_credential` | fire-and-forget, **no reply**; `auth_type`-dispatched, staged in memory — credentials are frozen per instance |
| `setup_claude_credentials` | `claude_credentials_setup {success, message, credential_type?}` |
| `setup_git_clone` | always emits `git_clone_complete` (even for an empty repo list); per-repo results + first-repo aggregates |
| `materialise_platformsmith_files` | `materialise_platformsmith_files_complete {files_written}` or `_failed {error_message, partial_files_written}` (partial files are NOT durable) |
| `spawn_claude_session` | `claude_session_started` or `claude_session_failed`, then an event stream of `session_output` |
| `claude_session_input` | fire-and-forget on success (**no ACK** — output arrives via `session_output`); `error_response` if the session is unknown, ended/terminated/failed/closing, or the send fails |
| `close_claude_session` | no direct success reply — `session_closed` arrives from the session monitor; `error_response` on failure |
| `check_claude_installation` | `claude_installation_status {installed, path?, version?}` |
| `registration_ack` | inbound ack for the runtime's own `registration`; stored internally, no reply |
| `build_image` | **builder mode only** — event stream `launch_build_started` → `launch_build_complete {image_tag}` or `launch_failed {phase?, error_message?}`; on a context-assembly failure or malformed payload, `launch_failed` arrives with no preceding `launch_build_started`. Outside builder mode: dropped with a WARN (no `error_response`). A second `build_image` in the same builder pod is silently ignored (single-build invariant) |

Any other command name → `error_response` with message `Unknown command: <name>`.

## Outbound events (runtime → controller, all `type:"message"`)

**Lifecycle.** On every (re)connection: `registration {name, version, platform, instance_uuid,
role}` is sent FIRST, **before the command router exists** — it binds the connection but is
**never readiness**. Then exactly one readiness event once the dispatcher is live:
`launch_ready {instance_uuid}` (unified product — injected UUID), or `launch_builder_ready`
(builder — echoed injected UUID when present), or `runtime_ready {runtime_name}` (legacy
self-minted-UUID path). A runtime is safe to command only after its readiness event. Peers see
registration + readiness again after every reconnect.

**Command replies.** `command_response`, `error_response`, `process_started`, `process_failed`,
`kill_process_response`, `list_processes_response`, `shutdown_ack`, `setup_complete`,
`claude_response`, `claude_credentials_setup`, `claude_installation_status`,
`git_clone_complete`, `materialise_platformsmith_files_complete` / `_failed` — the reply set of
the table above. `git_clone_complete` key fields: `success`, per-repo `repos[]`, and first-repo
aggregates `git_sha`, `resolved_branch`, `created_from_base`, `created_from`,
`platform_smith_dockerfile_present`, `exposed_container_ports`, `instance_uuid` (builder echo).

**Session streaming.** `claude_session_started` / `claude_session_failed` — these names are used
for **both Claude and Codex** sessions (historical, not agent-specific). `pid` is present only
for retained (Claude) sessions; Codex is spawn-per-turn, so absence means "no resident process".
`session_output` — key fields: `session_id`, a per-session **monotonic `sequence`** counter, and
the output content; ordering within a session is by `sequence`. `session_closed` — emitted by the
session monitor on any exit (requested close or crash).

**Builder.** `launch_build_started`, `launch_build_complete`, `launch_failed {phase?,
error_message?}` — all builder-mode only (the `build_image` event stream); each echoes
`instance_uuid` when injected.

**Bridge-originated** (runtime initiates, controller/orchestrator must answer on the message path):
- `git_mint_token_request` → answered by `git_mint_token_response` (token/error in `data`); an
  in-pod git operation blocks on it.
- `agent_tool_call {tool, args}` with `metadata.request_id` + `originating_session_id` → answered
  by `agent_tool_result {success, result?, error?}` matching `metadata.request_id`. 30s timeout,
  **no auto-retry**; the agent sees the timeout/error as the tool result.
- `a2a_message` → answered by `a2a_result {accepted, message_id?, seq?, error?}` — a
  **durability ack** (orchestrator persisted it), NOT a peer reply. Outbound `metadata` may carry an
  optional `to_session` (a destination session **name**, present only when the sender set it;
  resolved/validated by the orchestrator, not the runtime). Inbound peer messages arrive
  separately as `a2a_deliver` (metadata `session_id` targets the live session, `message_id` is the
  dedup + receipt key; `body` in `data`) and are surfaced into that session as user input.
- `a2a_delivered {status:"delivered"|"failed", error?}` (metadata `message_id` + `session_id` name)
  — the runtime's **delivery-outcome ack** for an inbound `a2a_deliver`, emitted once the body is
  handed to the target session (`delivered`) or on a genuine miss (`failed`). Runtime-initiated, no
  reply awaited. Distinct from `a2a_result` (that acks *durability* of an outbound send). Suppressed
  when the inbound `a2a_deliver` carried no `message_id` (degrade-safe) or had no correlatable
  `session_id`. A redelivery of an already-`delivered` `(message_id, session)` re-acks `delivered`
  idempotently (the runtime dedups so a session is never double-injected).

## Failure postures (deliberately asymmetric)

- **Command path**: unknown command → `error_response("Unknown command: X")`.
- **Message path**: unknown or malformed input (bad `request_id`, unknown session, orphan reply,
  missing fields) is **silently ignored / warn-and-drop** — never a response envelope.
  Exception: an `a2a_deliver` miss now emits an outbound `a2a_delivered{status:"failed"}` ack when
  the inbound carried a `message_id` (terminal signal for the orchestrator's receipt row), so it is
  no longer left merely pending; still, no error reaches the *sender* via the runtime.
- Outbound WS uses a bounded non-blocking channel: on saturation a message is **dropped**, never
  blocks. Session stdin and Codex mid-turn queues are bounded at 100.
- Secrets (tokens, credential payloads, auth bundles) are written/forwarded but never logged or
  echoed back on the wire; error strings are sanitised.
