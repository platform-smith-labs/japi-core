---
type: capability
title: "Webhook triggers"
tags: [webhook, trigger, ingress, system-originated, public-endpoint, hmac]
timestamp: 2026-07-09T10:49:10Z
description: "Public token-authed endpoint that starts a run of a webhook's bound published definition; tenant + definition come from the stored row, never the request"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/webhooks.go
  - internal/webhook/store.go
  - cmd/handlers/middleware.go
  - cmd/server/main.go
see_also:
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "starts a run of a definition the same way this ingress does", descriptive: false}
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "owns the published definition this ingress runs", descriptive: false}
---

# Webhook triggers

**What it does.** A public, token-authenticated ingress that lets an **external system** kick
off a workflow run by hitting one URL. It fires the published workflow definition bound to that
webhook — the caller supplies only a payload, never a tenant or a definition name. This is the one
route on the service that is **not** gateway-authed.

**How a peer interacts.** `POST /api/v1/webhooks/{webhook_uuid}` with header
`X-Webhook-Token: <token>` and a JSON body. The `{webhook_uuid}` is a **public, non-secret**
identifier that lives in the URL; the secret is the token, carried only in the header. This
public-uuid / secret-token split is deliberate — the uuid may appear in logs or configs; the token
must not.

**Observable behavior.** On success the ingress resolves the webhook row, verifies the token,
starts a run of the bound published definition for that row's company, and returns `200` with
`result: "started"` and an `execution_id`. The run starts **system-originated**: it carries **no
user identity**, and its tenant (company) comes entirely from the stored webhook row — the request
body and headers cannot influence which tenant runs. The body is passed into the run input under
the key **`webhook`** (the run sees `{ "webhook": <your-json> }`).

**Contract.** In: path `webhook_uuid` (required, uuid) + header `X-Webhook-Token` + an arbitrary
JSON body (read up to ~1 MiB; empty/unparseable body yields a null `webhook` input, not an error).
Out: `{ result, execution_id }`. The response is deliberately minimal — it never echoes the tenant
or definition. The returned `execution_id` is the same run handle other execution reads key off.

**Invariants.** The tenant + target definition are read **from the row**, never from the request —
enforced here, and the security boundary of the whole feature. The token is stored as a hash
(HMAC-SHA256 with a server-side pepper) and checked in **constant time**; the plaintext token is
never persisted. A run is only started if the webhook is **found, enabled, and published**. Auth is
the bearer token alone — no user or company header is trusted on this route.

**Failure modes.** `401` when `X-Webhook-Token` is missing or fails the constant-time check
(a known uuid with a stale/rotated token gets `401`, distinct from `404`). `404` when the webhook
uuid is unknown, disabled, or points at an unpublished definition — these are **indistinguishable**,
so there is no existence or cross-tenant enumeration leak. `404` **also** when the feature is not
enabled on the stack (see gotchas). `502` when the row lookup fails or the workflow fails to start.

**Gotchas.**
- **Gated — the route hides itself.** When `WEBHOOK_TRIGGER_ENABLED` is off (no webhook store
  wired) or the HMAC pepper is unset, the ingress returns `404` for **every** request. A `404` here
  does not tell a caller whether the feature exists on that stack.
- **Not idempotent.** Every valid trigger starts a **new** run and stamps the webhook's
  last-fired time — there is no dedup key. A caller that retries fires the workflow again. Retries
  are the caller's responsibility.
- **No user, no `_ps` context.** Because the run is system-originated, downstream logic that
  expects an initiating user will find none; the execution is tagged with the webhook's company
  only.

**See also / peers.** Same-repo **Workflow execution API** (this ingress starts a run through the
same tenant-scoped Conductor start path a normal execution uses) and **Workflow definition
registry** (owns the published definition this webhook runs). The caller is any **external system**
holding the uuid + token — there is no specific peer repo.
