---
type: gotcha
title: "Execution API is start + get only"
tags: [executions, api-surface, readme-drift]
timestamp: 2026-07-07T06:49:45Z
description: "Only start and get-status are implemented; there is no pause/resume/terminate/search despite the README's design intent"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_executions.go
---

# Execution API is start + get only

**The trap.** The README describes an execution lifecycle with pause / resume / terminate / search
controls. A peer that plans around those endpoints will 404 — they are **design intent, not
implemented**.

**What is true.** The live execution surface is exactly two endpoints:
`POST /api/v1/workflow-executions` (start, optionally idempotent via `Idempotency-Key`) and
`GET /api/v1/workflow-executions/{execution_id}` (read the engine status). There is no way through
this API to pause, resume, terminate, or search/list executions.

**What a peer must do.** Model execution interaction as fire-and-poll: start a run, then poll GET
for its status (the `status` field is the async readiness signal). Do not build flows that depend on
cancelling, pausing, or enumerating executions through this service — that capability does not exist
at this commit. Treat the README's broader lifecycle as a roadmap, not the current contract.
