---
type: capability
title: "Workflow execution API"
tags: [workflow, executions, tenant-isolation, idempotency, conductor]
timestamp: 2026-07-07T06:49:45Z
description: "Start a workflow execution from a definition and read its status, tenant-scoped, with idempotent start"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_executions.go
  - cmd/models/models.go
  - internal/tenant/proxy.go
  - internal/idempotency/store.go
  - internal/conductor/conductor.go
see_also:
  - {repo: ps-api, capability: "Gateway request proxy", intent: "ps-api/ps-ui call this API to start executions and poll status", descriptive: true}
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "registers the definition (by UUID) that start runs; owns the namespaced name"}
---

# Workflow execution API

**What it does.** Starts a run of a previously-registered workflow definition and lets the
caller read that run's status. This is the only execution surface — the Conductor engine that
actually runs the workflow is hidden behind it and never exposed.

**How a peer interacts.** Two endpoints only:
- `POST /api/v1/workflow-executions` — start a run. Optional `Idempotency-Key` request header.
- `GET /api/v1/workflow-executions/{execution_id}` — read a run's status.

There is **no pause / resume / terminate / search** endpoint. The service exposes start + get
only. (The README lists lifecycle controls as design intent; they are not implemented.)

**Observable behavior.**
- Start resolves the definition's required secrets as a gate (nothing is started if a required
  secret is unresolved), then launches the run in the engine and returns `201 Created` with a
  `Location` header pointing at the new run's GET URL. Response carries `execution_id` (the
  engine run id used for all subsequent reads) and `namespaced_name` (the tenant-scoped engine
  name of the definition).
- Start always runs the **latest** registered version of the definition (version selection is
  not caller-controllable).
- Get returns the run's current `status` plus its `output`. Readiness/completion is **async** —
  a peer polls this GET; the `status` field is the signal.

**Contract.**
- Start in — `key fields:` `workflow_definition_uuid` (required, which definition to run),
  `input` (free-form map passed to the run), `project_uuid` (optional; overrides the exec scope
  used for secret resolution).
- Start out — `execution_id`, `workflow_definition_uuid`, `namespaced_name`, `idempotent` (true
  only on an idempotent replay). Plus the `Location` header.
- Get out — `models.ExecutionStatus`: `execution_id`, `status`, `output`.
- Errors: invalid `workflow_definition_uuid` / `project_uuid` → 400; unknown definition → 404; a
  required secret unresolved → 422 (nothing started); reused `Idempotency-Key` with a different
  definition → 409; engine start failure → 502; engine status-fetch failure → 502.

**Invariants.**
- **Tenant isolation, enforced here** (the engine is tenant-blind). Every start namespaces the
  definition name per company and tags the run with the tenant; every Get verifies the run's
  tenant tag matches the caller before returning anything. An id that exists but belongs to
  **another tenant** is reported as **404**. A **genuinely unknown** id is not normalized to 404 —
  the engine's not-found surfaces as a **502** (engine status-fetch failure). So the two are
  *distinguishable* (404 = exists-but-not-yours, 502 = no such run); do not rely on a uniform
  not-found. (See the Gotchas note on this cross-tenant existence signal.)
- **Idempotent start** on `(company, Idempotency-Key)`: a retry with the same key and same
  definition returns the original `execution_id` and starts nothing new (`idempotent: true`,
  HTTP 200). Same key with a *different* definition → 409. Without the header, every POST starts
  a new run.
- Idempotency identity is `(company_uuid, key)` — a key can never collide across tenants.

**Failure modes.**
- Required secret missing → 422, no run created; the peer must supply/resolve the secret and
  retry.
- Engine unreachable / rejects the start → 502; the run did not start.
- Polling an id belonging to **another tenant** → 404. Polling a **nonexistent** id → 502 (the
  engine's not-found is surfaced as a status-fetch failure, not normalized to 404).

**Gotchas.**
- **Idempotency is in-memory and lost on a ps-workflow restart** (spike-grade). A retry that
  crosses a service restart will start a *second* run rather than replay — do not rely on it for
  exactly-once across deploys.
- `status` is the **engine's raw status string, passed through unnormalized** — running reads
  `RUNNING`; terminal runs read one of the engine's terminal states (e.g. `COMPLETED`, `FAILED`,
  `TERMINATED`, `TIMED_OUT`). This service does not define, map, or enumerate the set — treat the
  terminal vocabulary as engine-defined and match case-sensitively. UNKNOWN whether the engine
  emits additional states this service would surface verbatim.
- `namespaced_name` is the *engine-internal* tenant-scoped name; peers key runs by
  `execution_id`, not by this name. It is returned for observability, not as a lookup key.
- `input` supplied by the caller is forwarded to the run as-is; the tenant/originating-user
  identity is stamped by the service from the authenticated caller, not taken from `input`.
- **Cross-tenant existence signal.** Because an existing-but-foreign id returns 404 while a
  nonexistent id returns 502, the status endpoint leaks whether an `execution_id` exists in
  *another* tenant. Treat 404 vs 502 as "exists elsewhere" vs "no such run"; do not assume a
  uniform not-found.

**Business-critical data.** Start reads the target `workflow_definition` (by UUID, tenant-scoped)
to obtain its name and scope, and its secret-ref rows to run the required-secret gate. No
execution state is stored in this service's DB — run state lives in the hidden engine and is read
back through the tenant seam. (Tenant scoping applies as everywhere — see context.md.)

**See also / peers.** The definition that `workflow_definition_uuid` names is created by
**ps-workflow — Workflow definition registry** (which owns registration and the namespaced
name). **ps-api — Gateway request proxy** is the front door ps-ui/ps-api use to reach this API.
