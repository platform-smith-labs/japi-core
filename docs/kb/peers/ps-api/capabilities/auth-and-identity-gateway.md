---
type: capability
title: "Auth and identity gateway"
tags: [auth, jwt, identity, multi-tenancy, gateway]
timestamp: 2026-07-07T03:33:49Z
description: "JWT issuance/validation and the trusted identity headers ps-api injects toward orchestrator"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/auth.go
  - cmd/handlers/register.go
  - cmd/handlers/middleware.go
  - cmd/handlers/stream_auth.go
  - cmd/handlers/passthrough.go
  - pkg/httpclient/client.go
  - pkg/config/config.go
see_also:
  - {repo: orchestrator, capability: "Gateway-trusted identity headers", intent: "consumes the X-User-UUID/X-Company-UUID headers this gateway injects (TrustGatewayHeaders)", descriptive: true}
  - {repo: orchestrator, capability: "User and company registration", intent: "creates the user+company record that register proxies to before minting a JWT", descriptive: true}
  - {repo: ps-api, capability: "Realtime streams", intent: "SSE/WebSocket surface that reuses the same two-layer stream auth with the ?token= fallback"}
---

# Auth and identity gateway

**What it does.** ps-api is the platform's sole credential checkpoint: it authenticates end users,
mints the JWT the frontend carries, validates that JWT on every subsequent request, and converts the
validated identity into trusted headers for orchestrator. Peers behind the gateway never see a
password or a JWT — they trust the injected identity headers instead.

**How a peer interacts.**
- `POST /api/v1/auth/login` — email + password; credentials are verified against the gateway's DB.
- `POST /api/v1/auth/register` — proxied to orchestrator to create the user/company; on success the
  gateway mints a JWT locally (same response shape as login).
- Every protected route: `Authorization: Bearer <jwt>`. Streaming/proxied routes (SSE, WebSocket,
  reverse-proxy passthroughs) also accept `?token=<jwt>` because browser EventSource cannot set
  headers.

**Observable behavior.** Login/register return `{token, expires_at (RFC3339), user}` where `user`
carries `user_uuid`, `company_uuid`, `name`, `email`, `company_name`. Auth on typed, SSE, and
passthrough routes is **two-layer**: (1) JWT signature + expiry check, (2) a DB check that the user
actually belongs to the company named in the claims — a valid token alone is not enough (prevents
replaying a company-A token against company-B resources). Exception: the terminal WebSocket route
validates the JWT only (no DB membership check at the gateway). On registration failure the
orchestrator's status code and body pass through unchanged (400/409/etc.).

**Contract.** JWT claims carry `user_uuid`, `company_uuid`, `email`; tokens are signed with the
configured `JWT_SECRET` (required, ≥32 chars — the process refuses to boot otherwise), issuer
defaults to `platform-smith-api`, lifetime defaults to 24h (`JWT_EXPIRATION`). Toward orchestrator
the gateway injects `X-User-UUID` and `X-Company-UUID` on every upstream call — client-library
calls add `X-Request-ID` (and `X-Workspace-UUID` when a workspace is in scope); the raw
reverse-proxy passthrough forwards the identity pair only. These headers are the **trust root**: orchestrator
accepts them as authoritative identity, so orchestrator must only be reachable via the gateway.

**Invariants.**
- ps-api mints and validates the platform JWT; peers behind the gateway consume the injected
  identity headers. (Whether any peer independently validates end-user tokens is UNKNOWN from this repo.)
- Company-membership is re-verified in the DB on every authenticated request (terminal WS excepted), not just at login.
- Identity headers on upstream calls always come from validated claims, never from client-supplied headers.

**Failure modes.** Missing/malformed/expired token, or user-company mismatch → `401` (streaming and
passthrough routes return a bare `401 unauthorized`). Register: orchestrator rejection surfaces with
the orchestrator's own status; an unparsable orchestrator success response → `500`.

**Gotchas.**
- Deployment caution: any peer that verifies these JWTs must share the same `JWT_SECRET`
  (UNKNOWN from this repo whether orchestrator does) — a secret mismatch surfaces only as `401`.
- A token stays valid until expiry (default 24h); there is no revocation list — removing a user from
  a company is what cuts access (the per-request membership check fails).
- `?token=` puts the JWT in the URL: it can land in access logs of intermediaries; intended only for
  the browser-streaming cases that cannot send headers.
- The register proxy sends zero-value identity UUIDs upstream (no authenticated user yet); the real
  identity comes back from orchestrator and is only then baked into the JWT.

**Business-critical data.** The gateway's user/company tables back both credential verification at
login and the per-request user-belongs-to-company check; the JWT itself carries no roles or
permissions — authorization beyond company membership is UNKNOWN — TODO: check downstream handlers.
