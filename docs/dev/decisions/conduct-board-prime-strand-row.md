# Decision: conduct-board shows the parent's own "prime" strand as a gating row

**Date**: 2026-07-09
**Status**: Accepted
**Context**: In a ticket where the parent work item is *also* a worker (the "prime/conductor" repo
does execution work itself — e.g. `work-2607090251-a2a-delivery-tracking`, where runtime is both the
conductor and the strand that does the dedup / `a2a_delivered` code), `scripts/conduct-board.sh` only
ever rendered rows for **children** (items declaring `parent=<id>`). The parent's own strand had no
row, so the board never emitted a run-command for it and its phase never gated the barrier — the
conductor had to hand-surface "the runtime step the board can't show" every tick, and relays sent to
the parent-repo landed in the parent's own inbox with no board representation.

## Decision

When a parent work item **has children**, `conduct-board.sh` includes the **parent itself as a
first-class, barrier-gating roster member** — rendered first, marked `★` — so its own `phase_done`
counts in the barrier min and it gets a run-in-each-repo command exactly like a child. A **childless**
standalone item is unchanged (it is not being conducted yet).

## Rules

### 1. The prime row is a real member, not a display flourish

It folds through the identical per-member machinery (`phase_done`, `escalated`, relay counts) because
`work_dir(PARENT_REPO, PARENT_ID) == PARENT_DIR`. It therefore **gates the barrier**: the cohort
cannot advance to phase P+1 until the parent's own strand has also settled phase P.

```bash
# CORRECT: prime row prepended once children exist; gates the barrier like a child
local ROWS; ROWS="$(discover_children "$PARENT_ID" "$PARENT_REPO")"
if [[ -z "$ROWS" ]]; then ... ; return; fi        # childless item: unchanged, no prime row
ROWS="$PARENT_REPO|$PARENT_ID"$'\n'"$ROWS"          # prime/self row first

# WRONG: leaving the parent out of the roster (its execution work is invisible + never gates)
local ROWS; ROWS="$(discover_children "$PARENT_ID" "$PARENT_REPO")"   # children only
```

### 2. The prime row is marked, never silently blended

The `★` marker + a legend line distinguish the parent's own strand from its children, so the board
stays honest about the two hats (conductor + worker) living in one item.

```bash
# CORRECT
if [[ "$repo" == "$PARENT_REPO" && "$wid" == "$PARENT_ID" ]]; then R_prime[$i]=1; else R_prime[$i]=0; fi
[[ "${R_prime[$j]}" == "1" ]] && R_state[$j]="★ ${R_state[$j]}"

# WRONG: an unmarked parent row is indistinguishable from a child
```

### 3. Childless items are untouched

The prime row is prepended **after** the empty-roster early-return, so a standalone item with no
children keeps its original "no children yet — scaffold one" guidance and gains no board.

## Rationale

- **Honesty**: the board is the single source of "DO THIS NEXT". A parent that does execution work
  but shows no row under-reports the run — the conductor had to remember and narrate the missing
  strand, which is exactly the kind of hand-maintained state the derived-board model exists to kill.
- **Correctness (barrier gating)**: if the parent's strand does not gate, the cohort could advance a
  phase while the parent's own work lags — the barrier would be a lie. Gating makes the parent a peer.
- **Zero special-casing**: `work_dir(PARENT_REPO,PARENT_ID)==PARENT_DIR` means the existing folds
  work unchanged; the change is a roster prepend + a marker, not a new code path. Low risk.
- **Alternative rejected**: splitting the parent's execution work into a *separate child* work item
  (so all strands are children and the parent is a pure conductor) was considered and declined for
  this ticket — it would fragment an item mid-flight and duplicate the conductor seat. The prime-row
  approach keeps one item wearing both hats while still surfacing/gating the worker hat.

## Exceptions

1. **Childless standalone items** — no prime row; they are not being conducted, so the original
   scaffolding hint stands.

## Enforcement

- Code: `scripts/conduct-board.sh` (roster prepend + `R_prime` marker + legend).
- ⚠️ **Template sync**: `conduct-board.sh` is replicated byte-identically across the monorepo root
  (`solution`, the canonical copy `/conduct` runs) and every repo's `scripts/`, regenerated from the
  conductor **skillset**. This change was applied to both the runtime copy and the root copy, but the
  **skillset source must be updated too** or the next sync will clobber it. Tracked in the parent
  work log (`work-2607090251-a2a-delivery-tracking`).

## See Also

- [Parent/Child Work Items and Conduct](./parent-child-work-items-and-conduct.md)
- [Work Item](../../work/work-2607090251-a2a-delivery-tracking/) — the ticket that surfaced this
