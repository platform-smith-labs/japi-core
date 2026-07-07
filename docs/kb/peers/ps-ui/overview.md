---
type: overview
title: "ps-ui — Platform Smith web frontend"
tags: [frontend, react, spa, overview]
timestamp: 2026-07-07T06:27:35Z
description: "What ps-ui is, its role in the platform, and what a peer repo can and cannot expect from it"
repo: ps-ui
commit_sha: 1f5f197
evidence: [src/lib/api-client.ts, src/main.tsx, package.json, CLAUDE.md]
---

# ps-ui — Platform Smith web frontend

**What it is.** ps-ui is the Platform Smith web application: a React 19 single-page app (Vite,
TanStack Router, Tailwind) served on port 3000. It is the human-facing surface for every platform
capability — signing in, managing workspaces/projects, launching runtimes, driving Claude/Codex
coding sessions, and configuring integrations, git connections, secrets, and agent definitions.

**Its role for a peer repo.** ps-ui is a **pure consumer**: it holds no business logic of its own and
exposes **no wire API** that another service calls. Every backend interaction goes out through one HTTP
client to the **ps-api gateway** (`VITE_API_URL`, default `http://localhost:9004/api`), which proxies
to the orchestrator. So the peer-relevant question about ps-ui is never "how do I call it" — it is
**"what does ps-ui consume from me, and what will break in the UI if I change it?"**

**Why a backend peer reads this KB.** If you change a request/response contract on ps-api or the
orchestrator, the `self/capabilities/` concepts tell you which ps-ui feature depends on that endpoint
and which fields it reads or sends — so you can tell whether your change is UI-affecting. The KB
describes the **contracts ps-ui consumes**, not ps-ui's internal component tree.

**Shape of the KB.** `capabilities/` is organized by product feature (auth, workspaces & projects,
coding sessions, runtime launcher, readiness, integrations, git, secrets, agent definitions, …). Each
concept names the endpoints it calls and the fields it depends on. Cross-cutting consumption
conventions (auth, error handling, tenancy) live once in `context.md`.

**Not in scope.** ps-ui does not own any database, does not enforce authorization (the gateway and
orchestrator do — ps-ui's client-side guards are advisory UX, never a security boundary), and does not
define the backend contracts — it only consumes them.
