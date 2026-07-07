---
type: context
title: "System context & ubiquitous consumption conventions"
tags: [context, api-client, auth, tenancy, conventions]
timestamp: 2026-07-07T06:27:35Z
description: "Who ps-ui talks to and the cross-cutting rules every capability shares — stated once here"
repo: ps-ui
commit_sha: 1f5f197
evidence: [src/lib/api-client.ts, src/stores/auth.ts, src/lib/query-client.ts, docs/dev/decisions/no-uuid-in-ui.md, docs/dev/decisions/react-architecture-patterns.md]
---

# System context & ubiquitous consumption conventions

These facts hold for **every** capability. They are stated here once; individual `capabilities/`
concepts name only their own endpoints and do not restate any of this.

## Who ps-ui talks to

ps-ui talks to exactly one backend: the **ps-api gateway** at `VITE_API_URL` (default
`http://localhost:9004/api`). All endpoint paths in this KB (e.g. `POST /v1/auth/login`) are relative
to that base. ps-api authenticates and proxies to the **orchestrator**, which is the true owner of
platform state; ps-ui never contacts the orchestrator, controller, or runtimes directly. A backend
peer that changes a contract ps-ui consumes is almost always changing it on **ps-api / orchestrator**.

## Authentication — Bearer JWT on every call

Every request carries `Authorization: Bearer <jwt>` when a session exists. The JWT is obtained from
`POST /v1/auth/login` and persisted in browser localStorage (zustand `persist` key `ps-auth`). This is
ubiquitous — no capability concept repeats it. A `401` whose body is `{"error":{"code":401,...}}` with
an active token triggers an automatic client-side logout (token treated as expired); `401`s from the
login call itself (no active token) are ordinary wrong-credential errors and do **not** log out.

## Multi-tenancy & scoping (ubiquitous data facts)

The platform is multi-tenant. Every user belongs to a **company** (`company_uuid`) and operates within
a **workspace** (`workspace_uuid`); most resources (projects, runtimes, connections, secrets, agent
definitions) are scoped by workspace and/or company. ps-ui keeps the selected workspace in a persisted
store and threads its UUID into scoped endpoints. Authorization for all of this is enforced
**server-side** (gateway/orchestrator); ps-ui's route/membership guards are advisory UX that can
fail-open, never a security boundary. Individual concepts do not restate the tenant/workspace scoping —
assume it applies.

## Error handling — normalized shape

All backend calls go through one client that normalizes errors into `ApiClientError { status, message,
details }`. The client accepts **both** a flat `{"message":"..."}` and a nested
`{"error":{"code":N,"message":"..."}}` body, and tolerates an empty body on a 2xx (e.g. a 201 that
returns no JSON). A peer changing an error shape should stay within one of those two forms or ps-ui
will surface a generic `Request failed: <status>`.

## Server-state conventions

Server state lives in **TanStack Query** (React Query), keyed per resource (e.g. `['workspaces']`,
`['projects', workspace_uuid]`), with invalidation on mutation. Data-fetching lives only in
`*.container.tsx` files; presentational components receive data via props. A peer does not need these
internals except to know that ps-ui **caches** responses and re-fetches on invalidation, so a
contract's read shape must stay stable across a resource's lifecycle.

## Identifier discipline (no-UUID-in-UI)

Resources are keyed by UUID on the wire, but ps-ui **never displays raw UUIDs or internal IDs** to the
user (team decision `no-uuid-in-ui`). It shows human names and keeps UUIDs internal. Some routes key a
resource by **name** rather than UUID (notably coding sessions by `sessionName`). When a peer returns
an identifier, ps-ui needs a human-facing name field alongside it, or a resolvable name→id route.
