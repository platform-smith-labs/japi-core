# Decision: Research provenance tags and relay-first unknowns

**Date**: 2026-07-19
**Status**: Accepted
**Context**: On 2026-07-19, research for `work-2607141033-playbook-catalog-skeleton` (orchestrator repo) consulted the folded runtime KB (`docs/kb/peers/runtime/`, folded at commit 33f85d5 on 2026-07-06) for file-watcher behavior. The watcher shipped ~2026-07-13 — **after** the fold — so the KB contained zero watcher facts and would have silently yielded stale/absent conclusions. It was caught only because one research agent happened to check the fold date. This decision makes that check mandatory: force provenance on every cross-repo claim and turn unknowns into relays as a research **output**, operationalizing the existing "KB-first, relay on gap" law ([repo-isolation-kb-first-cross-repo](./repo-isolation-kb-first-cross-repo.md)) inside the research pipeline.

## Decision

Every cross-repo factual claim in research and requirements documents carries exactly one provenance tag (`[CODE <path:line>]`, `[KB@<fold-ref>]`, `[RELAY <slug>]`, `[UNKNOWN]`); KB use requires a recorded fold vintage; system-critical unknowns become drafted relays as research output; reply relays fold back into research addenda; and plans never build silently on a system-critical unknown.

## Rules

### 1. Rule A — Provenance tags on every cross-repo claim

Every cross-repo factual claim in a research or requirements document MUST carry exactly one tag:

| Tag | Meaning |
|-----|---------|
| `[CODE <path:line>]` | verified in THIS repo's code |
| `[KB@<fold-ref>]` | from the folded KB; `<fold-ref>` = `<peer>:<fold-sha>` (e.g. `[KB@runtime:e154f05]`) |
| `[RELAY <slug>]` | confirmed by a peer's reply relay |
| `[UNKNOWN]` | none of the above; treated as unverified |

An untagged cross-repo claim is a **validation failure** of the research document.

```markdown
<!-- CORRECT: claim anchored in this repo's code -->
watch_files paths are absolute pod paths [CODE pkg/protocol/protocol.go:541-549]

<!-- WRONG: no tag, and the KB predates the watcher — stale/absent knowledge riding as fact -->
the runtime watches files via inotify
```

### 2. Rule B — KB vintage check before any peer-KB use

Before using any `docs/kb/peers/<repo>/` content, record that KB's fold commit + fold date in a **KB Vintage** table in the research document. The fold sha/date is recorded in each peer KB's `docs/kb/peers/<repo>/index.md` header line ("Folded from `<repo>` @ `<sha>` (`<date>`)") and per-peer as `@<sha>` in `docs/kb/index.md`. If there is any evidence the researched area **postdates the fold** (newer decision docs, work items, protocol/code comments referencing features the KB lacks), downgrade those claims to `[UNKNOWN]`.

```markdown
<!-- CORRECT: vintage recorded, postdate risk assessed and acted on -->
| Peer KB | Fold commit | Fold date | Postdate risk |
|---|---|---|---|
| docs/kb/peers/runtime/ | e154f05 | 2026-07-07 | file-watcher shipped ~2026-07-13 → watcher claims downgraded to [UNKNOWN] |

<!-- WRONG: citing the peer KB with no vintage check -->
Per the runtime KB, session output is streamed over the controller WS. [KB@runtime]
(no fold commit/date recorded; no assessment of whether the area postdates the fold)
```

### 3. Rule C — System-critical UNKNOWNs become drafted relays (research output)

Every research document MUST end with a **Relay Candidates** section: each SYSTEM-CRITICAL `[UNKNOWN]` becomes a fully drafted relay (frontmatter `from`/`to`/`kind`/`phase`/`ask` + concrete questions phrased for code-level answers). For **parent-bound** work items, the relay file is actually written under `relays/outbound/` and `relay_sent` appended via `scripts/wlog.sh` per the `/work` relay rules. For **standalone** items, the drafts stay in the doc (no conductor channel exists). Non-critical unknowns are listed but NOT relayed — the existing "do not relay for routine confirmation" rule stands. A document with no system-critical unknowns still carries the section, stating `None` — a relay is never invented to fill it.

```markdown
<!-- CORRECT: system-critical unknown → fully drafted relay, code-level questions -->
### Drafted relay: watcher-event-shape
---
from: orchestrator/work-2607141033-playbook-catalog-skeleton
to: runtime
kind: blocks
phase: requirements
ask: "What is the exact watch_files event payload and debounce behavior?"
---
1. Which struct serializes the file-change event, and what are its fields (path form, op enum)?
2. Is there debouncing/coalescing, and at what interval?

<!-- WRONG: unknown buried as an assumption, no relay drafted -->
We assume the runtime debounces watcher events; if not, the orchestrator will dedupe. (system-critical, untagged, never asked)
```

### 4. Rule D — Fold reply relays back into research

When a reply relay arrives (inbound), fold its answers into the work item's research as a **new numbered research addendum**, upgrading `[UNKNOWN]` → `[RELAY <slug>]` where answered, and resolve the relay per the existing lifecycle (`relay_resolved` event; the file never moves).

```bash
# CORRECT: answers land as a registered research addendum, then the relay is resolved
Write "$WD/research/0003-watcher-event-shape-addendum.md"   # upgrades [UNKNOWN] → [RELAY watcher-event-shape]
scripts/wlog.sh "$WD" artifact_added kind=research path=research/0003-watcher-event-shape-addendum.md title="Relay fold-back: watcher event shape"
scripts/wlog.sh "$WD" relay_resolved direction=inbound slug=watcher-event-shape note="folded into research/0003"
scripts/wrender.sh "$WD"

# WRONG: resolve the relay without folding — the answer dies in the relay file, research stays [UNKNOWN]
scripts/wlog.sh "$WD" relay_resolved direction=inbound slug=watcher-event-shape
```

### 5. Rule E — Consume-side gates (plans and requirements)

`planv0` MUST NOT build a plan step on a system-critical `[UNKNOWN]` — it either **blocks on the drafted relay** or **carries the risk forward explicitly** in the plan. Requirements citing research inherit the tags of the claims they cite. Untagged cross-repo claims in cited documents are treated as `[UNKNOWN]`.

```markdown
<!-- CORRECT: risk carried forward explicitly -->
Phase 2 — built on UNKNOWN: runtime debounces watcher events — mitigation: orchestrator-side dedupe window; assert real behavior in phase-2 integration test before relying on it.

<!-- WRONG: plan step silently assumes the unverified claim -->
Phase 2: subscribe to watcher events (runtime already debounces, so no dedupe needed).
```

## Rationale

- **Stale KB fails silently.** A folded KB is a snapshot; anything shipped after the fold is invisible, and the KB returns confident-looking absence rather than an error. The `work-2607141033` incident showed the only defense was a lucky manual check — provenance tags plus a mandatory vintage table make the check structural, so a stale citation is grep-detectable rather than luck-detectable.
- **Repo isolation makes provenance the only audit trail.** Under [repo-isolation-kb-first-cross-repo](./repo-isolation-kb-first-cross-repo.md), an agent cannot re-verify a peer claim by reading the peer's source. The tag records *how* a claim was established (own code, folded KB @ vintage, live peer answer, or not at all), which is the only way a reviewer or downstream consumer can judge it.
- **Unknowns must move, not linger.** Turning each system-critical `[UNKNOWN]` into a drafted relay makes the gap itself a research deliverable with an owner and a lifecycle, instead of a footnote that planning later trips over. The consume-side gate (Rule E) closes the loop: an unknown either gets answered (relay → fold-back → `[RELAY <slug>]`) or is carried as a named, mitigated risk — never silently baked into a plan.

## Exceptions

1. **Trivial / non-system-critical facts need no relay** — Rule C's own carve-out: non-critical `[UNKNOWN]`s are listed in Relay Candidates but not relayed, preserving the existing "do not relay for routine confirmation" rule.
2. **Legacy `/epic` flows are untouched** — frozen commands and items under `docs/epics/` are never migrated to these rules.

## Enforcement

- `/research` and `/research_codebase` carry the rules and a document-level validation step (untagged cross-repo claim, missing KB Vintage table, or missing Relay Candidates section = validation failure); `/work` requires them of its auto-created research (Phase 2 and auto mode).
- `/planv0` enforces the Rule E gate in its validation checklist (Step 4d) and final check (Step 5); `/new_req` enforces tag inheritance.
- Code review: the exact tag spellings (`[CODE `, `[KB@`, `[RELAY `, `[UNKNOWN]`) are grep-able — a cross-repo claim without one in a research/requirements diff is a defect.

## See Also

- [Repo isolation — KB-first cross-repo knowledge](./repo-isolation-kb-first-cross-repo.md)
- [Append-only work event log](./append-only-work-event-log.md)
- [Parent/child work items and conduct](./parent-child-work-items-and-conduct.md)
