---
type: interface
title: "Downlink: controller ⇄ runtime containers (WS 9002)"
tags: [websocket, runtime, downlink, command-catalog, passthrough]
timestamp: 2026-07-07T00:00:00Z
description: "Commands the controller sends a runtime container, and the messages/events a runtime sends back (forwarded upstream)"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/protocol/commands.rs
  - src/protocol/runtime.rs
  - src/websocket/server.rs
provides_interfaces:
  - name: setup_devcontainer
    kind: ws-command
    peer: runtime
    intent: "Post-registration environment + optional static credential setup"
consumes_interfaces:
  - name: registration
    kind: ws-event
    peer: runtime
    intent: "Runtime announces itself; binds the connection for routing"
  - name: git_mint_token_request
    kind: ws-event
    peer: runtime
    intent: "Runtime asks upstream for a short-lived git token"
  - name: runtime_ready
    kind: ws-event
    peer: runtime
    intent: "PID-1 init done; runtime is operational"
---

# Downlink: controller ⇄ runtime containers (WS 9002)

The controller runs a WebSocket **server** on port 9002 (`CONTROLLER_WS_PORT`);
each runtime container connects **in** and registers. The controller then routes
commands down and forwards the runtime's messages/events up to the orchestrator
(see `uplink-orchestrator.md`). All traffic uses the shared 3-tier envelope (see
context.md). A runtime is addressed by its `runtime_name`.

Handshake order per connection: **registration** → controller pushes
post-registration setup → steady-state message loop.

## Controller → runtime commands

- **setup_devcontainer** — pushed right after registration; carries the
  generated devcontainer config and an **optional** static `auth_token`. Absent
  token → runtime still reports ready (readiness == sandbox operational, not
  credential-present). See capability `devcontainer-credential-setup`.
- **setup_coding_agent_credential** — the single generic carrier for an
  orchestrator-resolved coding-agent credential, forwarded **verbatim**
  (`auth_type` + secret `fields` + non-secret `config` + `connection_uuid`). The
  controller does **no** per-`auth_type` branching — the runtime dispatches.
  Fire-and-forget. Sent only when the launch resolved a credential.
- **setup_codex_credentials** — pushes a Codex ChatGPT-subscription `auth.json`
  bundle so the runtime writes `~/.codex/auth.json` (0600, only-if-missing).
  Fire-and-forget; the bundle is secret and never logged.
- **setup_git_clone** — pushed when the spawn carried git config; the runtime
  clones each repo and reports `git_clone_complete`. *Launch-family*
  (fire-and-forget from the controller's dispatch side).
- **relayed / fire-and-forget passthroughs** — the runtime-directed inbound
  commands from the uplink (`execute_command`, `execute_claude`,
  `spawn_claude_session`, `claude_session_input`, `close_claude_session`,
  `setup_claude_credentials`, `check_claude_installation`, and forwarded
  `git_mint_token_response`) are forwarded down verbatim; only `runtime_name` is
  read for routing. See `uplink-orchestrator.md` for their reply semantics.

## Runtime → controller messages/events

Correlated replies resolve a pending relay (→ upstream `task_response`); pure
events are enriched with `runtime_name` + `instance_uuid` and forwarded up.

- **registration** — announces `name`, `version`, `platform`, `instance_uuid`,
  optional observability `role`. Binds the connection for routing; the
  controller reconciles the echoed `instance_uuid` against the pre-minted one
  (prefer pre-minted) and forwards the reconciled registration upstream. See
  capability `runtime-registration-bridge`.
- **setup_complete** — devcontainer setup result (`success`, `message`,
  optional CLI `version`); marks the runtime ready locally, then forwarded up.
- **runtime_ready** — emitted once after PID-1 init; forwarded up (the
  orchestrator advances the runtime lifecycle on it).
- **command_response** — correlated reply to `execute_command`
  (stdout/stderr/exit_code).
- **claude_response** — correlated reply to `execute_claude`
  (stdout/stderr/exit_code).
- **claude_session_started** — correlated reply to `spawn_claude_session`
  (session_id, optional pid/model).
- **claude_session_failed** — correlated failure reply to a session start.
- **claude_credentials_setup** — correlated reply to `setup_claude_credentials`
  (success + credential type).
- **claude_installation_status** — correlated reply to
  `check_claude_installation` (installed / path / version).
- **error_response** — a runtime-side error; resolves the matching pending relay
  as a failure, or is forwarded up as a failed `task_response` if uncorrelated.
- **git_mint_token_request** — runtime asks upstream for a git token; forwarded
  up (the orchestrator answers with `git_mint_token_response`).
- **git_clone_complete** — clone finished; forwarded up (the orchestrator may
  auto-trigger a bootstrap session on it).
- **session_output** — streamed session output; forwarded up.
- **session_closed** — session ended; forwarded up.
- **launch_builder_ready / launch_build_started / launch_build_complete /
  launch_failed** — builder-pod launch progress. The controller forwards these
  **verbatim** (thin pipe): it never originates, synthesizes, dedups, or
  reorders them; `instance_uuid` is enriched into metadata for observability
  only (the orchestrator correlates off the runtime-threaded typed `data`). See
  capability `build-pipeline`.

## Generic passthrough (the default)
Any runtime message the controller does **not** recognize falls through to the
default arm and is forwarded upstream (enriched with `runtime_name`), **not**
dropped. New runtime→orchestrator event types therefore work end-to-end without
a controller change — the controller is deliberately dumb about their meaning.

## Gotchas
- Readiness is decoupled from credentials: a runtime can be ready with no
  coding-agent credential present.
- The controller reads only `runtime_name` for routing on forwarded commands; it
  does not parse or interpret payload contents (including secrets).
- Correlated relays that outlive their runtime resolve as `RuntimeDisconnected`
  failures on disconnect, never a hang.
