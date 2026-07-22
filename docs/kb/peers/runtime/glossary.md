---
type: glossary
title: "Runtime domain glossary"
tags: [runtime, glossary, terminology]
timestamp: 2026-07-09T10:42:29Z
description: "Domain terms a peer repo needs to reason about the runtime"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/config.rs
  - src/runtime_state.rs
  - src/core/protocol/payload.rs
  - src/core/protocol/envelope.rs
  - src/core/router/mod.rs
  - src/core/router/handlers.rs
  - src/core/router/seen_deliveries.rs
  - src/session/types.rs
  - src/session/io.rs
  - src/mcp_server/tool_seeds.rs
  - src/websocket/client.rs
  - docs/kb/kb-config.yaml
---

# Runtime domain glossary

**runtime** — the logical pod-level entity (named by `runtime_name`); its DB `runtime.status`
lags and must not be used for readiness.

**runtime instance** — one concrete boot of the runtime inside a pod, identified by
`instance_uuid` (orchestrator-injected via env, self-minted only on the legacy path).
Readiness, credentials, and correlation are all per-instance.

**greenfield mode** — the default operating mode: a normal product pod serving commands and
coding-agent sessions. Any unknown `PS_RUNTIME_MODE` value degrades to it with a WARN.

**builder mode** — build-only pod mode (`PS_RUNTIME_MODE=builder`): runs one in-pod
`docker build` and emits ordered `launch_*` events; supervises no customer CMD. `build_image`
is accepted only here, and only once per pod.

**session_id** — the orchestrator session **NAME**. Registry key inside the runtime and the
correlation field on `session_output`. Not a UUID.

**session_uuid** — the session's **DB UUID**. Carried as the MCP `X-PS-Session-ID` header, the
tool-seed key, and `originating_session_id` on outbound tool calls. Never interchangeable with
session_id.

**retained session** — a coding-agent session with a resident child process kept alive across
turns (the Claude model); input is written to its stdin.

**spawn-per-turn session** — a session with no resident process (the Codex model): each turn
spawns a fresh process. Conversation continuity is the orchestrator's job (it replays
`codex_thread_id` / `agent_session_id`).

**readiness event family** — the message that makes a runtime instance safe to command:
`launch_ready` (unified product, carries `instance_uuid`), `launch_builder_ready` (builder),
`runtime_ready` (legacy self-minted-UUID path). Re-emitted on every reconnect; registration
itself is never readiness.

**fire-and-forget command** — a command family with no success reply by design (e.g.
`claude_session_input`, `setup_codex_credentials`/`setup_coding_agent_credential`; the legacy
`setup_claude_credentials` is the exception — it replies). Success is observed via later effects
(streamed output, agent behaviour), not an ack. (`a2a_deliver` was formerly in this family but now
emits a separate `a2a_delivered` outcome ack — see the **A2A** term.)

**correlated reply** — an outbound `type:"message"` carrying the inbound command's `request_id`
so the sender can match response to request; the counterpart of fire-and-forget.

**relay** — a command that originates at the orchestrator and reaches the runtime through the
controller without being interpreted (note: the controller's envelope-metadata allowlist drops
unknown metadata fields — payload `data` passes through verbatim).

**A2A** — agent-to-agent messaging: a session sends via the `a2a_send` MCP tool (acked only for
**durability** — the orchestrator persisted it, not that the peer replied; an optional `to_session`
targets a specific destination session by name) and receives via `a2a_deliver` into a live session.
Inbound delivery is **dedup-guarded** on `(message_id, session name)` and **delivery-acked** back to
the orchestrator via `a2a_delivered` (`delivered`/`failed`); a delivery with no `message_id` on the
wire stays silent (degrade-safe).

**MCP tool seed** — the per-session set of granted tools delivered in the session spawn payload
and served by the in-pod MCP server for that session_uuid; in-memory, empty at process start.

**POD.md** — the human/agent-readable pod manifest the runtime writes to
`/var/run/platform-smith/POD.md` after clone, describing the pod's repos and layout.

**agent_files / secret_files** — files delivered inline in a session spawn payload and
materialised before the agent starts: agent_files are plain config/instruction files;
secret_files carry sensitive content (written, never logged).

**claude_session_started / claude_session_failed** — session lifecycle event names used for
**both** Claude and Codex sessions; the `claude_` prefix is historical, not agent-specific.
