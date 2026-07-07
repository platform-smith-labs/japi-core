---
type: eval-record
eval: task-scoping
timestamp: 2026-07-07T02:35:00Z
commit_sha: 910ed6a
result: pass
seam:
  capabilities: [route-registry, typed-middleware, jwt-authentication, handler-framework, database-layer, error-response-model]
  rationale: >-
    For a compile-time framework library the most peer-relevant seam is composing a new endpoint end
    to end. "Add an authenticated, body-validated, DB-writing POST endpoint" forces every hand-off a
    consumer actually chains: registration (route-registry) → middleware order (typed-middleware) →
    auth identity population (jwt-authentication → ctx.UserUUID) → typed handler contract
    (handler-framework) → context-propagated DB write (database-layer) → error/response wire shape
    (error-response-model). It cannot be answered by any single capability.
task: >-
  My service needs to add a new authenticated HTTP endpoint POST /widgets that accepts a JSON body
  {name, size}, validates it (name required, size positive int), inserts a row into my own widgets
  table stamped with the authenticated user's id, and returns the created widget as JSON — with
  proper error responses for validation and auth failures. Using japi-core, which capabilities do I
  call, in what order (registration, middleware, auth, DB, response), what do I read off the handler
  context, and what gotchas/invariants must I respect?
gaps: []
---

## Agent answer (verbatim)

A fresh agent was given ONLY `docs/kb/` (no repo source, README, tests, `extract/`, or `eval/`) and
the task above. It produced a complete, actionable scoping answer:

- **Concrete surface named**: `handler.NewRegistry` / `MakeHandler` / `RouteInfo` / `RegisterWithRouter`
  / `WithServices` (route-registry); `typed.RequireAuth(secret, validateUserCompany, next)`,
  `typed.ParseBody`, `typed.ResponseJSON` (typed-middleware, jwt-authentication); `HandlerContext[P,B]`
  with `struct{}` for empty params (handler-framework); `db.Connect` / `db.QueryOne[T]` /
  `db.WithTx` / `Querier` with `ctx.Context` first (database-layer); the `{error:{code,message,
  detail?,fields?}}` envelope, `core.NewAPIError` / `NewValidationError().AddField` / `Err*` sentinels,
  `core.IsUniqueConstraintError(err, name)` (error-response-model).
- **Order correct**: `RequireAuth` outermost → `ParseBody` → `ResponseJSON` innermost, with the
  explicit note that first-listed runs first and the code's own "last executes first" comment is
  misleading; 201 for POST from `ResponseJSON`.
- **Context reads correct**: `ctx.Body` (Nullable, present after ParseBody), `ctx.UserUUID` /
  `ctx.CompanyUUID` (Nullable, set by RequireAuth), `ctx.Context` (first arg to every DB call),
  `ctx.DB`, `ctx.Services` (type-assert; wrong assertion panics) — with full Nullable accessor
  semantics.
- **Seams closed**: auth → `ctx.UserUUID` → stamped onto the INSERT; `ParseBody` → `ctx.Body`;
  handler return → `ResponseJSON` writes it.
- **Gotchas surfaced**: Nullable never in model structs (marshals to `{}`); model needs
  `json:`+`validate:`+`db:` tags; `struct{}` param short-circuits parsing; parameterized SQL only;
  `jwtSecret` per-handler must match minting secret; `validateUserCompany` is a callback (not a
  `bool`); map DB errors to `APIError` or a raw 500 leaks internal text; `Err*` sentinels are shared
  read-only pointers (never `AddField` on one); don't double-write the response; register + rebuild or
  the route 404s.

(Full verbatim answer retained in the generation session log.)

## Judgement

| Criterion | Verdict | Justification |
|---|---|---|
| Name the concrete surface | ✅ PASS | Every function/type/middleware needed was named with its role. |
| Make the behavior clear | ✅ PASS | Composition order, sync flow, 201-on-POST, validation short-circuit, Nullable access all stated. |
| Surface relevant gotchas/invariants | ✅ PASS | 12-item checklist incl. the non-obvious traps (callback-not-bool, shared Err* sentinels, DB-error leakage, Nullable-in-models). |
| Close the cross-capability hand-off | ✅ PASS | Auth→userUUID→insert, ParseBody→ctx.Body, handler-return→ResponseJSON all correctly chained. |
| Not need to invent/guess/ask for source | ✅ PASS | Agent explicitly grounded every fact in a cited capability file; nothing was unanswerable from the KB. |

**Result: PASS (0 gaps).** This is the acceptance gate — the KB is accepted for commit at HEAD 910ed6a.
