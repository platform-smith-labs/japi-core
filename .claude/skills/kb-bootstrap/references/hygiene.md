# §0 hygiene rules — what to keep, what to cut

The governing discipline (requirements §0). DRAFT and VERIFY both cite this file; `kb-lint.sh`
enforces the mechanical parts. **BREVITY IS CLARITY** — every sentence must earn its place.

## The single test

> Would an agent in a *different* repo need this to reason about interacting with this capability?

If no → cut it. Clutter is a defect, not thoroughness.

## KEEP (business logic a peer needs)

- What the functionality does, in business terms.
- How a peer invokes/interacts with it (entry point, contract by name).
- Observable behavior, invariants, failure modes, gotchas.
- Business-critical data: only the tables/columns the **main logic** depends on, and why.

## OMIT (internal mechanics — no peer significance)

- Struct/DTO transformations, JSON↔struct mapping, serialization/marshalling.
- Internal call chains, per-function/per-class walkthroughs, private helpers.
- Boilerplate plumbing, config wiring, dependency-injection detail.
- **Ubiquitous data access** — a tenant/customer table joined on *every* query is stated **once** in
  `context.md`, never per capability; omit common join-chain detail and index/plumbing.

## NEVER (hard rules → lint FAIL where mechanical)

- **No source pointers in a concept body** — no `file:line`, no "see file X", no source links. A peer
  has no access to this repo's tree. (Grounding lives in the internal `evidence` frontmatter only.)
  → `kb-lint.sh` **fails** on a body source-pointer.
- **No fabrication** — a fact you cannot ground against the code/contracts is written literally as
  `UNKNOWN` (optionally `UNKNOWN — TODO: <hint>`), never guessed. VERIFY must not pass an unmarked
  unverifiable claim.
- **No pasted/restated schemas** — reference a contract by name/kind so a peer knows what to ask for;
  don't paste OpenAPI/proto/SQL. → lint **warns**.
- **Mark partial field lists** — an inline field list that is not the confirmed-complete wire shape
  MUST be prefixed `key fields:`; never let a partial list read as exhaustive.

## Behavior over implementation

State observable behavior in business terms (what happens, what the caller gets, what can go wrong) —
**not** by narrating internal steps. A bare signature/name is insufficient (research/0002 F5); pair
every interface reference with the behavior a peer needs.

## Brevity budget (NFR-1)

- Capability concept: soft cap **~120 lines**; other concepts ≤ ~150.
- Whole `self/` narrative bundle: ≤ ~15k tokens.
- Over-cap → lint **warning** prompting a trim. Shorter while still passing the task-scoping test is
  strictly better.

## Lint term-flags (warnings)

`kb-lint.sh` warns on likely internal-mechanic terms in a body: `unmarshal`, `marshal`, `DTO`,
`struct mapping`, `serialize`, `deserialize`, `dependency injection`. A warning means "justify or
cut," not an automatic failure — some may be legitimately peer-relevant.

The match is a context-blind substring/alternation (busybox-safe: no word boundaries), so it can
flag a term used in a *different sense* — e.g. the concurrency sense of "serialize" ("concurrent git
operations don't serialize"), not marshalling. When a flag is a reviewed false positive, exempt that
line with an inline HTML-comment marker naming the term and the reason:

    concurrent git operations don't serialize. <!-- lint-ok: serialize — concurrency sense, not marshalling -->

The linter suppresses the warning for any body line carrying a `lint-ok` marker; every other flagged
line still warns. Use it sparingly — the default is still "cut," not "annotate."
