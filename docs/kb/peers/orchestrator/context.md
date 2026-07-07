---
type: context
title: "System context — who talks to the orchestrator"
tags: [context, data-flow, multi-tenant]
timestamp: 2026-07-07T00:00:00Z
repo: orchestrator
commit_sha: 6843154
---
# System context

**Peers and the data flow.**
- **ps-api** — the frontend API gateway; forwards ps-ui's calls to the orchestrator's REST API.
- **controller** — connects to the orchestrator over WebSocket; manages Docker containers and bridges
  runtime traffic.
- **runtime** — runs inside containers as PID 1; reached via the controller and via A2A.
- Overall: `ps-ui → ps-api → orchestrator ⇄(WebSocket) controller → runtime`.

**Ubiquitous data fact (stated once here, omitted from every capability).** The platform is
multi-tenant: essentially all data and every request are scoped by company and workspace. Peers
should assume tenant/workspace scoping is always applied; capability entries do not repeat it and
only call out data access when a specific business-critical table/column matters.

**How peers reach it.** REST at `/api/v1/*` (via ps-api); WebSocket protocol commands for
controllers/runtimes. Cross-project agent messaging is workspace-scoped (never cross-workspace).
