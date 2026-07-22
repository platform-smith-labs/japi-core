---
type: overview
title: "ps-api — API gateway / auth proxy for the frontend"
tags: [gateway, auth, proxy, ps-api]
timestamp: 2026-07-09T10:35:01Z
description: "What ps-api is and its role between the browser frontend and the Platform Smith backend"
repo: ps-api
commit_sha: a4683c0
evidence:
  - cmd/server/main.go
  - cmd/handlers/passthrough.go
  - cmd/handlers/middleware.go
  - cmd/handlers/workflow_definitions.go
  - cmd/models/README.md
---

# ps-api — API gateway / auth proxy

ps-api is Platform Smith's **API gateway**: the single HTTP entry point (port 9004,
`/api/v1/*`) between the browser frontend and the backend. It is a Go service and
deliberately thin — it owns almost no domain logic and no database schema.

Its job is threefold:

1. **Terminate authentication.** Every request is authenticated here: JWT validation
   (Authorization Bearer; `?token=` fallback for browser EventSource/SSE) plus a
   database check that the user belongs to the claimed company. Unauthenticated
   requests never reach the backend.
2. **Serve reads (and some writes) directly from the database.** Most list/detail
   endpoints query the shared platform PostgreSQL database directly,
   always scoped to the caller's company.
3. **Proxy to backend services.** Most state-changing operations are forwarded over
   HTTP with the validated identity injected as trusted gateway headers (see
   context: gateway trust contract). There are **two proxy targets**: the
   **orchestrator** (runtimes, sessions, launches, most mutations) and
   **ps-workflow** (the workflow engine — definitions, executions, approvals,
   inbox), which ps-api fronts by relaying requests verbatim. Proxying comes in
   three flavors — raw passthrough, verbatim relay, and typed proxy (see glossary).

It also serves real-time surfaces (session/launch SSE streams, a terminal
WebSocket) and hosts the Slack connector, which acts as a second trusted gateway
asserting a mapped user's identity through the same path.

Resources are never addressed by tenant id (`company_uuid`) on the wire — resource
payloads keep it internal, though the caller's own `company_uuid` does appear in auth
responses and JWT claims — and the orchestrator's internal API surface is hard-blocked
at this boundary.
