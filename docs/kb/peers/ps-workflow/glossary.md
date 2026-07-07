---
type: glossary
title: "ps-workflow — glossary"
tags: [glossary, workflow, conductor, terminology]
timestamp: 2026-07-07T06:49:45Z
description: "Domain terms a peer needs to reason about ps-workflow"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/tenant/proxy.go
  - internal/platform/platform.go
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/session_events.go
  - internal/workers/nodes/sendprompt.go
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
re-parking. Examples: request-approval, session-send-prompt(wait=true), runtime-start, session-start.

**The tenant seam** — the single layer every tenant-scoped Conductor call passes through; it applies
name-namespacing, execution tagging, and tenant-checked status. The only handler-reachable path to the
engine.

**Correlation store** — the durable store mapping a runtime session to the Conductor task parked for
it, keyed on `company_id` + session name. Lets a later session event complete the right parked task;
survives restarts and is re-armed by a startup reconciliation sweep.

**run-agent-session** — the custom worker node that runs a coding-agent (Claude/Codex) session for a
workflow step; always this custom worker, never Conductor's LLM node. The agent does git/PR/commit/
test itself inside the session (via the prompt) — those are not separate nodes.

**Secret ref** — a `workflow_definition_secret_ref`: a named binding from a workflow definition to an
entry in the platform secret store, resolved at the execution scope when a workflow starts (a required
unresolved secret aborts the start).
