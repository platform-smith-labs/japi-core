---
type: capability
title: "Workflow signaling bridge (ps-workflow)"
tags: [workflow, ps-workflow, signals, session-completion, mcp-tool, bridge]
timestamp: 2026-07-09T10:40:45Z
description: "How in-runtime agent/session lifecycle events reach the ps-workflow engine so parked workflow tasks and await-signal nodes complete"
repo: orchestrator
commit_sha: 2fa8172
evidence:
  - cmd/websocket/tool_ps_signal.go
  - pkg/psworkflow/client.go
  - cmd/server/main.go
  - cmd/db/mcp_tools.go
  - cmd/websocket/handlers.go
see_also:
  - {repo: ps-workflow, capability: "Signal ingestion", intent: "owns POST /api/v1/signals; resolves company from the parked signal row and completes the await-signal node", descriptive: true}
  - {repo: ps-workflow, capability: "Task completion ingestion", intent: "owns POST /api/v1/task-completions; completes the parked run-agent-session / send-prompt task", descriptive: true}
---

# Workflow signaling bridge (ps-workflow)

**What it does.** The orchestrator is the HTTP bridge between a runtime's coding-agent session and
the **ps-workflow** engine. A runtime is deliberately HTTP-free, so the orchestrator forwards three
in-session lifecycle events to ps-workflow on the runtime's behalf, letting a parked workflow step
complete: an **agent signal** (the agent calls a tool to complete an await-signal node), a
**session close**, and a **per-turn completion**. The orchestrator only forwards — ps-workflow owns
the sessionId→workflow mapping, idempotency, tenant-scoping, and the actual completion.

**How a peer interacts.** Two directions:
- **An agent inside a session** calls the `ps_signal` MCP tool with `{correlation_id, payload}`. The
  runtime relays it to the orchestrator (as an `agent_tool_call`); the orchestrator does the outbound
  POST. `correlation_id` is an opaque handle the *workflow* injected into the agent's prompt when it
  parked the signal — the agent echoes it back verbatim; it is never minted or validated here.
- **ps-workflow** is the receiving peer for all three events. The orchestrator calls ps-workflow's
  own routes: `POST /api/v1/signals` for `ps_signal`, and `POST /api/v1/task-completions` for both
  `session_closed` and `agent_turn_completed` (one endpoint, discriminated by an `event` field).

**Observable behavior.**
- `ps_signal` is **synchronous** end-to-end: the agent receives exactly one tool result — success on
  a ps-workflow 2xx, else a safe generic error. A non-2xx (e.g. no matching open signal, or a
  cross-tenant reject) surfaces to the agent as a generic "signal delivery failed"; the underlying
  status/detail is logged server-side only, never wired.
- `session_closed` and `agent_turn_completed` are **best-effort and fire-and-forget**: a failure or
  timeout only logs and never blocks the session or surfaces an error. A peer must not treat the
  absence of a forwarded event as authoritative — ps-workflow's own idempotent completion is the
  source of truth.
- **Ships dark.** The whole bridge is opt-in, enabled only when `PS_WORKFLOW_BASE_URL` is configured.
  When unset there is no forwarder: `ps_signal` returns "signal forwarding not configured" and the two
  lifecycle events are silent no-ops.

**Contract.**
- `ps_signal` tool args: `{correlation_id (required), payload?}`. Body forwarded to ps-workflow:
  exactly `{correlation_id, payload}` (an absent/empty payload is defaulted to `{}`). The trusted
  company rides the outbound `X-Company-UUID` header, **never** the body.
- `session_closed` body (to `/api/v1/task-completions`): key fields `{event:"session_closed",
  session_id, company_uuid, runtime_name, state:"closed"|"crashed", exit_code?, signal?, reason}`.
- `agent_turn_completed` body (same endpoint): key fields `{event:"agent_turn_completed", session_id,
  company_uuid, turn_id?, stop_reason, summary?}`.
- `session_id` is the session **name** (the caller-provided id), used as the correlation key on the
  ps-workflow side — not the `session_uuid`. Best-effort UUIDs (`user_uuid`, `session_uuid`) are
  omitted from body/headers when unresolved.
- Errors: the tenant/authorization checks that decide whether a signal or completion is accepted are
  **enforced by ps-workflow**, not here — a cross-tenant or unmatched signal is rejected at
  ps-workflow and the orchestrator only relays the outcome (for `ps_signal`) or logs it (for the
  fire-and-forget events).

**Invariants.**
- **Company is never trusted from the wire.** For all three events the `company_uuid` comes from the
  trusted controller WebSocket connection (spoof-checked by the tool dispatcher for `ps_signal`) and
  rides the `X-Company-UUID` header; ps-workflow cross-checks it for tenant safety.
- **`ps_signal` is capability-scoped.** An agent can only complete a workflow whose opaque
  `correlation_id` it was explicitly handed at park time, so the tool is safe to grant broadly.
- **Grant-gated like other platform tools.** `ps_signal` is a baseline MCP grant for a primary
  session but is **omitted for a secondary (write-restricted) session** — see session-management.
- **Exactly-once completion is ps-workflow's job.** The orchestrator may forward a duplicate/late
  event; ps-workflow deduplicates (e.g. an open→closed CAS on the parked row).

**Failure modes.**
- `PS_WORKFLOW_BASE_URL` unset → `ps_signal` fails fast with "signal forwarding not configured"; the
  lifecycle events do nothing.
- ps-workflow down / slow → `ps_signal` returns a generic delivery error to the agent (it fails fast
  rather than stalling to the agent's tool timeout); `session_closed`/`agent_turn_completed` are
  dropped with a log line only.
- Forwarder saturated (too many concurrent in-flight forwards) → `ps_signal` returns "signal service
  busy" so the agent fails fast instead of hanging.

**Gotchas.**
- The agent never asserts its own company or session identity — both are derived server-side from the
  originating session; the agent supplies only `correlation_id` + `payload`.
- A `ps_signal` success means ps-workflow accepted the signal, not that the downstream workflow step
  has finished its own work.
- `session_closed` and `agent_turn_completed` share one ps-workflow endpoint and are told apart only
  by the `event` discriminator.

**See also / peers.** ps-workflow owns the receiving endpoints (`/api/v1/signals`,
`/api/v1/task-completions`) and all completion/idempotency logic; this capability is only the
orchestrator-side forwarder. Session close/turn events originate from the session lifecycle — see the
session-management capability.
