---
type: capability
title: "send-notification node"
tags: [workflow-node, notification, in-app, channels, tenant-scoped, conductor-worker]
timestamp: 2026-07-09T10:49:10Z
description: "Workflow node that emits a durable in-app notification, or delivers to an external channel via a per-stack sender"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/notify.go
  - internal/notification/store.go
  - internal/notification/pg_store.go
see_also:
  - {repo: ps-workflow, capability: "Workflow inbox", intent: "the read surface that lists the in-app notifications this node writes"}
---

# send-notification node

**What it does.** A custom Conductor worker node that emits a notification from within a running
workflow. The **in-app** channel (the default) writes a durable, tenant-scoped notification row that
IS the deliverable; the **external** channels (`slack` / `teams` / `email` / `webhook`) deliver
through a per-stack sender. Used as a workflow step to record or dispatch "this happened" for a
company or a specific user.

**How a peer interacts.** Reference the Conductor task type `send-notification` in a workflow
definition and supply annotation inputs under `inputParameters._ps`. Fields — `title` (required),
`body` (optional), `channel` (optional, default `in-app`; else `slack`/`teams`/`email`/`webhook`),
`target_user` (optional user UUID; in-app only), `data` (optional JSON object). A peer does not call
this node over HTTP; it runs when a workflow execution reaches the step.

**Observable behavior.** Synchronous — the node completes in the same task turn (NOT a park node).
- **in-app** → inserts one notification row and returns COMPLETED with output
  `{notification_uuid, channel}`.
- **external channel** → delivers through the channel sender and returns COMPLETED with
  `{delivered, channel, notification_id}`. On a stack where external channels are **not wired**, the
  node returns **NOT_LIVE** (a distinct Conductor FAILED with a not-live marker) — never a false
  "sent". The in-app channel is always live.

Missing `title` fails the task.

**Contract.** In (`_ps`): `title` (req), `body?`, `channel?` (default `in-app`), `target_user?`
(user UUID, in-app only), `data?` (object). Out — in-app: `{notification_uuid, channel}`; external:
`{delivered, channel, notification_id}`. Errors: missing `title` → fail; a **malformed**
`target_user` UUID → fail (fail-closed, see invariants); an external-channel delivery error → fail;
an external channel with no sender wired → NOT_LIVE.

**Invariants.** Every in-app row is scoped to the caller's company. `target_user` **absent** → a
company-wide row; `target_user` **present** → recorded as that user's notification (recipient
company-scoping is enforced at the insert path). A **malformed** `target_user` **fails the node**
rather than silently broadening to company-wide — visibility is never widened on a typo.

**Failure modes.** Missing `title` → task fails. Malformed `target_user` → task fails (fail-closed).
External-channel send error → task fails. External channel unwired on the stack → NOT_LIVE (branch
on the marker; not a delivery success and not a plain failure).

**Gotchas.** The default channel is `in-app`; you must pass an explicit `channel` for an external
destination. External channels are **per-stack gated** — treat a NOT_LIVE as "channel not available
here," not as a failure. `target_user` applies only to the in-app row; a malformed value is a hard
fail (not a silent broadcast).

**Business-critical data.** In-app rows land in the `workflow_notification` table (company-scoped;
`user_id` NULL = company-wide) with `channel`, `title`, `body`, `data`, and the originating
`workflow_id` + `task_ref_name`. (Tenant scoping applies as everywhere — see context.md.)

**Read path.** The in-app notifications this node writes are surfaced by **this repo's** workflow
inbox roll-up (see the sibling *Workflow inbox* capability) — a workspace/project-scoped list of
pending approvals + notifications. External-channel deliveries are not stored here (the sender owns
their delivery record).
