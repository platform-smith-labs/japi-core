---
type: capability
title: "Auto-generated Swagger/OpenAPI docs"
tags: [swagger, openapi, documentation, http, reflection]
timestamp: 2026-07-07T02:32:18Z
description: "Generates an OpenAPI 2.0 spec and Swagger UI from the RouteInfo + typed handler metadata a peer already declares — no separate annotations"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - swagger/generator.go
  - swagger/routes.go
  - handler/types.go
see_also:
  - {repo: japi-core, capability: "Route registry & registration", intent: "supplies the collected routes the spec is built from", descriptive: false}
  - {repo: japi-core, capability: "Typed handler framework", intent: "the ParamType/BodyType/ResponseBody generics the generator reflects over", descriptive: false}
---

# Auto-generated Swagger/OpenAPI docs

**What it does.** Produces an OpenAPI 2.0 (Swagger) specification and a hosted Swagger UI from the
routes a peer has already registered — no separate annotations or comment directives. The doc content
is derived entirely from each route's `RouteInfo` metadata plus the generic type parameters of its
typed handler, so the docs reflect exactly what is registered and nothing else.

**How a peer interacts.** After building a route `Registry` and mounting it on a chi router, call
`swagger.SetupSwaggerUI(r, registry)`. This registers two GET endpoints on the router:

- `GET /swagger.json` — the generated OpenAPI spec as JSON (regenerated per request from the current
  registry).
- `GET /swagger/*` — the interactive Swagger UI (browse at `/swagger/index.html`).

Use `swagger.SetupSwaggerUIWithPath(r, basePath, registry)` to mount both under a prefix (e.g.
`basePath="/api/docs"` → `/api/docs/swagger.json`, `/api/docs/swagger/*`).

**Observable behavior.** The generator reads all collected routes from the registry, groups them by
path, and emits one OpenAPI PathItem per path with an operation per HTTP method. For each operation it
uses reflection over the typed handler to introspect three type parameters:

- **Path/query parameters** — from the handler's param type; fields carrying a `param:"..."` tag
  become path parameters, `query:"..."` tag fields become query parameters.
- **Request body** — from the handler's body type; emitted as a `body` parameter referencing a
  generated `#/definitions/<TypeName>` schema.
- **Response body** — from the handler's first return value; struct → a `$ref` schema on the 200
  response, slice → an array-of-schema response.

Field-level details are also mined from struct tags: `json` names, `validate:"required"` → required
list, `validate:"min=/max="` → length/range constraints, `validate:"email"` → email format,
`description:"..."` → field description. `time.Time` and `uuid.UUID` are rendered as formatted
strings. Routes whose middleware chain includes a middleware named `RequireAuth` are marked as
requiring the `BearerAuth` (JWT-in-`Authorization`-header) security scheme. Every operation also gets
standard 400/401/403/500 error responses.

**Contract.** Per-route documentation quality is driven by `handler.RouteInfo` — key fields:
`Summary`, `Description`, `Tags` (all optional; auto-generated from method + path when blank), plus
`Method` and `Path` (which place the operation). Entry point:
`SetupSwaggerUI(r chi.Router, registry *handler.Registry)` and
`SetupSwaggerUIWithPath(r chi.Router, basePath string, registry *handler.Registry)`. There is also a
package-level `swagger.SwaggerInfo` var (Title/Description/Version/Host/BasePath/Schemes) — see the
gotcha below; it does **not** feed the served spec at this commit.

**Invariants.** The spec reflects only registered routes — a route not in the registry never appears,
and there is no hand-maintained doc file to drift. The JSON is generated fresh on each request to
`/swagger.json`, so it always matches the current registry. Auth marking is purely a function of a
middleware being named `RequireAuth`.

**Failure modes.** If spec generation errors, `/swagger.json` returns HTTP 500 with a plain-text
message and the UI cannot load a spec. A handler whose type parameters are empty structs contributes
an operation with no parameters/body/response schema (still valid, just sparse).

**Gotchas.**
- Doc richness is entirely a function of `RouteInfo` quality: a route with no `Summary`/`Description`/
  `Tags` gets machine-generated placeholders (e.g. a `Get Foo` summary, a single path-derived tag).
- The served spec's top-level `info` (title/description/version), host, base path, and schemes are
  **hardcoded** inside the generator (title "Junix API", host `localhost:8080`, basePath `/`).
  Mutating `swagger.SwaggerInfo` does **not** change the JSON served at `/swagger.json` at this commit
  — `SwaggerInfo` is effectively vestigial for the served output. Treat the served metadata as fixed
  unless the generator is changed.
- Auth detection is name-based: the middleware must be registered under the exact name `RequireAuth`
  for the `BearerAuth` requirement to appear; a differently-named auth middleware yields no security
  marking.
- Output is OpenAPI/Swagger **2.0**, not OpenAPI 3.x.

**See also.** japi-core — Route registry (the source of the collected routes); japi-core — Typed
handler framework (the generic param/body/response types this reflects over).
