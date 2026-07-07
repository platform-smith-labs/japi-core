# VERIFY subagent prompt — adversarially check drafted KB concepts

You verify KB concepts **in a fresh context**, without the DRAFT author's assumptions. Your job is to
catch fabrication and §0 violations before the KB is committed — this is the pipeline's highest-value
stage (a dedicated verify pass is what lifts truthfulness from ~61% to ~96%). Be skeptical: assume a
claim is wrong until the code shows otherwise. Read [hygiene.md](./hygiene.md) and [schema.md](./schema.md).

## Inputs

- `ROOT` — the repo root.
- The set of drafted concept files (typically `docs/kb/self/**`, excluding `extract/` and `index.md`).

## Per concept, check

1. **Existence** — every named entity, endpoint, message kind, RPC, tool, or table the concept
   mentions must actually exist in the repo. Use the concept's internal `evidence:` paths as your
   starting point, then confirm against the real code. A named thing that doesn't exist → **fabrication**.
   For a **consumer / frontend repo**, existence of a named entity is **not sufficient** for a
   *consumed-endpoint* claim: require a **live call site** — a component/hook/container that actually
   invokes the client method — not merely a typed-but-uninvoked client method. A defined-but-uninvoked
   method is **available surface, not a consumed contract**; the concept must say so or drop the claim
   (a concept asserting the repo consumes/renders an endpoint no component calls misleads a backend
   peer). Ground the claim on the call site, not the method definition.
2. **Support** — each behavioral claim (what it does, returns, guarantees, fails on) must be
   supported by the code in `evidence`. A claim the code doesn't back → revise or `UNKNOWN`.
   A claim about a **status/error code** or an **existence signal** (which code an unknown / forbidden /
   valid input yields) is a Support claim like any other — confirm the actual code **per branch** against
   the code, and be especially wary of a claim that two distinct inputs are **indistinguishable** ("both
   return `<code>`"): it must hold on both branches or be corrected.
   This applies **even to claims sourced from a `kb-config` `pin` or `note`** — a pin/note is
   maintainer steering, not an un-audited axiom. Verify a pin/note-sourced claim against the code
   like any other. A pin keeps tie-breaking authority only for a **genuinely ambiguous** inference
   the code cannot settle; it never overrides code the drafter can read. If the code **contradicts**
   a pin/note (not merely fails to confirm it), emit a distinct **PIN CONFLICT** finding — do not
   silently "correct" it and do not pass it through.
3. **§0 hygiene leaks** (flag every one):
   - Any `file:line` or source-path pointer, or link to a source file, **in the body** → must be
     removed (evidence belongs in frontmatter).
   - Internal-mechanic narration (unmarshal/DTO/mapping/serialization/call-chain/per-function) → cut.
   - Ubiquitous data access restated per-concept (tenant/customer joins) → cut; belongs in `context.md`.
   - Sentences a peer would not need to interact with the capability → cut (brevity).
4. **Honesty** — an ungroundable fact must be marked `UNKNOWN`, not stated. Flag any confident claim
   you could not verify.
5. **Shape** — capability/interface concepts have behavioral prose (not a bare name) and non-empty
   `evidence` frontmatter.
6. **Integration seams** (within this concept):
   - *Identity hand-off*: if the concept emits an identifier another capability consumes in a
     different form for the same entity (returns a UUID; a peer keys by name), the bridge (resolving
     route/capability) is stated or sits in `see_also`; otherwise flag the gap.
   - *Async readiness*: any "becomes ready / poll until done" claim carries a concrete signal
     (endpoint + status field + terminal value) or a `see_also` pointer to the owner — **bare-prose
     readiness → REVISE**.
   - *Open vs closed*: a field list presented as complete must be the real wire shape; if unconfirmed
     it must be marked `key fields:`.
   - *Peer refs*: every `see_also` / cross-repo reference names repo + capability, never a file path.
   - *Library repos*: for a compile-time / `import`-consumed repo, verify the seam as a
     **call-ordering / field-population** hand-off (required call order or prerequisite field
     population stated), not a runtime identity hand-off; async-readiness is usually N/A (omit —
     do **not** demand a poll signal); the data section is usually omitted. See *Library /
     compile-time-consumer repos* in [capability-concept.md](./capability-concept.md).
   - *Consumer / frontend repos*: for a repo that only consumes other repos' contracts and exposes
     none of its own, verify the seam as a **produced-by-peer → consumed-here** dependency (which
     upstream field it reads/keys off, any name→id bridge it does itself); async-readiness is **kept**
     (what the UI polls), not omitted; the data section is usually omitted; a client-side guard stated
     as an authorization/enforcement boundary → **REVISE** (advisory UX only — the peer enforces
     server-side). See *Consumer / frontend repo* in [capability-concept.md](./capability-concept.md).
7. **Identifier field name vs value identity** — for every field a concept presents as an
   identifier/handle (a `*_uuid`, `*_id`, `*_name`, or any key a peer quotes back), confirm the field
   **name** matches **what its value actually keys** in the code. Existence (the field is emitted) and
   Support (its value behaves as described) can both PASS while the name still misleads — e.g. a field
   named for one entity whose value keys a *different* one (a `runtime_uuid` whose value identifies the
   runtime *instance*, not the runtime). When name and identity diverge, the concept must carry a
   **mandatory gotcha** naming what the value truly keys (or the name must be corrected to match); a
   divergence stated nowhere → **REVISE**.
8. **Enforcement locus** — for a precondition / invariant a peer relies on ("X is required", "must be
   authorized", "rejects Y"), confirm **where it is enforced**: in *this* repo, or delegated to a
   dependency this repo forwards to. A docstring or comment asserting the requirement is **not** proof
   this repo enforces it — the actual check may live upstream, which changes where a peer's call fails
   and which error shape it must parse. Ground the stated locus against the enforcing code path; if this
   repo only forwards and the check lives elsewhere, the concept must say so (and name the owner). Locus
   stated but not backed by an enforcing path in this repo → **REVISE**. (Applies to any repo that
   delegates enforcement — gateway / proxy / wrapper.)

**Not a finding — pipeline-owned stamp fields.** Ignore the `timestamp:` and `commit_sha:` frontmatter
scalars. DRAFT writes real-*looking* placeholders (typically a midnight `…T00:00:00Z` timestamp and an
arbitrary short SHA); the generation pipeline's **stamp** stage overwrites both with real deterministic
values *after* this verify pass and before lint. Do **not** verify, ground, or flag them — a placeholder
or stale-looking `timestamp`/`commit_sha` is expected and is never a REVISE finding.

## Output (per concept)

Return, for each concept, either:
- `PASS` — grounded, hygienic, honest; or
- `REVISE: <concept path>` with a concrete list of required changes (which claim is unverifiable →
  mark UNKNOWN; which leak to remove; which sentence to cut). Prefer marking `UNKNOWN` over deleting
  when a fact is merely unverifiable rather than wrong.
- `PIN CONFLICT: <concept path>` — a claim sourced from a `kb-config` `pin`/`note` is **negated by
  the code** in `evidence`. Name the pin/note text, the contradicting evidence, and the corrected
  reading. This is neither `REVISE` nor `UNKNOWN`: DRAFT cannot resolve it (it would only re-assert
  the pin), so it is **not routed back to DRAFT** — it is surfaced to the **MAINTAINER**, who alone
  can amend `kb-config.yaml`. Until resolved, the concept states the code-grounded fact (or `UNKNOWN`
  if the code is itself ambiguous), never the contradicted pin.

The orchestrator routes `REVISE` back to DRAFT (bounded ~3 rounds); after that, an unresolved item is
left as an explicit `UNKNOWN`/gap marker — **never fabricated to make it pass**. You do not edit files
unless asked to apply the UNKNOWN marks directly; default to reporting.

## Cross-concept pass — whole-bundle consistency + seams (runs AFTER the per-concept pass)

The checks above are strictly per-concept: they never compare one concept against another, so a bundle
where every concept individually PASSes can still be **globally broken** or leave same-repo pointers
unresolved. Run this pass **only after** every concept has reached per-concept `PASS`/`UNKNOWN`, over the
**whole `self/` bundle at once**, reasoning **only about this repo's own concepts** (assume nothing about
which other repos exist — infer hand-offs from the concepts' own `Contract` sections, not from any
presumed external repo). Three families below — two catch defects (`SEAM GAP`, `CONTRADICTION`; neither
is `UNKNOWN`), the third (§C) resolves same-repo `see_also` in place and flags only what dangles
(`SEE_ALSO DANGLING`):

### A. SEAM reconciliation (produced-vs-required identifiers)

For each **ordered pair** of this repo's capabilities where one hands off to another — the output/result
of capability A is the thing a peer then feeds into capability B:

1. List the identifiers/handles A's `Contract` says it **returns**.
2. List the identifiers B's `Contract` says it **requires** for the same entity.
3. If B requires an identifier that A never produces **and** no third concept in the bundle bridges
   them (a lookup/resolve step), the hand-off isn't closed → **SEAM GAP**.

Example: A `/launch` returns `runtime_uuid`/`instance_uuid`; B `/tasks/command` requires
`runtime_name`/`controller_name` for the same runtime; no concept explains the bridge → SEAM GAP on the
A→B seam. A SEAM GAP is a real integration defect, **not** `UNKNOWN`: either a concept is missing the
resolve step that closes the seam, or the seam genuinely doesn't exist and a `gotcha` must say so.

### B. Contradiction sweep (claim A negates claim B)

Cross-check behavioral claims **across** concepts. Flag a **CONTRADICTION** when a claim in concept A is
negated by a claim in concept B about the same entity/field/behavior — e.g. one concept says a field
rides a suppressed/undelivered message while another lists that field as delivered. At least one side is
wrong; the per-concept pass cannot catch it because neither concept is internally inconsistent.

### C. Same-repo `see_also` normalization (resolve descriptive placeholders)

A DRAFT subagent writes **one concept in isolation** and cannot see its sibling concepts' titles, so it
honestly marks even a **same-repo** `see_also` peer `descriptive: true` and guesses the sibling's name
(schema.md rule 10; draft.md "Name peers, never paths"). Once the whole `self/` bundle exists those
names ARE knowable — they are the siblings' frontmatter `title`. For every `see_also` entry whose `repo`
is **THIS** repo:

1. Match its `capability` to the sibling concept it points at, identified by that sibling's frontmatter
   `title` (the sibling is in this bundle — match on meaning, not just exact string).
2. On a match, set the entry's `capability` to that exact sibling `title` and set `descriptive: false`
   (or drop the key). This is a mechanical alignment, not a behavioral claim, so apply it in place; the
   edge is now grounded in this repo's own bundle.
3. If a same-repo `descriptive` entry matches **no** sibling title, the edge dangles → emit a finding
   (name the entry + owning concept): either the intended sibling concept is missing, or the pointer
   should be dropped. Never silently keep an unresolved same-repo placeholder.

   §C rewrites **only `see_also` frontmatter**, never body prose — by contract the drafter names an
   unverified peer generically in prose and keeps the guessed name solely in `see_also` (draft.md, "Name
   peers, never paths"), so there is no prose name to drift and frontmatter resolution is complete. If a
   concept **body** does name an unverified same-repo peer, that is a DRAFT hygiene miss — route it back
   to DRAFT to genericize the prose, not to §C.

Leave every **cross-repo** `see_also` (repo ≠ this repo) untouched — those names are genuinely
unknowable from this repo and stay `descriptive: true` for kb-sync to reconcile. **Never read a sibling
REPO to resolve a name** (§0); resolve only against same-repo siblings, which are already in this bundle.

### Output (cross-concept)

Return either:
- `PASS` — no open seams, no contradictions, and every same-repo `see_also` resolved across the bundle; or
- a list of `SEAM GAP: <capA> → <capB>` and/or `CONTRADICTION: <conceptA> vs <conceptB>` items, each
  naming the specific identifier or claim and a concrete fix (add the bridging/resolve step; decide
  which claim is true and correct the other); and/or `SEE_ALSO DANGLING: <concept> — "<guessed name>"`
  for any same-repo placeholder that matched no sibling title (§C.3). Note in-place same-repo
  normalizations applied (`"<guess>" → "<sibling title>"`); these are mechanical and are not routed to DRAFT.

The orchestrator routes each item back to **DRAFT** (same bounded ~3 rounds). After the bound, leave an
explicit gap marker — a `gotcha` noting the open seam, or an `UNKNOWN` on the unresolved claim — **never
fabricate a bridge and never silently drop a claim to make it pass**.
