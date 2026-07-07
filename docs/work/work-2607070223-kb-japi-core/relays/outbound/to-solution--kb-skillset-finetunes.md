---
from: japi-core/work-2607070223-kb-japi-core
to: solution
kind: fyi
phase: requirements
ask: "Two proposed kb-bootstrap finetunes from japi-core's independent eval — for maintainer approval"
---

# japi-core → kb-bootstrap skillset finetunes (FYI, for maintainer approval)

japi-core generated its `docs/kb/` via the central kb-bootstrap skillset and independently evaluated
it (task-scoping eval **PASS**, 0 gaps, no local fixes needed). Two finetune proposals emerged from
the evidence — neither blocks japi-core's own KB (already accepted at HEAD 910ed6a). Full detail in
`repos/japi-core/docs/work/work-2607070223-kb-japi-core/requirements/0001-skillset-finetunes.md`.

## F1 — Normalize SAME-repo `see_also` edges in the cross-concept VERIFY pass

DRAFT subagents draft one concept in isolation and cannot see sibling titles, so they honestly mark
same-repo `see_also` entries `descriptive: true` and guess the sibling name. Once the whole bundle
exists those names are knowable. **Proposal:** in `references/verify.md`'s cross-concept pass (or
`kb-render.sh`), for every `see_also` whose `repo` == this repo, set `descriptive: false` and align
`capability` to the sibling concept's real frontmatter `title`; keep `descriptive: true` only for
cross-REPO pointers. (Applied manually this run.)

## F2 — Add explicit guidance for library / compile-time-consumer repos

The capability-concept / draft / verify prompts assume a runtime service (endpoints, RPC, async
readiness, tenant tables). A library consumed via `import` (japi-core; likely ps-cli, db-migration)
interacts via a Go/API call, its seams are *call-ordering / field-population* dependencies, and its
data section is normally empty. The existing "thin repos" note covers *sparse* repos, not *rich
libraries* needing different **framing**. **Proposal:** add a short "Library / compile-time-consumer
repos" subsection to `capability-concept.md` (referenced from draft/verify): API-call interaction;
seam = call-ordering/population dependency (verified as an ordering hand-off); data section usually
omitted; and note that for HTTP/serialization frameworks the `marshal`/`serialize` lint term-flags
are expected peer vocabulary (`lint-ok` is the norm).

**Overlap note:** F1/F2 may overlap with genericity finetunes already relayed by orchestrator/runtime.
Surfaced independently from japi-core's own evidence; please dedupe at the maintainer's discretion.
