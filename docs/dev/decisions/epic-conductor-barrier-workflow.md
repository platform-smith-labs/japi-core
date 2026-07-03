# Decision: Epic Conductor — Barrier-Synchronized Cross-Repo Workflow

**Date**: 2026-06-24
**Status**: Accepted
**Context**: The epic→work cross-repo workflow coordinates N child repos via file-based relay
messages (`upstream/to-*` / `from-*`) and per-repo `/work --sync`. In practice the human becomes
the scheduler with no scheduler's dashboard: with relays flying back and forth, it is unclear
which repo (terminal tab) to work on next, and decentralized `/work --sync` produced real bugs
(a lost epic parent, stale "delivered" claims, "who pulls what" confusion). This decision adds a
**central conductor** (the `epic-board`) and a **barrier-synchronized phase model**.

## Cross-repo loop (cheat-sheet)

Two commands, alternated. The **root tab** conducts; the **repo tab** does the work. You never run
`/epic sync` or `/work --sync` by hand.

```
┌─ root tab ──────────────────────────────────────────────────────────────┐
│  /epic board <epic>     sync-then-show: delivers pending relays,         │
│                         archives resolved, recomputes the barrier,       │
│                         prints "👉 DO THIS NEXT" (which repo tab)         │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │ board points you to repo B (🟢 ACT / 🔵 WORKING)
                               ▼
┌─ repo B tab ────────────────────────────────────────────────────────────┐
│  /work <B-work-id>      resume: processes OPEN upstream/from-* FIRST →    │
│                         validate (diff-gate) → act/update → reply relay   │
│                         (upstream/to-A--*) if A must change → archive the │
│                         answered relay → set **Epic Phase Done** → STOP   │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │ back to root
                               ▼
                     /epic board <epic>   (delivers B's reply to A, re-renders)
```

- **A → B:** repo A writes `upstream/to-B--<slug>.md` and STOPs. `/epic board` (root) delivers it to
  `B/upstream/from-A--<slug>.md`. In B, `/work <B-work-id>` processes it.
- **Open inbound takes precedence:** a `/work` resume handles open `from-*` relays *before* the
  phase prompt — so a **late-surfacing** dependency re-engages a repo that had already settled.
- **`confirms`/`fyi`** relays auto-resolve on delivery (no reply needed); only `blocks` relays hold
  the barrier. Round-cap: 3 exchanges/edge/phase → escalate to the human.

**Hands-off variant (both tabs in a loop).** The two tabs can each run on a `/loop` so the human
stops relaying commands entirely: `/loop /epic board <epic>` at the root, `/loop /work <work-id>
auto` in each repo. The worker's `auto` mode **executes** the current barrier phase (planning →
implementation → `/commit --force`) instead of printing the next command, then STOPs at its
`Epic Phase Done`. See Rule 6.

## Decision

An epic advances **one phase at a time across all its repos** (Requirements → Planning →
Implementation → Validation). No repo starts phase P+1 until **every** tracked repo has settled
phase P **and** all cross-repo relays for phase P are resolved. The **solution root is the sole
conductor**: it owns sync, computes the barrier/ready-set, and tells the human exactly what to run
in which repo. `/work --sync` is removed for epic-bound work items.

## Rules

### 1. The global phase barrier

The epic has a **target phase** `T = min(EpicPhaseDone across repos) + 1`. Repos whose
`Epic Phase Done` is below `T` must do phase `T`; repos already at/above `T` **wait at the
barrier**. The barrier opens (epic advances) only when **all** repos have settled the phase **and
there are zero open relays**.

Rationale for *global* (not per-edge): a new dependency can surface **late**, during the relay
settle of a phase. A global barrier guarantees no repo has already run ahead into Planning/
Implementation when that happens — the late dependency is caught while everyone is still at the
same gate.

```text
# CORRECT — barrier holds until all settle
phase=Requirements; orch=req✓ db=req✓ rt=req✓ ctl=req✓; open_relays=0
=> advance to Planning; run /planv0 in all repos

# WRONG — letting an independent repo run ahead
ctl finished requirements => ctl starts /implement_plan while db still reshaping the schema
=> ctl builds against a contract that later moves (rework)
```

### 2. The relay settle loop (per phase) + termination guard

Within a phase, every inbound relay triggers a **diff-gated mini-validation** in the receiving
repo: validate the ask against this repo's reality; if it forces a change, act and (if needed)
write a reply relay — which re-opens the settle. Loop until **fixpoint** (no repo has an open
outbound). A relay of `kind: confirms|fyi` is **auto-resolved on delivery** (no response needed)
— this is the diff-gate that prevents thrash. A **round cap** (default 3 exchanges per ordered
edge per phase) escalates to the human instead of looping forever.

```text
# CORRECT — converges
orch→db "ship 0028–0031" (blocks) ; db validates, replies "done, resolver=JOIN" (resolved) ; settle

# WRONG — no guard, oscillates
A→B ask ; B→A counter ; A→B re-ask ; ...  (no round cap, never settles)
```

### 3. Relays are append-only; resolved relays are archived, not deleted

An **open** relay is a file in `upstream/` (excluding `upstream/archive/`). `to-*` = pending
delivery; `from-*` = delivered, awaiting this repo's response. When resolved, the relay moves to
`upstream/archive/` (or its frontmatter `status:` flips to `resolved`). Never `rm` a delivered
relay — the history is needed for cycle/round detection and audit (deleting it is what made an
epic parent untraceable before).

```yaml
# CORRECT — relay carries machine-readable frontmatter
---
from: orchestrator/work-0066
to: db-migration/work-0030
kind: blocks        # blocks | confirms | fyi
phase: requirements
status: open        # open | resolved
round: 1
ask: "ship migrations 0028–0031"
---
<human-readable body>
```

### 4. The conductor owns sync; `/work --sync` is removed under an epic

Only the solution root runs sync (`/epic sync` / `epic-board sync`). It **pulls**: scans every
tracked repo's `upstream/to-*`, delivers each to its target's `from-*`, archives the source, and
reads each work manifest's `Epic Phase Done` to recompute the barrier. Child repos **never run a
sync command** — they only do their phase work and write `upstream/to-*` relays. `/work --sync`
remains valid **only for standalone (non-epic) work items**.

```text
# CORRECT — child writes a relay, conductor delivers it
(in repos/orchestrator) write upstream/to-db-migration--X.md ; STOP
(at solution root)       /epic sync   # delivers + recomputes barrier + prints next actions

# WRONG — child tries to sync under an epic
(in repos/orchestrator) /work --sync epic-0098   # removed for epic-bound work
```

### 5. Machine-readable phase field — the barrier signal is UNAMBIGUOUS, and children READ it (never recompute)

Every epic-bound work manifest carries `**Epic Phase Done**: <requirements|planning|implementation|
validation>` (the highest epic phase **this** repo has settled). The epic manifest carries the
**barrier signal** `**Epic Phase**: <phase> (<gate>)`, generated by `scripts/epic-board.sh --write`.

The signal **names the workable phase and its gate** — never a bare floor word:

```text
# CORRECT — self-describing go/no-go signal (what --write emits)
**Epic Phase**: implementation — OPEN (all repos settled planning, zero open relays — every repo may run implementation now)
**Epic Phase**: planning — HELD (3 open relay(s); resolve them before any repo starts planning)
**Epic Phase**: complete

# WRONG — the old bare floor word (ambiguous; caused this bug)
**Epic Phase**: planning     # reads as "we're IN planning, don't implement" even when implementation is OPEN
```

**`<phase> (OPEN)` is the single go signal**: every tracked repo may start `<phase>` now. `(HELD)`
means nobody starts it yet.

**Children READ the barrier; they NEVER recompute it.** A child answers "may I start phase P?" from
exactly two derived signals: (a) `**Epic Phase**: P (OPEN)` in the epic manifest, and (b) the board's
`run in each repo` block listing *its own* phase-P command. A child must **not** tally relays or
`Epic Phase Done` values itself — per-repo counting reproduces the pre-A1 miscount (a **source**
cannot see the **target's** `relay_resolved`, so it sees phantom open outbound legs) and makes repos
disagree: some hold, some proceed. The board is the only component that computes the barrier (it holds
the epic-global resolved-slug set — see
[epic-relay-one-resolution-closes-both-legs.md](./epic-relay-one-resolution-closes-both-legs.md)).
When a plan says "don't build ahead of the barrier," the barrier it means is
`**Epic Phase**: <that phase> (OPEN)`.

### 6. Autonomous worker mode (`/work <work-id> auto`) — execute, don't prompt; one phase, then stop

The review-gated default `/work <work-id>` **prints** the next command at a phase boundary
(`/planv0`, `/implement_plan`) and hands control to the human. `auto` mode instead **executes**
exactly one phase of work and STOPs, so a worker session in a `/loop` drives itself. Three
invariants make it safe to loop:

1. **One phase per invocation.** Never chain phases in a single run; the loop re-invokes.
2. **Barrier-safe.** The worker acts only up to the epic's `**Epic Phase**`; it never advances the
   barrier and never runs `/epic sync` or edits the epic manifest. Workers own `Epic Phase Done` +
   relays; the conductor owns `Epic Phase` + relay delivery. (Rule 4 still holds.)
3. **Loop-safe / idempotent.** Nothing actionable (at barrier, blocked, completed) → one status
   line, no mutation. Open inbound `from-*` relays are the highest-priority unit and preempt the
   phase (Rule 2 diff-gate), still one unit per run.

After an implementation phase completes, `auto` runs **`/commit --force`** (auto-commit only — it
does **not** push or open a PR; that stays with the human).

```text
# CORRECT — worker loop advances exactly to the barrier, then idles
epic Epic Phase=planning ; repo Epic Phase Done=requirements
=> /work <id> auto runs /planv0 logic, sets Epic Phase Done=planning, STOP
next iteration (barrier not yet moved): "✅ settled @ planning; waiting on conductor" STOP

# WRONG — running ahead of the barrier (defeats Rule 1)
epic Epic Phase=planning ; worker also runs /implement_plan in the same loop tick
=> builds against a plan the other repos haven't agreed to yet
```

### 7. Validation is two-layer — the epic owner (`solution`) drives the cross-repo e2e

A single repo cannot prove a cross-repo feature. The `validation` phase therefore has two layers:
**repo-local** validation (each child's unit/integration suite — the part only that repo can verify)
and **cross-repo e2e** validation (an end-to-end run of the epic's **Success Criteria** across live
pods — which only the conductor, `solution`, can see and drive). Each child, at validation, runs its
local suite and **declares its e2e needs to `solution`** via a `to-solution--{repo}-e2e-needs` relay
(which Success Criteria its strand exercises + the seams/fixtures/env it exposes); it does not author
or run the e2e. `solution` owns a validation work item, collects those relays, authors the e2e from
the epic Success Criteria + collected needs, drives it on a live stack, resolves each repo's relay as
covered+passing, and is the **final gate**: the epic completes on solution's e2e GO.

```text
# CORRECT — child declares, solution drives
(runtime, validation) run local suite ; write to-solution--runtime-e2e-needs (blocks) ; phase_done=validation
(solution, validation) collect all *-e2e-needs ; author + run A→B→C e2e on live stack ; resolve each ; phase_done=validation  ⇒ epic complete

# WRONG — a child tries to own the cross-repo proof (it can't see the other pods)
(runtime, validation) spin up the whole stack and hand-run an A→B→C test from inside runtime
=> partial/ad-hoc coverage, no single owner of the epic's Success Criteria
```

`scripts/epic-board.sh` emits the correct per-role validation command in its `run in each repo` block
(children: relay e2e-needs; `solution`: drive the e2e). See `/work` epic-aware Rule 5 and `/epic`.

## Rationale

- **Kills "which tab next?"** — the board computes the bottleneck (the repo with open inbound
  asks, or the laggards holding the barrier) and prints a single `DO THIS NEXT`.
- **Contract-first correctness** — freezing the requirements contract before anyone plans, and
  the plan before anyone implements, prevents "built against a contract that moved" rework.
- **One source of truth** — a single conductor with one pull-based sync path removes the
  decentralized-`/work --sync` bug class.
- **Honest about the tradeoff** — the global barrier serializes on the slowest repo. For a
  human driving one terminal tab at a time this idle cost is largely theoretical (the operator is
  already serial), and the late-dependency safety is worth more than the lost parallelism.

## Exceptions

1. **Standalone work items** (no `**Epic**:`) — keep `/work --sync` and ignore the barrier; there
   is no conductor.
2. **`confirms`/`fyi` relays** — auto-resolved on delivery; they do not hold the barrier.
3. **Human override** — the board is the recommended conductor; the operator may always override
   a gate (e.g. start an obviously-independent repo early), accepting the rework risk.

## Enforcement

- `scripts/epic-board.sh` computes and renders the barrier/ready-set; it is the conductor.
- `/epic` (board + pull-based sync + barrier) and `/work` (phase rules, frontmatter relays, no
  `--sync` under an epic; `auto` mode for loop-driven execution) encode the rules for Claude sessions.
- `/commit --force` is the non-interactive commit path `auto` mode uses (commit only — no push/PR).
- Code review + this document.

## See Also

- [Epic command](../../../.claude/commands/epic.md) · [Work command](../../../.claude/commands/work.md) · [Commit command](../../../.claude/commands/commit.md)
- [Wishlist skill](../../../.claude/skills/wishlist/SKILL.md)
- [epic-0098 (first epic run under this model)](../../epics/epic-0098-06231355-mailbox-walking-skeleton/manifest.md)
