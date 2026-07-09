---
type: capability
title: "Signal wait & unpark (Model-B)"
tags: [signal, await-signal, unpark, correlation-id, exactly-once, human-in-the-loop, webhook]
timestamp: 2026-07-09T10:49:10Z
description: "The await-signal node parks a workflow task on a minted correlation_id; POST /api/v1/signals unparks it — the general wait-for-an-external-decision primitive."
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/await_signal.go
  - cmd/handlers/signals.go
  - internal/signal/pg_store.go
  - internal/signal/sweep.go
  - internal/completion/completion.go
  - cmd/handlers/middleware.go
  - internal/workers/nodes/common.go
  - internal/workers/host.go
see_also:
  - {repo: ps-workflow, capability: "Human approval gate", intent: "await-signal generalizes and subsumes the legacy request-approval gate (shape=approval)", descriptive: false}
  - {repo: ps-workflow, capability: "Async session→task completion bridge", intent: "shares the same exactly-once CompleteTurn completion core", descriptive: false}
  - {repo: orchestrator, capability: "ps_signal forward", intent: "forwards an agent's ps_signal to POST /api/v1/signals with the trusted company header, no user", descriptive: true}
---

# Signal wait & unpark (Model-B)

**What it does.** Pauses a running workflow at a point where it must wait for an external decision or
event — a human approval, an agent's own signal, a peer message, or a webhook — and resumes it when
that signal arrives. This is the general "wait for the outside world" primitive; it subsumes the
legacy request-approval gate (which is now just `shape=approval`).

**How a peer interacts.** Two surfaces. Inside a workflow, the **`await-signal`** node parks the task
and emits a `correlation_id` the workflow injects into the world (an agent prompt, a notification
deep-link, or an expected webhook body). To resume it, a caller sends **`POST /api/v1/signals`** with
that `correlation_id`. The signal endpoint IS the completion signal — there is nothing to poll; the
world echoes the opaque `correlation_id` back when the awaited event happens.

**Observable behavior.** On park, `await-signal` holds the task in-progress and outputs
`awaiting_signal:true`, the `correlation_id`, the resolved `source`, and `shape`. The task stays
parked until either a signal lands or its deadline passes. A `COMPLETED` signal resumes the workflow
past the node; a `FAILED` signal (a deny, or a timeout) fails the parked task. If a `timeout_seconds`
deadline is set and elapses with no signal, a background sweep fails the wait (deny semantics) with
reason `await-signal timed out`.

**Contract.** Node inputs (`_ps` annotations): `source` (`human|agent|a2a|webhook`, default `agent`),
`shape` (`freeform|approval`, default `freeform`), `timeout_seconds` (int; `0` = no deadline). Node
park output key fields: `awaiting_signal`, `correlation_id`, `source`, `shape`. Endpoint request key
fields: `correlation_id` (required), `company_uuid` (optional/advisory — must match the authenticated
tenant if present), `status` (`COMPLETED` default | `FAILED`), `payload` (map; a `state` key is
auto-filled `completed`/`failed` when absent). Endpoint response: `result` (`completed` |
`noop_no_mapping` | `noop_terminal`) and the echoed `correlation_id`. Errors: `400` for a missing/empty
`correlation_id` or a `company_uuid` that mismatches the tenant; `401` when the `X-Company-UUID`
tenant header is missing/invalid; `502` when the underlying engine completion fails.

**Invariants.** Exactly-once: completion runs through the same store-guarded core the async session
bridge uses — concurrent signals (or a signal racing the timeout sweep) resolve so exactly one
performs the terminal transition; replays are idempotent no-ops. The `correlation_id` is
**deterministic** from the workflow+task identity, so a task redelivered after a lease lapse recomputes
the same id and re-parks without creating a duplicate wait. Everything is tenant-scoped: the store
lookup is keyed on `(company, correlation_id)`, which is itself the tenant gate — an unknown or
cross-tenant id resolves to a benign no-op with no enumeration. Auth is **company-only**: a Model-B
signal carries no user (the agent source is forwarded by a peer with the trusted company header but no
user identity); a user header is honored when present (the human path) but never required.

**Failure modes.** Signalling an unknown or already-resolved wait returns `200` with `result:
noop_no_mapping` or `noop_terminal` — not an error, but also **not** a real completion. A transient
engine failure surfaces as `502` and leaves the wait open for a later retry. A wait past its deadline
is failed by the sweep, so a workflow branch keyed on the node's failure (deny) runs.

**Gotchas.** The node is **env-gated** (`AWAIT_SIGNAL_LIVE`): when off, it does not park — it returns
NOT_LIVE (a Conductor `FAILED` carrying a `not_live` marker, an honest terminal state, never a false
success). A `noop_*` result means the signal hit nothing live — treat it as "did not complete a wait,"
not success. The `correlation_id` is **opaque** and not user-facing; it must be carried verbatim to
whatever eventually signals back. `await-signal` is the general form of the older request-approval
gate — new "wait for a decision" flows should use it, with `shape=approval` for the approval case.

**Business-critical data.** Open waits persist to the `signal` correlation table (db-migration
migration `0056`), unique on `(company, correlation_id)`, with an `open|completed|failed` status and a
nullable `deadline` the timeout sweep scans. The row is the exactly-once anchor: its status transition
is the atomic claim that guarantees one winner. (Tenant scoping is ubiquitous — see context.md.)

**See also.** ps-workflow **Human approval gate** (subsumed by this node's `shape=approval`);
ps-workflow **Async session-to-task completion bridge** (shares the same exactly-once completion core);
orchestrator **ps_signal forward** (delivers an agent's signal to the endpoint with the trusted
company header and no user).
