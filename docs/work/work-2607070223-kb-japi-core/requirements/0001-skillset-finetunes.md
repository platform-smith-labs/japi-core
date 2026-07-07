# Requirements — skillset finetunes (relay up) + local KB status

**Work Item**: work-2607070223-kb-japi-core
**Parent**: work-2607062158-kb-bootstrap-skillset @ solution
**Date**: 2026-07-07

Derived from the independent evaluation (`research/0001-kb-evaluation-research.md`). This work item's
mission is (a) generate japi-core's KB, (b) evaluate it, (c) relay skillset finetunes upstream for
maintainer approval. The KB passed the task-scoping acceptance eval with **no local fixes required**,
so the only outbound work is the finetune relay.

## Local KB fixes needed: NONE

The task-scoping eval (`docs/kb/self/eval/task-scoping-910ed6a.md`) passed all 5 criteria with 0 gaps.
No `implementation` phase (B-series local KB fixes) is required for japi-core — unlike peers whose
first eval surfaced answerability holes. The 3 UNKNOWNs are honest and correct (repo license; JWT
not-found status is consumer-owned; cross-repo consumer capability names deferred to kb-sync).

## Proposed skillset finetunes (relayed to solution, kind=fyi, for maintainer approval)

### F1 — Normalize SAME-repo `see_also` edges in the cross-concept VERIFY pass

**Problem.** DRAFT subagents draft one concept in isolation and cannot see sibling capability files
(they may not exist yet), so they honestly mark **same-repo** `see_also` entries `descriptive: true`
("name not verified from this repo's evidence") and guess the sibling's title. After the full bundle
exists, those sibling names ARE knowable, so the edges are left less machine-usable than they should
be.

**Proposed change.** Extend `references/verify.md` cross-concept pass (or `kb-render.sh`) with a step:
for every `see_also` entry whose `repo` == this repo, set `descriptive: false` and rewrite
`capability` to match the referenced sibling concept's actual frontmatter `title`. Leave
`descriptive: true` only for genuine cross-REPO pointers. (Applied manually this run for all 10
capability files.)

**Acceptance.** After a bundle generation, no same-repo `see_also` entry is `descriptive: true`, and
each same-repo `capability` value equals an existing sibling's `title`.

### F2 — Add explicit guidance for library / compile-time-consumer repos

**Problem.** `capability-concept.md`, `draft.md`, and `verify.md` are framed for **runtime services**:
"how a peer interacts" via endpoint/RPC/message-kind, async-readiness seams ("poll until ready"),
tenant/customer tables. A **library** consumed via `import` (japi-core, and likely ps-cli /
db-migration) has none of these: the interaction is a Go/API call, the seam is *required call
ordering / which call populates which field*, and the "business-critical data" section is normally
empty. The existing "Graceful on thin repos" note covers *sparse* repos but not *rich libraries* that
need a different **framing**, not fewer concepts.

**Proposed change.** Add a short "Library / compile-time-consumer repos" subsection (mirroring the
thin-repos note) to `capability-concept.md` and reference it from `draft.md`/`verify.md`:
- "How a peer interacts" = the exported API call, not a network endpoint.
- Seam = call-ordering / population dependency (e.g. middleware X must run before context field Y is
  populated), verified by the cross-concept pass as an ordering hand-off rather than an
  identifier-produced-vs-required hand-off.
- Data section usually omitted (the library owns no schema; state once in `context.md`).
- Note that for HTTP/serialization frameworks the `kb-lint.sh` internal-mechanic term-flags
  (`marshal`/`serialize`) are expected peer vocabulary; `lint-ok` markers are the norm, not a smell.

**Acceptance.** A library repo's DRAFT no longer forces runtime framing; VERIFY treats a
call-ordering dependency as a first-class seam.

**Note.** F1/F2 may overlap with the genericity finetunes already relayed by orchestrator/runtime
(parent Upstream Messages). Surfaced independently from japi-core's evidence; maintainer to dedupe.
