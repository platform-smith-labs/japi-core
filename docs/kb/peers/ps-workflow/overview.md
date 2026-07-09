---
type: overview
title: "ps-workflow — L2 Workflow Service"
tags: [workflow, conductor, multi-tenancy, orchestration, l2-service]
timestamp: 2026-07-09T10:49:10Z
description: "What ps-workflow is: the tenant-aware Go service (port 9005) in front of Conductor OSS and the only thing that talks to it"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - README.md
  - CLAUDE.md
  - cmd/server/main.go
  - cmd/services/services.go
  - internal/tenant/proxy.go
  - internal/platform/platform.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/session_events.go
  - cmd/handlers/signals.go
  - cmd/handlers/webhooks.go
  - cmd/handlers/workflow_task_catalog.go
  - internal/scheduler/scheduler.go
  - internal/taskcatalog/catalog.go
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
  surface is start (`POST /api/v1/workflow-executions`, with an optional `Idempotency-Key` header),
  status (`GET /api/v1/workflow-executions/{execution_id}`), and a run-context-enriched **list**
  (`GET /api/v1/workflow-executions`). Pause / resume / terminate / search are design intent in the
  README but are **not implemented** — do not assume they exist.
- **Multi-tenancy enforcement.** Conductor is tenant-blind; ps-workflow enforces isolation on every
  engine call (name-namespacing, execution tagging, tenant-checked status). See context.md.
- **Custom capability worker nodes.** The platform-specific node catalog: runtime/session lifecycle,
  the agent session (`run-agent-session` / `session-prompt`, a provision-aware unified turn), human
  approval, notification, and result collection. Expanded nodes now include **resolve-projects**
  (workspace→project fan-out feeding a dynamic branch), **git-open-pr** (the one privileged git node),
  **llm** (a runtime-less Anthropic call), **run-command** (deterministic in-runtime exec), **a2a**
  (agent-to-agent messaging), and **await-signal** (a generic wait-for-external-signal park that
  subsumes request-approval). The agent node is always the custom worker — never Conductor's built-in
  LLM node. Git / PR / commit / test remain in-session (in the prompt); only `git-open-pr` is a node.
  **Many advanced nodes are env-gated NOT_LIVE** until their cross-repo seams ship (e.g.
  `GIT_OPEN_PR_LIVE`, `LLM_NODE_LIVE`, `AWAIT_SIGNAL_LIVE`, `SESSION_PROMPT_LIVE`, `RUNTIME_STOP_LIVE`,
  `COLLECT_RESULT_LIVE`) — a workflow referencing one still starts, but the node reports NOT_LIVE.
- **Trigger surfaces (system-originated).** Beyond the peer-driven start API, executions can begin
  without a user: a **webhook trigger** (`POST /api/v1/webhooks/{uuid}`, public token-authed ingress)
  and a cron **scheduler** (`SCHEDULER_ENABLED`) that fires due workflows exactly-once. Both start
  system-originated executions carrying no user identity — the tenant comes from the stored trigger /
  schedule row. See context.md.
- **Visual-builder task catalog + inbox.** A deployment-static node catalog
  (`GET /api/v1/workflow-task-catalog`) describes the available node types and their live/NOT_LIVE
  availability for the ps-ui visual builder, and a **workflow-inbox** endpoint rolls up pending
  approvals and notifications for a user.
- **Async session→task bridge + signals.** Long-running park-style tasks (a running agent turn, a
  pending approval/signal, a runtime/session launch) hold their Conductor task IN_PROGRESS and are
  completed later, out-of-band — by a runtime session's completion event, or by `POST /api/v1/signals`
  unparking an `await-signal` node.

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
