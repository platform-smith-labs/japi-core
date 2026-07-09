---
type: capability
title: "Scheduled workflows"
tags: [scheduler, cron, exactly-once, system-originated, multi-tenant]
timestamp: 2026-07-09T10:49:10Z
description: "A background cron scheduler that fires due workflow definitions exactly-once as system-originated, tenant-scoped runs"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/scheduler/scheduler.go
  - internal/scheduler/cron.go
  - cmd/server/main.go
  - internal/tenant/proxy.go
see_also:
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "scheduled runs surface in the executions list like any other run", descriptive: false}
  - {repo: ps-workflow, capability: "Webhook triggers", intent: "the other system-originated way a run is started", descriptive: false}
---

# Scheduled workflows

**What it does.** A background scheduler inside this service fires workflow definitions on a cron
schedule. Each due schedule starts one run of its target workflow, tenant-scoped to the schedule's
company, with no acting user — a system-originated execution.

**How a peer interacts.** A peer does **not** call this over HTTP — there is **no schedule CRUD API
in this repo**. The scheduler is an autonomous internal loop: it wakes on a poll cadence, finds due
schedules, and fires them. Peer relevance is indirect: the runs it starts appear in the normal
executions surface, and firing is safe to run on many pods at once (see Invariants). Schedule rows
themselves are authored **elsewhere** — the owning surface is `UNKNOWN` from this repo.

**Observable behavior.** On each tick the scheduler drains every currently-due schedule, starting one
run per schedule and then advancing that schedule's next fire time to its next cron tick. A started
run is a normal workflow execution and is observed the same way any run is (poll the executions
surface); the scheduler itself returns nothing to a caller.

**Contract.** No request/response — this is a loop, not an endpoint. Per fire, the input passed to the
run is the schedule row's stored input payload (defaulting to empty). The run is stamped with the
schedule's company and **no user** (system origin). A malformed cron or a failed start is logged and
skipped, not surfaced to any caller.

**Invariants.**
- **Gated off by default** — the loop runs only when the scheduler gate (`SCHEDULER_ENABLED`) is on
  *and* workers are enabled; otherwise no schedule ever fires.
- **Exactly-once across pods/restarts** — a due schedule is claimed by an atomic database
  compare-and-swap (single-row lock, skip-locked) that also pushes its next fire time out under a
  short lease before firing, so only one pod fires a given due schedule and a crash mid-fire re-fires
  at most once, never a rapid double-fire.
- **Tenant-scoped** — every fire is bound to the schedule's company via the tenant seam; the scheduler
  never starts a run without a company.
- **Cron is UTC**, evaluated in the server clock; there is no per-schedule timezone.

**Failure modes.** A claim/database error ends the current tick quietly and retries next tick. A bad
cron leaves the short lease in place (the schedule effectively pauses ~1h rather than firing wrongly).
A failed workflow start is logged; the schedule's fire time was already advanced, so it is **not**
retried until its next cron tick.

**Gotchas.**
- **The cron dialect is a restricted subset**: only `*`, a single integer `N`, and `*/N` step per
  field (5 fields: `min hour dom mon dow`). Ranges (`1-5`) and lists (`1,15`) are **not** supported
  and are treated as a bad cron. `*/N` steps from the field minimum (so `*/2` on day-of-month fires
  on 1,3,5…).
- **`SCHEDULER_ENABLED` alone is not enough** — the worker host must also be enabled, or the goroutine
  is never launched.
- A schedule that a peer expects to fire but doesn't may simply not exist here — schedule authoring is
  not a surface this repo exposes (owner `UNKNOWN`).

**Business-critical data.** Depends on the `schedule` table (delivered by db-migration): a per-company
row carrying the target workflow name, the cron string, an optional input payload, an enabled flag,
and the next-fire timestamp used as the due/claim cursor. This repo **reads and advances** these rows;
it does not expose their creation. (Company scoping applies as everywhere — see context.)

**See also / peers.** ps-workflow — *Workflow execution API* (the scheduled run surfaces there like
any other execution). ps-workflow — *Webhook triggers* (the other system-originated trigger).
