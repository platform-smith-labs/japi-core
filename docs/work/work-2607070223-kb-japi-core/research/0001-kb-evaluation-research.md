# Independent evaluation — japi-core KB (kb-bootstrap run)

**Work Item**: work-2607070223-kb-japi-core
**Parent**: work-2607062158-kb-bootstrap-skillset @ solution
**Date**: 2026-07-07
**HEAD evaluated**: 910ed6a

## 0. Mandate & epistemics

Per the inherited scaffold rule (prior research is a pointer, not gospel), this evaluation grounds
every claim in **this repo's own code** and in the **behavior of the KB actually produced** — not in
the parent's design docs. The parent skillset is treated as a tool under test; findings below are
independent observations, reconciled against the skillset's stated contract only after being derived
from evidence.

## 1. What was generated

Ran the central `kb-bootstrap` skillset (monorepo root `.claude/skills/kb-bootstrap`) against
`repos/japi-core`. japi-core is a **compile-time Go framework library** (v3, `go get
github.com/platform-smith-labs/japi-core/v3`) consumed by orchestrator + ps-api — it owns no service
port and no database tables. Pipeline executed exactly as specified: EXTRACT (deterministic) → DRAFT
(one subagent per concept) → VERIFY per-concept (fresh, adversarial context) → VERIFY cross-concept
(seam + contradiction sweep) → STAMP → LINT → RENDER → task-scoping eval.

Output: `docs/kb/` with **13 concepts** — `overview`, `context`, `glossary`, and **10 capabilities**
(handler-framework, route-registry, typed-middleware, database-layer, error-response-model,
jwt-authentication, nullable-optional, cors-router, swagger-generation, observability). Lint: **0
fail, 0 warn, 3 honest UNKNOWNs**. Shape mirrors the minimal peer convention (overview + context +
glossary + capabilities/), with gotchas folded into each capability rather than a separate tree —
appropriate for a library whose "capabilities" are API surfaces.

## 2. Decisive test — task-scoping eval (PASS)

A **fresh KB-only agent** (no source, no README, no other repo) was posed the task a real consumer
faces: *"add an authenticated POST /widgets endpoint that validates a JSON body, inserts into my own
table stamped with the auth'd user, and returns the row with correct error responses."* This crosses
six capabilities and every hand-off a consumer chains.

The agent scoped it **completely and correctly from the KB alone** — full record at
`docs/kb/self/eval/task-scoping-910ed6a.md`. It named the concrete API surface, got the middleware
composition **order** right (first-listed outermost), closed all seams (auth→`ctx.UserUUID`→insert;
`ParseBody`→`ctx.Body`; handler-return→`ResponseJSON`), and surfaced the non-obvious traps
(`validateUserCompany` is a callback not a bool; `Err*` sentinels are shared read-only pointers;
unmapped DB errors leak as a 500; `Nullable` must not appear in model structs). All 5 pass-criteria
met, 0 gaps. **This is the acceptance gate — passed.** No local KB fixes were required (contrast with
peers who applied B-series fixes post-eval).

## 3. Ground-truth verification value (what the adversarial VERIFY caught)

The separate-context VERIFY stage earned its keep — DRAFT seeds that paraphrased the README were
corrected against the actual code:

- **Middleware composition order was stated inverted** in `handler-framework` and `glossary`
  (drafted from the code's own misleading doc-comment). VERIFY re-derived it empirically by driving
  `MakeHandler`: **first-listed middleware runs outermost/first**. Fixed everywhere and made
  consistent in the cross-concept pass.
- **`jwt.RequireAuth`'s 2nd param is a `validateUserCompany` callback, not a `validateUser bool`**;
  it sets both `ctx.UserUUID` and `ctx.CompanyUUID`; user/company-not-found status is consumer-owned
  (left UNKNOWN — a framework-honest call).
- **`core.List[T](w,data)` has no count arg**; `core.Error(w,r,status,message)`; `WriteAPIError`;
  `IsUniqueConstraintError(err, name)` + a foreign-key sibling — all corrected from the seed.
- **`db.Exec`/`db.HealthCheck` are not generic**, `HealthCheck` takes no ctx (own timeout); scany +
  pgx/v5 confirmed; pool defaults 25/25/5m/5m + the `MaxIdleConns ≤ MaxOpenConns` rule verified.
- **`swagger.SwaggerInfo` is not wired** into the served spec — `GenerateSpec` hardcodes
  title/version/host (OpenAPI 2.0, reflection-based body introspection). Documented as a gotcha.

## 4. japi-core LOCAL findings (source issues, surfaced during KB generation — out of this item's scope)

Documented in the KB as gotchas and noted in `docs/kb/log.md`; flagged here for a maintainer/future
work item (NOT skillset finetunes, NOT relayed upstream):

1. **`MakeHandler` doc-comment in `handler/types.go` is inverted** vs. runtime behavior ("last
   middleware executes first"). A latent code-comment bug; the runtime is authoritative.
2. **`swagger.SwaggerInfo` is dead at this HEAD** — README instructs setting it, but `GenerateSpec`
   ignores it. Either wire it or update the README.

## 5. Skillset finetune observations (relayed upstream for maintainer approval)

See `requirements/0001-skillset-finetunes.md` and the outbound relay
`relays/outbound/to-solution--kb-skillset-finetunes.md`. Summary:

- **F1 — cross-concept pass should normalize SAME-repo `see_also` edges.** DRAFT subagents work one
  concept at a time and cannot see sibling capability titles, so they honestly mark same-repo
  `see_also` entries `descriptive: true`. Once the whole bundle exists those names ARE knowable; the
  cross-concept VERIFY pass (or kb-render) should flip same-repo entries to `descriptive: false` and
  align the `capability` field to the sibling's real frontmatter `title`. Done manually this run.
- **F2 — add explicit guidance for library / compile-time-consumer repos.** The
  capability-concept / draft / verify prompts assume a *runtime service* (endpoints, RPC, message
  kinds, async readiness, tenant tables). For a library consumed via `import`, "how a peer interacts"
  = a Go API call, seams = required call-ordering / which function populates which field, and the data
  section is normally empty. A short "library repos" note (mirroring the existing "thin repos" note)
  would prevent drafters from forcing runtime framing. Relatedly, the internal-mechanic lint
  term-flags (`marshal`/`serialize`) fire heavily on a serialization framework where those ARE the
  peer vocabulary — expected, handled via `lint-ok`, but worth calling out in the library note.

Both may overlap with genericity finetunes already relayed by orchestrator/runtime; surfaced
independently here for the maintainer to dedupe.
