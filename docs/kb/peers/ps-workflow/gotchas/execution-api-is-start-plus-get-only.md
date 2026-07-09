---
type: gotcha
title: "Execution API has no pause/resume/terminate"
tags: [executions, api-surface, readme-drift]
timestamp: 2026-07-09T10:49:10Z
description: "Start, get-status, and list are implemented; there is no pause/resume/terminate despite the README's design intent"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/workflow_executions.go
---

# Execution API has no pause/resume/terminate

**The trap.** The README describes an execution lifecycle with pause / resume / terminate controls.
A peer that plans around those endpoints will 404 — they are **design intent, not implemented**.

**What is true.** The live execution surface is three endpoints:
`POST /api/v1/workflow-executions` (start, optionally idempotent via `Idempotency-Key`),
`GET /api/v1/workflow-executions/{execution_id}` (read the engine status), and
`GET /api/v1/workflow-executions` (list the company's runs, filterable by workspace/project/
execution_context/status with limit+offset). There is no way through this API to pause, resume, or
terminate a running execution.

**What a peer must do.** Model execution interaction as start → poll → (optionally) list: start a
run, poll GET for its status (the `status` field is the async readiness signal), and use the list
endpoint for run history / dashboards. Do not build flows that depend on cancelling, pausing, or
terminating executions through this service — that capability does not exist at this commit. Treat
the README's broader lifecycle controls as a roadmap, not the current contract.
