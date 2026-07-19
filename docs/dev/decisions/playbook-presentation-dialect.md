# Decision: Playbook presentation dialect — Tier-1 task-event vocabulary over watched files

**Date**: 2026-07-14
**Status**: Accepted
**Context**: The playbook substrate deleted task tables (`agentic-workflow-substrate-architecture.md`;
0013 addenda 5–7). The UI's task/sub-task view of a playbook run is therefore a **fold over
`watched_file_event`** rows (ingested lines of pod-local jsonl logs). Those rows are deliberately
vocabulary-free — so the tree needs a *presentation contract*. Per-playbook mappers in Go are
rejected (they destroy playbook-agnosticism); the contract rides the S1 declaration's **`file_key`**
role handle instead. Verified against real data: a ~40-line reducer over actual `work.jsonl` lines
produces the full tree (0013 addendum 7).

## Decision

Run visualization is **tiered by contract**. Tier 0 needs none and works for any playbook. Tier 1 is
a minimal, documented **task-event dialect** bound to a well-known `file_key`: any playbook whose
declared file emits it gets the full task/sub-task tree UI. Our own `/work` playbook's `work.jsonl`
already speaks the dialect verbatim. The fold is always **derived, rebuildable, and presentation-only**
— it never feeds claims, leases, readiness, or dispatch (the Class-1 fence).

## Rules

### 1. The tiers

| Tier | Contract | UI |
|---|---|---|
| **0** | none | generic run view: per-pod/per-declaration streams, event counts, last-activity/staleness, raw line timeline |
| **1** | declared `file_key = "work-log"` + the dialect below | task/sub-task tree: nodes, hierarchy, status, phase ladder, attention badges |
| **2** (post-V1) | custom mapping carried in the `playbook` definition (P1) | Tier-1 UI without adopting the dialect's event names |

### 2. The Tier-1 dialect (file_key `work-log`)

JSONL, one event per line: `{seq, ts, type, actor, ...}` with per-type fields:

```jsonc
// CORRECT — the five Tier-1 event types (any other type is ignored by the tree fold,
// but still shown in the Tier-0 timeline: forward-compatible)
{"seq":1,"ts":"…","type":"created","title":"…","parent":"<node-id>"}   // parent optional (root)
{"seq":2,"ts":"…","type":"status_changed","to":"implementation"}
{"seq":3,"ts":"…","type":"phase_done","phase":"requirements"}
{"seq":4,"ts":"…","type":"escalated","note":"…"}
{"seq":5,"ts":"…","type":"artifact_added","kind":"research","path":"…","title":"…"}
```

**Node identity** = the work-item directory segment of the declaration's `file_path`
(`…/work-<id>/work.jsonl` → node `work-<id>`); `parent` references such ids. One declared file =
one tree node's stream.

```jsonc
// WRONG — encoding arbitration in presentation events
{"type":"claim","task":"…"}   // claims/leases NEVER ride this plane (Class-1 fence)
```

### 3. Fold rules (normative for every consumer — verified against real logs)

1. **Idempotent + order-insensitive.** Dedup on `(file, seq)`; status = last-writer-wins by
   `(ts, seq)`; phases are a **set** (real logs contain duplicate `phase_done` events).
2. **Escalation clears.** `escalated` sets the attention badge; any later `status_changed` or
   `phase_done` on the node clears it (a latched badge was observed on real data — resolved
   escalations must not render forever).
3. **Orphans buffer, never drop.** A child whose `parent` node has no `created` yet renders under a
   synthetic root with a "parent pending" marker and re-parents on arrival (multi-pod sync order is
   not causal order).
4. **Malformed lines degrade, never poison.** A line that fails to parse is excluded from the
   Tier-1 fold (quarantine count surfaced), remains visible in Tier 0.

### 4. Where the fold runs

Per-run tree = **client-side fold** in ps-ui (REST backfill + SSE-over-poll live tail, on the
existing session/launch-events read pattern; reducer modeled on the ACP transcript reducer).
Cross-run boards = **SQL fold-on-read** in ps-api. Materialize a server-side read model **only** on
a forcing function (slow cross-run derived queries; unbounded per-run volume; a second consumer) —
it is always rebuildable from the log. **Never** a mutable task table (dual-write drift, no
rebuild, loses replay; 0013 addendum 7's precedent survey: Temporal/Jaeger/Langfuse all derive
trees from event logs).

## Rationale

The dialect gives the UI a stable contract while keeping the platform playbook-agnostic: the
knowledge lives in a declared file role, not in platform code. Our dogfood playbook needs zero
changes (its `work.jsonl` is the dialect). Deriving the view keeps the file/log as the single
origin (no dual write), and yields run playback (history scrubbing) for free.

## Exceptions

1. **Tier-2 custom mappings** (post-V1, via the `playbook` definition) may rename event
   types/fields — the fold rules (§3) still bind the mapped result.

## Enforcement

Fold-side: the §3 rules are mandatory in every consumer (ps-ui reducer, ps-api fold queries).
Publish-side: dialect lint lands with the B1 publish path / P1 definition module — until then,
conformance is convention (same posture as the single-writer jsonl discipline).

## See Also

- [agentic-workflow-substrate-architecture](./agentic-workflow-substrate-architecture.md)
- [append-only-work-event-log](./append-only-work-event-log.md)
- [playbook-terminology](./playbook-terminology.md)
- Design of record: `docs/work/work-2607051522-substrate-design-of-record/research/0013-…` addenda 5–7
