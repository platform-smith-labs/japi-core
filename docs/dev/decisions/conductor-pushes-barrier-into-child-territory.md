# Decision: The conductor PUSHES the barrier into each child's own territory

**Date**: 2026-07-21
**Status**: Accepted
**Context**: A deadlock in the `/conduct` + `/work … auto` barrier-synchronized workflow. Autonomous
worker loops could park permanently at a phase boundary because the barrier they must read lives in a
**different repo** they are forbidden to read under repo isolation.

## Decision

The barrier is shared coordination state. Instead of making an isolated child worker **pull** the
barrier across the repo boundary (reading the parent manifest), the conductor **pushes** it into a
file it already owns inside each child — a `barrier_advanced` event in the child's `relays.jsonl`.
The worker reads the barrier from its **own** repo.

## Rules

### 1. The barrier is pushed, never pulled

`/work … auto` reads the barrier from the latest `barrier_advanced` event in **its own**
`$WD/relays.jsonl` — never from the parent manifest.

```bash
# CORRECT: read the barrier locally (child's own conductor-owned log)
jq -rs '[ .[] | select(.type=="barrier_advanced") ] | last | {phase, state}' "$WD/relays.jsonl"

# WRONG: reach across the repo boundary into another pod's working tree
jq ... ../../solution/docs/work/<parent-id>/manifest.md   # violates repo isolation; unreadable in-product
```

### 2. Only the conductor writes `barrier_advanced` — in `relays.jsonl`

It rides the existing conductor-owned log (single-writer rule preserved). `rlog.sh` is the only writer;
workers never touch `relays.jsonl`.

```bash
# CORRECT: conductor pushes on every conduct-board.sh --write, idempotently
scripts/rlog.sh "$CHILD_WD" barrier_advanced phase=planning state=open

# WRONG: a worker writing its own barrier (wrong writer, wrong log)
scripts/wlog.sh "$WD" barrier_advanced ...        # barrier_advanced is not a work.jsonl event
```

### 3. The push is idempotent — only on change

`conduct-board.sh --write` appends a `barrier_advanced` to a child **only** when `(phase, state)`
differs from that child's last one. Re-running `--write` with no barrier movement writes nothing.

```bash
# CORRECT: compare last pushed token to the freshly computed one; append only if changed
[[ "$last_phase $last_state" != "$BPHASE $BSTATE" ]] && rlog.sh "$CHILD_WD" barrier_advanced phase="$BPHASE" state="$BSTATE"

# WRONG: append on every sync → unbounded relays.jsonl growth, noisy "advances" that never moved
rlog.sh "$CHILD_WD" barrier_advanced phase="$BPHASE" state="$BSTATE"   # every run
```

### 4. `state` is explicit: `open` | `held` | `complete`

`open` = the child may run this phase now. `held` = a relay/escalation blocks it; do not start,
only process inbound relays. `complete` = the run is validated. The prime/self row (the parent's own
strand) is **skipped** — it lives in the conductor's repo and reads the board directly.

### 5. The renderer ignores `barrier_advanced`

`wrender.sh` folds `relays.jsonl` only for relay sections (it `select`s `relay_*` types) and the
Change Log reads only `work.jsonl`. `barrier_advanced` is neither, so it never leaks into any
manifest section — verified.

## Rationale

- **Isolation-clean.** The worker's entire filesystem universe is its own repo. Pulling the parent
  manifest was a latent isolation violation that happened to work only in a shared-checkout dev
  layout; in-product (one pod per repo) it is simply unreadable.
- **Mirrors the product control-plane push model.** Shared coordination state is pushed to each
  participant's own territory, exactly as the platform pushes delivery state — the file-mode mirror
  of the substrate's ownership model.
- **Preserves the single-writer rule.** `relays.jsonl` already has exactly one writer (the
  conductor). Adding `barrier_advanced` to it changes nothing about ownership.
- **Kills the finish-early deadlock.** The failure was: a child settled phase P and built its next
  phase's code early, so it had no local trigger to re-settle, and could not see the barrier advance;
  the conductor could not write the child's `phase_done` (single-writer). With the push, the next
  `--write` deposits the advanced barrier into the child's own log and the next `auto` iteration acts.
- **Worst case is latency, not deadlock.** A child sees a barrier move at most one sync cycle late
  (the next `--write`), never never.

## Enforcement

- `rlog.sh` restricts its event vocabulary to `relay_received|relay_synced|barrier_advanced`; a
  worker's `wlog.sh` cannot emit it.
- `conduct-board.sh --write` is the sole producer, and only on change (rule 3).
- `work.md` (the `auto` algorithm) reads the barrier only from the child's own `relays.jsonl`.
- Propagated to every `repos/*` via `scripts/sync-workflow-tooling.sh` — the workers run the child
  copies, so the solution-root edit is inert until synced.

## See Also

- [Parent/child work items and conduct](./parent-child-work-items-and-conduct.md)
- [Epic conductor barrier workflow](./epic-conductor-barrier-workflow.md)
- [Repo isolation — KB-first cross-repo](./repo-isolation-kb-first-cross-repo.md)
- [Append-only work event log](./append-only-work-event-log.md)
