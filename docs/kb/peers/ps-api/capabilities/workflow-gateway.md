---
type: capability
title: "Workflow gateway"
tags: [workflow, gateway, proxy, ps-workflow, approvals, executions]
timestamp: 2026-07-09T10:35:01Z
description: "How a peer drives workflow definitions, executions, approvals, and the inbox through ps-api's verbatim proxy to ps-workflow"
repo: ps-api
commit_sha: a4683c0
evidence:
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/workflow_approvals.go
  - cmd/handlers/workflow_inbox.go
  - cmd/handlers/workflow_task_catalog.go
  - cmd/services/services.go
  - pkg/config/config.go
  - cmd/db/project_operations.go
  - cmd/db/workspace_operations.go
see_also:
  - {repo: ps-workflow, capability: "Workflow definitions and executions engine", intent: "owns all workflow logic, validation, run state, and the terminal execution states ps-api fronts", descriptive: true}
  - {repo: ps-api, capability: "Secrets and integration credentials", intent: "the secret-refs a workflow definition attaches resolve against the same credential surface", descriptive: false}
---

# Workflow gateway

**What it does.** Fronts the workflow engine (**ps-workflow**) for authenticated frontend/API
clients: author workflow definitions (company/workspace/project scoped), start and inspect workflow
executions, submit human approvals and task completions, read the per-scope approvals+notifications
inbox, and fetch the static task catalog. ps-api is a **pure gateway** here — it authenticates,
resolves project→workspace where a route needs it, mints trusted identity headers, and forwards the
request **verbatim** to ps-workflow. It owns no workflow logic, does no path rewriting, and does not
validate or transform request/response bodies.

**How a peer interacts.** All routes require a user JWT and are company-scoped. The public ps-api
path equals the ps-workflow path 1:1 under `/api/v1`.

- **Definitions (list/create per scope):**
  `POST`/`GET /api/v1/company/workflow-definitions`,
  `POST`/`GET /api/v1/workspaces/{workspace_uuid}/workflow-definitions`,
  `POST`/`GET /api/v1/projects/{project_uuid}/workflow-definitions` (project list includes inherited).
- **Definitions (item):** `GET`/`PUT`/`DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}`
  (DELETE archives, returns 204), plus `POST …/{uuid}/clone` (copy to an override) and
  `POST …/{uuid}/publish` (draft → live; takes no body).
- **Definition secret-refs:** `POST`/`GET …/{uuid}/secret-refs`,
  `GET …/{uuid}/secret-refs/status` (resolution status, no values decrypted),
  `DELETE …/{uuid}/secret-refs/{secret_ref_uuid}` (204).
- **Executions:** `POST /api/v1/workflow-executions` (start),
  `POST /api/v1/workspaces/{workspace_uuid}/workflow-executions` (workspace-context start, see below),
  `GET /api/v1/workflow-executions` (run history, company-scoped, filterable),
  `GET /api/v1/workflow-executions/{execution_id}` (single run status/output).
- **Approvals / completions:** `POST /api/v1/workflow-approvals` (decide a parked approval),
  `POST /api/v1/task-completions` (report a task done).
- **Inbox:** `GET /api/v1/workspaces/{workspace_uuid}/workflow-inbox` (rolls up its projects),
  `GET /api/v1/projects/{project_uuid}/workflow-inbox`. Each returns `{ approvals, notifications }`.
  There is **no company-level inbox route**.
- **Catalog:** `GET /api/v1/workflow-task-catalog` — the deployment-static per-task schema catalog;
  an authenticated passthrough with no tenant filtering.

**Observable behavior.** Every route is a verbatim relay: ps-workflow's status and body are returned
byte-for-byte, including non-2xx error bodies. ps-api adds no envelope of its own. Scope always
travels via trusted headers, never the request body.

**Observable behavior — async readiness.** Starting an execution is asynchronous: `POST` returns
ps-workflow's start response (which carries the run's `execution_id`); the run then progresses on
ps-workflow. Poll `GET /api/v1/workflow-executions/{execution_id}` for the run's status/output. The
concrete status field name and its terminal values are owned by ps-workflow and are `UNKNOWN` from
this repo — read them from ps-workflow's execution capability.

**Contract.** Request/response bodies are forwarded unmodified; ps-api neither validates nor documents
them — ask ps-workflow for the body shapes. `execution_id` is a **Conductor workflow id (an opaque
string, not necessarily a UUID)** — forward it as given; do not assume UUID form.
`GET /api/v1/workflow-executions` accepts (key fields) `workspace_uuid` | `project_uuid` (mutually
exclusive), `execution_context=workspace|project`, `status`, `limit` (≤200, default 50), `offset` —
forwarded verbatim for ps-workflow to apply. Path params ps-api itself validates as UUIDs:
`workspace_uuid`, `project_uuid`, `workflow_definition_uuid`, `secret_ref_uuid`.

**Invariants — enforcement locus.**
- **Authentication (JWT + DB company-membership) is enforced here**, at ps-api, on every route.
- **Project→workspace resolution is done here** for project-scoped definition/inbox routes; the
  resolved workspace is minted as the trusted `X-Workspace-UUID`.
- **Workspace ownership is validated here** for the workspace-context start route: ps-api confirms the
  workspace belongs to the caller's company (returns 404 if not) before minting `X-Workspace-UUID` —
  an anti-spoof guard, since the start body carries no scope.
- **All workflow business logic, body validation, run-state management, and tenant-filtering of
  results are delegated to ps-workflow.** The execution-history list is company-scoped because
  ps-workflow forces `correlationId=company` server-side from the trusted `X-Company-UUID` header, not
  because ps-api filters.

**Failure modes.** ps-workflow rejections (4xx/5xx) pass through verbatim with the upstream body
intact — a peer parsing an error receives ps-workflow's error shape, not ps-api's. ps-workflow
unreachable/timeout → **503/504** from the gateway.

**Gotchas.**
- **Client `Idempotency-Key` is dropped.** ps-workflow's start endpoint honours an optional
  `Idempotency-Key` request header for replay-dedup, but ps-api's transport forwards only trusted
  gateway headers (`X-User-UUID`/`X-Company-UUID`/`X-Request-ID`), **not** arbitrary client headers —
  so a client-supplied `Idempotency-Key` never reaches ps-workflow. With the header absent, ps-workflow
  starts a **new run on every call**; a peer must not rely on idempotent-retry dedup through this proxy.
- **Workspace-context runs need the workspace-scoped start route.** A company-level run that should be
  attributed to a workspace must POST to `…/workspaces/{workspace_uuid}/workflow-executions`, not the
  plain start route — scope rides the path (→ trusted header), and the plain route would lose the
  workspace attribution when there is no `project_uuid` in the body.
- **No company-level inbox** — only workspace and project inboxes exist.
- **`execution_id` is not a UUID.** Treat it as an opaque string when constructing the item read.

**See also / peers.** All workflow semantics — definition shapes, execution lifecycle and terminal
states, approval/inbox payloads — belong to **ps-workflow**; this gateway only relays them (see the
`see_also` entry). The secret-refs a definition attaches reference the same credential surface as the
**Secrets and integration credentials** capability in this repo.
