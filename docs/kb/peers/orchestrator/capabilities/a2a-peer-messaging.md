---
type: capability
title: "A2A peer messaging"
tags: [a2a, messaging, cross-project, conversations, websocket]
timestamp: 2026-07-07T00:00:00Z
description: "How one project's session sends a message to another project in the same workspace and receives a reply, plus the REST origination endpoint"
repo: orchestrator
commit_sha: 6843154
evidence:
  - cmd/websocket/a2a_message.go
  - pkg/protocol/protocol.go
  - cmd/handlers/conversations.go
  - cmd/handlers/workspace_summary.go
---

# A2A peer messaging

**What it does.** Agent-to-agent messaging: a session in one project sends a message to
another project in the *same workspace* and receives replies — the cross-project "ask a peer"
channel. The orchestrator is a persist-first dumb router; a reply is just another message in
the same conversation (no reply-specific machinery).

**How a peer interacts.** Two entry points into the same router:
- **From a live agent session** (the common path): the runtime emits the WebSocket command
  `a2a_message`. The agent supplies only `to_project` and the `{type, body}` content; the
  sender's identity and conversation are derived server-side from the originating session, never
  trusted from the agent. The orchestrator replies with an `a2a_result` durability ack and
  delivers to the target as an `a2a_deliver` message.
- **Via REST** (programmatic kickoff / first message with no pre-bound sender): `POST
  /api/v1/conversations/{conversation_uuid}/messages` with `{from_project, to_project, type,
  body, in_reply_to?}` → `202 Accepted` `{message_id, seq}`.

**Observable behavior.** The message is durably persisted *before* any delivery attempt. The
router then resolves the target project's live session and forwards it; if the target has no
live session the row stays `pending` and delivery is retried later (a down peer never fails the
send). The sender's ack/`202` means "durably accepted," not "delivered" or "read." A reply
arrives asynchronously as a *later* message back in the same conversation — waiting for it is
the caller's job, not a synchronous return.

**Contract.**
- WS `a2a_message` (inbound): `{originating_session_id, to_project, message_id, in_reply_to?}` +
  `data:{type, body}`. Ack is `a2a_result` `{accepted, message_id, seq}` or `{accepted:false,
  error}`. Outbound to target: `a2a_deliver` (carries conversation_id, from_project, message_id,
  seq, target session name).
- REST origination: request `OriginateMessageRequest` `{from_project, to_project, type, body,
  in_reply_to?}`; response `OriginateMessageResponse` `{message_id (server-generated), seq}`.
- Peer discovery: `GET /api/v1/workspaces/{workspace_uuid}/agents/summary` lists the workspace's
  runtime instances with a connection tri-state (`connected` | `heartbeat_stale` |
  `disconnected`) — use it to find candidate peers and whether they are live.

**Invariants.**
- **Persist-before-deliver** — no message is lost when the target is down.
- **Idempotent** on `(company, conversation, message_id)`; a redelivery returns the existing row
  rather than duplicating. `message_id` is the caller-enforced idempotency key on the WS path and
  server-generated on the REST path.
- **At-least-once delivery** — the target runtime dedupes by `message_id`.
- **Never crosses tenant or workspace** — sender/target must both be participants of the same
  conversation (which lives in one company+workspace); a mismatched sender is rejected as a spoof.
- **`seq`** is the row's monotonic replay cursor within the conversation.

**Failure modes.**
- Target has no live session → message left `pending`, no error surfaced; sender is not blocked.
- WS path rejections (as `a2a_result` `accepted:false` with a safe generic error): missing/invalid
  `to_project` or `message_id`; unknown originating session; **cross-company sender** ("outside
  connection scope"); `to_project` that resolves to no project ("unknown to_project").
- REST path: unknown/foreign conversation → `404`; `from_project` not a participant (anti-spoof) →
  `403`; `to_project` not a participant / unroutable → `422`.

**Gotchas.**
- Ack ≠ read receipt. `a2a_result.accepted:true` and REST `202` mean the row is durably persisted;
  the peer may not have received it yet.
- The agent never picks the conversation or asserts its own `from_project` on the WS path — both are
  derived from the originating session; a session not yet bound to a conversation is lazily bound.
- Replies land as a separate later turn in the same conversation, correlated by `in_reply_to`
  (an optional reference to a prior `message_id`, not an enforced FK).
- Delivery-failure replay from `pending` rows is a deferred milestone — a persisted-but-undelivered
  message is recoverable but may not auto-drain in every path yet. UNKNOWN whether pending messages
  are redelivered automatically on target reconnect in the current build.

**Business-critical data.** Messages persist as `conversation_message` rows keyed by conversation
and `message_id`, carrying `delivery_state` (`pending`/`delivered`), `seq` (replay cursor),
`in_reply_to`, and the verbatim `{type, body}` content forwarded byte-for-byte to the recipient.
Routing authorization keys off conversation *participants* (a project must be a participant to
originate or receive). Tenant scoping applies as everywhere — see context.
