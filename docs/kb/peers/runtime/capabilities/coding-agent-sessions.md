---
type: capability
title: "Coding-agent sessions (Claude + Codex)"
tags: [sessions, claude, codex, streaming, spawn-per-turn, coding-agent]
timestamp: 2026-07-06T23:40:38Z
description: "Interactive Claude (retained NDJSON process) and Codex (spawn-per-turn) sessions: spawn/input/close commands, streamed session_output, session_closed lifecycle"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/session/manager.rs
  - src/session/claude/session_type.rs
  - src/session/codex/session_type.rs
  - src/session/io.rs
  - src/session/monitor.rs
  - src/session/session_kind.rs
  - src/session/types.rs
  - src/core/router/handlers.rs
  - src/core/protocol/payload.rs
see_also:
  - {repo: runtime, capability: "agent-config-and-secret-materialisation", intent: "spawn-inline agent_files/secret_files written before exec; a bad file fails the spawn"}
  - {repo: runtime, capability: "coding-agent-credential-dispatch", intent: "stages the agent credentials this capability merges into the session env"}
  - {repo: runtime, capability: "mcp-tool-server", intent: "the per-session platform tools both agents reach; seeded from granted_tools at spawn"}
  - {repo: orchestrator, capability: "session-management", intent: "owns Codex conversation continuity (persists and replays the thread id) and the session NAME/UUID identities"}
---

# Coding-agent sessions (Claude + Codex)

**What it does.** Runs interactive coding-agent conversations inside the pod — Claude CLI or Codex —
executing the agent process(es), streaming every output line back upstream, and reporting session end.
This is the runtime's core executor capability.

**How a peer interacts.** Three commands, whose `claude_`-prefixed names are historical and serve
**both** agent types:
- `spawn_claude_session` — key fields: `session_id` (session NAME), `session_uuid` (DB UUID),
  `coding_agent_type` (`"claude"` default | `"codex"` — the agent selector), `initial_prompt?`,
  `working_dir?`, `args` (Claude CLI flags; ignored for Codex), `env`, `system_prompt?`,
  `codex_thread_id?` (resume a prior Codex conversation), `granted_tools`, `agent_files`, `secret_files`.
- `claude_session_input` — `{session_id, content, role="user", agent_session_id?}` (the last is the
  Codex resume handle the orchestrator replays).
- `close_claude_session` — `{session_id, force}`.

**Two process models (the load-bearing distinction).**
- **Claude = retained**: one long-lived CLI process per session speaking NDJSON (`stream-json` in and
  out, partial messages included). Input is written to its persistent stdin; the process holds the
  conversation itself.
- **Codex = spawn-per-turn**: no resident process. Each input spawns a fresh `codex exec` that emits
  JSONL and exits; the turn's output is forwarded raw. Continuity is via a **thread id** captured from
  the first turn's `thread.started` line and passed to `codex exec resume` on later turns. Durable
  continuity is the **orchestrator's** job — it persists the id and replays it as `codex_thread_id` on
  spawn (post-restart) and `agent_session_id` on input; the runtime's in-memory capture is a fallback.
  Input arriving while a turn is running is **queued** (bounded, 100) and drained FIFO as follow-up
  resume turns rather than dropped (rare residual: a queued item is dropped if its drain turn fails
  to spawn).

**Observable behavior.**
- Spawn replies with `claude_session_started{session_id, pid?, model?}` or
  `claude_session_failed{session_id, error}`. Codex sessions carry **no pid** — absence means "no
  resident process", it is normal, not an error.
- Output streams as `session_output{session_id, stream: stdout|stderr, data, sequence}` events, one
  per line, with a per-session **monotonic** `sequence` shared by both streams (and, for Codex, across
  turns). Envelope metadata carries `request_id` = the session NAME and `session_type` = `claude|codex`.
- `claude_session_input` success has **no ACK** — the only signal is the ensuing `session_output`
  stream. Failures come back as an `error_response`. Never treat input as request/response.
- Session end emits `session_closed{session_id, reason, exit_code?, signal?}`; on abnormal exit the
  `reason` embeds the last stderr lines as crash diagnostics. A Codex **per-turn** process exit does
  NOT emit `session_closed` — the session stays open until explicitly closed.
- `close_claude_session` success has no direct reply; `session_closed` is the completion signal
  (spawn-per-turn: emitted immediately; retained: after the process actually exits). Any close
  aborts an in-flight Codex turn (the per-turn process is killed, graceful or not); `force=true`
  additionally SIGKILLs a retained (Claude) process.

**Contract notes.** `session_id` is the orchestrator session NAME; `session_uuid` is the DB UUID used
for MCP tool identity — the full two-ID statement lives in context.md. Missing/invalid `session_uuid`
does not fail the spawn but leaves the session tool-blind (empty `tools/list`).

**Invariants.**
- Duplicate spawn of a live session name → the spawn fails (`claude_session_failed`, "already exists").
- Input to a session in Ended/Terminated/Failed/Closing state → `error_response`; unknown session →
  `error_response`.
- Close is two-phase: the session first turns Closing (rejects new input), then exactly one
  `session_closed` is emitted per session by a single owner (the exit monitor for retained; the close
  path for spawn-per-turn).
- Session env: `PLATFORM_SMITH_*` vars are stripped from every agent process; staged agent-credential
  env is merged **under** the per-session `env` (an explicit spawn var wins on collision). Working
  directory: explicit `working_dir`, else `/workspace`.
- Credential contents are written/forwarded but never logged or echoed in events.

**Failure modes.**
- Spawn failure (bad agent/secret file, missing binary, misconfig) → `claude_session_failed{error}`;
  no session is registered, so a retry does not hit "already exists".
- Codex mid-turn queue overflow (100 pending) → input rejected with an `error_response`; the
  in-flight turn is unaffected.
- Agent auth failure surfaces on the streams (stderr lines / Codex `turn.failed` JSONL) and, for a
  crashed retained process, in the `session_closed` diagnostics — not as a spawn-time error.
- Closing a Codex session with turns still queued drops the queue (the session is ending).

**Gotchas.**
- The `claude_*` command/event names are agent-agnostic; check the metadata `session_type` (or the
  spawn's `coding_agent_type`) to know which agent, not the name.
- Do not poll for a Codex pid or infer liveness from one — between turns there is no process at all.
- Ordering: consume `session_output` by `sequence`, not arrival; stdout and stderr interleave on one
  counter.
- If the orchestrator does not replay the Codex thread id, a runtime restart silently starts a fresh
  conversation — the runtime never persists it.
- Claude session behavior (model, tools, permissions) is baked into the spawn `args`/`system_prompt`;
  there is no mid-session reconfigure command — respawn instead.

**See also / peers.** runtime — *agent-config-and-secret-materialisation* (spawn-inline files),
*coding-agent-credential-dispatch* (staged credentials), *mcp-tool-server* (per-session platform
tools); orchestrator — *session-management* (session identities, Codex thread-id persistence/replay).
