---
type: capability
title: "Coding-session gateway"
tags: [sessions, coding-agent, claude, codex, gateway]
timestamp: 2026-07-07T03:45:26Z
description: "Lifecycle of Claude/Codex coding sessions through the gateway: create, list, input, events, rename, stop"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/sessions.go
  - cmd/models/session.go
  - cmd/db/session_writes.go
  - cmd/db/session_events.go
  - cmd/handlers/session_sse.go
  - cmd/server/main.go
see_also:
  - {repo: ps-api, capability: "Realtime streams", intent: "owns the session-events SSE stream — the live tail and the async stop/close signal"}
  - {repo: orchestrator, capability: "Session lifecycle", intent: "actually spawns/stops sessions; the gateway proxies create/input/stop to it", descriptive: true}
---

# Coding-session gateway

**What it does.** Front-door for coding-agent (Claude/Codex) sessions: creates a session on a
runtime, relays user input to it, and serves the session's transcript/state reads from the DB.
Writes that change a live session (create, input, stop) proxy to the orchestrator; reads (list,
get, events) and rename are served by ps-api directly.

**How a peer interacts.** REST under `/api/v1/sessions` (JWT auth):
- `POST /sessions` — create; `GET /sessions`, `GET /sessions/{name}` — list/get
- `POST /sessions/{name}/input` — send a message; `POST /sessions/{name}/stop?force=true` — stop
- `PATCH /sessions/{name}` — rename (sets `display_name`; direct DB write, no orchestrator)
- `GET /sessions/{name}/events` — one-shot event list; live tail is the SSE stream at
  `GET /sessions/{name}/events/stream` (see Realtime streams)

**Session identity — the seam peers hit.** Sessions are keyed on the wire by **name**
(`session_id`, supplied by the caller at create), not by UUID. Responses also carry a
server-side `session_uuid`, but every route addresses `{name}`. Names are unique only **per
runtime instance**: the optional `?runtime=<runtime_name>` query disambiguates on get, events,
stream, and stop; input disambiguates via its required body field `runtime_name`. When absent
the newest instance's row wins. Peers correlating with systems that
key by UUID must capture `session_uuid` from the create/get response.

**Observable behavior.**
- **Create** proxies to the orchestrator and returns its session record. `runtime_name` +
  `session_id` are required; `controller_name` is optional (omitted → orchestrator resolves the
  controller from the runtime — the "attach to a running runtime" path). Optional
  `agent_definition_uuid` selects the coding agent; the orchestrator freezes the resolved
  `coding_agent_type` at spawn (absent/unresolvable → null, treated as Claude).
- **Input** takes a body of `{runtime_name, content}` and returns a simple ack. The session must
  be in `started` state — enforced upstream by the orchestrator (the gateway forwards regardless,
  so a not-started rejection arrives in the orchestrator's error shape).
- **Stop is async**: returns **202 Accepted** immediately. The terminal signal is the session
  flipping to `state: closed` — observed via the events SSE ending with `stream_end`, or by
  re-reading `GET /sessions/{name}` until `state = closed`. Stopping an already-closed session
  is a safe no-op (unknown names still 404 at the tenant pre-check). `?force=true` requests a
  hard close (default graceful).
- **Events (one-shot)** reads are capped at 10,000 stored event rows (each row projects to 0..N
  transcript frames); deeper history must use
  the SSE stream. Cursor: `?after=<event_uuid>` (the last event seen). Absent → from the
  beginning; malformed → 400; well-formed but unknown → **410 Gone**, meaning the client must
  restart from scratch. REST and SSE emit byte-identical frames; raw per-harness envelopes are
  never exposed and noise rows are silently omitted.

**Contract.** Session record — key fields: `session_uuid`, `session_id` (the name),
`display_name`, `runtime_name`, `state`, `coding_agent_type` (nullable → Claude),
`runtime_kind`, `project_name`, `connection` (`connected|heartbeat_stale|disconnected`),
timestamps. Event — key fields: `event_uuid` (the cursor token), `event_type`, `severity`,
`data`, `created_at`. Tenant identifiers never appear on the wire.

**Invariants.** All routes are company-scoped; list only ever returns the caller's company.
Rename is the only session write the gateway commits itself. `coding_agent_type` is frozen at
spawn — changing the agent definition later never re-decodes an existing transcript.

**Failure modes.** Unknown name and cross-tenant access are indistinguishable: both return
**404** (existence hiding) — on get, events, rename, stop, and input. Sessions created with a
**personal** integration credential can only receive input from their owner; other company
members get **403** on input but can still read the transcript/events. Orchestrator unreachable
→ proxied routes fail with a mapped upstream error; DB-direct reads keep working.

**Gotchas.** The 202 from stop is an acknowledgment, not completion — never assume `closed`.
The one-shot events list is capped; a long transcript silently truncates at the cap (use the
stream). A 410 on `?after=` means the cursor is gone, not a transient error — restart the read.
Rename accepts the full session shape but only `display_name` is applied.

**Business-critical data.** `session` (keyed by `session_name` + runtime instance; holds
`state`, frozen `coding_agent_type`, personal-integration owner gate) and `session_event`
(append-only transcript; `event_uuid` is the wire cursor, the internal sequence key is never
exposed).
