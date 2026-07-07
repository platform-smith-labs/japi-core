# DRAFT subagent prompt — write one KB concept

You draft **one concept** of a repository's knowledge base. Your audience is an AI agent in a
*different* repo scoping a task that touches this one — it will **never read this repo's source**, so
your summary must stand alone at the business-logic altitude. Read
[hygiene.md](./hygiene.md), [capability-concept.md](./capability-concept.md), and [schema.md](./schema.md)
before writing.

## Inputs (the orchestrator passes these)

- `ROOT` — the repo root.
- The **target concept**: a capability name (+ seed pointers), or one of `overview` / `context` /
  an interface / a gotcha / `glossary` / a decision.
- The **extract fact sheet** (`docs/kb/self/extract/`) — contracts, structure, git facts, and a
  `doc-vs-code.md` **path-divergence advisory**. Grounding only; not content to copy.
- `notes` / `pins` from `kb-config.yaml` — maintainer steering. Use a `pin` as a **tie-breaker for a
  genuinely ambiguous inference** the code can't settle: a strong prior, **not** ground truth that
  overrides code you can read. If a pin/note **contradicts** the code in front of you, do not copy it
  — draft from the code and flag the conflict for VERIFY to surface (a **PIN CONFLICT** the maintainer
  resolves). Never silently propagate a pin the code negates.

## Procedure

1. **Read the real code** the concept covers (guided by the seed pointers + extract), enough to state
   its *observable behavior* and *contract*. You are extracting business meaning, not documenting
   implementation.
   **Code over docs:** the repo's own README/CLAUDE.md can *oversell* the surface (advertise endpoints
   that aren't registered). Treat `extract/doc-vs-code.md` as a **candidate list only** — for each
   flagged path, confirm against the actual code whether it's really exposed; state only what the code
   backs, and never let a doc claim the code contradicts stand (write `UNKNOWN` or omit it).
2. **Write the concept file** to its layout path (e.g. `docs/kb/self/capabilities/<slug>.md`) using
   the fixed section set from `capability-concept.md`. Frontmatter per `schema.md` — including the
   internal `evidence:` list of the file/contract paths you grounded in (this is the ONLY place paths
   appear).
3. **Obey §0 (hygiene.md) absolutely:**
   - Business logic + observable behavior only. **Omit** transforms, DTO/JSON mapping, serialization,
     internal call chains, per-function detail.
   - **No source pointers in the body** — no `file:line`, no "see file X", no source links. Paths live
     in `evidence:` frontmatter only.
   - **Data: business-critical only** — name only the tables/columns the main logic depends on and
     why. Do **not** restate ubiquitous joins (tenant/customer-on-every-query) — those belong once in
     `context.md`, not here.
   - **UNKNOWN, never guess** — if you cannot ground a fact in the code/contracts, write `UNKNOWN`
     (optionally `UNKNOWN — TODO: <hint>`).
   - Reference contracts **by name/kind** (endpoint, message, table) — never paste a schema.
4. **Be brief.** Capability ≤ ~120 lines; shorter is better. Every sentence must earn its place.

## Integration seams (don't describe a capability in isolation)

- **Identity hand-off.** When this capability emits an identifier another capability needs in a
  different form for the *same* entity (returns a UUID; a peer keys by name), state the bridge — the
  route/capability that resolves one to the other — or point to it via `see_also`.
- **Concrete readiness.** For any async readiness/completion, give a concrete signal (read
  endpoint/RPC + status field + terminal value) or a `see_also` pointer to the owner — never bare prose.
- **Open vs closed lists.** Prefix a non-exhaustive field list with `key fields:`; present a list as
  complete only when you confirmed the full wire shape.
- **Name vs identity.** When a field's name advertises one entity but its value keys another (a
  `runtime_uuid` that actually keys the runtime *instance*), don't let the name stand alone — add a
  gotcha stating what the value truly identifies, or use the accurate name. VERIFY flags a silent divergence.
- **Name peers, never paths.** Every cross-repo/cross-capability reference is by repo + capability
  NAME (in prose and in `see_also`) — never a file path (§0). If you have **not verified** the peer's
  actual capability name from THIS repo's own evidence/context (an explicit pin, a contract this repo
  emits — *never* by reading the sibling repo), the name is an honest best-guess placeholder: mark
  that `see_also` entry `descriptive: true` so kb-sync can reconcile it against the peer's real brief.
  Do not stuff a disclaimer into `intent`, and do not present an invented name as if confirmed.
  "Verified" means grounded in this repo's own evidence, not a guess — the skill never reads sibling
  repos to check a name. For a **same-repo** peer whose exact concept title you cannot see (you draft
  in isolation), still mark it `descriptive: true` with your best-guess name — the VERIFY
  **cross-concept pass** resolves it against the real sibling `title` once the whole bundle exists
  (verify.md §C). Only **cross-repo** placeholders survive to kb-sync. Keep any such best-guess name
  **only in `see_also`** — in the concept **body/prose**, refer to an unverified peer **generically**
  ("another capability in this repo", "the peer that resolves the name"), never by the guessed name. A
  guessed name baked into prose can't be reconciled by §C (which only rewrites frontmatter) and drifts
  silently; keeping the placeholder solely in `see_also` gives the name exactly one writer, so §C's
  frontmatter resolution is sufficient. A peer name **grounded in this repo's own evidence** may still
  appear in prose.
- **Library / compile-time-consumer repos.** If this repo is consumed via `import` / a direct API
  call rather than a wire, apply the *Library / compile-time-consumer repos* framing in
  [capability-concept.md](./capability-concept.md): interaction = a package/API call; seam =
  call-ordering / field-population dependency; async readiness usually N/A; data section usually
  omitted; framework `marshal`/`serialize` terms are expected peer vocab (use `lint-ok`).
- **Consumer / frontend repos.** If this repo only *consumes* other repos' contracts and exposes none
  of its own (a SPA / dashboard / thin client), apply the *Consumer / frontend repo* framing in
  [capability-concept.md](./capability-concept.md): interaction = backend contracts consumed (endpoints
  it calls + fields it depends on); seam = produced-by-peer → consumed-here; async readiness = what the
  UI polls (kept, not omitted); observable behavior = what the UI does with the response; data usually
  omitted; a client-side guard is advisory UX, never authorization.

## Graceful on thin repos

If the target repo has little genuine business logic (a library, config, or plumbing repo), say so
honestly: write a short concept or defer to `interfaces/`. **Do not invent behavior to fill sections.**
An honest `UNKNOWN` or an omitted empty section beats fabrication.

## Output

The concept file, written to disk. Return a one-line summary of what you wrote and any `UNKNOWN`s left
for the orchestrator/VERIFY to note. Do not commit.
