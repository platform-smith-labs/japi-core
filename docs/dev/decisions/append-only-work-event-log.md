# Decision: Work/Epic State Is an Append-Only Event Log, Not Hand-Edited Files

**Date**: 2026-07-01
**Status**: Accepted
**Context**: Running the epic→work workflow spent up to ~50% of each session on *internal bookkeeping* — the same status fact hand-edited across a work manifest, an epic Tracked-Repos table, and a wishlist README, plus LLM-derived sequential IDs and read-modify-write emoji edits. A multi-agent analysis (4 forensic command reads + 4 web-research angles + 4 codebase deep-dives) traced the cost to one root cause: **one fact written to many places, by the LLM, via brittle exact-string edits, on every transition.** This decision also future-proofs the workflow for the Platform Smith agent-to-agent (A2A) port, where it becomes one of several workflows agents coordinate over git-as-shared-state (see [the companion design doc](../../design/work-event-log-and-a2a-port.md)).

## Decision

A work item's state lives in a single **append-only event log** (`work.jsonl`, one per work item). Commands **append events**; they never hand-edit state. The human-readable `manifest.md` is a **generated view**, folded deterministically from the log by `scripts/wrender.sh`. Identity is a **timestamped slug** (`work-<YYMMDDHHMM>-<slug>`), never an LLM-derived sequential number. Content artifacts (research, requirements, plans, handoff docs) stay **markdown prose**; only *state* is structured.

## Rules

### 1. Identity is a timestamped slug — no sequential numbers

Work and epic IDs are `work-<YYMMDDHHMM>-<slug>` / `epic-<YYMMDDHHMM>-<slug>`, where the timestamp comes from `date` and the slug is a kebab summary. There is **no scan-max-increment** counter.

```bash
# CORRECT: deterministic, no global read, race-free across pods
SLUG="dark-mode"
WORK_ID="work-$(date +%y%m%d%H%M)-$SLUG"   # work-2607010322-dark-mode

# WRONG: LLM scans existing dirs, takes max, increments — slow, error-prone,
# and two agents in two pods pick the SAME number (write-write race).
# next = max(NNNN over docs/work/work-*) + 1   ← banned
```

Minute-level collisions on the timestamp are acceptable (the slug disambiguates; identical slug+minute in one repo is astronomically rare and self-evident). Chronological `ls` ordering is preserved by the `YYMMDDHHMM` prefix.

### 2. Every state-relevant fact is an event — "append an event, or it didn't happen"

Status, phase completion, artifact registration, and relay lifecycle are recorded **only** by appending to `work.jsonl` via `scripts/wlog.sh`. Nothing lives only in the agent's head, only in the manifest, or only in a folder location.

```bash
# CORRECT: append one event; manifest is regenerated from it
scripts/wlog.sh "$WD" status_changed to=implementation
scripts/wlog.sh "$WD" phase_done phase=requirements note="acceptance criteria signed off"
scripts/wrender.sh "$WD"

# WRONG: hand-edit the manifest's Status line (drifts from the log; a future
# control-plane reader that folds the log goes blind to this change)
#   - edit manifest.md:  **Status**: 🔄 In Implementation
```

This is the load-bearing invariant. If a command records state by editing the manifest or moving a file *instead of* appending an event, the log is no longer complete and the Platform Smith port (which reads the log, not the manifest) silently loses that fact.

### 3. The manifest is generated — never hand-edited

`manifest.md` is a pure projection of `work.jsonl`. It carries a generated-by banner. Commands regenerate it with `scripts/wrender.sh`; they never write it directly.

```bash
# CORRECT
scripts/wrender.sh "$WD"     # folds work.jsonl → manifest.md, deterministically

# WRONG: any tool/agent opening manifest.md in an editor to change state
```

The renderer only *places* prose the LLM already authored (event `note` fields). It never invents narrative — judgment is captured at append time, on the event, not re-derived on every render.

### 4. Structure the state, not the content

Only the event log is JSON. Research, requirements, plans, issues, and handoff docs remain **markdown prose** — that is what agents and humans reason over, and what git-resume depends on.

```text
# CORRECT
docs/work/<id>/work.jsonl                  ← structured state (JSONL)
docs/work/<id>/manifest.md                 ← generated view
docs/work/<id>/research/0001-*.md          ← prose content (unchanged)
docs/work/<id>/plans/master.md             ← prose content (unchanged)

# WRONG: JSON-ifying research/requirements/plans — destroys the readable
# artifacts the agent and the human (and git-resume) rely on.
```

### 5. Relays: folders encode direction, the log encodes lifecycle, files are immutable

Cross-repo relay *messages* are immutable markdown files under direction-named folders. Their *lifecycle* (sent → synced → resolved) lives in `work.jsonl`. Resolution is **not** a file move and **not** a delete.

```text
# CORRECT layout
docs/work/<id>/relays/outbound/to-<peer>--<slug>.md     ← immutable message (+ frontmatter)
docs/work/<id>/relays/inbound/from-<peer>--<slug>.md    ← immutable message

# lifecycle is events, not file moves:
scripts/wlog.sh "$WD" relay_sent     to=ps-api slug=theme-endpoint relay_kind=blocks phase=requirements ask="..." path=relays/outbound/to-ps-api--theme-endpoint.md
scripts/wlog.sh "$WD" relay_synced   slug=theme-endpoint          # delivered to peer; file STAYS
scripts/wlog.sh "$WD" relay_resolved direction=outbound slug=theme-endpoint note="acked"

# WRONG: move the file to an archive/ folder to mark it resolved.
# That puts open/resolved state in TWO places (folder location + log) — the
# exact dual-source-of-truth this decision removes. Pick ONE: the log.
```

Open relays are derived: `relay_sent`/`relay_received` minus `relay_resolved` (by `direction+slug`). `relay_synced` is a delivery annotation; it does **not** close the relay.

### 6. Deterministic mechanics are scripts, judgment stays with the LLM

ID minting, appending events, and folding the manifest are deterministic → `date`, `wlog.sh`, `wrender.sh`. Authoring research/plans/relay bodies, deciding *whether* a phase is done, and writing the one-line `note` on an event are judgment → the LLM.

```text
# Rule of thumb:
#   If removing the LLM changes the output → it was judgment (keep the LLM).
#   If it does NOT change the output       → it was bookkeeping (use a script).
```

The scripts are repo-committed and dependency-light (bash + jq) precisely so they travel into a sandbox pod with the repo when this workflow is ported to agent-config. This is NOT the host-side "psflow CLI" rejected earlier — that owned cross-file sync as the mutation path and would not follow the agent; these two scripts only append and project.

### 7. No DB changes; orchestrator stays a dumb router

This workflow is entirely git/file-based. It adds **zero** database tables/columns and **zero** orchestrator logic. The epic→work model must remain one of *many* workflows the platform can host, so no platform entity (no `ActivityEvent`, no `work_item` table) is coupled to it. Upward/control-plane sync, when the port happens, lives in an *agent or skill* reading the log — never in orchestrator Go. See the companion design doc §"Port options".

## Rationale

- **Eliminates the 50% tax by construction.** The fan-out (one status fact → manifest line + Epic-Phase-Done field + epic Tracked-Repos cell + wishlist row + change log, all hand-synced with "no automatic sync") collapses to one append + one deterministic regenerate. Status can no longer drift across copies because every copy is projected from one source.
- **Race-safe for multi-agent / multi-pod.** Slug IDs need no global counter (kills the write-write race two pods hit picking the same `NNNN`). Append-only logs merge in git without the conflicts that in-place emoji edits cause.
- **Matches the platform's own proven pattern.** Research `0068` found the most reliable state axis in Platform Smith is the launch machine: an **append-only `launch_event` log → projected HEAD**, and explicitly recommends promoting it to a reusable primitive. `conversation_message` (BIGSERIAL `seq`) is the same shape. This decision adopts that pattern for work state.
- **Forward-compatible with the A2A port.** wishlist `0003` locked "shared state = git; agents are stateless, resume from the branch." A complete, self-describing JSONL log on the branch is exactly what a future control-plane session (or a phase-boundary A2A message) needs to reconstruct work state — with zero platform commitment now.
- **Speed + correctness.** An LLM hand-writing JSON or matching an emoji string to edit is slow and occasionally malformed; `wlog.sh`/`wrender.sh` are instant and byte-stable, giving clean git diffs.

## Exceptions

1. **Epic rollup** is folded from child work logs (each child's last `phase_done` → the child manifest's `Epic Phase Done`, read by `scripts/epic-board.sh`), not from a `work.jsonl` of its own. An epic may carry a thin authored brief, but its Tracked-Repos/barrier state is derived, never hand-synced.
2. **Pre-existing legacy work items** (`work-NNNN-…` with hand-edited manifests) are not migrated. They remain readable; only new items use the event-log model. No big-bang rewrite.
3. **One-line `note` fields are authored prose** living inside an event — the single sanctioned place LLM narrative enters the state log. The renderer places them verbatim; it does not generate them.

## Enforcement

- Reviewed in code review and reinforced in `.claude/commands/*` and `.claude/CLAUDE.md`: state changes go through `wlog.sh` + `wrender.sh`; `manifest.md` is never hand-edited.
- The generated-by banner at the top of every `manifest.md` makes a hand-edit visible in review.
- `scripts/wrender.sh` is deterministic; a spurious manifest diff with no corresponding `work.jsonl` append is a red flag.

## See Also

- [Companion design doc — work event log & the A2A port](../../design/work-event-log-and-a2a-port.md)
- [Decision: epic conductor barrier workflow](./epic-conductor-barrier-workflow.md)
- [Decision: caller-injected peer identity](./caller-injected-peer-identity.md)
- wishlist `0003` — cross-pod agent coordination protocol (`docs/wishlist/0003_cross-pod-agent-coordination-protocol/`)
- Research `0068` (entity status & state-tracking event model), `0069` (task-tracking substrate)
