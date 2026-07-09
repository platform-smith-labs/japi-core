---
type: gotcha
title: "Webhook- and schedule-triggered runs carry no user identity"
tags: [executions, webhook, scheduler, system-originated, tenancy]
timestamp: 2026-07-09T10:49:10Z
description: "Executions started by a webhook trigger or the cron scheduler are system-originated — tenant comes from the trigger/schedule row, and no originating user_uuid is stamped"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/webhooks.go
  - internal/scheduler/scheduler.go
  - cmd/server/main.go
---

# Webhook- and schedule-triggered runs carry no user identity

**The trap.** Most executions are started by an authenticated user (via `POST
/api/v1/workflow-executions` with gateway headers), so the originating `user_uuid` is stamped into
`input._ps.user_uuid` and worker nodes can replay it on outbound orchestrator calls. **Webhook
triggers and scheduled fires do not have a user.** A workflow authored to run under both entry points
must not assume an originating user is present.

**What is true.** A webhook trigger (`POST /api/v1/webhooks/{webhook_uuid}`) and a scheduled fire are
**system-originated**: the tenant (company) comes from the stored webhook / schedule row, and the run
starts with a nil user. No `_ps.user_uuid` is stamped. Nodes that require the originating user to
mutate through the orchestrator (runtime/session lifecycle, run-command, git-open-pr, a2a send) will
fail the orchestrator's user-in-company check if they run in such a workflow with no user supplied.

**What a peer/author must do.** For a workflow reachable by webhook or schedule, do not rely on an
inherited `_ps.user_uuid`. Either restrict such workflows to nodes that need only company scope
(DB-direct reads, notifications, LLM, resolve-projects, await-signal), or supply an explicit service
identity where a mutating node needs one. Tenant isolation still holds — the run is tagged with the
row's company — but the *acting user* is absent by design.
