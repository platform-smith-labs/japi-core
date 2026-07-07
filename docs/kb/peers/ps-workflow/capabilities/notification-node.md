---
type: capability
title: "send-notification node"
tags: [workflow-node, notification, in-app, tenant-scoped, conductor-worker]
timestamp: 2026-07-07T06:49:45Z
description: "Workflow node that emits a durable, tenant-scoped in-app notification from inside a workflow"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/nodes/notify.go
  - internal/notification/store.go
  - internal/notification/pg_store.go
---

# send-notification node

**What it does.** A custom Conductor worker node that writes a self-contained, tenant-scoped
in-app notification from within a running workflow — a durable row that IS the deliverable
(v1 has no push/delivery step). Used as a workflow step to record "this happened" for a
company or a specific user.

**How a peer interacts.** Reference the Conductor task type `send-notification` in a workflow
definition and supply annotation inputs under `inputParameters._ps`. Fields — `title`
(required), `body` (optional), `target_user` (optional user UUID), `data` (optional JSON object).
There is no `channel` input — every v1 row is hardcoded `in-app`. A peer does not call this
node over HTTP; it runs when a workflow execution reaches the step.

**Observable behavior.** Synchronous — the node completes immediately in the same task turn
(NOT a park node): it inserts one notification row and returns COMPLETED with output
`{notification_uuid}`. The row is persisted as the whole point of the node; there is no
separate delivery attempt whose success it waits on. Missing `title` fails the task.

**Contract.** In (`_ps`): `title` (req), `body?`, `target_user?` (user UUID),
`data?` (object). Out: `{notification_uuid}` — the new row's `workflow_notification_uuid`.
`target_user` present → a per-user notification; absent (or a UUID that does not resolve
within the caller's company) → a company-wide row. Errors: unknown company (the tenant's
`company_uuid` matches no company row) fails the node.

**Invariants.** Every row is scoped to the caller's company. Recipient resolution is
company-scoped: a `target_user` that does not belong to the caller's company silently
degrades to company-wide rather than leaking across tenants — it never targets a foreign
user. Enforced here (in this repo's insert path).

**Failure modes.** Missing `title` → task fails. Unknown company for the caller → task fails.
A non-resolving `target_user` does NOT fail — it becomes a company-wide notification, which an
integrator may not expect.

**Gotchas.** There is no `channel` input — a supplied `channel` value is ignored; every v1 row
is `in-app`. A `target_user` that is nil, absent, or
foreign all collapse to the same company-wide outcome — there is no error distinguishing a
typo'd recipient from an intentional broadcast. The node emits `notification_uuid`, but this
repo does NOT expose a read/list endpoint for notifications.

**Business-critical data.** Rows land in the `workflow_notification` table (company-scoped;
optional `user_id` NULL = company-wide) with a hardcoded `in-app` channel, `title`, `body`, `data`,
and the originating `workflow_id` + `task_ref_name`.

**Read path.** UNKNOWN — no notification read/list capability exists in this repo (v1 row IS
the deliverable, "no UI consumer"). Which service surfaces `workflow_notification` to a UI is
not determinable from this repo's code. UNKNOWN — TODO: confirm whether ps-api/orchestrator
owns the read/surface path.
