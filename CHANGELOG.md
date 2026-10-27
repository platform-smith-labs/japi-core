# Changelog

## v4.0.0 (unreleased)

Module path is now `github.com/platform-smith-labs/japi-core/v4`. See
[Migrating from v3](README.md#migrating-from-v3).

### Breaking

- **Error responses are RFC 9457 Problem Details** (`application/problem+json`).
  `core.Problem` is the error type pipelines write; `core.WriteProblem` and
  `core.WriteError` write it from any `net/http` handler. Bodies carry `type`
  (type base + code), `title`, `status`, `detail` (4xx only), `trace_id` and
  `errors` (field errors with JSON Pointers), plus extension members declared for
  the code.
- **Unexpected errors are never echoed.** Any error that is not a `*core.Problem`
  (found with `errors.As`) is written as the generic 500 `internal_error` on every
  pipeline, including `core.HandlerFunc`, which used to return `err.Error()` to
  the client. A 5xx body never carries a constructed detail, field errors or the
  cause; they go to one log line and the span.
- Built-in middleware returns problems with fixed details: `unauthorized` (401),
  `bad_request` and `invalid_json` (400), `request_too_large` (413),
  `validation_failed` (422, with `errors[]`), and for uploads `file_too_large`,
  `unsupported_media_type`, `invalid_csv`. Validation failures are now 422, not
  400. Parser and validator text goes to the log only.
- A handler deadline is answered with 504 `timeout`.
- `router` installs `otel.Recoverer` instead of chi's Recoverer: a panic becomes
  the generic 500 problem with `trace_id`, and the panic value is never logged,
  recorded on a span or written.
- With `router.WithTracing`, `X-Request-ID` on every response is the trace ID, and
  so is the request ID in the context. An inbound `X-Request-ID` never becomes the
  trace ID.
- `middleware/http.RequestIDContextKey` is a typed key. Reading
  `ctx.Value("request_id")` no longer finds it; use `GetRequestID`,
  `GetRequestIDFromContext` and `ContextWithRequestID`.
- Error log lines use `http.request.method` instead of `method`, and the typed
  access log describes a failure by `error.type` and status rather than error
  text.

### Deprecated

- `core.APIError`, `NewAPIError`, `NewValidationError`, `AddField`,
  `WriteAPIError`, `core.Error`, the `ErrBadRequest`... variables and the
  `{"error": {...}}` envelope. They still compile, and a returned `*APIError` is
  still written in the old envelope, so services can migrate route by route. For
  a 5xx the envelope now carries only the status text. They will be removed in a
  later major version.

### Added

- `core.SetCodeRegistry` (titles, statuses and allowed extensions per code) and
  `core.SetTypeBase`. With a registry installed, an unknown code, a status that
  differs from the registered one or an undeclared extension panics under
  `go test`; in production an unknown code is written as `internal_error`.
  `core.BuiltinCodes` lists the codes japi-core itself uses.
- `core.Classify` describes an error by code and status for log lines.
- OpenTelemetry, a no-op until a service opts in:
  - `otel/sdk` (`Setup`), `otel/otlp`: tracer and meter providers from the
    standard `OTEL_*` variables.
  - `otel.Middleware` / `router.WithTracing`: server spans named after the route
    pattern, with an attribute allowlist; WebSocket and streaming handling.
  - `otel.Recoverer`, `otel.RequestIDFromTrace`, `otel.TraceID`.
  - `otel.Transport`, `NoPropagationTransport`, `InjectProxy` for clients and
    proxies.
  - `db.Config.Tracing`, `otel.WrapDriver`, `otel.WrapConnector`: database
    spans and durations without arguments or statement text.
    `db.Config.TracingRequireParent` / `otel.SQLRequireParentSpan` trace only
    under an existing span.
  - `otel.TraceHandler`: slog handler adding trace correlation fields.
- Response-writer wrappers support `Unwrap`, `Flush`, `FlushError` and `Hijack`.

### Fixed

- A panic inside `otel.Middleware` no longer records the panic value on the server
  span, and an error status set by the application is no longer cleared by the
  HTTP instrumentation.
- Logs no longer carry the client address, user agent, raw path or query string,
  database arguments or driver error text.
