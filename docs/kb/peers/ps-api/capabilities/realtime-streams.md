---
type: capability
title: "Realtime streams"
tags: [sse, websocket, streaming, sessions, launches, terminal]
timestamp: 2026-07-07T03:33:49Z
description: "The three live streams the browser consumes: session-event SSE, launch-timeline SSE, and the terminal WebSocket proxy"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/session_sse.go
  - cmd/handlers/session_sse_transform.go
  - cmd/handlers/launch_sse.go
  - cmd/handlers/terminal_ws.go
  - cmd/handlers/stream_auth.go
  - cmd/server/main.go
  - docs/dev/decisions/no-raw-harness-envelopes-on-the-wire.md
see_also:
  - {repo: ps-api, capability: "Auth and identity gateway", intent: "the two-layer stream auth and ?token= fallback these streams reuse"}
  - {repo: ps-api, capability: "Coding-session gateway", intent: "owns session lifecycle and the one-shot REST events read that serves byte-identical frames for backfill"}
  - {repo: ps-api, capability: "Runtime launch gateway", intent: "the launch flow that yields the instance_uuid keying the launch stream"}
  - {repo: orchestrator, capability: "Interactive terminal sessions", intent: "the upstream terminal WebSocket the proxy dials with trusted identity headers", descriptive: true}
---

# Realtime streams

**What it does.** The three live channels the frontend consumes: a coding-session transcript stream
(SSE), a launch progress timeline (SSE), and an interactive terminal (WebSocket). ps-api serves both
SSE streams by polling the platform DB directly (no orchestrator hop); the terminal is a
bidirectional proxy to orchestrator.

**How a peer interacts.**
- `GET /api/v1/sessions/{name}/events/stream` — session-event SSE (optional `?runtime=` disambiguator).
- `GET /api/v1/launches/{instance_uuid}/events/stream` — launch-timeline SSE.
- `GET /api/v1/terminal/{session_id}/ws` — terminal WebSocket.
Auth on all three: `Authorization: Bearer <jwt>`, or `?token=<jwt>` query fallback — browsers cannot
set headers on EventSource/WebSocket. Validation on the two SSE streams is two-layer: JWT
signature/expiry plus a DB check that the user belongs to the token's company. The terminal
WebSocket validates the JWT only at the gateway (no DB membership check; whether the upstream
enforces tenancy is UNKNOWN from this repo).

**Observable behavior — session SSE.** On connect: an immediate `session_status` event (`{state}`).
Then, as events land (DB poll ~2s cadence): `session_event` frames, `session_status` on each state
transition, and a heartbeat SSE *comment* every 15s. When the session reaches
`closed`/`crashed`/`failed`: a `stream_end` event (`reason: session_terminated`), then close. Frames
carry only the canonical ACP `session/update` projection (event types: `agent_message_chunk`,
`agent_thought_chunk`, `user_message_chunk`, `tool_call`, `tool_call_update`, `usage_update`,
`turn_end`, `error`) — raw harness output never appears on the wire (repo decision
*no-raw-harness-envelopes-on-the-wire*); telemetry/noise events are omitted entirely. The one-shot
REST events read serves the identical frame shapes for backfill.

**Resume — session SSE.** Each `session_event` frame's SSE `id:` is its `event_uuid`. A reconnecting
client sends `Last-Event-ID: <event_uuid>` and receives strictly-after events. An unknown or
malformed id (or a lookup failure) degrades to a full replay from the start — never an error.
Delivery is therefore at-least-once: clients must dedupe by `event_uuid` (one source event can fan
out to several frames sharing that uuid).

**Bootstrap race.** Session names prefixed `bootstrap-` get a bounded lookup retry (~5s) because the
UI may open the stream before orchestrator has created the session; any other unknown name 404s
immediately.

**Observable behavior — launch SSE.** On connect: an immediate `launch_status` snapshot
(`{status, failed_phase?}`). Then one `launch_event` frame per timeline row — key fields:
`launch_event_uuid`, `event_type`, `phase?`, `severity`, `data`, `created_at` — forwarded raw; the
UI owns the label map. SSE `id:` is the numeric launch-event sequence; `Last-Event-ID: <number>`
resumes strictly after it. When status reaches `ready`|`failed`: `stream_end`
(`reason: launch_ready|launch_failed`), then close. Covers all launch kinds (sandboxes and repo
imports).

**Observable behavior — terminal WS.** After upgrade, messages pump bidirectionally between the
browser and orchestrator's same-path terminal WebSocket; the validated identity travels upstream as
`X-User-UUID`/`X-Company-UUID` headers. Orchestrator unreachable → WS close code 1013 ("orchestrator
unavailable"). Keepalive: server pings every 30s, drops the connection after 60s without a pong.

**Contract.** Inputs: the path params above plus the JWT; no request body. Errors: 401 (missing/bad
token, or user not in the company), 400 (missing/invalid path param), 404 (session/launch not found
— including foreign-tenant resources, which are indistinguishable from nonexistent).

**Invariants.** Auth before any stream byte (two-layer on SSE; JWT-only on the terminal WS); all
DB reads tenant-scoped. Session frames only
ever carry projected ACP shapes. Resume cursors are monotonic; replay is safe by design (client
dedupe). Heartbeats are SSE comments, never data events.

**Failure modes.** A failing DB poll skips that tick and keeps the stream open — persistent DB
trouble looks like a silent stall with heartbeats, not an error event. `stream_end` is best-effort;
a client may observe the close without it. Terminal WS has no resume — a drop loses the stream and
reconnect starts fresh.

**Gotchas.** Event latency is up to ~2s (poll-driven, not push). `tool_call_update` output text is
truncated at 4096 bytes; the true byte count rides the frame's raw-output metadata for a "truncated"
indicator. Session resume ids are UUIDs, launch resume ids are numbers — not interchangeable. The
`?token=` fallback puts the JWT in a URL; it is the same token the header path accepts.

**Business-critical data.** Session stream: the session-event log (per-session; `event_uuid` is the
wire id, an internal sequence is the never-exposed keyset cursor) plus the session's `state` and its
frozen coding-agent-type discriminator (selects the per-harness projection for the session's life).
Launch stream: the launch-event log keyed by `instance_uuid` plus the launch status head.
