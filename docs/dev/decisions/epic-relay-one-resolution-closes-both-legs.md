# Decision: An epic relay is one thing — one resolution closes both legs; informational replies auto-resolve

**Date**: 2026-07-03
**Status**: Accepted
**Context**: During epic-2607011710-spawn-branch-checkout the conductor board accumulated 13 "open"
relay legs and the barrier never advanced, even though every substantive cross-repo question was
answered. Root cause: a relay's **outbound** leg (the source's `relay_sent`) had no way to close — the
source only ever emits `relay_synced` (delivered), never a resolution; resolution happened on the
**target** as `relay_resolved direction=inbound`. `scripts/epic-board.sh` counted each repo's legs
independently, so `relay_sent` counted as open forever. Compounding it, repos "acknowledged" a
`confirms`/`fyi` reply by sending **another** `confirms`/`fyi`, spawning new legs each round instead of
closing the loop.

## Decision

A cross-repo relay is **one object shared by two repos**, keyed by its `slug`. A **single**
`relay_resolved` for that slug (emitted by the acting/receiving party) settles the **whole** relay —
both the target's inbound leg and the source's outbound leg. Additionally, informational replies
(`kind=confirms`/`fyi`) are **auto-resolved at delivery** by `/epic sync`, because they require no
target action.

## Rules

### 1. One resolution closes both legs (A1) — derived in the board, no new event

An outbound leg (`relay_sent slug=X`) is **not** open once **any** `relay_resolved` with `slug=X`
exists **anywhere in the epic**. The source never emits its own `relay_resolved direction=outbound`.

```bash
# CORRECT: target resolves once; the board closes BOTH legs by slug.
scripts/wlog.sh "$TARGET_WD" relay_resolved direction=inbound slug=branch-passthrough
# epic-board.sh: epic_resolved_slugs collects {branch-passthrough,…}; relay_counts drops that slug
# from EVERY repo's open inbound AND outbound counts → the orchestrator's outbound auto-closes.

# WRONG: expecting the source to also close its own outbound leg (it never does → dangling forever).
scripts/wlog.sh "$SOURCE_WD" relay_resolved direction=outbound slug=branch-passthrough   # unnecessary
```

### 2. Informational replies auto-resolve at delivery (A2)

When `/epic sync` delivers a relay whose `kind` is `confirms` or `fyi`, it resolves it **immediately**
after recording receipt — no manual round.

```bash
# CORRECT: a confirms/fyi is an answer — deliver, then close it now.
scripts/wlog.sh "$TARGET_WD" relay_received from=runtime slug=oq7-push-creds relay_kind=confirms phase=requirements path=...
scripts/wlog.sh "$TARGET_WD" relay_resolved direction=inbound slug=oq7-push-creds   # A2: at delivery
scripts/wrender.sh "$TARGET_WD"

# WRONG: acknowledging a confirms/fyi by sending another confirms/fyi (opens a new leg every round).
scripts/wlog.sh "$SOURCE_WD" relay_sent to=runtime slug=ack-oq7 relay_kind=confirms ...   # never do this
```

### 3. Only `blocks` relays hold the barrier awaiting work

A `blocks` relay stays open until the target does the work and resolves it (Rule 1). `confirms`/`fyi`
never linger (Rule 2). If a reply genuinely requires new work from the peer, it is a **`blocks`**, not
a `confirms`.

## Rationale

- **Termination.** Without A1+A2 the relay graph is monotonically increasing — every acknowledgment
  adds legs, so the barrier can never reach zero. A relay is a request/response, not an append-only
  chat; it must be closeable in bounded steps.
- **Single source of truth.** Modeling a relay as one slug-keyed object (resolved once) removes the
  impossible bookkeeping of asking the source to observe the target's resolution.
- **No new events / backward compatible.** A1 is purely a derivation change in `epic-board.sh`
  (`epic_resolved_slugs` + slug-based exclusion in `relay_counts`/`relay_list`/`open_inbound_slugs`);
  existing logs replay correctly. A2 reuses the existing `relay_resolved` event.

## Exceptions

1. **Legacy `upstream/` file relays** — repos predating the event log fall back to file counts in
   `relay_counts`; A1/A2 don't apply there (no slugs). Those epics are not migrated.

## Enforcement

- `scripts/epic-board.sh` implements A1 (`epic_resolved_slugs`, and slug-based open-leg exclusion) —
  the derived board is the single computed view; do not hand-edit the Tracked Repos cells.
- `.claude/commands/epic.md` `/epic sync` Step 2/2b documents A1 (Step 2b) and A2 (Step 2, "auto-resolve
  informational replies at delivery"), plus the round-cap guard for `blocks` that won't settle.

## See Also

- [Append-only work event log](./append-only-work-event-log.md)
- Epic: `docs/epics/epic-2607011710-spawn-branch-checkout/` (where this surfaced)
