---
type: gotcha
title: "Runtime cross-cutting integrator traps"
tags: [fire-and-forget, session-ids, backpressure, env-stripping, durability, readiness, path-confinement, shutdown]
timestamp: 2026-07-09T10:42:29Z
description: "Cross-cutting traps for any peer that commands, observes, or correlates against the runtime — not owned by a single capability"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/router/mod.rs
  - src/core/router/handlers.rs
  - src/core/protocol/payload.rs
  - src/output.rs
  - src/websocket/client.rs
  - src/session/manager.rs
  - src/process/manager.rs
  - src/util/fs.rs
  - docs/kb/kb-config.yaml
---

# Runtime cross-cutting integrator traps

## 1. The fire-and-forget family — never await a reply

Two input families deliberately have **no success reply**. A peer that awaits a response
envelope to either of them hangs by design:

- `claude_session_input` — success is silent; the only signal is the subsequent stream of
  `session_output` events. Only failures produce an error.
- `setup_codex_credentials` / `setup_coding_agent_credential` — never reply at all; a bad
  credential surfaces later as an agent auth error inside the session. (The legacy
  `setup_claude_credentials` is the exception in the family: it replies with a
  `claude_credentials_setup` message carrying a success flag.)

`a2a_deliver` used to belong here but **no longer does**: it gets no direct *response envelope*, but
its delivery outcome is now separately confirmed by an outbound **`a2a_delivered`** ack
(`status:"delivered"` once handed to the session, `status:"failed"`+`error` on a miss), keyed by the
inbound `message_id`. An upstream `delivered` mark is therefore **no longer necessarily optimistic**
when a `message_id` rides the wire. The only still-silent cases are degrade-safe: **no `message_id`**
on the wire (older orchestrator) or an un-correlatable missing `session_id` — those behave as the old
warn-and-drop. Detail is in the a2a-messaging capability.

Detail lives in the coding-agent-sessions and a2a-messaging capabilities; the trap here is the
shared posture: these are one-way inputs, not request/response.

## 2. Unknown input: two deliberately different failure postures

- **Command path** (`type:"command"`): unknown command → an `error_response`
  (`"Unknown command: X"`). You get told.
- **Message path** (`type:"message"`): unknown input → **silently ignored**. You get nothing.

Do not infer "no error means it was handled" on the message path.

## 3. Two session identifiers — never conflate them

- `session_id` — the orchestrator session **NAME**: the runtime's session-registry key and the
  correlation field on `session_output`.
- `session_uuid` — the DB **UUID**: the MCP `X-PS-Session-ID` header value, the tool-seed key,
  and `originating_session_id` on outbound tool calls.

Sending one where the other is expected breaks MCP tool identity or output correlation with no
loud error (see gotcha 1/2 postures).

## 4. `claude_session_started` / `claude_session_failed` fire for Codex too

The event names are historical, not agent-specific — Codex spawns emit the same pair. Codex is
spawn-per-turn (no resident process), so its `claude_session_started` carries **no pid**. Do not
key logic on the event name to mean "this is Claude", and do not require a pid.

## 5. Bounded channels drop on saturation — nothing blocks, nothing retries

- Outbound WS channel: bounded at **1000**; on saturation the message is **dropped**
  (send fails, never blocks). A slow consumer loses events silently from the peer's view.
- Session stdin (retained sessions): bounded at **100**.
- Codex mid-turn inbound queue (spawn-per-turn): bounded at **100**; overflow rejects the input —
  surfaced as an `error_response` on the `claude_session_input` command path, and for `a2a_deliver`
  as an `a2a_delivered{status:"failed"}` ack when a `message_id` is on the wire (invisible only in
  the degrade-safe no-`message_id` case; per gotcha 1).

## 6. `PLATFORM_SMITH_*` env is stripped from sessions and spawned daemons — NOT from `execute_command` children

Coding-agent sessions and daemon processes spawned via `spawn_process` never see
`PLATFORM_SMITH_*` variables; **one-shot `execute_command` shell children DO inherit them** (no
stripping on that path). Do not rely on env-stripping for anything run through `execute_command`.
Anything a stripped child needs must ride a different channel — e.g. per-session Codex MCP
identity travels as `PS_SESSION_ID` precisely because a `PLATFORM_SMITH_`-prefixed variable would
never arrive.

## 7. Nothing survives a runtime restart

All correlation state is in-memory: pending MCP tool calls, pending a2a durability acks,
per-session tool seeds, the session registry, tracked processes. A restarted runtime
re-registers on reconnect but restores **none** of it — in-flight requests die, sessions are
gone, and staged credentials must be re-pushed via a fresh runtime instance.

## 8. `shutdown` terminates the pod's PID 1

The runtime sends `shutdown_ack` and then **exits the process (exit 0)**. Since the runtime is
PID 1, the entire pod ends. Treat `shutdown` as pod termination, not service restart.

## 9. Registration is not readiness

`registration` is sent **before** the command router exists. A runtime is safe to command only
after its readiness event: `launch_ready` (unified product), `launch_builder_ready` (builder),
or `runtime_ready` (legacy path). Commands sent between registration and readiness are not
guaranteed to be routed.

## 10. Path confinement hard-rejects — including customer-committed symlinks

Every file-delivery surface (agent/secret files at spawn, `.platform-smith/` batch delivery)
confines writes to its allowed root and **rejects symlinks recursively** along the target path.
A symlink the customer committed into their own repo clone can therefore hard-fail a spawn's
inline agent-file delivery — the spawn fails loudly (`claude_session_failed`) and the session
never starts. This is a deliberate security posture, not a bug to route around.
