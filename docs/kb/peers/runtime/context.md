---
type: context
title: "Runtime system context — connectivity, envelopes, modes, ubiquitous invariants"
tags: [runtime, websocket, context, invariants, sessions, security]
timestamp: 2026-07-09T10:42:29Z
description: "Who the runtime talks to and the ubiquitous facts stated once here, never repeated per capability"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/websocket/client.rs
  - src/utils/retry.rs
  - src/output.rs
  - src/core/protocol/envelope.rs
  - src/core/protocol/payload.rs
  - src/config.rs
  - src/runtime_state.rs
  - src/session/claude/session_type.rs
  - src/session/claude/mcp_config.rs
  - src/process/manager.rs
  - src/mcp_server/mod.rs
  - src/cred_server/mod.rs
  - src/core/router/handlers.rs
  - Dockerfile
  - docs/kb/kb-config.yaml
---

# Runtime system context

Facts here are **ubiquitous** — they apply to every capability and are stated only once, here.

## Connectivity

- The runtime's only control channel is a **single outbound WebSocket to the controller**, whose
  URL comes from the required `PLATFORM_SMITH_WS_URL` env var. The runtime never accepts inbound
  network connections from peers (its two side surfaces below are pod-local only).
- Connection loss triggers **infinite reconnection** with exponential backoff: 1s initial delay,
  ×2 per attempt, capped at 60s. The runtime never gives up and never crashes on WS failure.
- **Every (re)connect re-runs the full handshake**: registration is sent, then the readiness
  event is emitted. Peers must tolerate repeated registration/readiness from the same instance.
- **Registration is NOT readiness.** Registration is sent before the runtime's command router
  exists. A runtime is safe to command only after its readiness event: `launch_ready` (unified
  product path, carries `instance_uuid`), `launch_builder_ready` (builder mode), or
  `runtime_ready` (legacy self-minted-UUID path only; suppressed on the unified path).
- **Outbound is lossy under pressure**: the WS writer is fed by a bounded non-blocking channel —
  on saturation a message is **dropped**, never blocked on. Session stdin and Codex mid-turn
  queues are bounded at 100.

## Wire envelope

- Every outbound frame is `type:"message"`; inbound commands are `type:"command"`.
- A **single WS-writer task** owns the socket and auto-injects `runtime_name` and
  `instance_uuid` into `payload.metadata` of every outbound message. Capabilities never do this
  themselves; peers can rely on both fields being present on everything the runtime sends.
- Failure postures differ by path: an **unknown command** on the command path gets an
  `error_response` ("Unknown command: X"); unknown input on the message path is **silently
  ignored**. This asymmetry is deliberate.

## Operating modes

Exactly two modes, from `PS_RUNTIME_MODE`: **greenfield** (default) and **builder**. Any
unknown/legacy value degrades to greenfield with a WARN — it never crashes and never silently
becomes a build mode. Builder pods are build-only: no customer CMD is supervised.

## Two session-ID namespaces (never conflate)

- **session_id** = the orchestrator session **NAME**. It is the runtime's session-registry key
  and the correlation field on `session_output` streams.
- **session_uuid** = the session's **DB UUID**. It is the MCP `X-PS-Session-ID` header value,
  the tool-seed key, and the `originating_session_id` on outbound tool calls.

Every session-related capability uses one or both; which one matters per message is defined by
the wire contract, but the two namespaces above are fixed platform-wide.

## Nothing durable across restarts

All coordination state is **in-memory** and lost on pod/process restart: the session registry,
per-session MCP tool seeds, registration state, staged credentials, and all pending-correlation
maps (tool calls, A2A acks, git credential mints). On-disk artifacts are limited to:

- `/var/run/platform-smith/POD.md` (pod manifest),
- per-session dirs under `/var/run/platform-smith/sessions/` (**swept at every boot**),
- credential files written for coding agents and git.

Durable truth lives with the orchestrator. Notably, **coding-agent credentials are frozen per
instance** — pushed once post-registration and staged in memory; changing a credential requires
a fresh runtime instance.

## Process environment

- Default working directory for executed work is **`/workspace`**.
- The process runs as user **`psruntime`** (established by the image; default
  `HOME=/home/psruntime`).
- **`PLATFORM_SMITH_*` env vars are stripped from coding-agent session processes and spawned
  daemon processes** — but **NOT from one-shot `execute_command` shell children**, which inherit
  the runtime's full environment (including `PLATFORM_SMITH_*`). Per-session identity for the
  stripped children uses other vars (e.g. `PS_SESSION_ID`) for exactly this reason.

## Secrets

Secrets (tokens, credential payloads, auth bundles) are forwarded and written to their carriers
but **never logged**. Error strings sent on the wire are sanitised, customer-relative messages —
no raw stderr dumps or absolute internal paths.

## Pod-local side surfaces

- **MCP tool server**: loopback HTTP at `127.0.0.1:9099/mcp`, serving per-session seeded tools
  to coding agents running inside the pod; tool calls bridge over the WS to the orchestrator.
- **Git credential socket**: Unix domain socket at `/var/run/platform-smith/cred.sock`, served
  for the `ps-git-credential-helper` binary. Neither surface is reachable from outside the pod.
