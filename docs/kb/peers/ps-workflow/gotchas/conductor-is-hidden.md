---
type: gotcha
title: "Conductor OSS is hidden — never call it directly"
tags: [conductor, encapsulation, multi-tenancy, seam]
timestamp: 2026-07-07T06:49:45Z
description: "The durable engine is an internal implementation detail; peers only ever call ps-workflow's L2 HTTP API"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_executions.go
  - internal/tenant/proxy.go
  - internal/workers/host.go
---

# Conductor OSS is hidden — never call it directly

**The trap.** ps-workflow runs on Conductor OSS as its durable engine, but Conductor is a hidden
implementation detail. A peer that discovers the Conductor host/port and calls it directly would
bypass **all** of Platform Smith's multi-tenancy — Conductor is tenant-blind, so a raw call sees
and can act on every tenant's workflows.

**What is true.** The only supported surface is ps-workflow's own L2 HTTP API (`/api/v1`, port
9005). All tenant isolation — name-namespacing, execution tagging, tenant-checked status,
authz — lives in ps-workflow's tenant seam, which is the sole path to the engine. Handlers never
name the raw engine types; even reading an execution's status goes through the seam so a
cross-tenant id is rejected before any engine data is returned.

**What a peer must do.** Treat "the workflow engine" as ps-workflow. Never assume a Conductor API,
UI, or client is reachable or supported. Start/read executions, publish definitions, decide
approvals, and complete tasks only through the L2 endpoints. There is no fallback to "just call
Conductor" — doing so is a cross-tenant leak, not an optimization.
