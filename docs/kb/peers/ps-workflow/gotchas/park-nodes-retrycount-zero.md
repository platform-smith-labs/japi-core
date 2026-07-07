---
type: gotcha
title: "Park-style task defs must use retryCount:0"
tags: [conductor, taskdef, park, retry, teardown]
timestamp: 2026-07-07T06:49:45Z
description: "A Conductor retry on a park-style node re-fires its parked side effect and wedges the workflow, so those task defs register with retryCount:0"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/host.go
  - internal/workers/nodes/approval.go
  - internal/workers/nodes/sendprompt.go
  - docs/dev/decisions/park-style-taskdefs-need-retrycount-zero.md
---

# Park-style task defs must use retryCount:0

**The trap.** A workflow author who registers a park-style node (`request-approval`,
`session-send-prompt` with wait, `runtime-start`, `session-start`) with Conductor's **default**
retry policy (retryCount 3 / 60s) breaks it. These nodes hold the task IN_PROGRESS and are
completed asynchronously; a business FAILED (e.g. a rejected approval) or a lease lapse is treated
by the engine as retryable, so the engine **re-runs the whole node**, re-parking a task that was
already settled — the workflow wedges and, critically, an `optional:true` gate never falls through
to its always-cleanup teardown (a live runtime is orphaned).

**What is true.** ps-workflow's worker host registers every custom task def with **`retryCount:0`**
by default. These workers own their own durability — the reconciliation sweep re-arms parks after a
restart, and the worker's HTTP client retries transient calls — so engine-level retries are never
wanted. A node opts into retries only when it is idempotent *and* only fails transiently.

**What a peer/author must do.** When authoring a workflow, keep the gate `optional:true` for
teardown to run on every branch — but know that `optional` only falls through **after** retries are
exhausted, so the underlying task def must be `retryCount:0`. If you provision a stack whose task
defs predate this default, the host's `POST /metadata/taskdefs` will not overwrite an existing def
with the old `retryCount:3`; that stack needs a one-time `retryCount:0` PUT (or a taskdef delete +
host restart). A human decision is not retryable — re-running a decided `request-approval` can
never complete and hangs the run.
