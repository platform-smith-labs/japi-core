---
type: capability
title: "Async session→task completion bridge"
tags: [conductor, sessions, async-completion, idempotency, multi-tenant]
timestamp: 2026-07-09T10:49:10Z
description: "Completes a parked Conductor task when a runtime coding session reports turn-completion or close"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/session_events.go
  - internal/session/store.go
  - internal/completion/completion.go
see_also:
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "the park-style nodes that record the correlation this bridge later completes"}
  - {repo: orchestrator, capability: "Runtime session lifecycle events", intent: "forwards session_closed / agent_turn_completed to this endpoint", descriptive: true}
---

# Async session→task completion bridge

**What it does.** A live runtime coding session drives two workflow nodes that park on *session
events*: `session-send-prompt` (wait=true) and `run-agent-session`. This endpoint is their return
path: when the session reports a turn finished or the session closed, the bridge finds the parked
task and completes (or fails) it, letting the workflow resume. (Readiness parks like `runtime-start`
and `session-start` are NOT completed here — they resolve via their own DB-direct bounded pollers on
runtime/session status; this endpoint only handles the two session-event kinds below.)

**How a peer interacts.** The orchestrator forwards the runtime event as
`POST /api/v1/task-completions` with a JSON body. The trusted tenant is taken from the validated
gateway header (`X-Company-UUID`), never the body.

**Observable behavior.** Two event shapes on the same endpoint, discriminated by `event`:
- `event: "agent_turn_completed"` — one reply finished, the session stays open; completes the parked
  wait-style task as COMPLETED, carrying `stop_reason` / `result` / `turn_id` through as task output.
- otherwise (the session-close bridge) — the whole session ended; `state: "closed"` completes the
  parked task COMPLETED, `state: "crashed"` fails it FAILED.

The call returns synchronously with the outcome; there is no async readiness to poll here — this
endpoint *is* the signal that unblocks the workflow.

**Contract.** In (key fields): `session_id` (required), `event`, `state` (`closed`|`crashed`, for the
close bridge), `company_uuid` (advisory — if present it must equal the authenticated tenant or the
call is rejected 400), and, for `agent_turn_completed`, `stop_reason` / `result` / `turn_id`;
close-bridge extras include `reason` / `exit_code` / `signal`. Out: `{result, session_id}` where
`result` ∈ `completed` | `failed` | `noop_already_terminal` | `noop_no_mapping`. Errors: 401 missing
tenant context; 400 missing `session_id`, bad `state`, or mismatched `company_uuid`; 502 when the
store or the downstream Conductor completion fails transiently (the caller should retry).

**Invariants.**
- Exactly-once completion: a single store-guarded terminal transition, keyed on
  `(session_id, company)`. A live poller fallback and an inbound event race safely — exactly one wins.
- Tenant isolation is enforced here (Conductor is tenant-blind): all correlation access is keyed on
  the trusted company, so a caller can only ever act on its own mappings.
- Idempotent: a replay after a prior completion is a no-op with no Conductor call
  (`noop_already_terminal`).
- Durable correlation: the `session_id→Conductor-task` mapping (written by the parking node) is keyed
  on `company_id` and persisted, so it survives a ps-workflow restart; a startup reconciliation sweep
  re-arms poll-driven parks.

**Failure modes.**
- No correlation for `(session_id, company)` → **benign no-op** (`noop_no_mapping`, HTTP 200, zero
  Conductor calls) — NOT an error. An unknown session and a session owned by a different tenant are
  deliberately indistinguishable (no cross-tenant enumeration).
- Transient downstream Conductor failure (5xx/network) → the local terminal mark is rolled back and
  the endpoint returns 502 so the caller retries.
- Permanent Conductor 4xx (engine task already terminal/gone) → treated as idempotent success
  (`completed`), the mark is kept.

**Gotchas.**
- `session_id` carries the session **name** string the orchestrator emits (confirmed cross-repo),
  NOT a session UUID or runtime/instance identifier — key by that exact name value. `session_uuid`
  and `runtime_name` are separate optional fields.
- A `noop_no_mapping` (or `noop_already_terminal`) 200 does not mean *you* completed anything; only
  `completed` / `failed` reflect a real terminal transition this call performed.
- Only one outstanding prompt per session is assumed for the turn path — completion is by
  `(session_id, company)`, not by `turn_id`.

**Business-critical data.** The session↔task correlation store (`session_task_correlation` table,
durable Postgres backing) holds `session_id`, `company_uuid`, the parked Conductor `workflow_id` /
task ref, an open/terminal status, and a `kind` selecting how it completes (push-on-close vs a bounded
poll). The atomic open→terminal flip on this row is the idempotency and race-resolution primitive.
(Tenant scoping applies as everywhere — see context.md.)

**See also / peers.** ps-workflow *Coding-agent session nodes* write the correlation at park time;
`orchestrator` *Runtime session lifecycle events* is the forwarder that calls this endpoint.
