---
type: capability
title: "A2A peer messaging (in-pod endpoints)"
tags: [a2a, messaging, cross-project, a2a-send, a2a-deliver, durability-ack]
timestamp: 2026-07-09T10:42:29Z
description: "The pod-side ends of agent-to-agent messaging: the a2a_send tool (durability-acked outbound) and a2a_deliver (dedup-guarded, delivery-acked inbound into a live session)"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/mcp_server/tools.rs
  - src/mcp_server/pending_a2a_acks.rs
  - src/core/router/mod.rs
  - src/core/router/handlers.rs
  - src/core/router/seen_deliveries.rs
see_also:
  - {repo: orchestrator, capability: "A2A message persistence and routing", intent: "owns the conversation store, addressing, delivery/redelivery, and the a2a_result durability ack; name descriptive"}
  - {repo: runtime, capability: "In-pod MCP tool server", descriptive: false, intent: "the MCP server that advertises a2a_send independently of the seed and extracts the sender's session UUID"}
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
  `{to_project, body, in_reply_to?, to_session?}`. That is the agent's whole input; everything else
  is minted or derived. `to_session` optionally names a **specific session** (by session **name**)
  within the destination project; omit it to reach the project's primary session.
- **Inbound:** the orchestrator sends `a2a_deliver` down the controller WS on the **message** path,
  addressing a target session by its session **name** in metadata with the message body in data.
  The runtime surfaces the body into that live session as user input.

**Observable behavior.**
- **Outbound (`a2a_send`):** the runtime mints a unique `message_id` (idempotency key — never
  agent-supplied), emits `a2a_message` over the WS carrying the sender's session UUID (taken from
  the MCP request header, spoof-resistant) plus `to_project`/`in_reply_to`/`to_session` (the last
  carried **verbatim** and only when set — the runtime does not validate or resolve it), then awaits the
  orchestrator's `a2a_result` — a **durability ack meaning "the orchestrator persisted it", NOT a
  peer reply** — with a 30s timeout and no auto-retry. Accepted → the agent gets
  `{message_id, seq}`; rejected → a tool error carrying the orchestrator's reason. The call never
  blocks on the peer: a reply, if any, arrives later as an independent `a2a_deliver` (a reply is
  just another `a2a_send` addressed back).
- **Inbound (`a2a_deliver`):** **dedup-guarded and delivery-acked** (work-2607090251 A1/A2, was
  fire-and-forget). The runtime reads the `message_id` from metadata, **dedups on `(message_id,
  session name)`** so a redelivery is never double-injected into the session, and emits an
  **`a2a_delivered`** ack — `status:"delivered"` once the body is handed to the session input,
  `status:"failed"`+`error` on a genuine miss (unknown/non-receiving session, injection failure,
  malformed data). The un-correlatable missing-`session_id` case, and any delivery with **no
  `message_id`** on the wire, stay silent — degrade-safe, behaving as the old fire-and-forget path
  (no atomic cross-repo deploy required). The in-pod dedup set is a bounded last-mile guard for the
  LLM context; the durable dedup is the orchestrator's `conversation_message_delivery_status` receipt
  row. Undelivered/`failed` messages remain the orchestrator's to re-drive (sibling ticket A2).

**Contract.** `a2a_send` in: `{to_project, body, in_reply_to?, to_session?}` — empty `to_project`
is rejected before sending; `to_session` (a session **name** in the destination project) is
optional and, when set, rides in the outbound `a2a_message` metadata (omitted, never `null`, when
unset). The runtime does **not** validate `to_session` — the orchestrator resolves and routes it,
and **rejects** an unknown/invalid name (it is not silently redirected to the primary session).
Unknown extra keys in the tool args (e.g. a spoofed `from`/sender) are **ignored, not rejected** —
the sender identity is header-bound, never taken from args. Out: `{message_id, seq}` on acceptance. Wire pair out/in: `a2a_message` /
`a2a_result` (key fields of the ack: `accepted`, `seq`, `error`). Inbound: `a2a_deliver` — target
session name + `message_id` + `from_project` in metadata, `body` in data; answered by an outbound
**`a2a_delivered`** ack (metadata `message_id` + `session_id` name; data `{status:
"delivered"|"failed", error?}`). Conversation/routing fields (`conversation_id`, `from_project`,
`seq`) are filled by the orchestrator, never by the runtime.

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
- `a2a_deliver` only reaches a session that is live and receiving; a miss now emits
  `a2a_delivered{status:"failed"}` (when correlatable) rather than a silent drop, but **re-drive** of
  an undelivered message is still orchestrator-side (sibling ticket A2), not the runtime's.
- The runtime now reads `message_id` from `a2a_deliver` metadata for dedup + the ack, but the body
  **surfaced into the session** still carries no `message_id` — the *agent* cannot populate
  `in_reply_to` from anything in-band; whether the orchestrator embeds the originating id in the body
  is orchestrator-owned (UNKNOWN here). (The runtime's own dedup/ack does not depend on this.)

**See also / peers.** orchestrator — *A2A message persistence and routing* (name descriptive):
conversation store, `to_project` resolution, the durability ack, and pending/redelivery handling.
runtime — *In-pod MCP tool server*: the seam that advertises `a2a_send` and supplies the sender's
session UUID.
