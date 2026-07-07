# Context — work-2607070223-kb-japi-core

**Parent**: work-2607062158-kb-bootstrap-skillset @ solution (the KB-bootstrap skillset).
**Mission**: run kb-bootstrap on `repos/japi-core`, generate its `docs/kb`, evaluate it from a peer's
perspective, and relay skillset finetunes upstream for maintainer approval.
**Distribution**: skill is run CENTRALLY from the monorepo root (no per-repo install).

---

## Inherited scaffold rules (from work-2607051522-agentic-substrate/scaffold-rules.md)


These rules are **inherited by every work item scaffolded under this parent, at any depth**
(children, and children of mid-level nodes). The scaffold flows (`/conduct scaffold`,
`/work --parent-work`) copy this file's rules into each new descendant's `context.md` and record
the inheritance. Descendants that themselves become parents pass these rules on unchanged.

## Rule 1 — Epistemics: prior research is NOT gospel (maintainer directive, 2026-07-06)

Any prior research, decisions, or design documents referenced in a work item's context — including
the agentic-substrate design of record (research/0001–0012 and its decision ledger) — are
**NOT TO BE TAKEN AS GOSPEL TRUTH**. They are pointers to prior conversation, nothing more. Every
work item's research phase MUST:

1. **Independently research every fact** (web + codebase) as if the prior documents did not exist;
2. **Draw its own conclusions** from that evidence;
3. **Then critically match** those conclusions against the existing research and settled decisions,
   producing an explicit reconciliation (confirmed / revised / contradicted, with evidence) —
   where an independent conclusion disagrees with a settled decision, say so plainly and
   **escalate the conflict to the maintainer** rather than silently deferring to either side.

## Rule 2 — Terminology

Use the settled vocabulary: **playbook** / `playbook_run` (never "workflow" for our concept),
per [`docs/dev/decisions/playbook-terminology.md`](../../dev/decisions/playbook-terminology.md).
