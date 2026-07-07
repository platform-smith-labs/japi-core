---
type: overview
title: "ps-workflow — L2 Workflow Service"
tags: [workflow, conductor, multi-tenancy, orchestration, l2-service]
timestamp: 2026-07-07T06:49:45Z
description: "What ps-workflow is: the tenant-aware Go service (port 9005) in front of Conductor OSS and the only thing that talks to it"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - README.md
  - CLAUDE.md
  - cmd/server/main.go
  - internal/tenant/proxy.go
  - internal/platform/platform.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/session_events.go
---

# ps-workflow — L2 Workflow Service

**What it is.** ps-workflow is the Platform Smith **L2 Workflow Service** (Go, port **9005**): the
tenant-aware layer that sits **in front of Conductor OSS** and is the *only* component that talks to
it. Conductor is the durable workflow engine; ps-workflow hides it entirely — it is never exposed to
peers, and everything Platform Smith-specific (multi-tenancy, definition registry, custom node
workers, the async bridge) lives here.

**What it owns.**

- **Workflow-definition registry.** Canonical `workflow_definition` records live in the shared
  platform DB, scoped like `agent_definition` (system / company / workspace / project). Publishing a
  definition registers derived, tenant-namespaced Conductor JSON into the engine as a cache. Authoring
  is Conductor JSON + PS annotations (`inputParameters._ps`) — there is no PS IR or translator.
- **Execution API.** A peer starts an execution by definition UUID and reads its status. The live
  surface is start (`POST /api/v1/workflow-executions`, with an optional `Idempotency-Key` header) and
  status (`GET /api/v1/workflow-executions/{execution_id}`). Pause / resume / terminate / search are
  design intent in the README but are **not implemented** — do not assume they exist.
- **Multi-tenancy enforcement.** Conductor is tenant-blind; ps-workflow enforces isolation on every
  engine call (name-namespacing, execution tagging, tenant-checked status). See context.md.
- **Custom capability worker nodes.** The platform-specific node catalog (runtime/session lifecycle,
  the agent session, human approval, notification, result collection). The agent node is always the
  custom `run-agent-session` worker — never Conductor's built-in LLM node. Git / PR / commit / test are
  **not** nodes; the coding agent performs them inside its session (they belong in the prompt).
- **Async session→task bridge.** Long-running park-style tasks (a running agent turn, a pending human
  approval, a runtime/session launch) hold their Conductor task IN_PROGRESS and are completed later,
  out-of-band, when the corresponding runtime session posts a completion event.

**Peers who interact with it.**

- **ps-api / ps-ui** — call the HTTP API to author/clone/publish workflow definitions and to start and
  read executions.
- **orchestrator** — ps-workflow calls its HTTP API for all runtime/session **mutations** (launch,
  create-session, send-input, stop); ps-workflow reads runtime/session **status DB-direct** from the
  shared platform DB and never depends on ps-api. Orchestrator also forwards session completion events
  back into the bridge endpoint.
- **runtime coding sessions** — their turn-completed / session-closed events (forwarded via
  orchestrator) complete the parked Conductor tasks that the async bridge is holding.

**What it is not.** It does not embed or re-expose Conductor, and it does not own database schema —
migrations live in the `db-migration` repo, not here.
