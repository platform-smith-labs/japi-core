---
type: context
title: "ps-api system context"
tags: [gateway, tenancy, trust-contract, boot, orchestrator, ps-workflow, postgres]
timestamp: 2026-07-09T10:35:01Z
description: "Who ps-api talks to, the tenant-scoping and gateway-trust facts every capability assumes, and what it needs to boot"
repo: ps-api
commit_sha: a4683c0
evidence:
  - cmd/handlers/passthrough.go
  - pkg/httpclient/client.go
  - cmd/handlers/middleware.go
  - pkg/config/config.go
  - cmd/server/main.go
  - cmd/db/workspace_operations.go
see_also:
  - {repo: orchestrator, capability: "Trusted gateway headers", intent: "consumes the identity headers ps-api injects", descriptive: true}
  - {repo: db-migration, capability: "Database schema migrations", intent: "owns the platform_smith schema ps-api queries", descriptive: true}
---

# System context

## Who talks to whom

- **Upstream (consumer):** the browser frontend **ps-ui**. All its API
  traffic lands here. A **Slack connector** inside ps-api is a second ingress: it maps
  a Slack user to a platform user and asserts that identity through the same trusted
  path.
- **Downstream 1 — orchestrator:** HTTP client at `ORCHESTRATOR_BASE_URL` (required),
  with request timeout and exponential-backoff retries. Most mutations go here.
- **Downstream 2 — ps-workflow:** a second HTTP client at `PS_WORKFLOW_BASE_URL`
  (required), same transport as the orchestrator client. The workflow-* routes
  (definitions, executions, approvals, inbox, task catalog) forward verbatim here —
  see the Workflow gateway capability.
- **Downstream 3 — PostgreSQL:** the shared platform database. **ps-api owns NO
  schema and NO migrations** — the schema is owned by the **db-migration** repo; ps-api
  only reads/writes existing tables. Most reads (and some writes) are served directly
  from this database.

## Ubiquitous data facts (stated once — capabilities do not repeat them)

- **Every query is company-scoped.** The tenant (`company_uuid`) comes from validated
  JWT claims and is applied to every database read and write. Resources are never
  addressed by `company_uuid` on the wire and resource payloads keep it internal —
  though the caller's own `company_uuid` does appear in auth responses and JWT claims.
- **Cross-tenant and not-found are indistinguishable.** A resource that doesn't exist
  and a resource belonging to another company both return the same 404 — existence is
  hidden across tenants.

## Gateway trust contract

ps-api validates the JWT **and** verifies in the database that the user belongs to the
claimed company (401 on any failure). It then injects the validated identity toward the
orchestrator as headers: `X-User-UUID`, `X-Company-UUID`, `X-Request-ID` (and
`X-Workspace-UUID` when a workspace is in scope). **The orchestrator trusts these
headers only because they come from ps-api** — ps-api is the identity boundary; downstream
services are not expected to re-authenticate the end user (whether any peer independently
re-validates the JWT is UNKNOWN from this repo).

## Boot requirements (what an operator must provide)

- `JWT_SECRET` — required, **≥ 32 chars**; ps-api signs and verifies platform JWTs with
  it. (Any peer that independently verifies these tokens must share the same secret —
  whether orchestrator does is UNKNOWN from this repo.)
- `DB_HOST`, `DB_USER`, `DB_NAME` — required (`DB_PASSWORD` and SSL enforced in
  production; port defaults to 5432).
- `ORCHESTRATOR_BASE_URL` — required; boot fails validation without it.
- `PS_WORKFLOW_BASE_URL` — required; boot fails validation without it (the workflow
  proxy target must be injected by the operator, no default).
- `PS_ALLOW_PLAINTEXT_SECRETS=1` — dev-only gate for integration-credential writes
  until a KMS key provider lands. Without it the gateway still boots, but any
  integration-credential write fails closed with a clear error (never a silent
  plaintext write).

## Hard boundary

`/api/v1/internal/*` is blocked at this gateway with a bare 404 (no descriptive body)
and is **never** forwarded to the orchestrator, regardless of authentication.
