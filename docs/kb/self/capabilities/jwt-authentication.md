---
type: capability
title: "JWT authentication"
tags: [jwt, authentication, middleware, authorization, security]
timestamp: 2026-07-07T02:32:18Z
description: "Generate/validate JWTs and a handler middleware that enforces a bearer token and exposes the authenticated user + company to the handler"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - jwt/jwt.go
  - middleware/typed/auth.go
see_also:
  - {repo: japi-core, capability: "Typed middleware pipeline"}
  - {repo: japi-core, capability: "Typed handler framework"}
  - {repo: japi-core, capability: "Nullable optional type"}
  - {repo: japi-core, capability: "Error & response model"}
---

# JWT authentication

**What it does.** Provides HS256 JWT generation/validation and a composable handler middleware
(`RequireAuth`) that enforces a valid bearer token on a request and exposes the authenticated
user + company identity to the handler. The consumer (orchestrator, ps-api) owns the users/company
tables; this library only mints and verifies the token and, optionally, delegates an
existence check back to the consumer.

**How a peer interacts.**
- `jwt.GenerateToken(userUUID, companyUUID uuid.UUID, email, secret, issuer string, expiration time.Duration) (tokenString string, expiresAt time.Time, err error)` — mint a signed token.
- `jwt.ValidateToken(tokenString, secret string) (*Claims, error)` — verify signature + standard time claims, return the parsed claims.
- `jwt.ExtractClaims(tokenString string) (*Claims, error)` — parse claims **without** verifying the signature; explicitly for debugging, never for auth decisions.
- `typed.RequireAuth[P,B,R](jwtSecret string, validateUserCompany func(querier interface{}, userUUID, companyUUID uuid.UUID) error, next Handler[P,B,R]) Handler[P,B,R]` — a middleware wrapper passed into `MakeHandler`. After it runs, the handler reads `ctx.UserUUID` and `ctx.CompanyUUID` (each a `Nullable[uuid.UUID]`).

**Observable behavior.** `RequireAuth` reads the `Authorization` header, requires a `Bearer ` prefix,
extracts the token, and validates it against `jwtSecret`. On success it invokes the caller-supplied
`validateUserCompany` function (passing `ctx.DB`, the user UUID, and company UUID); if that returns
nil, it sets `ctx.UserUUID`/`ctx.CompanyUUID` from the claims and calls the next handler. Any failure
short-circuits with an API error and the next handler never runs. Token time claims (expiry,
not-before) are enforced by validation.

**Contract.**
- Token travels in the `Authorization: Bearer <token>` request header.
- Claims (`Claims` struct, complete wire shape): `user_uuid` (uuid — the authenticated user id the
  middleware reads), `company_uuid` (uuid), `email` (string), plus standard registered claims
  `exp`, `iat`, `nbf`, `iss`, `sub` (`sub` = the user UUID string). The user id is carried by
  `user_uuid` (and mirrored in `sub`).
- `RequireAuth` params: `jwtSecret` (must equal the generation secret), a `validateUserCompany`
  callback (consumer-supplied existence/authorization check — the library does **not** own any query
  or table; pass a no-op that returns nil to skip the DB check), and the `next` handler.
- Tokens are signed with HS256 at generation; `ValidateToken` rejects any token whose signing method is not HMAC-family (guards against `alg`-confusion such as RS256 or `none`; note it does not narrow to HS256 specifically — HS384/HS512 would also pass).

**Invariants.** The validation secret must match the secret used at generation, else validation
fails. Expiry (`exp`) and not-before (`nbf`) are enforced at validation. Generation uses HS256;
validation accepts only HMAC-family signing methods (rejects RS256/`none`/etc.). The authenticated
identity exposed to the handler always comes from verified claims (never from `ExtractClaims`).

**Failure modes** (as `core` API errors — see Error response model):
- Missing `Authorization` header, missing/empty `Bearer ` token, or invalid/expired token → **401**.
- `jwtSecret` empty (misconfiguration) → **500** ("Authentication configuration error").
- `validateUserCompany` returns an error → the middleware propagates that error verbatim; its status
  is defined by the consumer's callback. Convention (per the library's own doc) is **403** for
  user/company not found and **500** for DB errors, but the actual code is consumer-owned — UNKNOWN
  from this library's perspective.

**Gotchas.**
- The JWT secret is passed **per handler** (as a `RequireAuth` argument), not read from a global —
  every wired handler must receive a matching secret.
- The user/company existence check is **not** built in: it is an injected callback. There is no
  `validateUser bool` flag — a consumer that wants no DB check passes a function returning nil.
  Consumers that do check must supply the query and own the users/company table + id columns.
- `ExtractClaims` skips signature verification and must never gate access.
- `ctx.UserUUID`/`ctx.CompanyUUID` are `Nullable[uuid.UUID]`; a handler not wrapped by `RequireAuth`
  leaves them empty.

**Business-critical data.** None owned here. Any user/company existence check is delegated to the
consumer's callback against the consumer's own schema.

**See also.** japi-core "Typed middleware" (how `RequireAuth` composes into `MakeHandler`),
japi-core "Handler framework" (the `HandlerContext` carrying `UserUUID`/`CompanyUUID`/`DB`),
japi-core "Nullable optional" (the `Nullable[uuid.UUID]` identity fields), and japi-core
"Error response model" (the 401/403/500 API errors returned).
