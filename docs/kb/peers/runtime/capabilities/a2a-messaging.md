---
type: capability
title: "A2A peer messaging (in-pod endpoints)"
tags: [a2a, messaging, cross-project, a2a-send, a2a-deliver, durability-ack]
timestamp: 2026-07-06T23:40:38Z
description: "The pod-side ends of agent-to-agent messaging: the a2a_send tool (durability-acked outbound) and a2a_deliver (fire-and-forget inbound into a live session)"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/mcp_server/tools.rs
  - src/mcp_server/pending_a2a_acks.rs
  - src/core/router/mod.rs
  - src/core/router/handlers.rs
see_also:
  - {repo: orchestrator, capability: "A2A message persistence and routing", intent: "owns the conversation store, addressing, delivery/redelivery, and the a2a_result durability ack; name descriptive"}
  - {repo: runtime, capability: "In-pod MCP tool server", intent: "the MCP server that advertises a2a_send independently of the seed and extracts the sender's session UUID"}
---

# A2A peer messaging (in-pod endpoints)

**What it does.** Lets a coding agent in this pod message an agent in another project and receive
peer messages into its live session — the pod-side endpoints of the platform's agent-to-agent
channel. The orchestrator owns persistence, addressing, and routing; the runtime only sends and
surfaces.

**How a peer interacts.**
- **Outbound:** the agent calls the `a2a_send` MCP tool — advertised in every `tools/list` reply
  that carries a valid session UUID, independent of the granted-tools seed (a tool-blind session —
  no valid `X-PS-Session-ID` — sees no tools at all, `a2a_send` included) — with
  `{to_project, body, in_reply_to?}`. That is the agent's whole input; everything else is minted or
  derived.
- **Inbound:** the orchestrator sends `a2a_deliver` down the controller WS on the **message** path,
  addressing a target session by its session **name** in metadata with the message body in data.
  The runtime surfaces the body into that live session as user input.

**Observable behavior.**
- **Outbound (`a2a_send`):** the runtime mints a unique `message_id` (idempotency key — never
  agent-supplied), emits `a2a_message` over the WS carrying the sender's session UUID (taken from
  the MCP request header, spoof-resistant) plus `to_project`/`in_reply_to`, then awaits the
  orchestrator's `a2a_result` — a **durability ack meaning "the orchestrator persisted it", NOT a
  peer reply** — with a 30s timeout and no auto-retry. Accepted → the agent gets
  `{message_id, seq}`; rejected → a tool error carrying the orchestrator's reason. The call never
  blocks on the peer: a reply, if any, arrives later as an independent `a2a_deliver` (a reply is
  just another `a2a_send` addressed back).
- **Inbound (`a2a_deliver`):** strictly **fire-and-forget**. Any miss — missing target session id,
  unknown session, session in a terminal/closing state, or input-injection failure — is logged and
  **dropped with no response envelope** and no fallback session guessing. This is deliberately a
  different failure posture from session-input commands (which return an `error_response` on
  failure): the sender already got its durability ack at log time, and undelivered messages remain
  the orchestrator's to track and redeliver.

**Contract.** `a2a_send` in: `{to_project, body, in_reply_to?}` — empty `to_project` is rejected
before sending. Out: `{message_id, seq}` on acceptance. Wire pair out/in: `a2a_message` /
`a2a_result` (key fields of the ack: `accepted`, `seq`, `error`). Inbound: `a2a_deliver` — target
session name + `from_project` in metadata, `body` in data. Conversation/routing fields
(`conversation_id`, `from_project`, `seq`) are filled by the orchestrator, never by the runtime.

**Invariants.** Ack = durably logged, never delivered-or-read. `message_id` is runtime-minted, one
per send. The a2a ack correlation state is isolated from the generic tool-call path — a2a traffic
cannot interfere with `agent_tool_result` correlation. Outbound identifies the *sender* by session
**UUID**; inbound targets the *recipient* by session **name** (two different identifier spaces).

**Failure modes.** WS down → immediate tool error, nothing queued. No `a2a_result` within 30s →
"orchestrator timed out" tool error (the message may still have been persisted — the agent decides
whether to resend; `message_id` differs per attempt). Orchestrator rejection (e.g. a closed
conversation) → tool error with the reason. Inbound delivery misses are invisible to the sender.

**Gotchas.**
- Do not wait synchronously for a peer reply after `a2a_send` — success only means "logged".
  Replies land as a later, unrelated-looking `a2a_deliver` turn.
- A delivered message arrives inside the session as plain user input; there is no in-band envelope
  distinguishing it from a human message beyond what the orchestrator puts in the body. UNKNOWN —
  whether the orchestrator prefixes sender attribution into the body.
- `a2a_deliver` only reaches a session that is live and receiving; whether a message to a
  non-receiving session is redelivered later is orchestrator-side behavior, not the runtime's.
- The delivered payload carries no `message_id` — a recipient cannot populate `in_reply_to` from
  anything the runtime surfaces; whether the orchestrator embeds the originating message id in the
  body is orchestrator-owned (UNKNOWN here).

**See also / peers.** orchestrator — *A2A message persistence and routing* (name descriptive):
conversation store, `to_project` resolution, the durability ack, and pending/redelivery handling.
runtime — *In-pod MCP tool server*: the seam that advertises `a2a_send` and supplies the sender's
session UUID.
