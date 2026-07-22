---
type: capability
title: "Workflow inbox"
tags: [inbox, approvals, notifications, read-surface, ps-ui]
timestamp: 2026-07-09T10:49:10Z
description: "Level-scoped read roll-up of pending human approvals + in-app notifications, consumed by ps-ui to render an inbox"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/workflow_inbox.go
  - cmd/handlers/middleware.go
  - internal/approval/store.go
  - internal/approval/pg_store.go
  - internal/notification/store.go
  - internal/notification/pg_store.go
see_also:
  - {repo: ps-workflow, capability: "Human approval gate", intent: "the pending approvals this inbox lists + the decision endpoint that acts on them", descriptive: false}
  - {repo: ps-workflow, capability: "send-notification node", intent: "the in-app notifications this inbox lists", descriptive: false}
  - {repo: ps-ui, capability: "Workflow inbox view", intent: "renders this response as an inbox and drives approval decisions", descriptive: true}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "forwards the browser call here with trusted tenant headers", descriptive: true}
---

# Workflow inbox

**What it does.** A read-only roll-up that answers one question for a level of the tenant
hierarchy: "what needs a human's attention here?" It combines two families — pending human
approvals (parked workflows awaiting an approve/reject decision) and in-app notifications
emitted by running workflows — into one payload a UI renders as an inbox.

**How a peer interacts.** Two GET endpoints, chosen by the level being viewed:

- `GET /api/v1/workspaces/{workspace_uuid}/workflow-inbox` — rolls up that workspace **and all
  its projects**.
- `GET /api/v1/projects/{project_uuid}/workflow-inbox` — that project only.

Both are called server-to-server by the gateway, which forwards the browser's request with
trusted tenant headers (`X-User-UUID` + `X-Company-UUID`). The inbox is inherently
level-scoped: the level comes from the path, the tenant from the headers.

**Observable behavior.** Returns the current snapshot synchronously — no async readiness. Each
family is ordered newest-first. Only approvals/notifications belonging to workflow runs
attributed to the requested level surface (attribution is by the run's workspace/project); an
item whose run has no recorded run-context appears in **no** inbox. A UI renders the two lists
and, for each approval, offers approve/reject — the decision itself is a separate write against
the approval decision endpoint (not part of this read surface).

**Contract.** In: the level UUID in the path; tenant in the gateway headers. Out — `InboxResponse`,
a **frozen contract with ps-ui** (this is the confirmed-complete wire shape, not a partial list):

- `approvals[]` — each `{workflow_id, task_ref_name, run_name, approvers, status}`.
  `approvers` is an array of allowed decider user-ids; **an empty array means any tenant user may
  decide** (never null on the wire). `run_name` is the run's friendly definition name, `""` when
  the definition can't be resolved (e.g. archived). `status` is always `"pending"` for inbox rows.
- `notifications[]` — each `{notification_uuid, workflow_id, task_ref_name, title, body, channel,
  created_at}`. `channel` is `"in-app"` in v1. `created_at` is an RFC3339 timestamp.

Both arrays are always present and are `[]` (never null) when empty.

**Invariants.** Every query is tenant-scoped by company — a caller only ever sees its own
company's items, and cross-tenant enumeration is impossible (a foreign level UUID simply matches
no rows). `{workflow_id, task_ref_name}` identifies an approval within a run; a notification is
identified by `notification_uuid`.

**Failure modes.** Missing or malformed `X-User-UUID`/`X-Company-UUID`, or a user that doesn't
belong to the claimed company → **401** (the user-company relationship is re-validated against
the DB on every call). A malformed level UUID in the path → **400**. A failure querying either
underlying store → **502** ("failed to list approvals"/"failed to list notifications").

**Gotchas.** This is a read surface only — acting on an approval is a distinct decision call, so
a UI must not treat listing as deciding. `approvers: []` is permissive (anyone in the tenant),
not restrictive. `run_name` can be empty; don't rely on it as a stable key (use `workflow_id`).
An approval/notification with no run-context is silently invisible to every inbox rather than
appearing at a default level.

**See also.** Same-repo **Human approval gate** owns the pending approvals listed here and the
decision endpoint a UI calls to act on them; same-repo **Notification node** writes the in-app
notifications listed here. Cross-repo **ps-ui** renders this payload; **ps-api** proxies the
call with trusted tenant headers.
