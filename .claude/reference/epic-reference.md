# Epic Command Reference

> Templates & examples for the `/epic` command. Read on demand when scaffolding or when you need the full worked output of a subcommand. The procedural rules (ID minting, the derived-board rule, the relay model, resolution) live in `.claude/commands/epic.md` — this file is illustrative boilerplate only.

---

## `epic/context.md` Template (work-item side, `/epic` create Step 3.4)

Create `{repo}/docs/work/work-MMMM/epic/context.md`:

```markdown
# Epic Context — epic-NNNN

**Epic**: epic-NNNN
**Title**: {generated title}
**Role**: Primary Repo
**Created**: {YYYY-MM-DD}

## Original Request

{User's full prompt}

## Cross-Repo Guidance

This work item is part of a cross-repo epic. During research:

1. **Look for cross-repo impact**: If your research reveals that other repos need changes,
   write an immutable relay message for each target repo under `relays/outbound/`:
   - Filename: `relays/outbound/to-{target-repo}--{descriptive-slug}.md`
   - Content: What the target repo needs to do, any interface contracts (API endpoints,
     message formats, DB schemas), and why.
   - Then record the send as an event (do **not** edit the manifest by hand):
     `scripts/wlog.sh "$WD" relay_sent to={target-repo} slug={slug} relay_kind=blocks phase={phase} path=relays/outbound/to-{target-repo}--{slug}.md ask="..."`
     and regenerate the manifest with `scripts/wrender.sh "$WD"`.

2. **Check for incoming messages**: Before starting research/planning, read all
   `relays/inbound/from-*` files for context from other repos. When you act on one, close it with
   `scripts/wlog.sh "$WD" relay_resolved direction=inbound slug={slug}` (an **event**, not a file
   move — the message file stays put).

3. **After writing outbound relays**: the file + its `relay_sent` event are enough — the
   conductor (`/epic sync` at the monorepo root) delivers them to the target repos. Do not run
   `/work --sync` for epic-bound work.
```

---

## `/epic sync` — Step 4 Present Summary (worked example)

```
Synced epic-NNNN: {Title}

Relays delivered:
  - repo-a → repo-b: audit-events (new repo added)
  - repo-a → repo-c: ui-requirements

Repo Status:
| Repo         | Work Item  | Phase           | Status          |
|-------------|-----------|-----------------|-----------------|
| repo-a       | work-MMMM | 📝 Requirements | Done            |
| repo-b       | work-PPPP | 🎯 Proposed     | Ready to start  |
| repo-c       | work-QQQQ | 🎯 Proposed     | Ready to start  |

Next:
  - Run /work work-PPPP in repo-b/
  - Run /work work-QQQQ in repo-c/
```

---

## `/epic status` — dashboard (worked example)

```
Epic epic-NNNN: {Title}
Status: Active  |  Repos: {N}  |  Epic Phase: {barrier}  |  Last Updated: {date}

| Repo         | Work Item  | Phase          | Status            |
|-------------|-----------|----------------|-------------------|
| repo-a       | work-MMMM | 📝 Requirements | Done              |
| repo-b       | work-PPPP | 📚 Researching  | In progress       |
| repo-c       | work-QQQQ | 🎯 Proposed     | Ready to start    |

Pending Relays: {N}
{list of open outbound relays (relay_sent, not yet relay_synced) waiting to be delivered}

Recommended:
  1. {highest priority action}
  2. {next action}
```

### The `run in each repo` block (always emitted by `scripts/epic-board.sh`)

Every board render (`/epic board`, `/epic status`, `/epic sync`) ends with a per-repo action block so
the human never has to hand-assemble the command. It lists, for each repo with in-repo work
(🟢 ACT = open inbound ask, or 🔵 WORKING = owes the barrier's next phase), the dir to open a Claude
session in + the slash command with the **real work id** — and for ACT repos, the exact relay close:

```
run in each repo (open a Claude session in the dir, then run the command):
  runtime       cd repos/runtime          → in Claude:  /work work-2607012141-checkout-or-create
                  then close inbound: scripts/wlog.sh docs/work/work-2607012141-checkout-or-create relay_resolved direction=inbound slug=checkout-or-create && scripts/wrender.sh docs/work/work-2607012141-checkout-or-create
  ps-ui         cd repos/ps-ui            → in Claude:  /work work-2607012141-branch-input
                  then close inbound: scripts/wlog.sh docs/work/work-2607012141-branch-input relay_resolved direction=inbound slug=branch-input && scripts/wrender.sh docs/work/work-2607012141-branch-input
then, back at the solution root:  /epic sync epic-NNNN   (delivers replies, re-derives the barrier)
```

Present it verbatim. It is derived from the child work ids + open relays (never hand-written), so it
stays correct as the epic advances. ⏳ BLOCKED repos (own the outbound, waiting on replies) and
at-barrier repos are omitted — their next move is `/epic sync`, not in-repo work. The phase command
tracks the barrier: `/work <id>` at requirements, `/planv0 --work <id>` at planning,
`/implement_plan docs/work/<id>/plans/master.md` at implementation. **Validation is two-layer** (see
below), so the block emits a per-role command there.

### Validation phase — two-layer (child declares, `solution` drives the e2e)

At `validation` the board emits different commands by role. A **child repo** runs its local suite and
relays its e2e needs to solution; **`solution`** (the epic owner) authors + drives the cross-repo e2e
of the epic Success Criteria and is the final gate. Example block at validation:

```
run in each repo (open a Claude session in the dir, then run the command):
  runtime    cd repos/runtime   → run LOCAL suite; then relay e2e-needs → solution: write
             relays/outbound/to-solution--runtime-e2e-needs.md + scripts/wlog.sh docs/work/<id>
             relay_sent to=solution slug=runtime-e2e-needs relay_kind=blocks phase=validation
             ask="e2e coverage for runtime"; then settle: … phase_done phase=validation
  solution   cd .               → author + DRIVE the cross-repo e2e (epic Success Criteria) on a live
             stack — you are the e2e gate; resolve each repo's e2e-needs relay as covered+passing,
             then: … phase_done phase=validation
```

The `to-solution--{repo}-e2e-needs` relay body should list: which epic Success Criteria this strand
exercises, the seams/fixtures/env it exposes (e.g. "seed a repo lacking branch X to hit tier-2
create; assert push"), and the strand's local test status. See `/work` epic-aware Rule 5 and
`docs/dev/decisions/epic-conductor-barrier-workflow.md` Rule 7.

---

## `/epic next` — output examples

**If Status = `🎯 Pending scaffold`** (Sub-Epic ID column is `—` or empty):

```
Next sub-epic for epic-NNNN ({Title}):

  ▶ {Phase} — {Sub-Epic title}
    Status:   🎯 Pending scaffold
    Repos:    {Repos}
    Effort:   {Effort}

To scaffold from parent dir:
  /epic {primary-repo} "{Sub-Epic title} (Vision {Phase})" --parent epic-NNNN

Or from inside the primary repo:
  /work --epic-from-parent epic-NNNN "{Sub-Epic title} (Vision {Phase})"
```

(The `--parent` / `--epic-from-parent` flags are documented in the parent-epic linkage section; if the harness doesn't yet support them, the manual form is `/epic {repo} "..."` followed by editing the new epic's `Related to:` to point at the parent and adding the new sub-epic ID into the parent's roadmap row.)

**If Status = `🔄 Active`** (Sub-Epic ID is set):

```
Next sub-epic for epic-NNNN ({Title}):

  ▶ {Phase} — {Sub-Epic title}
    Status:    🔄 Active
    Sub-Epic:  epic-PPPP
    Work item: {repo}/docs/work/work-QQQQ

To resume:
  cd {repo}/  &&  /work work-QQQQ

Or check status:
  /epic status epic-PPPP
```

**If Status = `🔴 Blocked`**:

```
Next sub-epic for epic-NNNN is BLOCKED:

  ▶ {Phase} — {Sub-Epic title}
    Status:    🔴 Blocked
    Sub-Epic:  epic-PPPP

Read epic-PPPP manifest for blocker details. Resolve the blocker, then run /epic next again.
```

---

## `/epic update-sub` — Step 3 output (worked example)

```
Updated epic-NNNN Sub-Epic Roadmap:
  {Phase} → {Status}{ · Sub-Epic ID: epic-PPPP if provided}

Run /epic next epic-NNNN to see what's up next.
```

---

## `/epic next-milestone` — Step 4 output (worked example)

```
Advanced wishlist NNNN: Mx (epic-AAAA ✅) → M(x+1) (epic-BBBB, new)

Created epic-BBBB: {next-milestone title}
  Primary repo: {repo} → work-MMMM
  Wishlist back-link updated (Tracking row + Picked up registry).

Next: open a session in {repo}/ and run:
  /work work-MMMM
```

---

## Sub-Epic Roadmap (template — for parent epics)

A parent epic that spawns a sequenced set of child sub-epics MUST embed this section in its manifest. Place it after `## Tracked Repos` and before `## Dependencies`:

```markdown
## Sub-Epic Roadmap

Parent epic for a phased rollout. Each row below is a sub-epic that gets scaffolded as a separate epic when the prior phase completes. Use `/epic next epic-NNNN` to see what's up next; `/epic update-sub epic-NNNN {Phase} {Status}` to flip status.

| # | Phase | Sub-Epic | Repos | Effort | Status | Sub-Epic ID |
|---|-------|----------|-------|--------|--------|-------------|
| 1 | V1.1 | Workspace Overview Page | ps-ui | 3d UI | 🎯 Pending scaffold | — |
| 2 | V1.2 | Repo Picker + Import-as-Project | orchestrator + ps-ui | 2d BE + 2d UI | 🎯 Pending scaffold | — |
| 3 | V1.3 | Spawn Sandbox Dialog | ps-ui | 2d UI | 🎯 Pending scaffold | — |
| 4 | V1.5 | Command Palette V1 Grammar | ps-ui | 2d UI | 🎯 Pending scaffold | — |
```

**Status legend** (canonical for roadmap rows):
- 🎯 **Pending scaffold** — sub-epic not yet created
- 🔄 **Active** — sub-epic created and in progress
- 🔴 **Blocked** — sub-epic exists, blocked on dependency or external decision
- ⏸️ **On Hold** — paused; not blocked
- ✅ **Completed** — sub-epic done, all acceptance criteria met
- ❌ **Cancelled** — won't be implemented

When a sub-epic is scaffolded, its child manifest's `**Related to**:` field MUST link back to the parent epic, and the parent's roadmap row gets its `Sub-Epic ID` filled in via `/epic update-sub`.

---

## Epic Manifest Template

The manifest is a **mix**: the header metadata + `## Original Request` / `## Summary` /
`## Open Questions` / `## Related` / narrative `## Change Log` are **authored prose** (hand-written).
The `**Epic Phase**` barrier line and the `## Tracked Repos` table are **derived** — they live between
the `<!-- BEGIN BOARD -->` / `<!-- END BOARD -->` anchors and are **generated by
`scripts/epic-board.sh --write epic-NNNN`** from the child work logs. Author the anchors (and a
placeholder line) once; never hand-edit what the script writes between them.

```markdown
# Epic: epic-NNNN — {Title}

**Status**: Active
**Created**: {YYYY-MM-DD}
**Last Updated**: {YYYY-MM-DD}
**Primary Repo**: {repo-name}
**Wishlist**: {NNNN — milestone Mx if this epic implements a docs/wishlist/ item; omit this line otherwise}

## Original Request

{User's original prompt — exactly as provided}

<!-- BEGIN BOARD -->
<!-- Generated by scripts/epic-board.sh --write — DO NOT EDIT BY HAND. -->
<!-- The barrier signal names the WORKABLE phase + gate, e.g.
     "requirements — OPEN (kickoff — every repo may run requirements now)" /
     "implementation — HELD (2 open relay(s)…)" / "complete". `<phase> (OPEN)` = go. -->
**Epic Phase**: requirements — OPEN (kickoff — every repo may run requirements now)

## Tracked Repos

| Repo | Work Item | Epic Phase Done | Phase | Status | Open Relays |
|------|-----------|-----------------|-------|--------|-------------|
| {repo} | work-MMMM | — | 🎯 Proposed | Not started | 0 |
<!-- END BOARD -->

## Change Log

- {YYYY-MM-DD}: Epic created, primary repo: {repo} (work-MMMM)
```

> The exact columns the board renders are owned by `scripts/epic-board.sh`; treat the table above as
> illustrative of the placeholder, not a hand-maintained schema. There is no hand-kept `## Relay Log`
> table — relay history lives in the child `work.jsonl` logs (`relay_sent`/`relay_received`/
> `relay_synced`/`relay_resolved` events) and is summarized by the board.

---

## Epic Index Template

```markdown
# Epics Registry

Last Updated: {YYYY-MM-DD}

## Active Epics

| ID | Title | Primary Repo | Repos | Status | Created |
|----|-------|-------------|-------|--------|---------|
| epic-0001-04251432-oauth-social-login | {Title} | {repo} | {N} | Active | {date} |

## Completed Epics

{Move here when all repo work items are completed}

## Cancelled

{Move here when cancelled}
```
