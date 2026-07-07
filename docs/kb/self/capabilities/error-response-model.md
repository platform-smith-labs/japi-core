---
type: capability
title: "Error & response model"
tags: [japi-core, errors, http-response, json-envelope, validation]
timestamp: 2026-07-07T02:32:18Z
description: "Canonical APIError type plus response writers that produce consistent success/error JSON envelopes"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - core/errors.go
  - core/response.go
  - core/constants.go
  - core/handler.go
see_also:
  - {repo: japi-core, capability: "Typed middleware pipeline", intent: "produces field-level validation errors as APIError", descriptive: false}
  - {repo: japi-core, capability: "Typed handler framework", intent: "the handler adapter that writes a returned error to the response", descriptive: false}
---

# Error & response model

**What it does.** Defines the one canonical API error type (`APIError`) that every handler in a
japi-core service returns on failure, plus a small set of response writers that emit consistent JSON
envelopes for success and error. This is the shape a consuming service's HTTP clients actually see on
the wire, so it is the externally observable contract of every japi-core-based API (orchestrator,
ps-api).

**How a peer interacts** (Go calls, compile-time library):
- Construct errors: `core.NewAPIError(code, message, detail...)` (detail is an optional variadic
  string); `core.NewValidationError(message).AddField(field, msg)` for field-level validation;
  package-level shortcuts `core.ErrBadRequest`, `core.ErrUnauthorized`, `core.ErrForbidden`,
  `core.ErrNotFound`, `core.ErrInternal` (each a `*APIError`).
- Write responses: `core.Success[T](w, data)` (200), `core.Created[T](w, data)` (201),
  `core.NoContent(w)` (204), `core.List[T](w, data)` (200, list envelope), and for errors either
  return the `*APIError` from the handler (the framework writes it) or call
  `core.Error(w, r, status, message)` / `core.WriteAPIError(w, r, apiErr)` directly.
- DB constraint helpers: `core.IsUniqueConstraintError(err, constraintName)` and
  `core.IsForeignKeyConstraintError(err, constraintName)` classify a pgx error so a handler can map it
  to a 400 `APIError` instead of leaking a 500.

**Observable behavior** (the JSON an HTTP client receives):
- Success (`Success`/`Created`): the payload is written **as-is** — no wrapper envelope — with status
  200 or 201.
- List (`List`): wrapped as `{ "data": [...], "count": N }` where `count` is the item count of that
  page (derived from the slice length, not a total-row count), status 200.
- NoContent: status 204, empty body.
- Error (any path): wrapped as `{ "error": { ...APIError fields... } }`, HTTP status = the error's
  `code`. Errors are logged server-side by severity (`code >= 500` → error log, `>= 400` → warn).
- Field-level validation surfaces as the error object's `fields` map: `{ field-name: message }`.
  Multiple errors on the same field are joined into one string with ` || `.

**Contract.**
- `APIError` JSON fields (confirmed complete): `code` (int, HTTP status), `message` (string),
  `detail` (string, omitted when empty), `fields` (object of field-name → message, omitted when
  empty).
- Success envelope: raw `T`. List envelope fields (confirmed complete): `data` (array), `count` (int).
- Error envelope: single top-level `error` key holding the `APIError` object.
- Status codes set by constructors/shortcuts (confirmed): `NewValidationError` → 400; `ErrBadRequest`
  → 400, `ErrUnauthorized` → 401, `ErrForbidden` → 403, `ErrNotFound` → 404, `ErrInternal` → 500.
  `NewAPIError` uses whatever `code` the caller passes. Constants `StatusDatabaseConstraintViolation`
  = 400 and `StatusDatabaseError` = 500 are the recommended codes when mapping DB errors.

**Invariants.**
- Every error response has the same top-level shape (`{ "error": {...} }`) and the HTTP status always
  equals the embedded `code` — a client can rely on status and body agreeing.
- `code` and `message` are always present on an error; `detail` and `fields` are present only when
  populated.
- The package-level `Err*` shortcuts are shared pointers to fixed values — treat them as read-only
  sentinels; mutating one (e.g. calling `AddField` on `ErrBadRequest`) would corrupt the shared
  instance for all callers (build a fresh `NewValidationError`/`NewAPIError` instead).

**Failure modes.**
- A handler that returns a **non-`APIError`** error is caught by the framework adapter and converted to
  a 500 whose `detail` carries the original error's text — so raw internal error strings can leak into
  the response body unless the handler maps them first.
- `IsUniqueConstraintError` / `IsForeignKeyConstraintError` return false for any non-pgx error, and
  match on constraint-name substring — a wrong (non-empty) constraint name silently fails to classify,
  letting the error fall through to a 500; conversely an **empty** name substring-matches every
  constraint of that error code, so pass the exact name.

**Gotchas.**
- A handler may deliver an error two ways: **return** the `*APIError` (the handler-framework adapter
  writes it via the same writer), **or** call `core.Error` / `core.WriteAPIError` directly. Both
  produce the identical `{ "error": {...} }` envelope; do not double-write.
- `core.Error(w, r, status, message)` takes the request (for log context) and a status+message — not a
  logger and not an error value. To emit an already-built `APIError`, use `core.WriteAPIError`.
- `core.List` does **not** take a count argument; it computes `count` itself from the slice length, so
  the envelope's `count` reflects the returned page, not a total.
- Success responses are unwrapped: clients read the object directly, but list responses are wrapped —
  a peer must special-case the `{data,count}` shape for collection endpoints.

**See also / peers.** Within japi-core: **Typed middleware** (parses/validates request bodies and
raises the field-level validation `APIError` this model carries) and the **Handler framework** (the
adapter whose `ServeHTTP` writes a returned error through this writer).
