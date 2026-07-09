---
type: capability
title: "Workflow execution API"
tags: [workflow, executions, tenant-isolation, idempotency, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "Start a workflow execution from a definition, read its status, and list the company's executions — tenant-scoped, with idempotent start"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/workflow_executions.go
  - cmd/models/models.go
  - internal/tenant/proxy.go
  - internal/idempotency/store.go
  - internal/runcontext/store.go
  - internal/conductor/conductor.go
see_also:
  - {repo: ps-api, capability: "Gateway request proxy", intent: "ps-api/ps-ui call this API to start executions, poll status, and list runs", descriptive: true}
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "registers the definition (by UUID) that start runs; owns the namespaced name"}
---

# Workflow execution API

**What it does.** Starts a run of a previously-registered workflow definition, lets the caller
read a run's status, and lists the company's runs. This is the only execution surface — the
Conductor engine that actually runs the workflow is hidden behind it and never exposed.

**How a peer interacts.** Three endpoints:
- `POST /api/v1/workflow-executions` — start a run. Optional `Idempotency-Key` request header.
- `GET /api/v1/workflow-executions/{execution_id}` — read a run's status.
- `GET /api/v1/workflow-executions` — list the company's runs (filterable).

There is **no pause / resume / terminate** endpoint. Start / get / list only. (The README lists
lifecycle controls as design intent; they are not implemented — see the Gotchas note.)

**Observable behavior.**
- Start resolves the definition's required secrets as a gate (nothing is started if a required
  secret is unresolved), then launches the run in the engine and returns `201 Created` with a
  `Location` header pointing at the new run's GET URL. Response carries `execution_id` (the
  engine run id used for all subsequent reads) and `namespaced_name` (the tenant-scoped engine
  name of the definition). Start always runs the **latest** registered version (not
  caller-controllable). A definition that is not yet published is rejected before any engine call.
- On a successful start the service also writes a best-effort **run-context sidecar** row
  recording which workspace/project/definition the run targeted; a write failure only degrades
  later enrichment and never fails the start.
- Get returns the run's current `status` plus its `output`. Readiness/completion is **async** — a
  peer polls this GET; the `status` field is the signal. Get **also** best-effort enriches the
  response with `workflow_definition_uuid` and the definition's `conductor_json` (so a run viewer
  can draw the read-only DAG); a run with no sidecar row (a pre-sidecar run) simply omits those
  DAG fields and returns engine-only status.
- List returns the company's runs, newest engine page, each row enriched from the run-context
  sidecar (friendly un-namespaced name, execution_context, workspace/project). It supports
  `status`, `limit` (default 50, max 200), and `offset`, plus scope filters `workspace_uuid` /
  `project_uuid` (mutually exclusive → 400) and `execution_context` (`workspace`|`project`). A
  workspace filter rolls up that workspace's project runs too. A run **without** a run-context
  row cannot match an active scope filter, so pre-sidecar runs are excluded when a scope filter is
  applied (they still appear in an unfiltered list, with engine-only fields).

**Contract.**
- Start in — `key fields:` `workflow_definition_uuid` (required, which definition to run),
  `input` (free-form map passed to the run), `project_uuid` (optional; overrides the exec scope
  used for secret resolution). A trusted `X-Workspace-UUID` header may attribute a company-level
  run to an operating workspace.
- Start out — `key fields:` `execution_id`, `workflow_definition_uuid`, `namespaced_name`,
  `idempotent` (true only on an idempotent replay). Plus the `Location` header.
- Get out — `models.ExecutionStatus`: `key fields:` `execution_id`, `status`, `output`,
  `workflow_definition_uuid?`, `conductor_json?` (the last two only when a sidecar row resolves).
- List out — `{executions[], total}`; each item `ExecutionListItem` has `key fields:`
  `execution_id`, `workflow_definition_uuid?`, `name` (friendly, un-namespaced), `status`,
  `execution_context?`, `workspace_uuid?`, `project_uuid?`, `start_time`, `end_time?`. Context
  fields are null for runs without a sidecar row. `total` is the engine's tenant-scoped count when
  unscoped, or the filtered page count when a scope filter narrows it app-side.
- Errors: invalid `workflow_definition_uuid` → 400; unknown definition → 404; unpublished/draft
  definition → 422 (nothing started); unknown/foreign `project_uuid` → 404 (no existence leak); a
  required secret unresolved → 422 (nothing started); reused `Idempotency-Key` with a different
  definition → 409; engine start failure → 502; engine status-fetch failure → 502; engine search
  failure (list) → 502; `workspace_uuid` + `project_uuid` both set (list) → 400.

**Invariants.**
- **Tenant isolation, enforced here** (the engine is tenant-blind). Every start namespaces the
  definition name per company and tags the run with the tenant; Get and List go through the same
  tenant seam. List forces the engine search to the caller's company (the seam pins
  correlationId=company) — no cross-tenant runs can appear.
- **Cross-tenant existence signal on Get.** An id whose tenant tag belongs to **another** tenant is
  reported as **404** ("not found", no leak). A **genuinely unknown** id is not normalized to 404 —
  the engine's not-found surfaces as a **502** (status-fetch failure). So 404 (exists-but-not-yours)
  and 502 (no such run) are *distinguishable*; do not rely on a uniform not-found. (See Gotchas.)
- **Idempotent start** on `(company, Idempotency-Key)`: a retry with the same key and same
  definition returns the original `execution_id` and starts nothing new (`idempotent: true`,
  HTTP 200). Same key with a *different* definition → 409. Without the header, every POST starts a
  new run. Idempotency identity is `(company_uuid, key)` — a key can never collide across tenants.

**Failure modes.**
- Required secret missing → 422, no run created; the peer must supply/resolve the secret and retry.
- Definition not published → 422, no run created.
- Engine unreachable / rejects the start → 502; the run did not start.
- Polling an id belonging to **another tenant** → 404. Polling a **nonexistent** id → 502.
- List: engine search unreachable → 502. Run-context enrichment failure → runs still returned with
  engine-only fields (name may be the un-namespaced engine type, scope fields null).

**Gotchas.**
- **Idempotency is in-memory and lost on a ps-workflow restart** (spike-grade). A retry that crosses
  a service restart will start a *second* run rather than replay — do not rely on it for exactly-once
  across deploys.
- **Run-context enrichment is best-effort.** Runs started before the sidecar existed (or with a
  failed sidecar write) have no context row: their List rows fall back to the un-namespaced engine
  name with null scope fields, they are **excluded** from any scoped List filter, and their Get omits
  the DAG fields. Do not treat a missing `workflow_definition_uuid`/scope as "no such definition".
- `status` is the **engine's raw status string, passed through unnormalized** — running reads
  `RUNNING`; terminal runs read one of the engine's terminal states (e.g. `COMPLETED`, `FAILED`,
  `TERMINATED`, `TIMED_OUT`). This service does not define, map, or enumerate the set — treat the
  terminal vocabulary as engine-defined and match case-sensitively. The list `status` filter accepts
  `RUNNING|COMPLETED|FAILED|PAUSED|TERMINATED|TIMED_OUT`.
- `namespaced_name` is the *engine-internal* tenant-scoped name; peers key runs by `execution_id`,
  not by this name. `name` in a List row is the friendly, un-namespaced definition name — never key
  off it either.
- `input` supplied by the caller is forwarded to the run as-is; the tenant/originating-user identity
  is stamped by the service from the authenticated caller, not taken from `input`.
- **Cross-tenant existence signal.** Because an existing-but-foreign id returns 404 while a
  nonexistent id returns 502, the Get endpoint leaks whether an `execution_id` exists in *another*
  tenant. Treat 404 vs 502 as "exists elsewhere" vs "no such run"; do not assume a uniform not-found.

**Business-critical data.** Start reads the target `workflow_definition` (by UUID, tenant-scoped) to
obtain its name/scope and its secret-ref rows for the required-secret gate, and writes a
`workflow_run_context` sidecar row (unique on company + run id) recording the run's
workspace/project/definition/execution-context. That sidecar is the single source List and Get read
to enrich runs; the friendly definition name is resolved from it. No execution *status* is stored
here — run state lives in the hidden engine and is read back through the tenant seam. (Tenant scoping
applies as everywhere — see context.md.)

**See also / peers.** The definition that `workflow_definition_uuid` names is created by
**ps-workflow — Workflow definition registry** (which owns registration and the namespaced name).
**ps-api — Gateway request proxy** is the front door ps-ui/ps-api use to reach this API.
