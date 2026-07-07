---
type: capability
title: "Conversations gateway (multi-project)"
tags: [conversations, multi-project, gateway, workspaces]
timestamp: 2026-07-07T03:33:49Z
description: "Create multi-project conversations and originate messages into them via the API gateway"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/conversations.go
  - cmd/db/conversation_operations.go
  - cmd/models/conversation.go
  - cmd/handlers/workspace_summary.go
see_also:
  - {repo: orchestrator, capability: "Conversation message routing / ensure-session", intent: "owns message persistence, participancy validation, and lazy per-participant session creation on first delivery", descriptive: true}
  - {repo: ps-api, capability: "Realtime streams", intent: "where the session activity produced by a delivered message becomes observable to a client"}
---

# Conversations gateway (multi-project)

**What it does.** Creates a *conversation* — a workspace-scoped address book of participating
projects whose agents can message each other — and forwards message origination into an existing
conversation. A conversation is pure addressing state: creating one binds no session and spawns
nothing.

**How a peer interacts.** All routes require the gateway's user JWT auth.
- `POST /api/v1/workspaces/{workspace_uuid}/conversations` — create a conversation.
- `POST /api/v1/workspaces/{workspace_uuid}/conversations/{conversation_uuid}/messages` —
  originate a message (kickoff / programmatic-test seam; production messages flow agent-to-agent
  inside the orchestrator, not through this route).

**Observable behavior.**
- *Create* is a direct DB write by ps-api (no orchestrator call). Default participants = **every
  live (non-archived) project** in the workspace. `project_uuids` selects a subset;
  `participants` additionally pins a per-participant harness via an `agent_definition_uuid`
  (when non-empty, `participants` is authoritative and `project_uuids` is ignored — no merge).
  Returns `201` with a `Location` header and the created view.
- Optional `branch` / `base_branch` are stored verbatim on the conversation (no git-ref
  validation here); the orchestrator reads them at pod-spawn to pick the branch to check out
  (and, if absent, which base to create it from). Omitted → per-project default, no base.
- *Message origination* is an auth-then-verbatim proxy to the orchestrator, which persists the
  message and runs route → ensure-session → deliver. Returns `202 {message_id, seq}`. Session
  activity is therefore **asynchronous and lazy**: a participant's coding session is created by
  the orchestrator on first delivery, not at conversation-create or at POST time. ps-api exposes
  no readiness read for that delivery; observe the resulting session via the session gateway
  (see_also) or the orchestrator's conversation reads.

**Contract.** Create in — key fields: `project_uuids?`, `participants?[{project_uuid,
agent_definition_uuid?}]`, `branch?`, `base_branch?`. Create out: `{conversation_uuid,
participants[{project_uuid, agent_definition_uuid?}]}` (internal serial ids never leak).
Message body is forwarded opaquely; the orchestrator owns all structural validation, including
requiring `from_project`. Upstream 4xx/5xx are returned verbatim; orchestrator transport
failure surfaces as `503` (network/retries exhausted) or `504` (timeout).

**Invariants.** Participant set is resolved inside the conversation's own (company, workspace)
tenant scope — a conversation never crosses workspace boundaries. Participant inserts are
internally idempotent (a create retry cannot duplicate a participant); there is no separate
add-participants endpoint. Anti-spoof on origination is **participancy-based, not identity-derived**: the
client-supplied `from_project` must be a conversation participant (orchestrator rejects 403),
because this path has no runtime to infer a sender from.

**Failure modes.** Unknown or cross-tenant workspace → `404` (indistinguishable, by design).
`400` listing the offending UUIDs when a requested project is not live in the workspace, an
agent definition is not in the caller's company, or a project repeats across `participants`.
A `400` after the conversation insert leaves an inert zero-participant conversation row whose
UUID is never returned — deliberate, harmless.

**Gotchas.**
- `GET /api/v1/workspaces/{workspace_uuid}/conversations/recent` is a **different sense of
  "conversation"**: it lists the workspace's recently active *coding sessions* (title/first-prompt
  cards, `?limit=` 1–20, default 5, out-of-range → `400` with `error_code=invalid_limit`) — it
  does not list multi-project conversation entities.
- Creating a conversation produces zero runtime activity; nothing happens until a message is sent.
- `202` on a message means durably accepted upstream, not delivered or processed.
- A participant without a pinned agent definition gets the default harness (Claude).

**Business-critical data.** `conversation` (holds `git_branch`/`git_base_branch` — ps-api is the
sole writer; the orchestrator reads them at pod-spawn) and `conversation_participant` (unique per
conversation+project; optional pinned agent definition drives which harness ensure-session uses).
