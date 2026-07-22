---
type: interface
title: "ps-workflow HTTP API"
tags: [http, rest, workflow-definitions, executions, approvals, task-completions, signals, webhooks, inbox]
timestamp: 2026-07-09T10:49:10Z
description: "The REST surface a peer calls to author/publish workflow definitions, start/list/read executions, decide approvals, complete parked tasks, land signals, trigger via webhook, read the inbox, and read the node catalog"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_definitions_scoped.go
  - cmd/handlers/workflow_definition_secret_refs.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/workflow_approvals.go
  - cmd/handlers/session_events.go
  - cmd/handlers/signals.go
  - cmd/handlers/webhooks.go
  - cmd/handlers/workflow_inbox.go
  - cmd/handlers/workflow_task_catalog.go
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
  - {name: "GET /api/v1/workflow-executions", kind: rest-endpoint, intent: "list company executions, scope-filtered + enriched from run-context"}
  - {name: "GET /api/v1/workflow-executions/{execution_id}", kind: rest-endpoint, intent: "read an execution's engine status + best-effort definition/DAG enrichment (tenant-checked)"}
  - {name: "POST /api/v1/task-completions", kind: rest-endpoint, intent: "complete a parked Conductor task on session-close / turn-completion"}
  - {name: "POST /api/v1/workflow-approvals", kind: rest-endpoint, intent: "record a human approve/reject decision and resume the parked task"}
  - {name: "POST /api/v1/signals", kind: rest-endpoint, intent: "Model-B signal unpark of an await-signal task keyed on (company, correlation_id)"}
  - {name: "POST /api/v1/webhooks/{webhook_uuid}", kind: rest-endpoint, intent: "public token-authed webhook trigger — start a run of a published definition"}
  - {name: "GET /api/v1/workspaces/{workspace_uuid}/workflow-inbox", kind: rest-endpoint, intent: "workspace inbox: pending approvals + notifications (rolls up its projects)"}
  - {name: "GET /api/v1/projects/{project_uuid}/workflow-inbox", kind: rest-endpoint, intent: "project inbox: pending approvals + notifications"}
  - {name: "GET /api/v1/workflow-task-catalog", kind: rest-endpoint, intent: "deployment-static node catalog for the visual builder (not tenant-filtered)"}
see_also:
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "the definition CRUD/clone/publish endpoints below front this registry"}
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "start + list + get status endpoint behavior detail"}
  - {repo: ps-workflow, capability: "Human approval gate", intent: "the workflow-approvals decision endpoint drives this gate"}
  - {repo: ps-workflow, capability: "Async session→task completion bridge", intent: "the task-completions endpoint is the async bridge ingress"}
  - {repo: ps-workflow, capability: "Signal wait & unpark (Model-B)", intent: "the signals endpoint lands a Model-B unpark", descriptive: false}
  - {repo: ps-workflow, capability: "Webhook triggers", intent: "the webhooks endpoint is the public trigger ingress", descriptive: false}
  - {repo: ps-workflow, capability: "Workflow inbox", intent: "the workflow-inbox endpoints roll up pending approvals + notifications", descriptive: false}
  - {repo: ps-workflow, capability: "Workflow task catalog", intent: "the workflow-task-catalog endpoint serves the node catalog", descriptive: false}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "ps-api/ps-ui reach this API through the gateway", descriptive: true}
---

# ps-workflow HTTP API

**What it is.** The REST surface of the L2 Workflow Service on port **9005**, base path
**`/api/v1`**. It is the only way a peer authors/publishes workflow definitions, starts/lists/reads
executions, decides human approvals, delivers async task completions, lands signals, triggers runs
via webhook, and reads the inbox and node catalog. The Conductor engine behind it is never exposed.

**Auth.** Three middlewares gate this surface:

- **TrustGatewayHeaders** (the default — all definition, execution, approval, task-completion, inbox,
  and catalog routes): the caller presents the shared-JWT identity as gateway headers
  **`X-User-UUID`** and **`X-Company-UUID`**, re-validated (user ∈ company) against the platform DB
  on every request. Missing/invalid headers or a user–company mismatch → 401. Tenant and originating
  user are always taken from these headers, never from a request body.
- **TrustCompanyHeader** (only `POST /api/v1/signals`): trusts **`X-Company-UUID`** alone;
  `X-User-UUID` is optional (honored if valid, never required) — the agent-forwarded signal carries
  no user. Missing company → 401.
- **Webhook token** (only `POST /api/v1/webhooks/{webhook_uuid}`): no gateway headers at all — the
  one public route; auth is the `X-Webhook-Token` bearer, constant-time HMAC-verified.

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

Request/response bodies are the `CreateWorkflowDefinitionRequest` / `UpdateWorkflowDefinitionRequest`
/ `CloneWorkflowDefinitionRequest` / `CreateWorkflowDefinitionSecretRefRequest` shapes — reference by
name, not pasted here.

## Workflow executions (start + list + get)

- `POST /api/v1/workflow-executions` — start a run from `workflow_definition_uuid`. Optional
  `Idempotency-Key` header makes a retry replay the original run (200) instead of starting a new one
  (409 if the key was used for a different definition). An unpublished definition → 422; a required
  unresolved secret → 422, nothing started. Success → 201 with a `Location` header and `execution_id`.
- `GET  /api/v1/workflow-executions` — list this company's runs, tenant-forced at the engine seam and
  enriched from the run-context sidecar. Query filters (all optional): `workspace_uuid` and
  `project_uuid` (**mutually exclusive** → 400), `execution_context` (`workspace|project`), `status`,
  `limit` (default 50, max 200), `offset`. Returns `ListExecutionsResponse` — `executions[]`
  (`ExecutionListItem`) + `total`. A run **without** a run-context row cannot match a scope/context
  filter and is excluded when one is active; `total` reflects the filtered count when a scope filter
  narrows the page, else the engine's tenant-scoped total.
- `GET  /api/v1/workflow-executions/{execution_id}` — read the run's engine status (`ExecutionStatus`);
  a run whose tenant tag does not match the caller returns 404 (no existence leak). Best-effort
  run-context enrichment adds `workflow_definition_uuid` and the definition's `conductor_json` (for
  the read-only DAG viewer); a pre-sidecar run simply omits those fields.

There is **no** pause / resume / terminate endpoint — see the
`execution-api-is-start-plus-get-only` gotcha (list is search-backed, not a control op).

## Task completions (async bridge ingress)

- `POST /api/v1/task-completions` — the canonical async-session bridge. The orchestrator forwards two
  event shapes here: `session_closed` (whole session ended) and `agent_turn_completed` (one agent
  reply finished). It completes the parked Conductor task for that session. Idempotent and
  tenant-scoped; an unknown or cross-tenant session is a benign no-op with zero engine calls.

## Workflow approvals (human decision)

- `POST /api/v1/workflow-approvals` — record the **first** approve/reject decision for a parked
  `request-approval` task and resume it (approved → engine COMPLETED, rejected → engine FAILED).
  Idempotent (first-wins); a decider not in the gate's approver list → 403; an unknown or cross-tenant
  approval → 404.

## Signals (Model-B unpark)

- `POST /api/v1/signals` — land a signal that unparks an await-signal task keyed on
  `(company, correlation_id)`. Company-only auth (`TrustCompanyHeader`). Body (`SignalRequest`):
  `correlation_id` (required), `company_uuid` (advisory — if present must match the trusted tenant),
  `status` (`COMPLETED`|`FAILED`, default COMPLETED), `payload`. Returns `SignalResponse`:
  `{result, correlation_id}` where `result` ∈ `completed | noop_no_mapping | noop_terminal`. Missing
  company → 401; bad `correlation_id` or a `company_uuid` mismatch → 400; the completion failing at
  the engine → 502. An unknown/cross-tenant correlation id is a benign no-op (no enumeration).

## Webhook triggers (public token-authed ingress)

- `POST /api/v1/webhooks/{webhook_uuid}` — the ONLY public route. No gateway headers; auth is the
  `X-Webhook-Token` bearer (HMAC-verified). The body is arbitrary JSON and becomes the run input under
  key `webhook`. Tenant + published definition are resolved **from the stored row**, never the
  request; the run is system-originated (no user). Success → `{result: "started", execution_id}`. A
  bad token → 401; an unknown/disabled/unpublished trigger → 404 (indistinguishable, no existence
  leak) — as is any request when `WEBHOOK_TRIGGER_ENABLED` is off (the route hides itself); an engine
  failure → 502.

## Workflow inbox (approvals + notifications roll-up)

- `GET /api/v1/workspaces/{workspace_uuid}/workflow-inbox` — rolls up the workspace **and all its
  projects**.
- `GET /api/v1/projects/{project_uuid}/workflow-inbox` — that project only.

Both return `InboxResponse` = `approvals[]` (actionable pending approvals — decide via
`POST /api/v1/workflow-approvals`) + `notifications[]` (informational in-app notifications). Scope is
taken from the path param; a store failure → 502.

## Workflow task catalog (visual-builder node catalog)

- `GET /api/v1/workflow-task-catalog` — the deployment-static node catalog (per-task input/output
  schemas, typed handles) the ps-ui visual builder renders its palette from. Authenticated
  (`TrustGatewayHeaders` → 401 without gateway headers) but **not tenant-filtered**: the catalog is
  identical for every tenant on a deployment (per-task availability reflects server env, not the
  caller).

**Failure-shape note.** `publish`, `start`, list, the completion/approval resume paths, signals,
webhook triggers, and the inbox surface an engine/store failure as **502** — the peer receives a
gateway-error shape, not the raw engine error.

**See also.** Endpoint behavior detail lives in the same-repo capabilities **Workflow definition
registry**, **Workflow execution API**, **Human approval gate**, **Async session→task completion
bridge**, **Signal wait & unpark**, **Webhook triggers**, **Workflow inbox**, and **Workflow task
catalog**. Node task types (what a definition's steps invoke) are in the sibling **node catalog**
interface.
