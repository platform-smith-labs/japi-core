---
type: capability
title: "In-pod MCP tool server"
tags: [mcp, tools, agent-tool-call, spawn-seed, claude, codex, loopback]
timestamp: 2026-07-09T10:42:29Z
description: "Loopback-only MCP HTTP server at 127.0.0.1:9099/mcp: per-session seeded tools/list, generic tools/call forwarded to the orchestrator over the controller WS"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/mcp_server/mod.rs
  - src/mcp_server/tools.rs
  - src/mcp_server/tool_seeds.rs
  - src/mcp_server/pending_calls.rs
  - src/constants.rs
  - src/session/claude/mcp_config.rs
  - src/session/codex/mcp_config.rs
see_also:
  - {repo: runtime, capability: "A2A peer messaging (in-pod endpoints)", descriptive: false, intent: "a2a_send is the one tool dispatched in-runtime instead of through this forwarder"}
  - {repo: runtime, capability: "Controller WebSocket link", descriptive: false, intent: "the transport every bridged tool call rides; its drop/reconnect semantics apply"}
  - {repo: orchestrator, capability: "Agent tool-call dispatch and grant enforcement", intent: "computes the per-session granted_tools spawn-seed and is the authoritative enforce-on-call gate answering agent_tool_call"}
---

# In-pod MCP tool server

**What it does.** Gives the pod's coding agents (Claude and Codex) their platform tools: one MCP
server inside the pod that advertises each session's granted tool set and forwards every tool call
to the orchestrator for execution. The runtime itself executes no tool logic (two exceptions below).

**How a peer interacts.**
- **In-pod agents** (the direct consumers) speak streamable-HTTP MCP to `http://127.0.0.1:9099/mcp`
  — loopback only, never reachable from outside the pod. Every request must carry an
  `X-PS-Session-ID` header holding the session **UUID** (the DB session UUID — *not* the
  orchestrator session name). The runtime wires this up itself: Claude sessions get a per-session
  mcp-config file (passed via `--mcp-config`) with the header baked in; Codex sessions get a
  runtime-managed `$CODEX_HOME/config.toml` whose header value is sourced from the per-spawn
  `PS_SESSION_ID` env var (Codex is spawn-per-turn, so identity is fresh each turn).
- **The orchestrator** (the backend) participates two ways: it ships each session's granted tool
  set (`granted_tools`: name + description + JSON Schema) inside the session **spawn payload**
  ("spawn-seed"), and it answers bridged calls — the runtime emits `agent_tool_call{tool, args}`
  with `originating_session_id` over the controller WS and expects a correlated
  `agent_tool_result` back on the message path.

**Observable behavior.**
- `tools/list` is served **per session** from the in-memory spawn-seed registry keyed by
  `X-PS-Session-ID`, with each seeded tool's `input_schema` passed through verbatim. **Fail-closed:**
  a missing/invalid header gets a truly **empty** tool set (no error — the session stays alive,
  just tool-blind); an unseeded session with a valid header advertises only `a2a_send`. `a2a_send`
  (platform primitive, independent of the seed) is appended to every reply that carries a valid
  session UUID — the header-less/invalid-header reply omits it too.
- `tools/call` is a **generic forwarder**: any tool name and args are sent as-is to the
  orchestrator; adding a new platform tool needs zero runtime changes. The call blocks awaiting the
  correlated `agent_tool_result` with a **30s timeout and no auto-retry** — on timeout the agent
  gets a tool error and decides whether to retry.
- Result mapping: `success` → one MCP text block holding the result JSON as text; failure → an
  MCP tool error carrying the orchestrator's error text (this is also how a grant rejection
  surfaces).

**Contract.** In (per call): tool name + JSON-object args + `X-PS-Session-ID` header (rejected on
call if missing, empty, malformed, or the nil UUID). Bridged wire pair: `agent_tool_call` out,
`agent_tool_result` in — key fields of the result: `success`, `result` (JSON), `error`. Seed entry —
key fields: `name`, `description`, `input_schema` (must be an object-at-root schema; non-object
seeds are dropped from advertisement with a warning).

**Special-cased tools.**
- `ps_transfer_files` — the only tool with a server-side transform: the runtime reads each
  `local_path` from the pod's disk and attaches `content` before forwarding (the orchestrator
  cannot read the pod filesystem). Whole-batch failure on any bad file — no partial shipping.
  `remote_path` must be relative, no `..`, no NUL; `destination` must be `"orchestrator"`.
- `a2a_send` — dispatched **in-runtime** on its own command and correlation map, not through this
  forwarder (see the A2A capability).

**Invariants.** Advertisement is cosmetic — the authoritative grant gate is the orchestrator's
enforce-on-**call**; the runtime never checks grants. No runtime→orchestrator HTTP egress: the seed
rides the existing authenticated WS spawn path. Seeds are in-memory only — a re-seed replaces the
prior set, and a runtime restart starts empty (and since the runtime is PID 1, a restart takes the
pod's sessions with it — there is no surviving session to re-query).

**Failure modes.** Controller WS down → immediate tool error ("WS not connected"), the call is
never queued. No reply within 30s → "orchestrator timed out" tool error; a late result is dropped
with a warning. Orchestrator rejection (`success:false`) → tool error with the orchestrator's
message.

**Gotchas.**
- Two session identifiers coexist: this server keys everything on the session **UUID**
  (`X-PS-Session-ID`); session output/input elsewhere uses the session **name**. Never conflate.
- An empty `tools/list` is not an error condition — it is the fail-closed answer for a
  header-less/invalid-header request. An unseeded session with a valid header still sees `a2a_send`,
  so a truly empty list implies a header problem, not an empty grant.
- A tool visible in `tools/list` can still be rejected at call time (enforce-on-call); peers must
  not treat advertisement as authorization.
- Files sent via `ps_transfer_files` are read from disk **at call time**, unrestricted within the
  single-tenant pod.

**See also / peers.** orchestrator — *agent tool-call dispatch and grant enforcement* (name
descriptive; owns the granted-tools computation, tool execution, and the `agent_tool_result`
reply). runtime — *A2A peer messaging (in-pod endpoints)* for `a2a_send`; *Controller WebSocket
link* for the transport's drop-on-saturation and reconnect behavior.
