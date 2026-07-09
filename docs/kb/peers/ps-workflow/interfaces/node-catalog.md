---
type: interface
title: "Custom node catalog (Conductor task types)"
tags: [conductor, nodes, task-types, runtime, session, agent, approval, signal, a2a, git, llm]
timestamp: 2026-07-09T10:49:10Z
description: "The custom capability-worker task types a peer references by name inside a Conductor workflow definition, and their call-ordering seam"
repo: ps-workflow
commit_sha: b1f4682
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
  - internal/workers/nodes/resolve_projects.go
  - internal/workers/nodes/git_open_pr.go
  - internal/workers/nodes/llm.go
  - internal/workers/nodes/run_command.go
  - internal/workers/nodes/session_prompt.go
  - internal/workers/nodes/a2a.go
  - internal/workers/nodes/await_signal.go
provides_interfaces:
  - {name: "run-agent-session", kind: conductor-task-type, intent: "the agent node — provision+run a coding session; PARKS until close (STILL A STUB)"}
  - {name: "runtime-start", kind: conductor-task-type, intent: "provision a runtime; PARKS until READY"}
  - {name: "runtime-status", kind: conductor-task-type, intent: "read runtime status; sync, no park"}
  - {name: "runtime-stop", kind: conductor-task-type, intent: "release a runtime (GATED / may be NOT_LIVE); no park"}
  - {name: "session-start", kind: conductor-task-type, intent: "create a coding-agent session; PARKS until started"}
  - {name: "session-stop", kind: conductor-task-type, intent: "request graceful close, bounded poll; no park"}
  - {name: "session-status", kind: conductor-task-type, intent: "read session state; sync, no park"}
  - {name: "session-get-last-message", kind: conductor-task-type, intent: "read the last agent message; sync, no park"}
  - {name: "session-send-prompt", kind: conductor-task-type, intent: "send a prompt; PARKS until reply when wait=true (default)"}
  - {name: "session-prompt", kind: conductor-task-type, intent: "unified provision-aware agent turn (runtime+session+turn); PARKS; GATED SESSION_PROMPT_LIVE"}
  - {name: "resolve-projects", kind: conductor-task-type, intent: "workspace → project array for a FORK_JOIN_DYNAMIC fan-out; sync, no park; LIVE"}
  - {name: "run-command", kind: conductor-task-type, intent: "deterministic in-runtime shell exec (exit_code/stdout/stderr); sync, no park; runner-gated"}
  - {name: "git-open-pr", kind: conductor-task-type, intent: "the ONE privileged git node — open a PR via org GitHub App; sync; GATED GIT_OPEN_PR_LIVE"}
  - {name: "llm", kind: conductor-task-type, intent: "runtime-less Anthropic Messages call for summarize/structure; sync; GATED LLM_NODE_LIVE"}
  - {name: "a2a", kind: conductor-task-type, intent: "agent-to-agent messaging (send/broadcast/start-conversation); sync; NOT_LIVE if client nil"}
  - {name: "await-signal", kind: conductor-task-type, intent: "Model-B generic wait-for-signal (subsumes request-approval); PARKS; GATED AWAIT_SIGNAL_LIVE"}
  - {name: "request-approval", kind: conductor-task-type, intent: "human gate; PARKS until the approvals endpoint decides"}
  - {name: "send-notification", kind: conductor-task-type, intent: "write an in-app notification; sync, no park"}
  - {name: "collect-result", kind: conductor-task-type, intent: "harvest a session's result artifacts (GATED / may be NOT_LIVE); no park"}
see_also:
  - {repo: ps-workflow, capability: "Runtime lifecycle nodes", intent: "runtime/session lifecycle node behavior detail"}
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "the agent session node behavior"}
  - {repo: ps-workflow, capability: "Human approval gate", intent: "request-approval + the decision endpoint"}
  - {repo: ps-workflow, capability: "collect-result node", intent: "result-artifact harvest gated node"}
  - {repo: ps-workflow, capability: "Resolve-projects fan-out node", intent: "workspace→projects expansion for FORK_JOIN_DYNAMIC"}
  - {repo: ps-workflow, capability: "Git open-PR node", intent: "the one privileged git node + GitHub App seam"}
  - {repo: ps-workflow, capability: "LLM node (runtime-less)", intent: "runtime-less structured LLM call"}
  - {repo: ps-workflow, capability: "Run-command node", intent: "deterministic in-runtime exec node"}
  - {repo: ps-workflow, capability: "A2A messaging node", intent: "agent-to-agent send/broadcast node"}
  - {repo: ps-workflow, capability: "Signal wait & unpark (Model-B)", intent: "await-signal park + the signals endpoint"}
  - {repo: orchestrator, capability: "Runtime and session lifecycle", intent: "owns the mutations these nodes call and the status the reads observe", descriptive: true}
---

# Custom node catalog (Conductor task types)

**What it is.** The platform-specific capability workers ps-workflow hosts. A peer does **not**
call these over HTTP — it references a task type **by name** as a `SIMPLE` step inside a workflow
definition's Conductor JSON, and supplies per-step inputs under `inputParameters._ps`. Each task
type is a single global queue; tenancy comes from `_ps` (company/user/scope), never the task name.
The agent node is always a custom worker — never Conductor's built-in LLM node. Git commit / push /
test are **not** nodes; the coding agent performs them inside its session (put them in the prompt).
The one exception is PR creation, which is a node (`git-open-pr`) because its scope lives in the org
GitHub App, off the per-session token.

## Task types (does it PARK? is it env-GATED?)

- **run-agent-session** — the original agent node: PARKS until the session closes. **Still a STUB** —
  it mints a session name, records the correlation, and parks, but does **not** yet provision a
  runtime or launch a real session; completed out-of-band by the task-completions bridge.
- **runtime-start** — provisions a runtime and **PARKS** until the launch reaches READY. Emits the
  runtime identity.
- **runtime-status** — reads runtime status DB-direct; **sync, no park**. Unknown/cross-tenant →
  clean `found=false`, never a failure.
- **runtime-stop** — trailing always-cleanup; idempotently releases a runtime. **No park. GATED** —
  NOT_LIVE until the orchestrator stop route is deployed.
- **session-start** — creates a coding-agent session and **PARKS** until it reaches `started`.
- **session-stop** — requests graceful close then a **bounded synchronous poll**; **no park** (never
  hangs — completes with a close-requested report on the bound).
- **session-status** — reads session state DB-direct; **sync, no park**.
- **session-get-last-message** — reads the last agent message DB-direct; **sync, no park**.
- **session-send-prompt** — sends a prompt. With `_ps.wait=true` (**default**) it **PARKS** until the
  reply; with `wait=false` it sends and completes immediately.
- **session-prompt** — the unified **provision-aware** agent turn: composes runtime-start +
  session-start + the turn (with `response_schema` + `wait_for`) in one node. **PARKS. GATED**
  `SESSION_PROMPT_LIVE` (runner must also be wired). Transitional replacement for
  session-send-prompt; **dual-runs** with the atomic lifecycle nodes during deprecation.
- **resolve-projects** — expands a workspace into its project array for a `FORK_JOIN_DYNAMIC` fan-out
  (the flagship "run an agent across every repo in a workspace"). **Sync, no park; LIVE** (no gate).
  Empty/unknown workspace → empty array, not a failure. `_ps.only_with_repo=true` restricts to
  git-bound projects.
- **run-command** — deterministic in-runtime shell exec, returning `exit_code`/`stdout`/`stderr` for a
  downstream `SWITCH` (distinct from the agentic prompt nodes). **Sync, no park; runner-gated** —
  NOT_LIVE until the orchestrator exec path is wired (no separate env flag).
- **git-open-pr** — the **ONE privileged git node**: opens a pull request via the **org-installed
  GitHub App** (create-only scope), never the per-session git token. **Sync. GATED** `GIT_OPEN_PR_LIVE`
  (opener must also be wired). A repo whose App is not installed is a **non-fatal skip** (`created=false`
  + `skipped_reason`), not a failure, so a trailing notify can report it.
- **llm** — a **runtime-less** Anthropic Messages call over workflow data for summarize/structure; with
  `response_schema` it exposes the validated reply at `.output.response.*` to feed a `SWITCH`. **Sync.
  GATED** `LLM_NODE_LIVE` (caller must also be wired — nil ⇒ NOT_LIVE even with the flag on).
- **a2a** — agent-to-agent messaging via the orchestrator conversation surface. `_ps.action` selects
  `send` / `broadcast` / `start-conversation` (only `send` is dependably available today — the peer
  surface for the others may be missing). **Sync**; `broadcast` pairs with `await-signal` to collect
  replies. **NOT_LIVE if the client is nil.**
- **await-signal** — the **Model-B** generic wait-for-signal: mints a correlation id, records an OPEN
  wait, and **PARKS** until a `(company, correlation_id)` signal arrives via
  `POST /api/v1/signals`. **Subsumes request-approval** (`shape=approval`). **GATED** `AWAIT_SIGNAL_LIVE`.
- **request-approval** — durable human gate: records a pending approval and **PARKS**; completion is
  push-only via `POST /api/v1/workflow-approvals`.
- **send-notification** — writes a self-contained in-app notification row; **sync, no park**.
- **collect-result** — harvests the session's `scope=session, kind=result` artifacts for a downstream
  branch; **no park. GATED** — NOT_LIVE until the orchestrator session-artifact read route is deployed.

## Integration seam (call ordering + identity hand-off)

A typical agent workflow chains these with a required ordering — each step consumes an identifier the
previous one emits:

**runtime-start → session-start → session-send-prompt → collect-result** (with `session-stop` /
`runtime-stop` as the trailing always-cleanup tail). `session-prompt` collapses the first three into
one provision-aware node.

- **runtime-start emits** `runtime_uuid` (the stable runtime identity), `runtime_name`,
  `controller_name`.
- **session-start consumes** `_ps.controller_name`, `_ps.runtime_name`, and `_ps.session_id` (the
  session **name**, caller-supplied and known before create).
- **session-send-prompt / session-status / session-get-last-message / session-stop / collect-result
  consume** `_ps.session_id` (the same session **name**). send-prompt also needs `_ps.runtime_name`.
- **run-command consumes** `_ps.runtime_name` — it runs inside an already-provisioned runtime, so it
  sits **after** runtime-start (or a session provision) in a chain.
- **runtime-stop consumes** `_ps.runtime_uuid` (the identity runtime-start emitted).

**Fan-out ordering.** `resolve-projects` runs **before** a `FORK_JOIN_DYNAMIC` that fans out agent
nodes: it emits `output.projects[]` (each `{project, name, repo, branch}`), which the dynamic-fork
iterates so each fork instance runs a per-project agent chain.

**git-open-pr inputs.** Consumes `_ps.repo` + `_ps.head` (the branch the agent pushed in-session) +
`_ps.title` (all required), plus optional `_ps.base` (default `main`) and `_ps.body`. It sits at the
tail of a per-repo agent chain, after the agent has pushed its branch.

Two name-vs-value traps a workflow author must honor — see the `node-identifier-names-vs-values`
gotcha: `_ps.session_id` carries a session **name** (not a UUID), and `runtime_uuid` is the stable
runtime identity that the platform seam resolves to the newest runtime *instance* for mutations.

## Behavior a peer relies on

- **PARK vs terminal.** A parked node holds the Conductor task IN_PROGRESS; completion arrives later
  (a poller, the reconciliation sweep's re-arm, or an external endpoint such as
  `POST /api/v1/signals` or `POST /api/v1/workflow-approvals`). Park-style task defs register with
  `retryCount:0` — see the `park-nodes-retrycount-zero` gotcha.
- **Mutating nodes require `_ps.user_uuid`.** Any node that calls the orchestrator (runtime-start,
  session-start, session-send-prompt, session-prompt, session-stop, runtime-stop, run-command,
  git-open-pr) must carry the originating `_ps.user_uuid` or the orchestrator rejects it 401.
  DB-direct read nodes and resolve-projects do not.
- **NOT_LIVE is honest, not success.** A gated node (runtime-stop, collect-result, session-prompt,
  run-command, git-open-pr, llm, a2a, await-signal) reports a distinct Conductor FAILED carrying a
  `not_live` marker — never a false COMPLETED — until its dependency is wired on the stack. A trailing
  cleanup's NOT_LIVE is ignorable; a mid-chain node's is not.

**See also.** Behavior detail lives in the ps-workflow **Runtime lifecycle nodes**, **Coding-agent
session nodes**, **Human approval gate**, **collect-result node**, **Resolve-projects fan-out node**,
**Git open-PR node**, **LLM node (runtime-less)**, **Run-command node**, **A2A messaging node**, and
**Signal wait & unpark (Model-B)** capabilities. The mutations these nodes drive (and the status their
reads observe) are owned by **orchestrator**.
