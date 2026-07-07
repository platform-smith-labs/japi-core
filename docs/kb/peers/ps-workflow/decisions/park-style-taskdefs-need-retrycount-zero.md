---
type: decision
title: "Park-style task defs register with retryCount:0"
tags: [decision, conductor, taskdef, retry, teardown]
timestamp: 2026-07-07T06:49:45Z
description: "Custom park-style worker task defs default to retryCount:0 so an engine retry can't re-fire a settled park and wedge the workflow"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - docs/dev/decisions/park-style-taskdefs-need-retrycount-zero.md
  - internal/workers/host.go
---

# Park-style task defs register with retryCount:0

**Consequence for a peer.** The worker host registers every custom node's Conductor task def with
`retryCount:0` by default. This matters to a workflow author: an `optional:true` gate only falls
through to always-cleanup teardown **after** retries are exhausted, so with the default-3 policy a
rejected `request-approval` (or any park-style node) would re-park and hang instead of tearing down.
With `retryCount:0` the branch proceeds immediately. Two operational notes: (1) a node opts back
into retries only when idempotent and transient-failing; (2) on a stack whose task defs predate this
default, registration will **not** overwrite an existing `retryCount:3` def — that stack needs a
one-time `retryCount:0` PUT (or a taskdef delete + host restart) before park behavior is correct.
