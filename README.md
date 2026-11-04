# japi-core

Type-safe Go framework for building REST APIs with PostgreSQL.

Module path: `github.com/platform-smith-labs/japi-core/v4`
Requires Go 1.25 or later.

    go get github.com/platform-smith-labs/japi-core/v4@latest

## Errors

Every error response is an [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457)
Problem Details body, `Content-Type: application/problem+json`:

```json
{
  "type": "https://example.com/errors/validation_failed",
  "title": "Validation failed",
  "status": 422,
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "errors": [{ "detail": "name must be at least 3 characters", "pointer": "#/name" }]
}
```

- `type` is the type base (`core.SetTypeBase`) followed by the error code. Clients
  branch on the code, the last path segment of `type`. Without a type base,
  `type` is `about:blank`.
- `title` is fixed per code (`core.SetCodeRegistry`), never request-specific.
- `detail` is written for 4xx only. A 5xx body is type, title, status and
  `trace_id`: constructed details, field errors and causes go to the log and the
  span.
- `trace_id` is the request's trace ID, also sent as `X-Request-ID` when tracing
  is on.

Return a `*core.Problem` from a handler; anything else becomes the generic 500
`internal_error`:

```go
if errors.Is(err, sql.ErrNoRows) {
    return nil, core.NotFound("widget_not_found", "No widget with that ID.")
}
if err != nil {
    return nil, core.Internal(err) // the client sees only "Internal server error"
}
```

## Migrating from v3

1. Change imports from `github.com/platform-smith-labs/japi-core/v3` to `/v4`.
2. Replace constructors. The old ones (`APIError`, `NewAPIError`,
   `NewValidationError`, `WriteAPIError`, `core.Error`, `ErrBadRequest` and the
   other `Err*` variables) are removed in v4:

   | v3 | v4 |
   |---|---|
   | `core.NewAPIError(400, msg)` | `core.BadRequest(code, detail)` |
   | `core.NewAPIError(401, msg)` | `core.Unauthorized(code, detail)` |
   | `core.NewAPIError(403, msg)` | `core.Forbidden(code, detail)` |
   | `core.NewAPIError(404, msg)` | `core.NotFound(code, detail)` |
   | `core.NewAPIError(409, msg)` | `core.Conflict(code, detail)` |
   | `core.NewAPIError(422, msg)` | `core.Unprocessable(code, detail)` |
   | `core.NewAPIError(429, msg)` | `core.TooManyRequests(code, detail)` |
   | `core.NewAPIError(500, msg, err.Error())` | `core.Internal(err)` |
   | `core.NewAPIError(502/503/504, msg, err.Error())` | `core.BadGateway` / `core.Unavailable` / `core.GatewayTimeout(code, err)` |
   | `core.NewValidationError(msg).AddField(f, m)` | `core.Validation(core.FieldError{Detail: m, Pointer: "#/" + f})` |
   | `core.Error(w, r, status, msg)` / `core.WriteAPIError(w, r, e)` | `core.WriteProblem(w, r, p)` or `core.WriteError(w, r, err)` |
   | any status | `core.NewProblem(code, status, detail)` |

   Never pass `err.Error()` as a detail: attach it with `.WithCause(err)`, which
   is logged and recorded on the span but never sent.
3. Choose codes: lowercase `snake_case` tokens, stable once published. Register
   them, with their titles, statuses and any extension members, through
   `core.SetCodeRegistry`; set `core.SetTypeBase` to where they are documented.
4. Field errors move from the `fields` map to the `errors` array. `pointer` is a
   JSON Pointer into the request body; query and path parameters have no pointer
   and are named in `detail`.
5. Clients: parse `application/problem+json`, read the code from the last
   segment of `type`, show `title` (and `detail` for 4xx), and quote `trace_id`
   when reporting a problem. The old `{"error": {...}}` envelope is no longer written.
6. Request-body validation failures are 422 `validation_failed` (were 400), and
   `ctx.Value("request_id")` no longer finds the request ID: use
   `httpmiddleware.GetRequestID`.
