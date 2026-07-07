---
type: capability
title: "Authentication & session lifecycle"
tags: [auth, jwt, session]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui logs in/registers a user, stores the JWT, injects it as a Bearer, and auto-logs-out on token expiry."
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/auth.ts
  - src/stores/auth.ts
  - src/lib/api-client.ts
  - src/routes/_auth.tsx
  - src/routes/login.tsx
  - src/routes/signup.tsx
  - src/lib/auth/redirect.ts
  - src/components/auth/LoginForm.container.tsx
  - src/main.tsx
see_also:
  - {repo: ps-api, capability: "Auth login + JWT issuance", intent: "issues the JWT and user identity ps-ui stores on login/register", descriptive: true}
  - {repo: ps-api, capability: "JWT validation on gateway requests", intent: "returns the 401 that drives ps-ui's auto-logout when a token expires", descriptive: true}
---

# Authentication & session lifecycle

**What it does.** Signs a user into Platform Smith and holds the session for the whole SPA:
email/password login and self-service registration, JWT persistence across reloads, automatic
Bearer attachment on every backend call, and automatic sign-out when the session expires.

**Backend contracts consumed.**

- `POST /v1/auth/login` — body `{email, password}`. On success ps-ui reads response `token`
  (the JWT), `expires_at`, and `user` (key fields: `user_uuid`, `company_uuid`, `name`, `email`,
  `company_name`). ps-ui persists `token` + `user` and treats the call as the session origin.
- `POST /v1/auth/register` — body `{name, email, password, company_name}`. Returns the **same**
  login response shape (`{token, expires_at, user}`) and is consumed identically — a successful
  register immediately establishes an authenticated session (no separate login round-trip).

ps-ui depends on `token` being a Bearer-usable JWT and on the `user` sub-object carrying at least
`user_uuid`/`company_uuid`/`company_name` for display and downstream scoping. `expires_at` is
received but ps-ui does not currently act on it — expiry is discovered reactively via 401 (below).

**Observable behavior.**
- On login/register success the token+user are written to a persisted store and the app navigates
  to a validated redirect target (else `/`).
- The stored token is attached as `Authorization: Bearer <token>` on **every** subsequent request
  ps-ui makes through its API client (all backend capabilities, not just auth).
- Logout (manual or automatic) clears the auth store (and its localStorage copy), clears the active
  workspace selection, and drops the entire client query cache so no prior account's data leaks into
  the next session; the user is redirected to `/login`.

**Auto-logout on expiry.** When any backend call returns HTTP `401` **and** a session token is
currently present **and** the error body is the nested shape `{"error":{"code":401,...}}`, ps-ui
treats it as an expired session: it runs the full logout and redirects to `/login`. Concurrent 401s
are de-duped so logout fires once. A `401` from `POST /v1/auth/login` (wrong credentials) is
excluded — no token exists yet at that point, so the guard does not trip; that 401 instead surfaces
as an inline "Invalid email or password" form error.

**Failure modes.**
- Login/register `401` → inline invalid-credentials form errors; no session created.
- Any non-401 error → the error message from the response body (ps-ui reads either flat
  `{message}` or nested `{error:{message}}`) is surfaced to the user.
- Expired-token `401` mid-session → silent forced logout + redirect to `/login` (the in-flight
  request still rejects to its caller).

**Gotchas.**
- The expiry auto-logout only triggers on the **nested** `{error:{code:401}}` body. A bare `401`
  with a flat/absent body is NOT treated as expiry — the session is left intact and the individual
  request just fails. A backend changing its 401 error envelope shape will change this behavior.
- Registration returns a full session token — there is no email-verification / pending state gate in
  ps-ui; the account is live immediately on `POST /v1/auth/register` success.
- JWT is persisted in browser localStorage under the key `ps-auth`; `isAuthenticated` is derived from
  token presence on rehydration, so a manually-cleared token silently deauthenticates on next load.
- Route protection is client-side only: the `_auth` route group redirects unauthenticated users to
  `/login?redirect=<path>`, and `/login`+`/signup` bounce already-authenticated users to `/`. This is
  a UX guard, not a security boundary — the backend must still enforce auth on every endpoint.
- Post-auth redirect targets are allowlist-validated (only `/dashboard`, `/settings`, `/projects`,
  `/profile`, `/`; protocol-relative and traversal paths rejected); anything else falls back to `/`.
  Notably `/workspaces/...` deep-links are deliberately NOT honored as a redirect target.
