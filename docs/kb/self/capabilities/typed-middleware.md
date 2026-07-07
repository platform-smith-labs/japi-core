---
type: capability
title: "Typed middleware pipeline"
tags: [middleware, validation, request-parsing, response, generics, go]
timestamp: 2026-07-07T02:32:18Z
description: "Composable generic middleware that parse request params/body/headers into the typed context with struct-tag validation, and write the typed response"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - middleware/typed/request.go
  - middleware/typed/response.go
  - middleware/typed/csv.go
  - middleware/typed/json.go
  - middleware/typed/validator.go
  - middleware/validation/setup.go
  - handler/types.go
see_also:
  - {repo: japi-core, capability: "Typed handler framework", intent: "the handler + MakeHandler composition these middleware plug into"}
  - {repo: japi-core, capability: "Error & response model", intent: "the APIError/validation-error types these middleware return on failure", descriptive: false}
  - {repo: japi-core, capability: "Nullable optional type", intent: "the Nullable wrapper parsed values arrive in", descriptive: false}
---

# Typed middleware pipeline

**What it does.** A set of generic higher-order middleware that populate a request's typed
`HandlerContext` (params, body, headers) with automatic struct-tag validation on the way in, and
write the handler's typed return value on the way out. A peer composes these around a handler to
get parsing + validation + response-writing without hand-rolling any of it.

**How a peer interacts.** Pass middleware into `MakeHandler(registry, routeInfo, handlerFn,
middleware...)` after the handler function. Parsing middleware: `typed.ParseParams` (URL path +
query), `typed.ParseBody` (JSON request body), `typed.ParseHeaders` (all request headers),
`typed.ParseCSV` and `typed.ParseJSON` (a file uploaded as multipart form field named `file`).
Response middleware: `typed.ResponseJSON` and `typed.ResponseJSONFile(filename)` (same, but as a
browser download). Auth and request-id/logging are separate middleware with their own capabilities —
not covered here.

**Observable behavior.**
- *Composition order.* `MakeHandler` wraps the middleware around the base handler in reverse
  argument order, making the **first-listed** middleware the **outermost**. On the request path,
  middleware run in listed order (first argument first, base handler last); return values unwind in
  the reverse order. Practically, list parsing middleware first and the response middleware last, so
  parsing runs before the handler and the response middleware — being innermost — wraps the handler
  directly to capture and write its return value. (The inline "last middleware executes first"
  code comment is misleading; the entry order is first-listed-first.)
- *Validation.* Parsing middleware validate the populated struct with go-playground `validator/v10`
  struct tags. On failure, params/body validation returns a structured validation error listing each
  offending field with a human-readable message (multiple messages per field joined); CSV/JSON-file
  parsing returns a plain bad-request error naming the failing row/reason. Validation error field
  names follow the field's `json` tag (falling back to snake_case of the Go field name).
- *Parsed values arrive as `Nullable`.* Successful parses set `ctx.Params` / `ctx.Body` /
  `ctx.Headers` as populated `Nullable` values; when a handler declares no params/body (the type is
  an empty struct), the middleware sets the corresponding `Nullable` to empty and skips parsing.
  `ParseBody` also exposes the raw bytes as `ctx.BodyRaw`.
- *Response status.* `ResponseJSON`/`ResponseJSONFile` pick 201 for POST and 200 otherwise, then
  write the handler's return value. They only act on a nil-error return; a handler error is passed
  through untouched for the framework adapter to render.

**Contract.** A peer annotates the param/body struct with tags the middleware read:
- `param:"name"` — URL path parameter; `query:"name"` — query-string parameter (ParseParams).
- `json:"name"` — JSON body field (ParseBody / ParseJSON file).
- `csv:"name"` — CSV column; the body type must be a slice of the row struct (ParseCSV).
- `validate:"..."` — go-playground validation rules (`required`, `email`, `min`/`max`, `uuid`,
  `url`, `eqfield`, etc.).

Inputs: the HTTP request. Outputs: a mutated `HandlerContext` with the typed, validated value.
Errors (returned as the framework's API/validation error types, rendered by the handler adapter):
missing required param/query → bad request; unparseable param (bad int/uuid/etc.) → bad request;
missing-but-expected body → bad request; malformed JSON → bad request; struct-tag validation
failure → structured validation error; missing/`wrong-extension`/empty upload file → bad request.

**Invariants.** Parse middleware are additive and order-independent among themselves (each sets a
distinct context field); a value is only present in the context if its Parse middleware ran and
succeeded. Path/query params support string, integer, unsigned, float, bool, and `uuid.UUID` field
types only. The validator instance is process-global within the `typed` package.

**Failure modes.** A validation or parse failure short-circuits the chain — the base handler never
runs and the peer receives the error response. CSV parsing loads the whole file (32 MB multipart
cap) and validates every row, failing on the first invalid one (row number reported 1-based
including the header).

**Gotchas.**
- `ctx.Body` / `ctx.Params` / `ctx.Headers` are **absent until the matching Parse middleware is in
  the chain** — omit `ParseBody` and the body is never populated even if the request carries one.
- Validators are stock go-playground `validator/v10`. japi-core ships **no** custom or DB-backed
  validators (e.g. uniqueness / existence checks) — registering those against the app's schema is
  the consumer's job; the `middleware/validation` package is documentation/examples only.
- `ParseCSV` and `ParseJSON` read a multipart **upload** (form field `file`), not the raw request
  body — distinct from `ParseBody`, which decodes the JSON request body directly.
- Response middleware do not translate handler errors; error rendering belongs to the handler
  framework's adapter, not this pipeline.
