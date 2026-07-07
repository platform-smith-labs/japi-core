---
type: decision
title: "Mutating nodes must carry _ps.user_uuid"
tags: [decision, nodes, auth, orchestrator, user-uuid]
timestamp: 2026-07-07T06:49:45Z
description: "Any node that mutates via orchestrator must be authored with _ps.user_uuid or the call is rejected 401"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - docs/dev/decisions/workflow-nodes-need-user-uuid-for-orchestrator-calls.md
  - internal/workers/nodes/session_start.go
  - internal/workers/nodes/runtime_start.go
---

# Mutating nodes must carry _ps.user_uuid

**Consequence for a peer.** A workflow author must thread `_ps.user_uuid` (the originating identity,
stamped into `workflow.input._ps` at start) into **every** orchestrator-mutating node —
`runtime-start`, `session-start`, `session-send-prompt`, `session-stop`, `runtime-stop`. The
orchestrator re-validates user ∈ company on state-changing calls, so a step missing `user_uuid`
fails **401 "User does not belong to company"** — which, on a teardown step, can abort the chain and
orphan a live runtime. DB-direct **read** nodes (`runtime-status`, `session-status`,
`session-get-last-message`) are exempt (they make no gateway call). When reviewing a workflow
definition, confirm each mutating task references both `_ps.company_uuid` and `_ps.user_uuid`.
