---
type: context
title: "ps-workflow — system context & ubiquitous data facts"
tags: [context, multi-tenancy, auth, conductor, orchestrator, shared-db]
timestamp: 2026-07-07T06:49:45Z
description: "Who ps-workflow talks to, and the tenancy/auth/data facts stated once so no capability repeats them"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/tenant/proxy.go
  - internal/platform/platform.go
  - cmd/handlers/middleware.go
  - cmd/handlers/workflow_executions.go
  - cmd/handlers/session_events.go
  - docs/dev/decisions/ps-workflow-reads-db-direct-no-ps-api.md
  - docs/dev/decisions/workflow-nodes-need-user-uuid-for-orchestrator-calls.md
  - docs/dev/decisions/correlation-store-keys-on-company-id.md
---

# ps-workflow — system context

ps-workflow is a backend service that fronts **Conductor OSS** and talks to three peers: **ps-api /
ps-ui** call its HTTP API (author definitions, start/read executions); **orchestrator** receives its
runtime/session mutations and forwards session completion events back; **runtime coding sessions**
generate those completion events. Conductor sits *behind* ps-workflow and is never exposed. These
facts hold on **every** call — state them here once; no capability concept repeats them.

## Multi-tenancy — enforced entirely here (Conductor is tenant-blind)

Conductor has no notion of tenants; 100% of isolation is enforced by ps-workflow on every engine call:

- **Name-namespacing.** A tenant's workflow names are prefixed with the company UUID before
  registration into Conductor, so two tenants' identically-named workflows never collide.
- **Execution tagging.** Every started execution is tagged with the tenant (both the Conductor
  `correlationId` and `input._ps.company_uuid`) so a status read can be filtered to the caller.
- **Tenant-checked status.** A status read whose execution tag does not match the caller is reported
  as **not-found** — unknown and cross-tenant are indistinguishable (no existence leak).

There is a single handler-reachable path to Conductor (the **tenant seam**); the raw engine client is
not injected into handlers, so a bypass is a compile error, not a review miss. The one deliberate
exception is the tenant-**blind** worker poll loop, which serves the global node queue and discovers
the tenant from the polled task itself.

**Auth.** Callers authenticate with the shared-platform JWT (`company_uuid` / `user_uuid` claims).
HTTP handlers read the trusted identity from the **`X-User-UUID` / `X-Company-UUID`** gateway headers,
never from the request body (a body-supplied tenant is at most advisory and must match). The
authenticated `user_uuid` is stamped into `input._ps.user_uuid` at execution start so workers can
replay the originating identity on outbound calls.

## Data access — DB-direct reads, orchestrator mutations

- **Reads are DB-direct.** Node read-paths (runtime status, session state, last agent message) read
  straight from the shared **`platform_smith`** Postgres — the same pattern ps-api uses. ps-workflow
  has **no dependency on ps-api** (a backend depending on the frontend gateway would be a layering
  inversion). Every DB-direct read is tenant-scoped on `company_uuid`, returning not-found for both a
  miss and a cross-tenant row.
- **Mutations go to orchestrator.** State-changing runtime/session operations (launch, create-session,
  send-input, stop, release) are HTTP calls to the **orchestrator** API, carrying the originating
  identity as gateway headers. Orchestrator re-validates user ∈ company, so a mutating node **must**
  carry `_ps.user_uuid` or the call fails 401; DB-direct read nodes do not need it.
- **Schema-as-contract.** There is no shared table-model package; each service hand-writes its own SQL.
  An additive schema change breaks no reader, but a breaking change to a shared column fans out to
  every direct reader. **Migrations live in the `db-migration` repo**, not here.

## The correlation store (async bridge state)

The async bridge that completes parked Conductor tasks is backed by a **durable** sessionId→Conductor-
task correlation store. It is keyed on the integer **`company_id`** plus the session name, matching
the platform's tenant-table convention. It
survives a ps-workflow restart, and a startup reconciliation sweep re-arms in-flight parks. Completion
is idempotent and tenant-scoped: an unknown/cross-tenant session or a replay is a benign no-op.
