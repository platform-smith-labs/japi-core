# KB change log

Newest first. One entry per `kb-bootstrap` generation run.

## 2026-07-07 · HEAD 910ed6a

- **Run**: initial bootstrap of `docs/kb/` for japi-core.
- **Concepts regenerated**: 13 — overview, context, glossary, and 10 capabilities
  (handler-framework, route-registry, typed-middleware, database-layer, error-response-model,
  jwt-authentication, nullable-optional, cors-router, swagger-generation, observability).
- **Pipeline**: EXTRACT → DRAFT (one subagent per concept) → VERIFY per-concept (fresh context) →
  VERIFY cross-concept (seam + contradiction sweep) → STAMP → LINT → RENDER.
- **Lint**: 0 fail, 0 warn.
- **UNKNOWN markers**: 3 — repo license (README placeholder); JWT user/company not-found HTTP status
  (consumer-owned via the `validateUserCompany` callback, not framework-enforced); concrete consumer
  capability names in orchestrator/ps-api (cross-repo, left for kb-sync to reconcile).
- **Open SEAM GAPs**: 0.
- **Notes for maintainers (japi-core local, surfaced during VERIFY)**: (1) `MakeHandler`'s doc-comment
  in `handler/types.go` describes middleware execution order inverted vs. actual runtime behavior
  (first-listed runs outermost/first); (2) `swagger.SwaggerInfo` is not wired into the served spec —
  `GenerateSpec` hardcodes title/version/host, so setting `SwaggerInfo.*` (as the README suggests) has
  no effect at this HEAD. Both are documented as gotchas in the KB.
