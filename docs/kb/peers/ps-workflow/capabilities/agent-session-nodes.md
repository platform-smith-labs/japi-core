---
type: capability
title: "Coding-agent session nodes"
tags: [workflow, nodes, agent-session, conductor, coding-agent]
timestamp: 2026-07-09T10:49:10Z
description: "The custom worker nodes that start, prompt, read, and stop a coding-agent session inside a workflow"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/session_start.go
  - internal/workers/nodes/session_stop.go
  - internal/workers/nodes/sendprompt.go
  - internal/workers/nodes/session_prompt.go
  - internal/workers/nodes/session_prompt_runner.go
  - internal/workers/nodes/reads.go
  - internal/workers/host.go
  - internal/workers/run_agent_session.go
  - cmd/handlers/session_events.go
see_also:
  - {repo: ps-workflow, capability: "Async session→task completion bridge", intent: "external completion that un-parks send-prompt/agent turns and session close"}
  - {repo: ps-workflow, capability: "Runtime lifecycle nodes", intent: "provisions the runtime whose handle session-start consumes"}
---

# Coding-agent session nodes

**What it does.** A family of custom Conductor worker nodes that let a workflow drive a
coding-agent session: start it, send it prompts, read its state/last message, and stop it. The
coding agent (Claude Code / Codex) does its real work — reading code, editing, running tests,
committing, opening PRs — *inside* the session; those are NOT workflow nodes and must be requested
in the prompt text.

**Key design fact.** The agent node is ALWAYS the custom `run-agent-session` worker, never
Conductor's built-in LLM node. `run-agent-session` parks the task and is un-parked externally by
the session task-completions bridge when the session closes. (In this repo `run-agent-session` is a
stub that mints a session id and parks; runtime provisioning + launch are soft-gated.)

**How a peer interacts.** These are workflow nodes, not HTTP endpoints — a peer authors them into a
workflow definition. Each is a global Conductor task type carrying a `_ps` annotation:
- `session-start` — `_ps`: `controller_name`, `runtime_name`, `session_id` (the session NAME) all
  required; optional `initial_prompt`, `model`, `agent_definition_uuid`.
- `session-send-prompt` — the **atomic** turn node. `_ps`: `session_id` + `content` are the only
  fields validated locally (missing either FAILs the task); `runtime_name` is forwarded to the
  downstream input endpoint and required *there*, not checked here; `wait` (bool, DEFAULT true).
- `session-prompt` — the **unified, provision-aware** turn node (P3, GATED — see below). `_ps`:
  `content` OR `prompt` (required, either); `session` mode (`new|resume|ref`, DEFAULT `ref`);
  `wait_for` (`turn|signal`, DEFAULT `turn`); `response_schema` (optional). When `session=new`,
  provisioning inputs: `project_uuid` (required), `environment_uuid`, `agent_definition_uuid`,
  `branch`. Mutation/provisioning forwards `user_uuid` as with the atomic mutation nodes.
- `session-status` — `_ps.session_id` required.
- `session-get-last-message` — `_ps.session_id` required.
- `session-stop` — `_ps.session_id` required.

**Observable behavior.**
- `session-start` creates the session (sync, at `pending`) then PARKS until the session state
  reaches `started`. Completes COMPLETED with the session identity; FAILs if the session hits a
  terminal state (`failed`/`closed`/`crashed`) before starting, or if a max-wait bound elapses.
- `session-send-prompt` with `wait=true` PARKS until the agent's turn completes, then completes with
  the turn `message` + `stop_reason`. With `wait=false` it sends the prompt and returns immediately
  (`accepted=true`) — fire-and-forget, no turn result.
- `session-prompt` (when LIVE) consolidates runtime provisioning + session-start + one turn into a
  single node. With `session=new` it PARKS immediately, then a background driver runs three bounded
  phases — provision (~20m) → session reaches `started` (~5m) → agent turn result (~2h) — and
  completes the parked task through the shared completion core (the same bridge un-parks it). On
  success it returns the turn `message` + `stop_reason` PLUS the runtime identity it provisioned. It
  does NOT auto-stop the runtime it created (a trailing run-command / runtime-stop / PR node reuses
  it). When GATED off it completes NOT_LIVE (see gotcha).
- `session-status` and `session-get-last-message` are synchronous DB-direct reads: they return
  `found` plus (respectively) `state`, or the last agent `message` + `stop_reason`.
- `session-stop` requests a graceful close then polls the state, bounded by a max-wait. It NEVER
  hangs and NEVER FAILs on a slow close: if the session does not confirm `closed` within the bound
  it completes COMPLETED with `closed=false` and an explanatory note (report, not failure).

**Completion signals (for a peer polling / branching downstream).** The session `state` field is the
readiness signal. Terminal values: `started` (session ready), `closed` (graceful teardown),
`crashed`/`failed` (abnormal end). `session-start` treats `started` as success; `session-stop`
treats `closed` as a clean close (`closed=true`) but `crashed`/`failed` still mean teardown is
effectively done. A parked `session-send-prompt` completes when the async task-completions bridge
delivers an `agent_turn_completed` event (its own bounded poller is a fallback).

**Contract.** Inputs are `_ps` fields (above). Outputs (key fields):
`session-start` → `{state, session_id, session_uuid}`; `session-send-prompt` (wait) →
`{state, message, stop_reason}`, (no-wait) → `{accepted, session_id, waited:false}`;
`session-prompt` (COMPLETED) → `{state:"completed", message, stop_reason, session_id, runtime_name,
runtime_uuid}` plus `{schema_valid, response|schema_error}` when `response_schema` is set,
(FAILED) → `{state:"failed", reason}`; `session-status` → `{found, state}`;
`session-get-last-message` → `{found, message, stop_reason}`;
`session-stop` → `{session_id, state, closed, note?}`. Errors: missing required `_ps` fields FAIL
the task. Session mutations (create / send input / stop) go to the orchestrator HTTP API and forward
`user_uuid` from `_ps` as a gateway header; the node does not check its presence — authorization is
the orchestrator's, which rejects a nil/foreign user against the company. Reads go
DB-direct.

**Invariants.**
- One outstanding `wait=true` prompt per (company, session): a second workflow/task sending to the
  same session mid-turn is refused; a redelivery of the SAME parked task re-parks (never
  double-drives).
- Park-style nodes (`session-start`, `session-send-prompt` wait=true) register with `retryCount:0` —
  a Conductor retry would re-run the whole node and re-park a settled task; durability comes from the
  startup reconciliation sweep re-arming open parks, not from engine retries.
- Read/park failure to find the session for the tenant is a benign result, not a leak — reads return
  `found=false` (see below), never enumerate cross-tenant.
- `session-prompt` `session=new` is idempotent on redelivery: a redelivered parked task re-parks on
  the existing turn correlation and never re-provisions (no second runtime/session, no second driver).

**Failure modes.** `session-send-prompt` wait=true FAILs on max-wait deadline or on mid-turn session
death (`closed`/`crashed`/`failed`) with no result frame; if a result frame did land before the
session died it is accepted (COMPLETED). `session-start` FAILs on deadline or pre-start terminal
state. `session-stop`'s `closed=false` on bound is a KNOWN slow-close outcome (runtime graceful close
may not exit-0), NOT an error — do not branch it as failure.

**Gotchas.**
- `_ps.session_id` is the session **NAME** (caller-supplied, stable, known before create — it is the
  idempotency key), not a UUID. The distinct `session_uuid` is returned as output.
- `wait=false` send returns `accepted=true` = the input was delivered, NOT that the agent's turn
  finished. To get the reply, use `wait=true` (default) or read `session-get-last-message` later.
- The read nodes (`session-status`, `session-get-last-message`) treat an unknown or cross-tenant
  session as a clean COMPLETED with `found=false` (for a downstream SWITCH) — never a FAIL. Branch on
  `found`.
- `session-prompt` is the forward-looking **unified** turn node but is **GATED** (`SESSION_PROMPT_LIVE`):
  on a stack where it is off (or the provisioning runner is unwired) it completes **NOT_LIVE** without
  running — there, use the atomic `runtime-start → session-start → session-send-prompt` chain instead.
  It dual-runs with the atomic nodes during a typeVersion migration.
- `session-prompt` `session` mode names are a trap: the DEFAULT `ref` means "continue an EXISTING
  session" and, like `resume`, is **not wired** in this version — both are honest FAILs, not false
  success. Only `session=new` (provision + first turn) works today; `wait_for=signal` is likewise an
  honest FAIL (only `wait_for=turn` is wired). Continue an existing session with `session-send-prompt`.
- Because `session-prompt` can provision, its output carries the runtime identity it created
  (`runtime_name` / `runtime_uuid`) that the atomic `session-send-prompt` does NOT — downstream
  run-command / runtime-stop / PR nodes key off that.
- Git / PR / commit / test are NOT nodes. There is no "run-tests" or "open-PR" node; put those
  instructions in the prompt `content`.

**See also.** ps-workflow *Async session task-completions bridge* (un-parks send-prompt/agent turns
and session close via forwarded `agent_turn_completed` / `session_closed` events); ps-workflow
*Runtime lifecycle nodes* (provisions the runtime that `session-start` consumes via
`controller_name` + `runtime_name`).
