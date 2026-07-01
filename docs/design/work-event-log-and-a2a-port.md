# Design: The Work Event Log & Its Platform Smith (A2A) Port

**Document type**: Design / architecture reference
**Created**: 2026-07-01
**Status**: Active — V1 (terminal/Claude, git-as-state) shipping; platform port deferred
**Owns the decision**: [append-only-work-event-log](../dev/decisions/append-only-work-event-log.md)
**Read this when**: porting the epic→work workflow into Platform Smith agent-config / A2A, or
extending the event vocabulary, the renderer, or the relay model.

> This is the precise, self-contained spec. It is written so a future session — possibly an agent
> inside a sandbox pod — can reconstruct the full model without re-deriving it from chat history.
> Keep it in sync with `scripts/wlog.sh`, `scripts/wrender.sh`, and `.claude/commands/*`.

---

## 1. Why this exists (the problem, in one paragraph)

The epic→work workflow is a set of `.claude` slash-commands that drive cross-repo development.
Historically a single "phase done" fact was hand-edited by the LLM into 3–7 places (work manifest
`Status` line + `Epic Phase Done` field + epic `Tracked Repos` cell + wishlist row + change logs),
and IDs were LLM-derived sequential numbers (`work-0074-…`). Forensic analysis attributed ~50% of
session time to this bookkeeping, and it is **unsafe under concurrency**: sequential IDs race when
two pods create items at once, and in-place emoji edits conflict in git. The fix is to make state an
**append-only event log** with a **deterministic generated manifest**, and identity a **timestamped
slug**. This is also the representation the Platform Smith A2A vision already assumes — "shared state
= git; agents stateless; resume from branch" (wishlist `0003`).

---

## 2. On-disk layout (per work item)

```
docs/work/work-<YYMMDDHHMM>-<slug>/
├── work.jsonl              ← THE STATE. Append-only event log. Single source of truth.
├── manifest.md             ← GENERATED VIEW (scripts/wrender.sh). Never hand-edited.
├── epic/
│   └── context.md          ← prose: this item's role in its epic (authored once at onboarding)
├── research/NNNN-*.md      ← prose content (unchanged by this design)
├── requirements/NNNN-*.md  ← prose content
├── issues/NNNN-*.md        ← prose content
├── plans/master.md, phase-*.md
├── implementation/status.md (optional prose; status-of-record is work.jsonl)
└── relays/
    ├── outbound/to-<peer>--<slug>.md     ← immutable relay messages (+ YAML frontmatter)
    └── inbound/from-<peer>--<slug>.md
```

Only `work.jsonl` is authoritative for *state*. Everything `.md` is either generated (`manifest.md`)
or prose *content* (everything else). The split is the whole design: **structure the state, keep the
content prose.**

---

## 3. Identity

| Kind | Format | Example | Minted by |
|------|--------|---------|-----------|
| Work item | `work-<YYMMDDHHMM>-<slug>` | `work-2607010322-dark-mode` | `date +%y%m%d%H%M` + kebab slug |
| Epic | `epic-<YYMMDDHHMM>-<slug>` | `epic-2606301200-uiux-revamp` | same |

- **No sequential counter, no scan.** The timestamp gives chronological `ls` order; the slug gives
  meaning and disambiguation. Minute-level collisions are tolerated (slug differs; identical
  slug+minute in one repo is vanishingly rare and self-evident).
- **Slug = address.** It maps cleanly onto Platform Smith's `to_project` string addressing, and is
  stable across pods (a pod that resumes from the branch reads the same id).
- Resolution of a short reference (`work-2607010322-dark-mode` or just the slug) to a directory is a
  glob, not a computation: `docs/work/*<slug>*/` — a deterministic lookup, never LLM arithmetic.

---

## 4. The event log (`work.jsonl`)

One JSON object per line, appended by `scripts/wlog.sh`. Every event carries:

| Field | Type | Meaning |
|-------|------|---------|
| `seq` | int | monotonic, 1-based; assigned = (line count + 1). The replay cursor. |
| `ts` | string | UTC RFC3339 (`2026-07-01T03:22:07Z`), assigned by the script. |
| `type` | string | event type (below). |
| `actor` | string | git email (or override). Who appended it. |
| `note` | string? | optional one-line LLM-authored narrative (the only prose in the log). |

### 4.1 Event types (the vocabulary)

| `type` | Required fields | Optional | Meaning / when appended |
|--------|-----------------|----------|--------------------------|
| `created` | `title`, `slug`, `kind` (`work`\|`epic`), `repo`, `owner` | `epic`, `wishlist`, `priority`, `effort` | item created (always `seq:1`) |
| `status_changed` | `to` (status key) | `from`, `note` | lifecycle transition (incl. blocked/on_hold/cancelled/completed) |
| `phase_done` | `phase` (`requirements`\|`planning`\|`implementation`\|`validation`) | `note` | the epic-barrier signal — this item settled a phase |
| `artifact_added` | `kind` (`research`\|`requirements`\|`issue`\|`plan`\|`implementation`\|`other`), `path` | `title` | a content artifact was created |
| `relay_sent` | `to` (peer), `slug`, `relay_kind` (`blocks`\|`confirms`\|`fyi`), `phase`, `path` | `ask`, `note` | outbound cross-repo message authored |
| `relay_received` | `from` (peer), `slug`, `relay_kind`, `phase`, `path` | `ask`, `note` | inbound cross-repo message landed |
| `relay_synced` | `slug` | `note` | outbound relay delivered to peer (file stays put) |
| `relay_resolved` | `direction` (`inbound`\|`outbound`), `slug` | `note` | relay closed (the lifecycle terminal) |
| `meta_changed` | (any of `owner`/`epic`/`wishlist`/`priority`/`effort`) | `note` | header metadata updated post-creation |
| `note` | `body` | — | freeform journal line attached to the item |

> Adding a type is a one-line change in `wrender.sh`'s `summ` function (for the change log) and,
> if it drives a header/section, a new fold query. Unknown types still append safely and render in
> the change log via the `else .type` fallback — forward-compatible.

### 4.2 Status keys → badges (the only place the emoji vocabulary lives — in `wrender.sh`)

`proposed`→🎯 · `researching`→📚 · `requirements`→📝 · `planning`→🎨 ·
`implementation`→🔄 In Implementation · `completed`→✅ · `blocked`→🔴 · `on_hold`→⏸️ · `cancelled`→❌

### 4.3 Phase order (for the epic barrier)

`requirements (1) → planning (2) → implementation (3) → validation (4)`. The item's *current*
settled phase = the last `phase_done` event's `phase`.

### 4.4 Worked example

```jsonl
{"seq":1,"ts":"2026-07-01T03:22:07Z","type":"created","actor":"dev0@platformsmith.com","title":"Add dark mode toggle","slug":"dark-mode","kind":"work","repo":"ps-ui","owner":"dev0@platformsmith.com","epic":"epic-2606301200-uiux","priority":"High","effort":"M"}
{"seq":2,"ts":"2026-07-01T03:25:00Z","type":"status_changed","actor":"dev0@platformsmith.com","to":"requirements","note":"3 acceptance criteria agreed"}
{"seq":3,"ts":"2026-07-01T03:30:00Z","type":"phase_done","actor":"dev0@platformsmith.com","phase":"requirements"}
{"seq":4,"ts":"2026-07-01T03:31:00Z","type":"relay_sent","actor":"dev0@platformsmith.com","to":"ps-api","slug":"theme-endpoint","relay_kind":"blocks","phase":"requirements","ask":"Need GET/PUT /v1/users/me/theme","path":"relays/outbound/to-ps-api--theme-endpoint.md"}
```

---

## 5. The two scripts (the deterministic mechanics)

### 5.1 `scripts/wlog.sh <work-dir> <event-type> [key=value …]`
Appends exactly one event. Assigns `seq` + `ts` + `actor`. Builds JSON safely via `jq` (a `note`
with quotes/newlines cannot corrupt the log). Never reads other files, never edits the manifest,
never moves files. **The only sanctioned writer of `work.jsonl`.**

### 5.2 `scripts/wrender.sh <work-dir>`
Folds `work.jsonl` → `manifest.md`. Pure projection: identical log → byte-identical manifest
(verified). Sections: header (status/phase/meta/dates), Artifacts, Open Relays (sent/received minus
resolved, by `direction+slug`; `synced` outbound marked `✓`), Upstream Messages (all
`relay_received`), Change Log (every event with its `note`). Writes atomically (temp + `mv`).

Both are bash + `jq` only — deliberately dependency-light so they travel into a sandbox pod with the
repo. They are invoked by the `.claude` commands; a `/work`-style skill orchestrates "append → render
→ commit" but does not do the fold in-model.

### 5.2b `scripts/windex.sh [work-root]`
Regenerates the work-item **registry** (`<work-root>/index.md`, default `docs/work`). Pure projection
over each item's generated `manifest.md` — it harvests the already-rendered fields (so the status/emoji
vocabulary stays only in `wrender.sh`), sorts by Last Updated, and writes a roll-up table. Deterministic;
`index.md` is generated, never hand-edited. Needs no `jq` (reads manifests, not logs).

### 5.3 Generated manifest shape (reference)

```
# Work Item: <title>
<!-- GENERATED … DO NOT EDIT BY HAND. -->
**ID** · **Status** · **Created** · **Last Updated** · **Owner** · **Epic** · **Wishlist**
**Epic Phase Done** · **Priority** · **Estimated Effort**
## Artifacts        (from artifact_added)
## Open Relays      (folded; table)
## Upstream Messages(from relay_received)
## Change Log       (every event, seq order, with notes)
```

---

## 6. Epic rollup (derived, never hand-synced)

An epic does not own a `work.jsonl`. Its state is **folded from child work logs**:

- Each child's last `phase_done` → the child's generated `manifest.md` `**Epic Phase Done**` line.
- `scripts/epic-board.sh` already reads that line across tracked repos and computes the barrier:
  `Epic Phase = min(child Epic Phase Done) + 1`, open only when **zero open relays** remain anywhere.
- The epic's `Tracked Repos` table and `Epic Phase` are therefore a **rendered board**, not a
  hand-maintained table. `/epic` runs the board; it does not hand-edit cells.

The relay open/closed inputs to the barrier come from each child's folded **Open Relays** (i.e. from
`work.jsonl`), so the whole epic view is a projection over child logs — no upward write-back.

---

## 7. The Platform Smith / A2A port (deferred — design-of-record)

**Constraint locked by the maintainer:** the epic→work workflow is *one of many* workflows the
platform will host. So **no platform entity may be coupled to it** — no `ActivityEvent` rows, no
`work_item` table, no orchestrator logic. Orchestrator stays a **dumb router**; the DB is unchanged.
Cross-pod/upward sync lives in an **agent or skill**, never in Go.

The port rests on a property this design already guarantees: **`work.jsonl` is a complete,
self-describing, ordered (`seq`) log committed to the conversation branch.** That makes both viable
sync mechanisms feasible without any new platform primitive:

### Option A — phase-boundary A2A message (push)
At each `phase_done`, the work agent additionally fires one `a2a_send` to the conversation owner
(coordinator) with a short summary. Phase boundaries are low-frequency (~4 per item) → cheap,
non-chatty. **Feasibility: high.** Requires only that `phase_done` is an explicit checkpoint in the
command (it is). Risk: depends on the work agent reliably emitting the message — mitigate by making
it a non-skippable step at the phase boundary (later: a Stop/phase hook in-pod).

### Option B — control-plane reader (pull)
A separate control-plane Claude session (or the coordinator) reads each branch's `work.jsonl` and
projects state, using `seq` as the cursor (`WHERE seq > last_seen ORDER BY seq` — the exact
deferred-replay pattern from wishlist `0003`). **Feasibility: high, and more robust** — it does not
depend on the work agent remembering anything. Requires only branch read access (git-as-state gives
it). Risk: the reader must run/poll; acceptable for a control plane.

**Recommended:** design for B's substrate (the log is already complete), treat A as an optional
low-latency notification layered on `phase_done`. They are not exclusive: always append the event
(B), optionally also fire the message (A).

### Are the V1 commands in sync with this? — Yes, conditional on Rule #2
Because every command's only state mutation is "append an event," and `phase_done` is a first-class
event, a future port needs **zero command rewrites** for the substrate — only an added emit (A) or an
external reader (B). The one requirement is the completeness invariant ("append an event, or it
didn't happen"); if any command ever records state by editing the manifest or moving a file, B goes
blind to that fact. Enforce it.

### The through-line (same vocabulary, three eras)

| Concern (one vocabulary) | V1 — git-as-state (now) | Port — A2A (later) |
|---|---|---|
| `status_changed`, `phase_done` | line in `work.jsonl` on branch | same line; coordinator learns via A or B |
| `relay_sent` / `relay_received` | line in `work.jsonl` + immutable relay file | the `a2a_send` envelope / `conversation_message` row; file mirrors it |
| manifest / epic rollup | folded view (`wrender.sh` / `epic-board.sh`) | coordinator folds the same logs |
| identity | `work-<YYMMDDHHMM>-<slug>` | same slug = `to_project`-style address |

A command that today *appends a line* tomorrow *also emits an `a2a_send`* — zero semantic change.
The relay file = immutable message (content), the `work.jsonl` event = its delivery/resolution state
— the same shape as `conversation_message` (immutable) + `delivery_state` (lifecycle) in the DB
mailbox. That parallel is intentional: it is what makes the port a thin addition, not a rewrite.

---

## 8. Relationship to existing platform mechanics (for the porter)

- **Runtime A2A** (`a2a_send` MCP tool → orchestrator persist-first → `a2a_deliver`): pods are
  filesystem-isolated; cross-pod truth is the websocket mailbox + the per-branch logs. A "single
  status file" is single-source **per item on its branch**, never a global shared file.
- **Status model** (research `0068`): the launch machine (append-only `launch_event` → projected
  HEAD, single-writer, `classify()` guard) is the platform's most reliable axis and the pattern this
  design mirrors. If work state is ever promoted to a first-class platform feature, reuse that
  primitive — but that is explicitly **out of scope** here (no DB change).
- **wishlist `0003`** (mailbox/coordinator): git-as-state, stateless agents, reply-as-message, a
  coordinator that observes the log and advances the task. Option B's reader is that coordinator.
- **agent-config**: the two scripts + the rewritten commands/skills are materialized into a pod via
  `SpawnRuntimeData.AgentFiles` like any other repo file — which is *why* the mechanics are
  repo-committed bash, not a host CLI.

---

## 9. Invariants checklist (enforce in review)

1. State changes go through `wlog.sh`; `manifest.md` is generated by `wrender.sh`, never hand-edited.
2. IDs are `work-/epic-<YYMMDDHHMM>-<slug>`; no sequential-number scanning.
3. Relays: folders = direction, log = lifecycle, files immutable; resolution is an event, not a move.
4. Only state is JSON; research/requirements/plans/handoff docs stay markdown prose.
5. Epic state is folded from child logs; no upward write-back.
6. No DB change, no orchestrator logic; upward sync (if/when) is an agent/skill.
7. Every state-relevant fact is an event — "append an event, or it didn't happen."

## 10. See also

- Decision: [append-only-work-event-log](../dev/decisions/append-only-work-event-log.md)
- Scripts: `scripts/wlog.sh`, `scripts/wrender.sh`, `scripts/windex.sh`, `scripts/epic-board.sh`
- Commands: `.claude/commands/{work,epic,planv0,implement_plan,commit,journal}.md`
- wishlist `0003` (cross-pod coordination), research `0068`/`0069`, epics `0100`/`0113`/`0114`
