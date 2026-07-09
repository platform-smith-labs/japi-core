# Decision: Parent/Child Work Items + /conduct (epic entity removed)

**Date**: 2026-07-05
**Status**: Accepted
**Context**: Brainstorm epic epic-2607031635 (agentic workflow substrate). The L0-skill decomposition
analysis (work item research/0008) showed `/epic` was two things fused: a container entity holding
almost no state of its own, and a conductor role — the prototype of the product's
conductor-as-command. The entity is removed; the role is promoted. This supersedes the epic *entity*
model while preserving the barrier semantics of
[epic-conductor-barrier-workflow](./epic-conductor-barrier-workflow.md) verbatim and building on
[append-only-work-event-log](./append-only-work-event-log.md).

## Decision

The epic entity is removed for new work. A **work item is either standalone or a child**
(`parent=<work-id>` + `parent_project=<repo>` on its `created` event); nesting is **N-level**
(relaxed 2026-07-06 from the original 2-level profile): any node may parent children, with a
create-time chain validation and a settling rule (Rule 1). The parent work item plays the epic
role (board, roadmap, wishlist link, conductor journal); a new **`/conduct`** command carries all of
`/epic`'s conductor functionality. **Every jsonl has exactly one writer**: a new conductor-only
`relays.jsonl` per child holds delivery events; `work.jsonl` stays worker-owned. **`escalated`**
joins the event/status vocabulary as the out-of-play-until-human terminal.

## Rules

### 1. N-level nesting with chain validation + the settling rule (relaxed 2026-07-06)

The L0 contract (the verb set: create/sync/report/block/resolve/escalate) is written
N-level-generic — matching the product substrate's self-referential `parent_id`. As of 2026-07-06
the file-mode tooling matches it: **any node may parent children** (program → sub-effort →
per-repo strand; the original 2-level profile was a "for now" simplification retired on its first
real use case — one coordinated sub-effort fanning out across repos while the program tracks
several such sub-efforts, each needing its own barrier). Two rules replace the old guard:

- **Chain validation at child-create**: walk the intended parent's `parent=` links upward — the
  chain must terminate at a **standalone root**, must not revisit any node (**no cycles**), and
  must not include the item being created.
- **Settling rule for mid-level nodes**: a node with children may not settle its final/validation
  `phase_done` toward its own parent while its children board is incomplete — its children's
  completion is its evidence. Earlier phases settle on conductor judgment. `conduct-board.sh`
  prints the settling hint when a mid-level node's children complete.

Boards are per-level by construction: child discovery matches `parent=<this-node>` only, so each
node conducts exactly its direct children.

```bash
# CORRECT: three levels — program → sub-effort → repo strand
scripts/wlog.sh "$SUB" created title="..." parent=work-...-program parent_project=solution ...
scripts/wlog.sh "$LEAF" created title="..." parent=work-...-sub-effort parent_project=solution ...

# WRONG: a parent= chain that cycles or never reaches a standalone root — reject at create
scripts/wlog.sh "$A" created ... parent=<B>   # where B's chain already contains A
```

### 2. Writer partition by FILE (one writer per jsonl)

| Log | Events | Sole writer |
|---|---|---|
| child `work.jsonl` | `created`*, `status_changed`, `phase_done`, `artifact_added`, `relay_sent`, `relay_resolved`, `escalated`, `note`, `meta_changed` | child-side tooling (`wlog.sh`). *`created` may be seeded by the conductor at scaffold time — the **birth exception** — after which the conductor never writes it again |
| child `relays.jsonl` | `relay_received`, `relay_synced` | **conductor only** (`rlog.sh`, from `/conduct sync`) |
| parent `work.jsonl` | conductor decisions (tick notes, round-cap `escalated`, status) | the conductor |

Reads are unrestricted: `wrender.sh` folds both logs into the manifest;
`conduct-board.sh` folds both when deriving the barrier. Relay delivery stays **push**
(conductor-driven), as today.

```bash
# CORRECT: conductor records delivery in ITS log on the target
scripts/rlog.sh "$TARGET_WD" relay_received from=alpha slug=alpha-needs ...

# WRONG: conductor appending delivery into the child's work.jsonl (the legacy /epic behavior —
# a two-writer file; frozen for legacy epics, never used for parent-bound items)
scripts/wlog.sh "$TARGET_WD" relay_received from=alpha slug=alpha-needs ...
```

### 3. `escalated` is distinct from `blocked`

🔴 Blocked = waiting on something **expected to resolve** (open relay, dependency) — the conductor
keeps the item in play. 🚨 Escalated = bounded attempts exhausted or human decision required —
**out of play**: excluded from the barrier min, no run-command emitted, run cannot complete while
any child is escalated. Only a human `status_changed` resumes (or cancels) it. The `/conduct sync`
round-cap (3+ rounds on one edge in one phase) lands as `escalated` on the **parent's** log.

```bash
# CORRECT: worker exhausts attempts → terminal-until-human
scripts/wlog.sh "$WD" escalated note="verify gate failed 3x, same signature — need decision"

# WRONG: spinning on retries, or parking as blocked when no resolution is in flight
scripts/wlog.sh "$WD" status_changed to=blocked note="tests keep failing"   # conductor keeps re-prompting forever
```

### 4. The board is derived, rendered into the PARENT's manifest

Membership = children's `parent=` declarations (no roster). `scripts/conduct-board.sh --write
<parent-id>` injects the `**Barrier Phase**` line + children table between
`<!-- BEGIN BOARD -->`/`<!-- END BOARD -->` anchors in the parent manifest; `wrender.sh` preserves
that region verbatim across re-renders. Children READ the barrier; they never recompute it. A1
(one resolution closes both legs), A2 (auto-resolve `confirms`/`fyi` at delivery), and the
two-layer validation split are preserved verbatim from the epic model.

### 5. Sub-epic roadmaps → sequenced sibling parents

3-level nesting is retired (for now). A multi-milestone effort is a chain of **sibling standalone
parents** sequenced by a roadmap artifact (wishlist-milestone pattern), advanced by
`/conduct <parent> next`.

### 6. Freeze-don't-migrate

Everything under `docs/epics/` and every `epic=`-linked work item stays on the frozen `/epic` +
`epic-board.sh` model (which is why `epic-board.sh` and the legacy delivery behavior are untouched).
Never convert a legacy epic.

## Rationale

The parent/child model makes the file-mode dogfood **isomorphic to the product substrate**
(epic-2607031635 settled design): one generic node type with a parent link, a derived board, a
conductor command, writer-partitioned state (delivery platform-owned, node state worker-owned —
exactly `conversation_message` vs `ps_task`), and `done|escalated|timed-out`-style terminals. The
epic entity held no state of its own (its manifest was already a projection); removing it deletes a
concept, not a capability — and the parent, being a real work item, gains an event log, giving the
conductor the decision journal the epic's hand-authored Change Log never was. The writer partition
also fixes a latent two-writer violation (`/epic sync` appended into child `work.jsonl`).

## Exceptions

1. **Legacy epics** — operate under the old rules via `/epic` until they complete (frozen, not
   migrated).
2. **The birth exception** — the conductor may seed a child's `created` event at scaffold time
   (the child's log does not exist yet; ownership hands off immediately after).
3. **A2 auto-resolution** — the conductor appends `relay_resolved` on the *target's* `work.jsonl`
   for `confirms`/`fyi` relays at delivery; this is a conductor-performed action recorded as the
   settled fact it is, retained verbatim from the epic model.

## Enforcement

- `/work --parent-work` / `/conduct scaffold` validate the ancestry chain (standalone root, no
  cycles) before creating; `conduct-board.sh` surfaces a mid-level node's own parent in its header
  and prints the settling hint on children-complete.
- `scripts/rlog.sh` accepts only `relay_received|relay_synced`; `wlog.sh` remains the only
  worker-side writer.
- Code review + the command docs (`.claude/commands/conduct.md`, `work.md`) carry the rules.

## See Also

- [epic-conductor-barrier-workflow](./epic-conductor-barrier-workflow.md) — barrier semantics (preserved)
- [append-only-work-event-log](./append-only-work-event-log.md) — the event-log law
- [agentic-workflow-substrate research/0008](../../work/work-2607031635-agentic-workflow-substrate/research/0008-conduct-decomposition-proposal-of-record.md) — the settled proposal of record
