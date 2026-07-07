---
type: decision
title: "Status reads are DB-direct; no dependency on ps-api"
tags: [decision, layering, db-direct, orchestrator, ps-api]
timestamp: 2026-07-07T06:49:45Z
description: "ps-workflow reads runtime/session status straight from the shared platform DB and mutates via orchestrator — it never calls ps-api"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - docs/dev/decisions/ps-workflow-reads-db-direct-no-ps-api.md
  - internal/workers/nodes/reads.go
see_also:
  - {repo: orchestrator, capability: "Runtime and session lifecycle", intent: "owns the mutations ps-workflow calls and the tables it reads", descriptive: true}
---

# Status reads are DB-direct; no dependency on ps-api

**Consequence for a peer.** ps-workflow's node read paths (runtime/session/last-message/artifact
status) read the shared `platform_smith` Postgres directly, and its mutations go to the
**orchestrator** HTTP API. It has **no** dependency on ps-api — no ps-api client, base URL, or
token. A peer wiring or deploying ps-workflow does not need ps-api reachable; it does need shared-DB
read access and the orchestrator mutation routes. Because reads make the DB **schema the contract**,
an additive column change is safe but a breaking change to a shared runtime/session column fans out
to ps-workflow as a direct reader (coordinate such migrations in `db-migration`).
