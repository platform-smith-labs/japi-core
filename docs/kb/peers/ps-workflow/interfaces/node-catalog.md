---
type: interface
title: "Custom node catalog (Conductor task types)"
tags: [conductor, nodes, task-types, runtime, session, agent, approval]
timestamp: 2026-07-07T06:49:45Z
description: "The custom capability-worker task types a peer references by name inside a Conductor workflow definition, and their call-ordering seam"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/host.go
  - internal/workers/run_agent_session.go
  - internal/workers/nodes/runtime_start.go
  - internal/workers/nodes/runtime_stop.go
  - internal/workers/nodes/reads.go
  - internal/workers/nodes/session_start.go
  - internal/workers/nodes/session_stop.go
  - internal/workers/nodes/sendprompt.go
  - internal/workers/nodes/approval.go
  - internal/workers/nodes/notify.go
  - internal/workers/nodes/collect_result.go
provides_interfaces:
  - {name: "run-agent-session", kind: conductor-task-type, intent: "the agent node — provision+run a coding session; PARKS until close"}
  - {name: "runtime-start", kind: conductor-task-type, intent: "provision a runtime; PARKS until READY"}
  - {name: "runtime-status", kind: conductor-task-type, intent: "read runtime status; sync, no park"}
  - {name: "runtime-stop", kind: conductor-task-type, intent: "release a runtime (GATED / may be NOT_LIVE); no park"}
  - {name: "session-start", kind: conductor-task-type, intent: "create a coding-agent session; PARKS until started"}
  - {name: "session-stop", kind: conductor-task-type, intent: "request graceful close, bounded poll; no park"}
  - {name: "session-status", kind: conductor-task-type, intent: "read session state; sync, no park"}
  - {name: "session-get-last-message", kind: conductor-task-type, intent: "read the last agent message; sync, no park"}
  - {name: "session-send-prompt", kind: conductor-task-type, intent: "send a prompt; PARKS until reply when wait=true (default)"}
  - {name: "request-approval", kind: conductor-task-type, intent: "human gate; PARKS until the approvals endpoint decides"}
  - {name: "send-notification", kind: conductor-task-type, intent: "write an in-app notification; sync, no park"}
  - {name: "collect-result", kind: conductor-task-type, intent: "harvest a session's result artifacts (GATED / may be NOT_LIVE); no park"}
see_also:
  - {repo: ps-workflow, capability: "Runtime lifecycle nodes", intent: "runtime/session lifecycle node behavior detail"}
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "the agent session node behavior"}
  - {repo: ps-workflow, capability: "Human approval gate", intent: "request-approval + the decision endpoint"}
  - {repo: ps-workflow, capability: "collect-result node", intent: "result-artifact harvest gated node"}
  - {repo: orchestrator, capability: "Runtime and session lifecycle", intent: "owns the mutations these nodes call and the status the reads observe", descriptive: true}
---

# Custom node catalog (Conductor task types)

**What it is.** The platform-specific capability workers ps-workflow hosts. A peer does **not**
call these over HTTP — it references a task type **by name** as a `SIMPLE` step inside a workflow
definition's Conductor JSON, and supplies per-step inputs under `inputParameters._ps`. Each task
type is a single global queue; tenancy comes from `_ps` (company/user/scope), never the task name.
The agent node is always `run-agent-session` — never Conductor's built-in LLM node. Git / PR /
commit / test are **not** nodes; the coding agent performs them inside its session (put them in the
prompt).

## Task types (does it PARK?)

- **run-agent-session** — the agent node: provisions/runs a coding-agent session and **PARKS**
  until the session closes; completed out-of-band by the task-completions bridge.
- **runtime-start** — provisions a runtime and **PARKS** until the launch reaches READY (a durable
  poller drives completion). Emits the runtime identity.
- **runtime-status** — reads runtime status DB-direct; **sync, no park**. Unknown/cross-tenant →
  clean `found=false` result, never a failure.
- **runtime-stop** — the trailing always-cleanup; idempotently releases a runtime. **No park.**
  **GATED**: returns NOT_LIVE until the orchestrator stop route is deployed on the stack.
- **session-start** — creates a coding-agent session and **PARKS** until it reaches `started`.
- **session-stop** — requests a graceful close then does a **bounded synchronous poll** to report
  the terminal state; **does not park** (never hangs — completes with a close-requested report on
  the bound).
- **session-status** — reads session state DB-direct; **sync, no park**.
- **session-get-last-message** — reads the session's last agent message DB-direct; **sync, no
  park**.
- **session-send-prompt** — sends a prompt to the session. With `_ps.wait=true` (**default**) it
  **PARKS** until the agent's reply, then completes with the result; with `wait=false` it sends and
  completes immediately.
- **request-approval** — a durable human gate: records a pending approval and **PARKS**; completion
  is push-only via `POST /api/v1/workflow-approvals`.
- **send-notification** — writes a self-contained in-app notification row; **sync, no park**.
- **collect-result** — harvests the session's `scope=session, kind=result` artifacts for a
  downstream branch; **no park**. **GATED**: returns NOT_LIVE until the orchestrator
  session-artifact read route is deployed on the stack.

## Integration seam (call ordering + identity hand-off)

A typical agent workflow chains these with a required ordering — each step consumes an identifier
the previous one emits:

**runtime-start → session-start → session-send-prompt → collect-result** (with `session-stop` /
`runtime-stop` as the trailing always-cleanup tail).

- **runtime-start emits** `runtime_uuid` (the stable runtime identity), `runtime_name`,
  `controller_name` in its output.
- **session-start consumes** `_ps.controller_name`, `_ps.runtime_name`, and `_ps.session_id` (the
  session **name**, caller-supplied and known before create).
- **session-send-prompt / session-status / session-get-last-message / session-stop /
  collect-result consume** `_ps.session_id` (the same session **name**). send-prompt also needs
  `_ps.runtime_name`.
- **runtime-stop consumes** `_ps.runtime_uuid` (the identity runtime-start emitted).

Two name-vs-value traps a workflow author must honor — see the `node-identifier-names-vs-values`
gotcha: `_ps.session_id` carries a session **name** (not a UUID), and `runtime_uuid` is the stable
runtime identity that the platform seam resolves to the newest runtime *instance* for mutations.

## Behavior a peer relies on

- **PARK vs terminal.** A parked node holds the Conductor task IN_PROGRESS; completion arrives
  later (a poller, the reconciliation sweep's re-arm, or an external endpoint). Park-style task
  defs register with `retryCount:0` — see the `park-nodes-retrycount-zero` gotcha.
- **Mutating nodes require `_ps.user_uuid`.** Any node that calls the orchestrator (runtime-start,
  session-start, session-send-prompt, session-stop, runtime-stop) must carry the originating
  `_ps.user_uuid` or the orchestrator rejects it 401. DB-direct read nodes do not.
- **NOT_LIVE is honest, not success.** A gated node (runtime-stop, collect-result) reports a
  distinct Conductor FAILED carrying a `not_live` marker — never a false COMPLETED — until its
  cross-repo route is deployed. Authors can treat a trailing cleanup's NOT_LIVE as ignorable.

**See also.** Behavior detail lives in the **ps-workflow — Runtime lifecycle nodes**,
**Coding-agent session nodes**, **Human approval gate**, and **collect-result node** capabilities.
The mutations these nodes drive (and the status their reads observe) are owned by **orchestrator**.
