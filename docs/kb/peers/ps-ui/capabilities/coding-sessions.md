---
type: capability
title: "Coding sessions (ACP streaming chat)"
tags: [coding-sessions, acp, streaming, sse, chat, consumer]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui drives an interactive Claude/Codex coding session: sends prompts and applies a streamed ACP event transcript"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/sessions.ts
  - src/lib/acp/events.ts
  - src/lib/acp/apply-acp-event.ts
  - src/hooks/use-session-sse.ts
  - src/hooks/use-session-transcript.ts
  - src/stores/pending-prompts.ts
  - src/stores/command-center-tabs.ts
  - src/routes/_auth/workspaces/$workspaceUuid/sessions/$sessionName.tsx
see_also:
  - {repo: orchestrator, capability: "Coding session lifecycle & event log", intent: "owns session state + the persisted event stream this UI replays", descriptive: true}
  - {repo: ps-api, capability: "Session SSE projection & harness-to-ACP transform", intent: "serves the /events/stream SSE and projects raw harness output into the ACP event_type vocabulary this UI applies", descriptive: true}
  - {repo: runtime, capability: "Coding-agent session execution", intent: "runs the actual Claude/Codex harness whose output becomes these events", descriptive: true}
  - {repo: controller, capability: "Runtime WebSocket bridge", intent: "bridges orchestrator to the runtime that produces session output", descriptive: true}
---

# Coding sessions (ACP streaming chat)

**What it does.** Realizes the interactive coding-agent chat: a user sends prompts to a
Claude/Codex session running in a container and watches the agent's reply — text, reasoning,
tool calls, file diffs, token usage — stream in live. ps-ui is a pure consumer of the backend
session API; it owns no session state, only the transcript rendering and the send path.

**How a peer interacts (backend contracts consumed).** All against ps-api `/v1` (JWT — see
context.md), each session addressed **by name** (never uuid) with an optional `runtime` qualifier:
- `POST /v1/sessions` — create/launch a session. `key fields:` `runtime_name`, `session_id`,
  `initial_prompt?`, `model?`, `session_type?` (`claude|shell`), `agent_definition_uuid?`.
- `GET /v1/sessions` — list (enriched projection). `GET /v1/sessions/{name}?runtime=` — one session.
- `POST /v1/sessions/{name}/input` — send a prompt. Body `{runtime_name, content}`.
- `GET /v1/sessions/{name}/events?after=&runtime=` — transcript backfill; returns a **raw
  `SessionEvent[]`** (no wrapper). Also the polling fallback.
- `GET /v1/sessions/{name}/events/stream?token=&runtime=` — the **live SSE stream** (see below).
- `POST /v1/sessions/{name}/stop?runtime=` — graceful terminate (agent stops, runtime stays up).
- `PATCH /v1/sessions/{name}?runtime=` — rename; `display_name` is the only mutable field.

**Observable behavior (streaming signal a peer must preserve).** The transcript is driven by an
**SSE stream**, not by `session.state`. The `EventSource` connects to `/events/stream` and listens
for three **named SSE events**:
- `session_event` — one transcript frame; `data` is a JSON envelope `{event_uuid, event_type,
  data, created_at, severity?, phase?}`.
- `session_status` — `{state}` session-status change.
- `stream_end` — **terminal signal**: the UI closes the EventSource and marks disconnected.

Inside each `session_event`, `event_type` is the **ACP `session/update` vocabulary** ps-ui applies
(non-exhaustive, `key fields:`): `agent_message_chunk`, `agent_thought_chunk`, `user_message_chunk`,
`tool_call`, `tool_call_update`, `usage_update`, `plan`, plus the retained Platform-Smith frames
`turn_end` and `error`. A turn is closed by **`turn_end`** (carries `stopReason`) — that is the
per-turn terminal signal that flips the "thinking" state off and re-enables the composer. Bootstrap
lifecycle frames (`bootstrap_session_*`, `*_failed`) ride the same envelope. Unknown `event_type`s
are parsed-and-ignored (forward-compat), so adding a new frame kind does not break the UI.

**Observable behavior (send + optimism).** On send, ps-ui appends a **provisional** user bubble
immediately, then reconciles it against the server-echoed `user_message_chunk` by trimmed-content +
a timestamp window. A freshly-launched session's `initial_prompt` is stashed in a **pending-prompts**
store keyed by session name and rendered optimistically until real frames arrive. Consecutive
`agent_message_chunk` deltas coalesce into one streaming assistant message.

**Contract (identity hand-off).** Session **name** is the routing key everywhere (route param is
`$sessionName`); `session_uuid` is the stable id used only for cache-matching. A name is
**per-runtime-instance** and can be reused across runtimes — hence the `runtime` qualifier on every
read/mutate. `stop` returns **202 = "requested", not "stopped"**; the terminal `closed` arrives later
via the SSE `stream_end`/`session_status` or status polling.

**Invariants.** Frames are deduped by a **composite key `(event_uuid, event_type, toolCallId)`**,
NOT bare `event_uuid` — one backend row fans out to multiple ACP frames sharing a uuid (a `result`
row → `usage_update` + `turn_end`; an assistant row with N tool_use blocks → N `tool_call` frames).
Uuid-only dedup would drop `turn_end` and freeze the session on "Thinking". Re-application is
idempotent (the `seen` set), so backfill-then-SSE replay of the same uuids is safe.

**Failure modes.** SSE failing ≥5 times in 60s → the UI **falls back to polling** `GET .../events`
every 2s + status every 3s (transcript keeps flowing, just coarser). A 403 on send = a non-owner
tried to message a **personal-integration** session (owner-only); the provisional is dropped, not
retried. `stop` 404/409 is a safe idempotent no-op.

**Gotchas.**
- Sessions are keyed by **NAME, not uuid**, in routes and every session endpoint; omit `runtime` and
  ps-api resolves the newest instance of a reused name (can target the wrong runtime).
- The transcript is **not** driven by `session.state` — an eager session stays `pending` while it
  streams; gate the composer on terminal states only, and key streaming/connection off SSE frames.
- The SSE endpoint takes the JWT as a **`token` query param** (EventSource cannot set headers), not
  an `Authorization` header like the REST calls.
- ACP frame **ordering matters**: `turn_end` must survive dedup or the turn never closes; the reducer
  also reorders backfilled `user_message_chunk`s (recorded mid-turn) to before the turn they prompted.
- Field names inside ACP frames are **camelCase** (`toolCallId`, `stopReason`); the envelope stays
  snake_case (`event_uuid`, `event_type`).
- Command-center tabs stream only a **bounded live-SSE set** (the ~4 most-recent tabs) to stay under
  the HTTP/1.1 6-connection cap; backgrounded tabs pause their EventSource and catch up on re-entry.

**See also.** orchestrator — *Coding session lifecycle & event log* (owns state + the persisted
stream). ps-api — *Session SSE projection & harness-to-ACP transform* (serves the SSE, projects raw
harness output into these `event_type`s). runtime — *Coding-agent session execution*. controller —
*Runtime WebSocket bridge*.
