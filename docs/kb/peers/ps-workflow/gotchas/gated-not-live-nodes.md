---
type: gotcha
title: "runtime-stop and collect-result may return NOT_LIVE"
tags: [nodes, cross-repo-gate, not-live, runtime-stop, collect-result]
timestamp: 2026-07-07T06:49:45Z
description: "Two nodes register but report a NOT_LIVE terminal state until their orchestrator routes are deployed on the target stack"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/nodes/runtime_stop.go
  - internal/workers/nodes/collect_result.go
  - internal/workers/host.go
---

# runtime-stop and collect-result may return NOT_LIVE

**The trap.** `runtime-stop` and `collect-result` are registered task types, so a workflow that
references them publishes and runs fine — but on a stack where their **orchestrator dependency
route is not yet deployed**, they do not do their job. A peer expecting a real teardown or a real
artifact harvest may instead get a distinct not-live outcome.

**What is true.** Each node is guarded by a live flag (`RUNTIME_STOP_LIVE` / `COLLECT_RESULT_LIVE`).
When the flag is off, the node returns a **NOT_LIVE** result — surfaced as a Conductor **FAILED**
carrying a `not_live: true` output marker — never a false COMPLETED. This is deliberate: a
not-yet-deployed route returns 404 for all ids, which the seam would otherwise misread as
"already gone" (false teardown success) or "no artifacts" (false empty harvest). The honest FAILED
prevents that.

**What a peer/author must do.** Do not treat these two nodes as guaranteed-live on every stack.
Because a trailing `runtime-stop` is typically `optional:true`, a NOT_LIVE cleanup can be tolerated
and let the run finish. For `collect-result`, a NOT_LIVE means "no harvest happened," not "zero
artifacts" — branch on the marker, don't assume an empty result set. The flags flip to live once
the orchestrator routes ship on that stack; the nodes' `_ps` input contracts are unchanged either
way.
