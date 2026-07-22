# Decision: Conductor pushes the barrier into each child's own territory

**Date**: 2026-07-21
**Status**: Accepted
**Context**: Autonomous `/work <id> auto` worker loops in the parent/child conductor model
(`.claude/commands/conduct.md` + `.claude/commands/work.md`, driven by `scripts/conduct-board.sh`)
could deadlock at a phase boundary. A child decided its next step by reading the barrier from the
**parent** work item's manifest — which lives in a **different repo**. Under repo isolation (each
repo is its own pod with no filesystem access to siblings; see
[repo-isolation-kb-first-cross-repo.md](./repo-isolation-kb-first-cross-repo.md)) a strictly
isolated worker cannot read that manifest, so it never sees the barrier advance. This **supersedes**
the earlier `{child-WD}/barrier.md` file-push sketch (a hand-written markdown file the conductor
command wrote every sync) mentioned in
[parent-child-work-items-and-conduct.md](./parent-child-work-items-and-conduct.md): the barrier push
is now a deterministic `barrier_advanced` event emitted by `conduct-board.sh --write`, not a
hand-authored file.

## Decision

Shared barrier state is **pushed by the conductor into each child's own conductor-owned
`relays.jsonl`** as a `barrier_advanced` event (fields `phase` + `state`), and `/work auto` reads the
barrier from **its own** `relays.jsonl` — never by reaching across the repo boundary for the parent
manifest.

## Rules

### 1. The conductor PUSHES the barrier; the child never PULLS it

`conduct-board.sh --write` computes the canonical barrier token from the same fold it already does,
then appends a `barrier_advanced` event into each **non-prime** child's `relays.jsonl`. The child
reads it locally.

```bash
# CORRECT: /work auto reads the barrier from ITS OWN relays.jsonl (same repo/pod)
jq -rs 'map(select(.type=="barrier_advanced"))|last // {} | "\(.phase // "") \(.state // "")"' \
  "$WD/relays.jsonl"

# WRONG: reaching across the repo boundary for the parent manifest (isolation violation → deadlock)
grep '^\*\*Barrier Phase\*\*:' ../../solution/docs/work/<parent-id>/manifest.md
```

### 2. `barrier_advanced` is conductor-owned and idempotent

`relays.jsonl` keeps its single-writer rule: only the conductor writes it (`scripts/rlog.sh`).
`barrier_advanced` joins `relay_received`/`relay_synced` in the conductor vocabulary. The push
appends **only when `(phase,state)` changed**, so re-running `--write` with no barrier change writes
nothing.

```bash
# CORRECT: append only on change (idempotent)
if [[ "$last_phase" != "$BPHASE" || "$last_state" != "$BSTATE" ]]; then
  scripts/rlog.sh "$child_wd" barrier_advanced phase="$BPHASE" state="$BSTATE"
fi

# WRONG: a child appends its own barrier_advanced (breaks single-writer; the child must never write relays.jsonl)
scripts/rlog.sh "$WD" barrier_advanced phase=planning state=open   # ← never, from a worker
```

### 3. The renderer ignores `barrier_advanced` — no manifest noise

`wrender.sh` folds `relays.jsonl` only for the relay sections (it selects `relay_*` types) and the
Change Log reads only `work.jsonl`. `barrier_advanced` matches neither, so it never leaks into any
manifest.

```bash
# CORRECT: barrier_advanced is invisible to the renderer
scripts/wrender.sh "$child_wd"
grep barrier_advanced "$child_wd/manifest.md"   # → no output (absent)
```

### 4. State vocabulary the worker acts on

`barrier_advanced.state` ∈ `{open, held, complete}`:

- `open` → target = the barrier `phase`; if the child already settled ≥ it, it is *at the barrier* → STOP.
- `held` → do **not** start the barrier phase (open relays / escalations upstream) → STOP.
- `complete` → the run is finished → STOP.
- No `barrier_advanced` yet → treat the barrier as the kickoff phase (`requirements`, `open`) and proceed.

## Rationale

- **Isolation-clean.** The child only ever reads files in its own repo. The cross-repo read that
  caused the deadlock is gone.
- **Mirrors the product control-plane push model.** In the PlatformSmith substrate, coordination
  state is platform-owned and **pushed** to workers, not pulled by them. Pushing the barrier into the
  child's tree is the file-mode mirror of that same ownership model.
- **Preserves single-writer.** Each `jsonl` still has exactly one writer: children write
  `work.jsonl`; the conductor writes `relays.jsonl` (delivery **and** barrier). No one crosses the
  partition.
- **Kills the finish-early deadlock.** The deadlock required both (a) the conductor advanced the
  barrier and (b) the child had no local work left to trigger a settle, so it parked "at the barrier"
  forever while the conductor waited for a `phase_done` it is forbidden to write. With the barrier
  visible in the child's own tree, the next `/work auto` iteration reads the advance and acts.
- **Worst case is latency, not deadlock.** If the conductor has not synced yet, the child simply sees
  a stale/absent barrier and idles for one sync cycle — it never blocks permanently.

## Enforcement

- `scripts/rlog.sh` allowlist (`case` statement) — the only place `barrier_advanced` may be written,
  and only via the conductor.
- `scripts/conduct-board.sh --write` — the sole producer of `barrier_advanced` (skips the ★ prime/self
  row; idempotent on change).
- `.claude/commands/work.md` (Conductor-aware work + `auto` algorithm) — the worker reads the barrier
  from its own `relays.jsonl`; it must **not** read the parent manifest.
- `.claude/commands/conduct.md` — documents the `--write` push.
- `scripts/sync-workflow-tooling.sh` — propagates all of the above (and this decision doc) into every
  `repos/*` submodule, since each repo runs its own copy.

## See Also

- [Parent/Child Work Items and Conduct](./parent-child-work-items-and-conduct.md)
- [Repo Isolation — KB-first cross-repo](./repo-isolation-kb-first-cross-repo.md)
- [Append-only Work Event Log](./append-only-work-event-log.md)
- [Epic Conductor Barrier Workflow](./epic-conductor-barrier-workflow.md) (legacy semantics preserved)
