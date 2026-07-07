---
name: kb-bootstrap
description: >-
  Generates or regenerates a repository's knowledge base at docs/kb/ — clinical, business-logic-only
  summaries of each capability, written for AI agents in OTHER repos to understand this repo's
  externally observable behavior, interfaces, and gotchas WITHOUT reading its source. Runs a
  deterministic extract → per-capability draft → adversarial verify → lint → render pipeline. Use
  when asked to bootstrap, generate, refresh, or update a repo's KB, project brief, or capability
  docs for cross-repo/peer consumption. NOT for exhaustive per-function code documentation.
disable-model-invocation: true
argument-hint: "[target-repo-path]"
---

# kb-bootstrap — generate a repo's cross-repo knowledge base

You are the **orchestrator** of a five-stage pipeline that produces `docs/kb/` for one repository.
The KB's sole audience is an agent in a *different* repo scoping a task that touches this one — so it
is a **summary of business logic and observable behavior, never exhaustive code documentation**.

**Read first (the contract):** [references/hygiene.md](./references/hygiene.md) (the §0 keep/omit/never
rules — non-negotiable), [references/capability-concept.md](./references/capability-concept.md) (the
core deliverable's shape + GOOD/BAD example), [references/schema.md](./references/schema.md),
[references/layout.md](./references/layout.md), [references/kb-config.md](./references/kb-config.md).

`${CLAUDE_SKILL_DIR}` is this skill's directory; the helper scripts (`kb-extract`, `kb-stamp`,
`kb-lint`, `kb-render`) live in `${CLAUDE_SKILL_DIR}/scripts/`.

## Golden rules (violating any is a defect)

1. **Business logic only. Brevity is clarity.** Omit internal mechanics (transforms, mapping,
   serialization, call chains, per-function detail). If a peer wouldn't need a sentence to interact
   with this repo, cut it.
2. **No consumer-facing source pointers.** Never write `file:line` or source links into a concept
   body. Grounding goes in the internal `evidence:` frontmatter only.
3. **UNKNOWN over guess.** Never fabricate. An ungroundable fact is written literally as `UNKNOWN`.
4. **Never commit on lint failure.** Never touch `peers/`, `notes/`, or `kb-config.yaml`.

## Pipeline

### 0. Resolve + read steering
- `ROOT` = the argument if given, else the current repo root. All scripts take `ROOT` as `$1`.
- If `ROOT/docs/kb/kb-config.yaml` exists, read it: honor `exclude`, seed from `capability_hints`,
  inject `notes` + `pins` into every DRAFT prompt as maintainer steering — a strong prior for
  ambiguous inference, **not** ground truth over readable code. A pin the code contradicts is a
  **PIN CONFLICT** VERIFY surfaces to the maintainer (see references/verify.md), never copied.

### 1. EXTRACT (deterministic)
Run `bash ${CLAUDE_SKILL_DIR}/scripts/kb-extract.sh "$ROOT"`. This writes `docs/kb/self/extract/`
(contracts, manifests, structure, git facts). Read those files — they are your grounding fact sheet,
**not** KB content to publish.

### 2. Enumerate capabilities
From the extract fact sheet + `capability_hints` + the repo's own docs (README/CLAUDE.md) + a
targeted read of entry points, list the repo's **material functionalities** — the things a peer
would interact with. Order them so a capability others depend on is drafted first (rough topological
order). **Do not invent capabilities to fill a template** (see graceful-degradation below).

### 3. DRAFT (LLM, fan out — one subagent per capability)
For each capability, spawn a subagent using [references/draft.md](./references/draft.md) as its prompt,
passing: `ROOT`, the capability name + seed pointers, the extract fact sheet, and any `notes`/`pins`.
Each subagent writes one `docs/kb/self/capabilities/<slug>.md`. Also draft the singular concerns
(`overview.md`, `context.md` — including the repo's ubiquitous data facts stated **once here**,
`glossary.md`), `interfaces/`, `gotchas/`, and `decisions/` (one-line summaries referencing
`docs/dev/decisions/` ADRs — never restated).

### 4a. VERIFY — per concept (LLM, SEPARATE context — mandatory)
Spawn a verify subagent using [references/verify.md](./references/verify.md) over the drafted
concepts. It existence-checks named entities against the repo via each concept's `evidence`, confirms
behavioral claims, and flags §0 leaks (source pointers, internal mechanics, non-peer-relevant
sentences), marking unverifiable facts `UNKNOWN`. It returns per-concept `pass` or a revision list.
Route failures back to DRAFT (bounded — ~3 rounds per concept; then leave an `UNKNOWN`/gap marker,
never fabricate).

### 4b. VERIFY — cross-concept (LLM, SEPARATE context — mandatory)
Once every concept has reached per-concept `PASS`/`UNKNOWN`, spawn a **second** verify subagent — fresh
context, the **whole `self/` bundle** — using the **"Cross-concept pass"** section of
[references/verify.md](./references/verify.md). Per-concept VERIFY (4a) checks each concept against its
own `evidence` and never compares two concepts, so a bundle of individually-passing concepts can still
be un-wireable or self-contradictory. This pass, reasoning **only about this repo's own concepts**:
- **SEAM reconciliation** — for each ordered capability hand-off, reconciles the identifiers one
  capability *returns* against those the next *requires*, flagging a **SEAM GAP** when the hand-off
  isn't closed by any concept.
- **Contradiction sweep** — flags a **CONTRADICTION** when one concept's claim negates another's.
- **Same-repo `see_also` normalization** — resolves each same-repo `see_also` placeholder to the
  sibling's real `title` (now knowable across the bundle), flagging a **SEE_ALSO DANGLING** when it
  matches no sibling. Cross-repo placeholders are left for kb-sync.

Route each `SEAM GAP` / `CONTRADICTION` back to DRAFT (bounded — same ~3 rounds; then leave an explicit
gap marker — a `gotcha` for an open seam, an `UNKNOWN` for an unresolved claim — never fabricate a
bridge or drop a claim). Re-run 4b after revisions until the bundle is clean, **before** proceeding.

### 5. STAMP → LINT → RENDER → commit
- `bash ${CLAUDE_SKILL_DIR}/scripts/kb-stamp.sh "$ROOT" <concepts-regenerated-this-run…>` to stamp
  real `timestamp:` + `commit_sha:` frontmatter (replacing DRAFT placeholders). Pass the concept files
  (re)generated this run to keep diffs minimal; with no file args it restamps all concepts. Run this
  ONCE, before lint — never from kb-render (which must stay byte-identical).
- `bash ${CLAUDE_SKILL_DIR}/scripts/kb-lint.sh "$ROOT"`. On **FAIL**, fix the named concepts
  (DRAFT/VERIFY) and re-lint. **Do not proceed while lint fails.**
- `bash ${CLAUDE_SKILL_DIR}/scripts/kb-render.sh "$ROOT"` to regenerate the `index.md` projections.
- Append one entry to `docs/kb/log.md` (newest first): date, HEAD short-sha, concepts regenerated,
  UNKNOWN count, open SEAM GAP count.
- **Commit** the `docs/kb/` changes through the repo's normal flow (or leave staged if the caller
  asked). Do not commit if lint still fails.

### 5b. ACCEPTANCE EVAL (task-scoping — the decisive gate)
After LINT passes, run the task-scoping eval per [evals/task-scoping.md](./evals/task-scoping.md): in a
**fresh KB-only context** (no source) derive a scenario from THIS repo's own `self/capabilities/` (a
chained capability pair that crosses a seam), pose the peer task, and run a **fresh KB-only agent** (no
source, no other repo) to answer it. Then run the **ground-truth check** — a source-side pass (this
repo's source is the oracle) that verifies the specific claims the KB-only agent relied on are actually
true; a claim the source contradicts is a KB defect that **FAILS the eval even when scoping succeeded**.
Write the outcome as a record under `docs/kb/self/eval/task-scoping-<HEAD-short-sha>.md`. **On
`result: fail` (a missing scoping criterion OR a ground-truth defect), route the gaps back to
DRAFT/VERIFY, regenerate, re-lint, and re-run — do not commit the KB while the latest record is
`fail`.**

### 6. Discovery pointer
Idempotently ensure the repo's `CLAUDE.md` (and/or `AGENTS.md`) carries a ≤5-line pointer, e.g.:
> **Knowledge base**: `docs/kb/` summarizes this repo's capabilities for peer repos. Before a
> cross-repo task, read a peer's `docs/kb/index.md` → `self/capabilities/`. Regenerate with
> `/kb-bootstrap`.

Add it only if absent; never duplicate.

## Regeneration (re-runs are idempotent)

- `self/extract/**` — regenerated **wholesale** every run (deterministic).
- Narrative concepts — regenerate **only where their `evidence` drifted** (kb-lint's drift report
  names them); an unchanged tree yields no semantic diff.
- `self/eval/**` — one **new** acceptance record appended per run (keyed by HEAD sha); prior records
  are left intact.
- **Never** modify `peers/**`, `notes/**`, or `kb-config.yaml`.

## Graceful on low-business-logic repos

A thin library / plumbing / config repo legitimately has few "capabilities." **Do not invent
capability narrative to fill the roster.** Produce a sparse `capabilities/` and lean on
`interfaces/` (what it exposes) + `overview`/`context`. Coverage follows the repo's real business
logic, not a template — an honest short KB beats a padded one.

## Notes

- The KB **content** is model-agnostic (a peer may run a different agent); keep Claude-specific
  mechanics in this skill, never in the KB body.
- DRAFT/VERIFY are internal subagent roles driven by the `references/` prompts — not separate
  user-facing skills.
