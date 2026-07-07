---
type: interface
title: "ps-workflow HTTP API"
tags: [http, rest, workflow-definitions, executions, approvals, task-completions]
timestamp: 2026-07-07T06:49:45Z
description: "The REST surface a peer calls to author/publish workflow definitions, start/read executions, decide approvals, and complete parked tasks"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_definitions_scoped.go
  - cmd/handlers/workflow_definition_secret_refs.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/workflow_approvals.go
  - cmd/handlers/session_events.go
  - cmd/handlers/middleware.go
provides_interfaces:
  - {name: "POST /api/v1/company/workflow-definitions", kind: rest-endpoint, intent: "create a company-scoped workflow definition"}
  - {name: "GET /api/v1/company/workflow-definitions", kind: rest-endpoint, intent: "list company-scoped definitions"}
  - {name: "POST /api/v1/workspaces/{workspace_uuid}/workflow-definitions", kind: rest-endpoint, intent: "create a workspace-scoped definition"}
  - {name: "GET /api/v1/workspaces/{workspace_uuid}/workflow-definitions", kind: rest-endpoint, intent: "list workspace-scoped definitions"}
  - {name: "POST /api/v1/projects/{project_uuid}/workflow-definitions", kind: rest-endpoint, intent: "create a project-scoped definition"}
  - {name: "GET /api/v1/projects/{project_uuid}/workflow-definitions", kind: rest-endpoint, intent: "list project-scoped definitions"}
  - {name: "GET /api/v1/workflow-definitions/{workflow_definition_uuid}", kind: rest-endpoint, intent: "get one definition (any scope)"}
  - {name: "PUT /api/v1/workflow-definitions/{workflow_definition_uuid}", kind: rest-endpoint, intent: "partial-update / un-archive / set-mandatory a definition"}
  - {name: "DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}", kind: rest-endpoint, intent: "archive a definition (soft delete, 204)"}
  - {name: "POST /api/v1/workflow-definitions/{workflow_definition_uuid}/clone", kind: rest-endpoint, intent: "clone-to-override into a strictly more specific scope"}
  - {name: "POST /api/v1/workflow-definitions/{workflow_definition_uuid}/publish", kind: rest-endpoint, intent: "register the tenant-namespaced definition JSON with the engine"}
  - {name: "POST /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs", kind: rest-endpoint, intent: "declare a secret ref on a definition"}
  - {name: "GET /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs", kind: rest-endpoint, intent: "list a definition's declared secret refs"}
  - {name: "GET /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs/status", kind: rest-endpoint, intent: "per-ref resolution status without decrypting"}
  - {name: "DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs/{secret_ref_uuid}", kind: rest-endpoint, intent: "remove a secret ref (204)"}
  - {name: "POST /api/v1/workflow-executions", kind: rest-endpoint, intent: "start an execution from a definition UUID (idempotent via header)"}
  - {name: "GET /api/v1/workflow-executions/{execution_id}", kind: rest-endpoint, intent: "read an execution's engine status (tenant-checked)"}
  - {name: "POST /api/v1/task-completions", kind: rest-endpoint, intent: "complete a parked Conductor task on session-close / turn-completion"}
  - {name: "POST /api/v1/workflow-approvals", kind: rest-endpoint, intent: "record a human approve/reject decision and resume the parked task"}
see_also:
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "the definition CRUD/clone/publish endpoints below front this registry"}
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "start + get status endpoint behavior detail"}
  - {repo: ps-workflow, capability: "Human approval gate", intent: "the workflow-approvals decision endpoint drives this gate"}
  - {repo: ps-workflow, capability: "Async session→task completion bridge", intent: "the task-completions endpoint is the async bridge ingress"}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "ps-api/ps-ui reach this API through the gateway", descriptive: true}
---

# ps-workflow HTTP API

**What it is.** The REST surface of the L2 Workflow Service on port **9005**, base path
**`/api/v1`**. It is the only way a peer authors/publishes workflow definitions, starts and reads
executions, decides human approvals, and delivers async task completions. The Conductor engine
behind it is never exposed.

**Auth.** Every route is tenant-authenticated the same way: the caller presents the shared-JWT
identity as gateway headers **`X-User-UUID`** and **`X-Company-UUID`**, which are re-validated
(user ∈ company) against the platform DB on every request. Missing/invalid headers or a
user–company mismatch → 401. Tenant and originating user are always taken from these headers,
never from a request body.

## Workflow definitions (scoped CRUD + clone + publish + secret refs)

Definitions are scoped **company / workspace / project** (mirrors `agent_definition`). Create is
scope-specific; get/update/archive/clone/publish/secret-refs are scope-agnostic (keyed by
`workflow_definition_uuid`).

- `POST /api/v1/company/workflow-definitions` — create a company-scoped definition.
- `GET  /api/v1/company/workflow-definitions` — list company-scoped (optional `include_archived`).
- `POST /api/v1/workspaces/{workspace_uuid}/workflow-definitions` — create a workspace-scoped definition.
- `GET  /api/v1/workspaces/{workspace_uuid}/workflow-definitions` — list workspace-scoped.
- `POST /api/v1/projects/{project_uuid}/workflow-definitions` — create a project-scoped definition.
- `GET  /api/v1/projects/{project_uuid}/workflow-definitions` — list project-scoped.
- `GET  /api/v1/workflow-definitions/{workflow_definition_uuid}` — get one (any scope); 404 if not this tenant.
- `PUT  /api/v1/workflow-definitions/{workflow_definition_uuid}` — partial update / un-archive / set-mandatory; at most one mandatory lineage per company (409 on conflict).
- `DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}` — archive (soft delete → 204).
- `POST /api/v1/workflow-definitions/{workflow_definition_uuid}/clone` — clone-to-override; target scope must be **strictly more specific** than the source (422 otherwise).
- `POST /api/v1/workflow-definitions/{workflow_definition_uuid}/publish` — register the tenant-namespaced definition JSON with the engine (idempotent); rejects a doc lacking a non-empty `name` (422), engine failure → 502.
- `POST /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs` — declare a secret ref.
- `GET  /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs` — list declared refs.
- `GET  /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs/status` — per-ref resolution status (resolved / missing / placeholder / type_mismatch); never decrypts a value.
- `DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}/secret-refs/{secret_ref_uuid}` — remove a ref (204).

Request/response bodies are the `models.CreateWorkflowDefinitionRequest` /
`UpdateWorkflowDefinitionRequest` / `CloneWorkflowDefinitionRequest` /
`CreateWorkflowDefinitionSecretRefRequest` shapes — reference by name, not pasted here.

## Workflow executions (start + get only)

- `POST /api/v1/workflow-executions` — start a run from `workflow_definition_uuid`. Optional
  `Idempotency-Key` header makes a retry replay the original run (200) instead of starting a new
  one (409 if the key was used for a different definition). A required unresolved secret → 422,
  nothing started. Success → 201 with a `Location` header and `execution_id`.
- `GET  /api/v1/workflow-executions/{execution_id}` — read the run's engine status; a run whose
  tenant tag does not match the caller returns 404 (no existence leak).

There is **no** pause / resume / terminate / search endpoint — see the
`execution-api-is-start-plus-get-only` gotcha.

## Task completions (async bridge ingress)

- `POST /api/v1/task-completions` — the canonical async-session bridge. The orchestrator forwards
  two event shapes here: `session_closed` (whole session ended) and `agent_turn_completed` (one
  agent reply finished). It completes the parked Conductor task for that session. Idempotent and
  tenant-scoped; an unknown or cross-tenant session is a benign no-op with zero engine calls.

## Workflow approvals (human decision)

- `POST /api/v1/workflow-approvals` — record the **first** approve/reject decision for a parked
  `request-approval` task and resume it (approved → engine COMPLETED, rejected → engine FAILED).
  Idempotent (first-wins); a decider not in the gate's approver list → 403; an unknown or
  cross-tenant approval → 404.

**Failure-shape note.** `publish`, `start`, and the completion/approval resume paths surface the
engine's failure as **502** when the engine is unreachable or rejects the call — the peer receives
a gateway-error shape, not the raw engine error.

**See also.** Endpoint behavior detail lives in the **ps-workflow — Workflow definition
registry**, **Workflow execution API**, **Human approval gate**, and **Async session→task
completion bridge** capabilities. Node task types (what a definition's steps invoke) are in the
sibling **node catalog** interface.
