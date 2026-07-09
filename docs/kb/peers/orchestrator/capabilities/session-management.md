---
type: capability
title: "Interactive session management"
tags: [sessions, claude, agent, shell, rest-api, runtime]
timestamp: 2026-07-09T10:40:45Z
description: "How a peer creates, feeds input to, lists, and closes interactive agent/shell sessions on a running runtime via the REST API"
repo: orchestrator
commit_sha: 2fa8172
evidence:
  - REST_API.md
  - cmd/handlers/sessions.go
  - cmd/db/sessions.go
  - cmd/models/session.go
  - cmd/models/task.go
  - cmd/db/mcp_tools.go
see_also:
  - {repo: orchestrator, capability: "Workflow signaling bridge (ps-workflow)", intent: "a session close / per-turn completion is forwarded to ps-workflow to complete a parked workflow task"}
---

# Interactive session management

**What it does.** Runs interactive coding-agent (Claude/Codex) and shell sessions *inside an
already-running runtime container*. A peer starts a session, feeds it turns of input, watches it
progress, and stops it — the runtime container stays up so multiple sessions can share it and
outlive any one session.

**How a peer interacts.** All routes are under `/api/v1/sessions` (the orchestrator listens on
:9003, internal-only behind the ps-api gateway; caller identity — company + user — arrives via
trusted gateway headers, not a request field):
- `POST /api/v1/sessions` — create/attach an agent session on a running runtime.
- `POST /api/v1/sessions/{name}/input` — send one turn of input to a session.
- `POST /api/v1/sessions/{name}/stop[?force=true]` — stop the agent *process* (not the container).
- `POST /api/v1/sessions/{name}/close` — graceful close (equivalent to a non-force stop).
- `POST /api/v1/sessions/shell` — start an ephemeral shell session (returns a terminal WS path).
- `GET /api/v1/sessions` and `GET /api/v1/sessions/{name}` — list / fetch. **Deprecated** here;
  ps-api now owns these reads DB-direct. Do not build new callers on the orchestrator reads.

**Observable behavior.** Create is *attach, not launch*: it targets an existing runtime by
`runtime_name` and dispatches a spawn onto it — no container is started. The session row is created
synchronously in state `pending` and returned immediately; the actual agent spawn is queued to the
controller and runs asynchronously. Calling create again for the same runtime starts an *additional*
session on it (many sessions per runtime). Input and close return the queued Task; the agent's
output and state changes are observed out-of-band (poll the session, or the SSE/event stream keyed
by `session_uuid`). Stop is fire-and-forget: it returns the session **unchanged** (still
pending/started) and the terminal state flips later when the runtime reports the session closed.

**Contract.**
- Create — in: `{runtime_name, session_id, controller_name?, initial_prompt?, model?, working_dir?,
  session_args?, session_env?, agent_definition_uuid?}`. `controller_name` is optional (resolved from
  the runtime when omitted). Out: the `Session` (state `pending`, carrying `session_uuid`).
- Input — in: `{runtime_name, content}` (path `{name}` = the session id). Out: the queued `Task`.
- Stop — no body; optional `?force=true`. Out: the current `Session` (unchanged at dispatch time).
- Close — no body. Out: the queued `Task`.
- Shell — in: `{runtime_name, controller_name, cols?, rows?, shell?, working_dir?}`. Out:
  `{task_uuid, session_id, terminal_ws_path}` — connect the terminal WebSocket at that path.
- Errors follow the standard `{success:false, error}` envelope: unknown session → 4xx "Session not
  found"; runtime not connected / no live controller → 4xx (no orphan session row is created).

**Invariants.** All routes are company-scoped by the gateway identity — a peer only ever sees/acts on
its own company's sessions. Create never launches a container; it requires the named runtime to be
already connected. Coding-agent harness type (`coding_agent_type`) and `model` are frozen at spawn
and never re-derived on read. A graceful stop/close lands as `closed`; a force stop (SIGKILL) lands
as `crashed`. Terminal state is authoritative only from the runtime's async close event.
`session_role` (`primary` | `secondary`, default `primary`) is frozen at create and is **not**
overwritten on a re-drive of the same session key.

**Session role and tool posture.** A `secondary` session (e.g. an ephemeral judge session) is
write-restricted **by convention**: its baseline MCP platform-tool grant is empty, so write-capable
platform tools (`ps_transfer_files`, `save_artifact`, `ps_signal`) are rejected at call time with
"tool not granted". A `primary` (or empty/absent role, for back-compat) keeps the full baseline. This
is a complementary MCP-layer restriction, **not** a hard security boundary and it does not gate the
coding agent's built-in Read/Write/Bash.

**Failure modes.** Runtime disconnected at create → clear 4xx "Runtime not connected", no session
persisted. Stop/close against a session whose runtime is gone → 4xx (agent already unreachable), not
a 500. Stopping an already-terminal session is an idempotent no-op (no dispatch) — a double "Stop" is
safe. Input mid-turn on a Codex session may return a retryable invalid-state (turns are serialized). <!-- lint-ok: serialize — concurrency sense (one turn at a time), not marshalling -->

**Gotchas.** `session_id` is the caller-provided name and is the `{name}` used on every session-scoped
route — not the `session_uuid`. Create's return in `pending` does **not** mean the agent is running;
it means the spawn was accepted — watch the session/event stream for `started`/`failed`. Stop returns
success without confirming termination. Shell sessions are *ephemeral* — they have **no** session DB
row, so the `GET /sessions` reads and stop/close routes do not apply to them; drive them purely over
the returned terminal WS path. Brownfield runtimes hard-require a `working_dir`; if the orchestrator
can't resolve one for such a runtime the spawn is rejected by the runtime as a contract violation.

**Business-critical data.** Agent sessions persist to the `session` table keyed by `session_uuid`,
carrying `session_name` (the caller's id), `runtime_name`, `state`
(`pending`→`started`→`closed`/`crashed`/`failed`), `coding_agent_type`, `model`, and `session_role`
(`primary`/`secondary`, frozen at create). State is
uniqueness-scoped per `(company, runtime_instance, session_name)`; a re-create for the same key
converges onto the existing row rather than duplicating. (Company/user tenancy scoping applies as
everywhere — see context.md.)
