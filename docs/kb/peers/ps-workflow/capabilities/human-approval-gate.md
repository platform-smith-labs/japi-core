---
type: capability
title: "Human approval gate"
tags: [workflow, approval, human-in-the-loop, park-style, durable]
timestamp: 2026-07-07T06:49:45Z
description: "Durable human-in-the-loop gate: a parked request-approval node plus the decision endpoint that resumes it"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/nodes/approval.go
  - cmd/handlers/workflow_approvals.go
  - internal/approval/store.go
  - internal/approval/pg_store.go
see_also:
  - {repo: ps-ui, capability: "Approval decision UI", intent: "renders the pending gate and POSTs the human decision", descriptive: true}
  - {repo: ps-workflow, capability: "Async session→task completion bridge", intent: "sibling park-style node; both resume a parked Conductor task push-only"}
---

# Human approval gate

**What it does.** A durable human-in-the-loop pause in a workflow: the `request-approval` node
parks the workflow at a decision point until a human approves or rejects, then the branch either
continues (approved) or fails (rejected). Two halves — the worker node that parks, and an HTTP
decision endpoint a human/UI calls to resume it.

**How a peer interacts.**
- *Authoring side:* place a `request-approval` node in the Conductor workflow definition. Its only
  PS input is `_ps.approvers` — a list of user_uuids allowed to decide (empty = any user of the
  tenant may decide).
- *Decision side:* `POST /api/v1/workflow-approvals` with `{workflow_id, task_ref_name, decision}`
  (`decision` is `"approved"` or `"rejected"`; optional `reason`). The tenant and the deciding user
  come from validated gateway headers, never the body.

**Observable behavior.** When reached, the node records a pending, tenant-scoped approval and parks
the Conductor task IN_PROGRESS (no poller — completion is push-only). It stays parked indefinitely
until a decision arrives. The first decision wins and is recorded atomically; that decision resumes
the parked task — **approved → the task COMPLETES** (branch continues), **rejected → the task FAILS**
(the workflow branch fails). The completed/failed task's output carries the decision provenance —
key fields: `decision`, `decided_by` (the winning decider's user_uuid), `reason`.

**Contract.**
- *Node input:* `_ps.approvers` (open — this is the only field this node reads). Node park output:
  `{workflow_id, awaiting_approval}`.
- *Endpoint in:* `{workflow_id, task_ref_name, decision, reason?}`. *Out:*
  `{result, decision, workflow_id}` where `result` is `"decided"` (this call performed the flip) or
  `"already_decided"` (a prior decision won) and `decision` is the **winning** value.
- *Errors:* missing tenant/user context → 401; missing `workflow_id`/`task_ref_name` or bad
  `decision` value → 400; unknown or cross-tenant approval → 404 (no existence leak); decider not in
  a non-empty approvers list → 403; store error → 502; transient failure resuming the Conductor task
  → 502.

**Invariants.** Decide is exactly-once (a single conditional UPDATE guarded on `status='pending'`) —
exactly one concurrent caller gets `result="decided"`, the rest see `already_decided` with the same
winning decision. Tenant isolation is enforced here on every call (Conductor is tenant-blind — see
context): the approval is keyed on `(company, workflow_id, task_ref_name)`, and the decider's tenant
membership is checked inside the decision SQL. The endpoint's resume of the parked task is idempotent
at the engine, so it is retried safely on every decision (including duplicates) to recover a prior
failed resume.

**Failure modes.** A duplicate/late decision is not an error — it returns `already_decided` carrying
the original winner's decision and provenance. If resuming the engine task fails permanently (the
task is already terminal, e.g. the first decider already completed it), that is treated as idempotent
success; only a transient engine failure surfaces as 502. A decider outside the approvers list gets
403; a wrong-tenant or unknown target gets 404.

**Durability.** The pending approval and its decision survive a service restart — there is no
in-memory poller; the parked task is only ever advanced by an inbound decision call. Resuming the
Conductor task runs on a detached 30s-timeout context so a client disconnect can't orphan the resume.

**Gotchas.**
- `retryCount:0` (park style): the task def must not let Conductor auto-retry the parked task —
  advancement is push-only via the endpoint.
- No built-in timeout/auto-expiry on the gate itself — it waits forever for a human decision. Any
  deadline must be modeled in the workflow around this node. UNKNOWN whether an expiry policy exists.
- **Reject fails the branch.** A rejected decision resolves the engine task to FAILED, not a clean
  "no" branch — model the reject path accordingly.
- The prompt/context text shown to the approver is not persisted or returned by this node —
  UNKNOWN whether it is carried elsewhere (e.g. surfaced by the UI from the execution state).

**Business-critical data.** The `workflow_approval` table (migration 0050) is the durable
correlation between a parked task and its decision — keyed on `(company_id, workflow_id,
task_ref_name)`, holding `status` (pending|approved|rejected), the allowed `approvers`, and the
winning `decided_by`/`decision_reason`. (Tenant scoping applies as everywhere — see context.)

**See also / peers.** ps-ui (Approval decision UI) renders the pending gate and submits the decision.
The sibling park-style completion bridge in this repo (session-closed) resumes a parked Conductor
task the same push-only way.
