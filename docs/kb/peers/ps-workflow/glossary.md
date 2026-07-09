---
type: glossary
title: "ps-workflow — glossary"
tags: [glossary, workflow, conductor, terminology]
timestamp: 2026-07-09T10:49:10Z
description: "Domain terms a peer needs to reason about ps-workflow"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/tenant/proxy.go
  - internal/platform/platform.go
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/session_events.go
  - internal/workers/nodes/sendprompt.go
  - internal/signal/pg_store.go
  - internal/scheduler/scheduler.go
  - internal/runcontext/store.go
  - internal/taskcatalog/catalog.go
  - cmd/handlers/signals.go
  - cmd/handlers/webhooks.go
  - cmd/handlers/workflow_task_catalog.go
  - docs/dev/decisions/park-style-taskdefs-need-retrycount-zero.md
  - docs/dev/decisions/correlation-store-keys-on-company-id.md
---

# ps-workflow — glossary

**Workflow definition** — a canonical, scope-owned (`system`/`company`/`workspace`/`project`) record
of a workflow, authored as Conductor JSON plus PS annotations (`inputParameters._ps`); publishing it
registers derived, tenant-namespaced JSON into Conductor. Mirrors `agent_definition`; overrides are
clone-to-override (no merging).

**Workflow execution** — one running instance of a published definition, started by definition UUID
and tagged with the tenant. A peer starts it and polls its status by execution id.

**Capability worker node / task type** — a custom Platform Smith worker that backs a workflow step
(e.g. `run-agent-session`, runtime/session lifecycle, approval, notification), registered as a
Conductor task type. Distinct from Conductor's built-in task types, which ps-workflow does not use for
the agent step.

**Park (IN_PROGRESS parked task)** — a node that does not finish synchronously: it holds its Conductor
task IN_PROGRESS and is completed later out-of-band (via the async bridge or a poller). Park-style
task defs register with `retryCount: 0` so a failed/optional park falls through to cleanup instead of
re-parking. Examples: await-signal, request-approval, session-send-prompt(wait=true), session-prompt,
runtime-start, session-start.

**The tenant seam** — the single layer every tenant-scoped Conductor call passes through; it applies
name-namespacing, execution tagging, and tenant-checked status. The only handler-reachable path to the
engine.

**Correlation store** — the durable store mapping a runtime session to the Conductor task parked for
it, keyed on `company_id` + session name. Lets a later session event complete the right parked task;
survives restarts and is re-armed by a startup reconciliation sweep.

**run-agent-session** — the custom worker node that runs a coding-agent (Claude/Codex) session for a
workflow step; always this custom worker, never Conductor's LLM node. The agent does git/PR/commit/
test itself inside the session (via the prompt) — those are not separate nodes.

**Model-B signal / await-signal** — the generic wait-for-external-signal pattern: an `await-signal`
node parks keyed on a `correlation_id`, and `POST /api/v1/signals` later delivers a matching signal to
unpark it (subsumes human approval as a special case). Env-gated NOT_LIVE until its cross-repo seams
ship.

**Signal correlation store** — the durable **(company, correlation_id)** store an `await-signal` park
lives in; the signals endpoint unparks against it. Reuses the **same exactly-once completion core** as
the async bridge, so unpark is idempotent and tenant-scoped.

**Task catalog** — a **deployment-static** descriptor of the available node types and their
live/NOT_LIVE availability on this stack, served at `GET /api/v1/workflow-task-catalog` for the ps-ui
visual builder. Built once at startup from the same env gates the workers use.

**Webhook trigger** — a stored, public **token-authed** ingress (`POST /api/v1/webhooks/{uuid}`) that
starts a workflow execution. **System-originated**: no user identity — the tenant comes from the stored
trigger row. The route 404s unless webhook triggers are enabled on the stack.

**Schedule / scheduler** — a cron **scheduler** (`SCHEDULER_ENABLED`) that fires due workflows from the
schedule store **exactly-once** via a DB compare-and-swap claim. **System-originated** like a webhook —
tenant from the schedule row, no user identity.

**Run-context sidecar** — a per-execution `workflow_run_context` row written at start-execution and
read back to enrich the list/get execution responses and the workflow-inbox roll-up.

**FORK_JOIN_DYNAMIC fan-out** — Conductor's dynamic fan-out (one branch per input item), fed by the
`resolve-projects` node which expands a workspace into its project array.

**System task** — an engine-executed built-in Conductor task (HTTP / WAIT / INLINE / JSON_JQ_TRANSFORM
/ SET_VARIABLE) rather than a custom PS worker. Tenant-blind and carries **no `_ps` context field**.

**Secret ref** — a `workflow_definition_secret_ref`: a named binding from a workflow definition to an
entry in the platform secret store, resolved at the execution scope when a workflow starts (a required
unresolved secret aborts the start).
