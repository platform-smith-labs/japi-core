---
type: capability
title: "A2A messaging node"
tags: [a2a, messaging, cross-project, workflow-node, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "The a2a workflow node — send a message from a workflow to another project's agent; replies collected via await-signal(source=a2a)"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/a2a.go
  - internal/workers/nodes/a2a_client.go
  - internal/workers/nodes/common.go
see_also:
  - {repo: ps-workflow, capability: "Signal wait & unpark (Model-B)", intent: "how a workflow collects the A2A replies"}
  - {repo: orchestrator, capability: "A2A conversation surface", intent: "owns message origination, delivery, and user↔company re-validation", descriptive: true}
---

# A2A messaging node

**What it does.** A workflow node (task type `a2a`) that lets a running workflow send an
agent-to-agent message to another project's agent — the cross-project "ask a peer" channel,
initiated from workflow logic rather than from a live agent session. The node does not run the
coordinator loop; it only originates the message onto the orchestrator's conversation surface.

**How a peer interacts.** A workflow authors an `a2a` task whose `inputParameters._ps` carries
`action` and the send fields. Only the `send` action is live this cycle. key `_ps` fields for
`send`: `action` (must be `send`), `conversation_id`, `from` (alias `from_project`), `to` (alias
`to_project`), `message` (alias `body`), `type` (optional, default `"text"`), and `user_uuid` (the
originating user). This node is a *producer* — replies are not returned by it (see Gotchas).

**Observable behavior.** `send` is synchronous and terminal: the node originates the message and
completes in one step (no parking, no poll). On success the task completes with the message's
durable identity. `start-conversation` and `broadcast` are declared but NOT built on this stack —
they fail honestly (see Failure modes), never returning a fabricated success.

**Contract.** In (`send`): key `_ps` fields above; `conversation_id`, `from`/`from_project`, and
`to`/`to_project` are required (empty → error). Aliases resolve first-non-empty with the `_project`
variant preferred; `message`/`body` likewise. Out (`send`): `action` (`"send"`), `message_id`,
`seq`, `delivered` (`true`). The actual message origination — persistence, delivery, and the
`user_uuid`↔company re-validation — is performed by the orchestrator A2A conversation surface, so a
malformed send or a foreign/unknown target surfaces as *that* upstream's error, wrapped here as a
FAILED task.

**Invariants.** The node requires a wired A2A client; when unwired it is inert rather than silently
no-op (see Failure modes). `send` never parks — a completed `a2a` send task means the message was
durably accepted by the conversation surface, not that it was read. The originating `user_uuid` must
be supplied for `send`; tenant/company scoping applies as everywhere (see context.md) and the
originating user is re-validated against the company by the orchestrator, not here.

**Failure modes.**
- Client not wired on this stack → the task is reported NOT_LIVE (not FAILED) — an explicit "a2a not
  live on this stack" signal, so a peer can distinguish "capability absent" from "call rejected".
- `action` not one of `start-conversation | send | broadcast` → FAILED with a validation reason.
- `action = start-conversation` or `broadcast` → FAILED with an honest "not available on this stack:
  only send is built" reason (these are the deferred orchestrator P4 surface). This is a FAILED task,
  distinct from the NOT_LIVE unwired case.
- `send` missing `conversation_id` / `from` / `to` → FAILED.
- Origination rejected upstream (e.g. foreign or unknown target, invalid user) → FAILED carrying the
  orchestrator's error text.

**Gotchas.**
- Only `send` is live. Do not author `start-conversation` or `broadcast` yet — they FAIL honestly.
- The node does not collect replies. Delivery-accepted (`delivered: true`) is not a read receipt and
  not a reply. To gather answers, pair the send with an `await-signal` node using `source=a2a`
  (broadcast would pair with `count=N`); the reply arrives as a signal that unparks the waiter.
- `delivered: true` is always the send success shape — it asserts durable acceptance by the
  conversation surface, nothing about the peer having a live session or having responded.
- `from`/`to` accept both the bare and `_project`-suffixed names; the `_project` form wins when both
  are set.

**See also / peers.**
- ps-workflow — *Signal wait & unpark (Model-B)*: how a workflow parks for and collects the A2A
  reply(ies) via `await-signal(source=a2a)`.
- orchestrator — *A2A conversation surface*: owns message origination, cross-session delivery, and
  the `user_uuid`↔company re-validation this node forwards to.
