---
type: capability
title: "Message bridging (runtime-directed commands + runtime output)"
tags: [controller, websocket, relay, runtime, cross-repo, orchestrator]
timestamp: 2026-07-07T00:00:00Z
description: "Which of the three delivery paths a runtime-directed command takes, and the timing/failure semantics each guarantees"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/orchestrator/relay.rs
  - src/orchestrator/executor.rs
  - src/orchestrator/websocket_client.rs
  - src/websocket/server.rs
  - docs/dev/decisions/relay-pipeline-pattern.md
---

# Message bridging (runtime-directed commands + runtime output)

**What it does.** The controller is a thin bridge between the orchestrator and the runtime
containers it manages: it carries orchestrator-issued, runtime-directed commands downstream and
forwards runtime-emitted output/events upstream. For relay and fire-and-forget commands it reads
**only** `runtime_name` (for routing) and forwards the rest of the payload byte-identical — payload
semantics belong to the orchestrator and runtime, not the controller.

**How a peer interacts.** The orchestrator sends a task naming a `runtime_name`. Every
runtime-directed command falls into exactly one of three delivery paths; adding a new command means
picking one up front (see Contract). The command catalog itself lives in the interface concepts —
this concept is about *which path* and *what to expect from it*.

## Observable behavior

- **Relay (correlated request→response).** The command is forwarded to the runtime; the controller
  awaits the runtime's single correlated reply and converts it into one `task_response` upstream.
  Correlation is by `request_id`/`task_id` — **never** by command name. Bounded by a timeout
  (`CONTROLLER_COMMAND_TIMEOUT_SECS`, default 300s). One request yields exactly one `task_response`.
- **Fire-and-forget.** The command is forwarded and a **synthetic success ACK** returns immediately,
  no reply awaited. The ACK confirms only that the WebSocket write succeeded — it carries no
  stdout/stderr/exit_code. Any real outcome arrives later as an event on the forwarding path.
- **Event forwarding.** Runtime→orchestrator events (streams, lifecycle, launch-build outcomes) are
  forwarded verbatim, enriched with `runtime_name` + `instance_uuid` in metadata (the payload `data`
  is never rewritten). A **generic passthrough default** forwards any *unrecognized* runtime message
  rather than dropping it, so a new runtime event reaches the orchestrator without a controller change.

## Contract — the 3-path classification (pick one when adding a command)

1. **Relay** — the runtime emits exactly ONE reply per request, correlated by `request_id`, fully
   capturing the result, with no intermediate progress events. Correlated commands include the
   execute/spawn/setup/check family (each with a named response kind). If any condition fails, it is
   not a relay.
2. **Fire-and-forget** — the "result" is just "the write succeeded"; the runtime action produces
   async events but no correlated reply. Input-delivery and session-close are of this kind. The
   **launch/build family** (build-image, git-clone setup) also forwards this way but its synthetic
   ACK is **suppressed** — outcomes arrive later as separate forwarded events.
3. **Event forwarding** — a runtime-originated event with no matching controller-initiated request.
   Forwarded verbatim + metadata-enriched. A git-token mint is a bidirectional relay *of this kind*
   (the runtime's mint request is forwarded up; the orchestrator's minted token comes back down).

Why session-input and session-close must NOT be "upgraded" to relay: forcing an uncorrelated command
through relay hangs until timeout then reports a spurious failure on a perfectly healthy runtime.

## Invariants

- **Single correlation key.** There is exactly one — `request_id` in the pending map. No second
  correlation scheme (e.g. by `session_id`) may be introduced.
- **Blind-forward.** For relay + fire-and-forget the controller strips `runtime_name` and forwards
  the remaining payload byte-identical; unknown/future fields survive verbatim (no field-drop).
- **Exactly one `task_response` per relayed request** (no duplicate, no empty placeholder).
- **Never silently dropped.** An unrecognized runtime message is forwarded via the generic default.

## Failure modes

- **Runtime disconnects mid-relay** → the pending request resolves as `RuntimeDisconnected` (a
  failure surfaced immediately), never a hang.
- **Relay timeout** → the request fails with a timeout after `CONTROLLER_COMMAND_TIMEOUT_SECS`; the
  pending entry is cleaned up (no leak).
- **Late/stale reply** arriving after timeout (no pending entry) → still forwarded upstream as a
  `task_response`, not dropped; the orchestrator deduplicates on its side.
- **Runtime handler fails before its normal reply** → a universal error signal resolves the pending
  request as a failed `task_response` immediately (not a 300s timeout). One shared signal covers all
  relay commands — no per-command error wiring.
- **Unparseable orchestrator message** → the controller best-effort probes the raw text to recover
  the `task_id` and synthesizes a FAILED `task_response` so the task doesn't hang in SENT. If the
  `task_id` is unrecoverable, the message is dropped (logged with an audit marker).

## Gotchas

- A fire-and-forget ACK is **not** a result — it means "delivered," not "done." Do not wait on it
  for the command's real outcome; that arrives as a later event correlated by the runtime's own key
  (e.g. `session_id`), not by `request_id`.
- Launch/build commands return no `task_response` at all — the orchestrator must await their
  forwarded `launch_*` / clone-complete events instead.
- Two-response-string commands (a distinct success vs failure wire string) are still relay:
  whichever the runtime emits first wins the `request_id` correlation; command name is never used to
  correlate.
