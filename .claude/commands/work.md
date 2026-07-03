# Work Item Management Command

**Purpose**: Create and manage unified work items that group related research, requirements, plans, and implementation artifacts.

> ## 🧾 State is an append-only event log
>
> **Never hand-edit `manifest.md`.** It is GENERATED from `<WD>/work.jsonl` by `scripts/wrender.sh`.
> Record every state change by **appending an event** with `scripts/wlog.sh`, then **regenerate** with
> `scripts/wrender.sh "$WD"`. The renderer owns all badges, checkboxes, dates, and the change log.
>
> **IDs are `work-<YYMMDDHHMM>-<slug>`** (epics: `epic-<YYMMDDHHMM>-<slug>`) — a timestamped slug,
> **no sequential numbers, no scan-max-increment**. Resolve an existing id by **glob**, never arithmetic.
>
> Throughout this doc, `$WD` is the work item directory (e.g. `docs/work/<id>`) and `<id>` is the full
> minted id. See **docs/dev/decisions/append-only-work-event-log.md** for the law and
> **docs/design/work-event-log-and-a2a-port.md** for the full spec.

## Command Usage

```bash
/work "Natural language prompt"       # Auto-create work + research + requirements
/work <id>                            # Resume (review-gated: prints the next command, hands back to you)
/work <id> auto                       # Autonomous: EXECUTE one phase, append events, STOP (loop-friendly)
/work --epic <id>                     # Promote existing work item to cross-repo epic
/work show <id>                       # Show work item details
/work list                            # List all work items
/work update <id> --status X          # Update work status (appends status_changed)
```

## ID Format

Work IDs follow the format: **`work-<YYMMDDHHMM>-<slug>`** (epics: **`epic-<YYMMDDHHMM>-<slug>`**).

- **`YYMMDDHHMM`** — current local time from `date +%y%m%d%H%M` (2-digit year, month, day, hour 24h, minute, no separators).
- **`<slug>`** — kebab-case slug derived from the prompt: lowercase, hyphen-separated, 2–5 distinctive words, ≤30 chars (e.g. `oauth-social-login`).

Mint it in one deterministic shot — **no global read, no counter**:

```bash
SLUG="oauth-social-login"
WORK_ID="work-$(date +%y%m%d%H%M)-$SLUG"   # e.g. work-2607010322-oauth-social-login
```

> ### ⚠️ No sequential numbers — anywhere
>
> There is **NO** scan-max-NNNN, **NO** zero-padding, **NO** increment. Two agents on two branches
> never collide on a counter because there is no counter; the timestamp + slug keep ids unique on
> merge, and `YYMMDDHHMM` preserves chronological `ls` order. Minute-level collisions are tolerated
> (the slug disambiguates).
>
> **Legacy items** (`work-NNNN-…` with hand-edited manifests) are **not migrated** — they remain
> readable side-by-side. Never rename or migrate them. Only *new* items use the slug id + event log.

When this command promotes a work item to a cross-repo epic (`/work --epic`), the new epic id uses the
matching format: **`epic-<YYMMDDHHMM>-<slug>`** — reuse the work item's slug for traceability.

## Resolving Existing IDs

When a subcommand takes an id argument (`/work <id>` resume, `/work show`, `/work update`,
`/work --epic <id>`), the user may pass the full id or just the slug. Resolution is a **glob, never
arithmetic**:

1. **Try exact match** — `docs/work/{arg}/work.jsonl`. If found, `WD=docs/work/{arg}`.
2. **Else glob by slug** — `docs/work/*{arg}*/work.jsonl` (matches full id or bare slug; also matches
   legacy `work-NNNN-…` dirs that contain the slug).
3. **If exactly one match**, use that directory. If zero, error: "Work item {arg} not found." If
   multiple, error and list matches.

The same glob rule applies to **epic ids** (resolve against `../docs/epics/`). For legacy items that
still carry a hand-edited `manifest.md` and no `work.jsonl`, fall back to matching `manifest.md`.

Throughout this document, `<id>` is shorthand for the resolved directory name and `$WD` for its path.

## Hierarchy & optional parents

The workflow has three nesting tiers, **each parent optional**:

```
wishlist item   (a deferred idea; may spawn 0..N epics, one per milestone)   ← docs/wishlist/ (monorepo root)
   └─ epic       (cross-repo coordination; may own 1..N work items)            ← docs/epics/ (monorepo root)
        └─ work  (single-repo execution)                                       ← {repo}/docs/work/
```

- A **work item** may be **standalone** (`/work "prompt"` — no epic), **epic-owned** (created by
  `/epic`), or wishlist-derived (via its epic). Parent epic is **optional**.
- An **epic** may be **standalone** (no wishlist) or **wishlist-derived**. Parent wishlist is **optional**.
- A **wishlist item** may map to **0..N epics** over time (incremental, one per milestone).

**Linkage fields** are carried as event metadata (the `created`/`meta_changed` `epic=` and `wishlist=`
keys), and rendered into the manifest header by `wrender.sh` — never hand-edited:
- work manifest header: `**Epic**` and/or `**Wishlist**` lines (present only when set on the log)
- epic brief: its own `**Wishlist**` line

**Upward status sync is derived, not pushed.** Each work item's state lives in its own `work.jsonl`;
the epic rollup is **folded from child logs** by `scripts/epic-board.sh` (read-only) — a child's last
`phase_done` is read off its generated manifest. There is **no** `/work --sync` write-back into epic
or wishlist tables. Run `scripts/epic-board.sh` (or `/epic`) at the monorepo root to see the rolled-up
view. See **Epic-aware work** below.

## Behavior

### When user runs: `/work "Natural language prompt or problem description"`

This is the **PRIMARY and RECOMMENDED** usage. User provides a freeform description of what they want to build or problem they're facing.

**Examples**:
- `/work "I want to add OAuth social login to the app"`
- `/work "Running into performance issues with database queries"`
- `/work "Want to build a payment integration with Stripe"`

You MUST execute this **3-phase automatic workflow**:

#### Phase 1: Create Work Item

1. **Mint the Work ID** — format `work-<YYMMDDHHMM>-<slug>` (see **ID Format** above)
   - **slug**: kebab-case from the user's prompt — lowercase, hyphen-separated, 2–5 distinctive words,
     ≤30 chars. Strip stopwords; keep nouns/verbs. Example: `"I want to add OAuth social login to the
     app"` → `oauth-social-login`.
   - **Mint deterministically** (no scan, no counter):
     ```bash
     SLUG="oauth-social-login"
     WORK_ID="work-$(date +%y%m%d%H%M)-$SLUG"
     WD="docs/work/$WORK_ID"
     ```

2. **Extract Title from Prompt**
   - Analyze the user's prompt
   - Generate concise title (3-8 words)
   - Example: "I want to add OAuth login" → "OAuth Social Login Integration"

3. **Scaffold the directory and append the `created` event**
   - Create the folder skeleton (relays subdirs only if this is or may become epic-bound; always safe
     to create them):
     ```bash
     mkdir -p "$WD"/{research,requirements,issues,plans,relays/outbound,relays/inbound}
     ```
   - Append the creation event, then render the manifest:
     ```bash
     scripts/wlog.sh "$WD" created title="<Generated Title>" slug="$SLUG" kind=work \
       repo=<this-repo> owner=<owner-email> request="<the user's verbatim prompt>" \
       [epic=<epic-id>] [wishlist=<n>] [priority=<P>] [effort=<S|M|L>]
     scripts/wrender.sh "$WD"
     ```
     Omit `epic=`/`wishlist=` for a standalone item; include them only when the parent exists.
   - **Always pass `request=`** with the user's original prompt verbatim — `wrender.sh` surfaces it as
     the `## Original Request` section (load-bearing context for git-resume). `wlog.sh` JSON-encodes it
     safely, so quotes/newlines in the prompt are fine. Do **not** hand-write a manifest — `wrender.sh`
     generates it, with a "DO NOT EDIT BY HAND" banner.

4. **Registry is generated — do not hand-maintain an index**
   - There is **no** `docs/work/index.md` row to edit. Regenerate the roll-up with
     `scripts/windex.sh docs/work` (or the repo's work root, e.g. `scripts/windex.sh repos/<repo>/docs/work`) —
     it folds every item's generated `manifest.md` into `index.md`. Never hand-edit `index.md`.

5. **Notify User**
   - Output: "✅ Created $WORK_ID: {Generated Title}"
   - Output: "📋 Original Request: {User's prompt}"
   - Output: "🔍 Starting automatic research and requirements gathering..."

#### Phase 2: Automatic Research

**Immediately after creating work item**, you MUST:

1. **Spawn Research Agent**
   - Use Task tool with subagent_type appropriate for the domain
   - Pass the user's prompt as research context
   - Include work id in the task: `--work <id>`
   - Research should cover:
     - Understanding the problem/requirement
     - Exploring existing codebase for related patterns
     - Investigating best practices and approaches
     - Analyzing technology options if applicable

2. **Create Research Document**
   - Folder: `$WD/research/`
   - Filename: `$WD/research/0001-{slug}-research.md`
   - Content follows standard research document structure (markdown prose — unchanged by the event log)
   - References work item: `Work Item: <id>`
   - **IMPORTANT**: Initial research auto-created, more can be added with `/research --work <id>`

3. **Record state via events** (never hand-edit the manifest):
   ```bash
   scripts/wlog.sh "$WD" status_changed to=researching
   scripts/wlog.sh "$WD" artifact_added kind=research path=research/0001-{slug}-research.md title="Initial Research"
   scripts/wlog.sh "$WD" status_changed to=requirements note="research complete"
   scripts/wrender.sh "$WD"
   ```
   The renderer derives the status badge, the Artifacts list, the workflow checkboxes, the change log,
   and Last Updated from these events. Do **not** touch those manifest fields by hand.

#### Phase 3: Automatic Requirements

**Immediately after research completes**, you MUST:

1. **Spawn Requirements Agent**
   - Use Task tool with appropriate agent (ux-researcher, architect-reviewer, qa-expert)
   - Base requirements on:
     - User's original prompt
     - Research findings (from `$WD/research/*.md`)
     - Existing codebase patterns discovered
   - Include work ID context

2. **Create Requirements Document**
   - Folder: `$WD/requirements/`
   - Filename: `$WD/requirements/0001-{slug}-req.md`
   - Content follows standard requirements structure (markdown prose — unchanged by the event log):
     - Overview and objectives
     - Functional requirements
     - Non-functional requirements
     - User stories / use cases
     - Acceptance criteria
     - Constraints and assumptions
   - References work item: `Work Item: <id>`
   - References research: Link to relevant research docs
   - **IMPORTANT**: Initial requirements auto-created, more can be added with `/new_req --work <id>`

3. **Record state via events** (never hand-edit the manifest):
   ```bash
   scripts/wlog.sh "$WD" artifact_added kind=requirements path=requirements/0001-{slug}-req.md title="Initial Requirements"
   scripts/wrender.sh "$WD"
   ```
   Status is already `requirements` from Phase 2; the renderer folds the new artifact, the workflow
   checkbox, the change log, and Last Updated. If this work item is **epic-bound** and you have settled
   the requirements phase, also append `scripts/wlog.sh "$WD" phase_done phase=requirements` (see
   **Epic-aware work**), then re-render.

#### Phase 4: Return Control to User

After both research and requirements are complete:

1. **Present Summary**
   - Output: "✅ Research completed: $WD/research/0001-{slug}-research.md"
   - Output: "✅ Requirements documented: $WD/requirements/0001-{slug}-req.md"
   - Output: ""
   - Output: "📊 Work Item Status: 📝 Requirements (Ready for Planning)"

2. **Request User Review**
   - Output: "Please review the research and requirements documents in $WD/"
   - Output: "You can add more research or requirements with:"
   - Output: "  /research --work <id> \"Additional research topic\""
   - Output: "  /new_req --work <id> \"Additional requirements\""
   - Output: ""
   - Output: "When you're ready to proceed, run:"
   - Output: "`/planv0 --work <id>`"
   - Output: ""
   - Output: "This will create an implementation plan based on ALL research and requirements."

3. **Support Iteration**
   - User may ask questions, request changes to research or requirements
   - Update documents based on feedback
   - Only proceed to planning when user explicitly runs `/planv0 --work <id>`

### When user runs: `/work <id>` (existing work item id)

**Resume an existing work item from its current status.** This makes `/work` idempotent — it picks up where the last session left off.

> **Review-gated (default) vs autonomous.** The steps below are the **review-gated** default: at a
> phase boundary they **print the next command** (`/planv0`, `/implement_plan`) and hand control back
> to you. To make `/work` **execute** that next step instead of printing it — so a worker session
> drives itself in a loop — use **`/work <id> auto`** (see **Autonomous mode** below). Both paths
> share the same event-log recording and barrier/relay rules; they differ only in whether a phase
> boundary stops for you or runs the work.

1. Resolve `<id>` → `$WD` (see **Resolving Existing IDs**). Read `$WD/manifest.md` for the rendered
   view, and `$WD/work.jsonl` if you need the precise event history (the manifest is folded from it).
2. If `epic/` folder exists, read `epic/context.md` for cross-repo context (authored prose).
3. **Open inbound relays take precedence (epic-bound only).** If the item has an `**Epic**` link AND
   the manifest's **Open Relays** section lists **open inbound** relays (a `relay_received` with no
   matching `relay_resolved` for that `direction=inbound`+`slug`), do **NOT** jump to the phase-status
   prompt in step 4. **Process them first** per **Epic-aware work** below — an open inbound ask is
   actionable work in THIS repo regardless of phase (this is how a *late-surfacing* dependency
   re-engages a repo that had already settled). Only after every open inbound relay is validated →
   acted-on → answered (reply relay if needed) → **resolved** (an event), re-evaluate the settled phase
   and proceed to step 4.
4. Check status (the rendered badge, derived from the last `status_changed`) and continue:
   - **🎯 Proposed** → Start Phase 2 (Research). Use `epic/` and inbound relays if present.
   - **📚 Researching** → Check what research exists in `research/`. If incomplete, continue. If complete, move to Phase 3 (Requirements).
   - **📝 Requirements** → Check what requirements exist. If complete, prompt: "Requirements ready. Run `/planv0 --work <id>` to create implementation plan."
   - **🎨 Planning** → "Planning phase. Run `/planv0 --work <id>` to create or review the plan."
   - **🔄 In Implementation** → "Implementation in progress. Run `/implement_plan $WD/plans/master.md` to continue."
   - **✅ Completed** → "This work item is already completed."
   - **🔴 Blocked** → Display blockers (from the latest `status_changed to=blocked` note), suggest resolution.

**Epic-aware work (barrier-synchronized conductor model)**:
When the work item has an `**Epic**` link, this repo is one strand of a **barrier-synchronized** epic.
Honor these rules in EVERY phase (requirements, planning, implementation), not just research. Relay
**messages** are immutable files under direction-named folders; relay **lifecycle** is events in
`work.jsonl`. Resolution is an **event, never a file move/delete**.

1. **Read inbound first.** Read all `epic/` files + every **open inbound** relay file
   (`relays/inbound/from-*.md`) whose `relay_received` has no matching `relay_resolved`.
2. **Mini-validate every open inbound relay** (diff-gate): check the ask against THIS repo's reality.
   - If it forces a change → do the needful for the current phase. If your response requires the
     sender to change something, write a reply outbound relay (next rule). Then close it:
     ```bash
     scripts/wlog.sh "$WD" relay_resolved direction=inbound slug=<slug> note="<how addressed>"
     scripts/wrender.sh "$WD"
     ```
   - If it's a pure `confirms`/`fyi` (no action needed) → just resolve it the same way. **Never move
     the file to an archive/ folder** — the file stays put; the log records the resolution.
3. **Write outbound relays** when you need another repo to do/know something. Author an **immutable**
   message file `relays/outbound/to-{target-repo}--{slug}.md` with YAML frontmatter + prose body:
   ```yaml
   ---
   from: {this-repo}/{this-id}
   to: {target-repo}
   kind: blocks        # blocks (holds the barrier) | confirms | fyi
   phase: {current epic phase}
   ask: "<one-line ask>"
   ---
   ```
   (followed by the human-readable detail — contracts, schemas, why), then append the event:
   ```bash
   scripts/wlog.sh "$WD" relay_sent to={target-repo} slug=<slug> relay_kind=blocks \
     phase=<phase> ask="<one-line ask>" path=relays/outbound/to-{target-repo}--{slug}.md
   scripts/wrender.sh "$WD"
   ```
   When the peer picks the message up (delivery), append `relay_synced slug=<slug>` — the file STAYS;
   `relay_synced` is a delivery annotation, it does **not** close the relay.
4. **Do NOT advance to the next phase yourself.** When you finish the current phase, append the
   barrier signal and **STOP**:
   ```bash
   scripts/wlog.sh "$WD" phase_done phase=<requirements|planning|implementation|validation> note="<what settled>"
   scripts/wrender.sh "$WD"
   ```
   Then tell the user: *"{this-repo} has settled {phase}. Run `scripts/epic-board.sh` (or `/epic`) at
   the monorepo root — the conductor folds the child logs and advances the epic when all repos settle."*
   The global barrier means no repo proceeds to the next phase until every repo settles this one. The
   `**Epic Phase Done**` line in your manifest is **rendered** from your last `phase_done` event — never
   hand-edit it, and never hand-edit the epic's Tracked-Repos cells (those are folded by
   `scripts/epic-board.sh`).
5. **The VALIDATION phase is TWO-LAYER (epic-bound only).** A single repo cannot prove a cross-repo
   feature — the epic's real acceptance is an end-to-end run of its **Success Criteria** across live
   pods, which only the epic owner (**solution**) can drive. So validation splits by who you are:
   - **You are a child repo** (orchestrator, runtime, ps-ui, …): (a) run your **local** suite
     (unit/integration — the part only you can verify); (b) **declare your e2e needs to solution** —
     write `relays/outbound/to-solution--{this-repo}-e2e-needs.md` (`kind=blocks`, `phase=validation`)
     stating which epic Success Criteria your strand must exercise, the **seams/fixtures/env** you
     expose (e.g. *"seed a repo lacking branch X to hit tier-2 create; assert push"*), and your local
     test status; log it (`relay_sent … slug={this-repo}-e2e-needs`). Do **NOT** author or run the
     cross-repo e2e yourself. Then settle: `phase_done phase=validation`.
   - **You are solution** (the epic owner's validation work item): you are the **e2e gate**. Collect
     every `from-*-e2e-needs` relay, author the cross-repo e2e from the epic's **Success Criteria +
     the collected needs**, stand up the stack, and **drive all the tests**. Resolve each repo's
     e2e-needs relay (`relay_resolved direction=inbound`) as its scenario is covered **and passing**.
     Settle `phase_done phase=validation` **only** when the full e2e is GREEN — that GO is the epic's
     completion (and the commit) gate.

**If no epic context exists** (standalone work item), proceed normally — the barrier + conductor apply
only to epic-bound work. (Backward compatible.)

---

### When user runs: `/work <id> auto` (autonomous mode)

**Same as `/work <id>` resume, but at a phase boundary it EXECUTES exactly one phase of work, records
the outcome as `work.jsonl` events, and STOPS — instead of printing the next command for you to run.**
This is the mode to put in a loop (e.g. `/loop /work <id> auto`): each worker session drives itself
through research → requirements → planning → implementation → commit without you relaying commands.
Accepts `auto` (positional) or `--auto`.

Everything about state is unchanged from the rest of this command: **append events via `scripts/wlog.sh`,
regenerate with `scripts/wrender.sh`, never hand-edit `manifest.md`.** `auto` only removes the review
gate; the recording model is identical.

#### Invariants (must hold every invocation)

1. **One phase per invocation, then STOP.** Do not chain phases. The loop re-invokes for the next one.
   (Keeps each iteration cheap to reason about and barrier-safe.)
2. **Barrier-safe for epic-bound work.** Never advance past the epic's current `**Epic Phase**`. Do the
   barrier phase, append `phase_done phase=<target>`, STOP. The conductor (`/epic board`/`/epic sync` at
   the monorepo root) is the **only** thing that opens the barrier — `auto` mode **never** runs `/epic
   sync`, never edits the epic manifest, and never hand-edits its own `Epic Phase Done` (that line is
   rendered from the `phase_done` event).
3. **Loop-safe / idempotent.** If there is nothing actionable this iteration (at the barrier waiting on
   other repos, blocked, or completed), **report it in one line and STOP without appending any event.**
   Re-running must not create duplicate work or re-do a finished phase.

#### Algorithm

1. **Read state.** Resolve `<id>` → `$WD` and read `$WD/manifest.md` (the generated view; fold detail
   from `$WD/work.jsonl` if needed). Note `Status`, `**Epic**`, `**Epic Phase Done**`. If `**Epic**:
   <epic-id>` is present, also read the epic manifest's generated `**Epic Phase**` field (resolve the
   epic dir via the monorepo root, same as `/epic`). Read `epic/context.md` if present.

2. **Open inbound relays are the highest-priority unit of work (epic-bound).** If the manifest's
   **Open Relays** lists any **open inbound** relay (a `relay_received` with no matching
   `relay_resolved` for `direction=inbound`+`slug`), process them per **Epic-aware work** above
   (validate → act → reply relay if needed → append `relay_resolved` + `wrender.sh`), and **STOP** —
   relays are this invocation's one phase-unit. Do not also advance a phase in the same run.

3. **Pick the target phase (exactly one).**
   - **Epic-bound:** compare phase ordinals `requirements(1) < planning(2) < implementation(3) <
     validation(4)`.
     - If `ord(Epic Phase Done) ≥ ord(Epic Phase)` → this repo is **at the barrier**. Output
       `✅ {repo} settled @ {Epic Phase Done}; waiting on conductor/other repos.` and **STOP** (no event
       appended).
     - Else **target = the epic's `Epic Phase`** (the barrier phase). Never pick a phase beyond it.
   - **Standalone:** target = the next pending phase implied by `Status` (see mapping below).

4. **Execute the target phase — and only that phase — to completion.** Record via events, then STOP:

   | Target phase | Action (run the command's logic inline) | Events appended (then `wrender.sh "$WD"`) |
   |---|---|---|
   | `requirements` (from 🎯 Proposed / 📚 Researching) | Run **Phase 2 + Phase 3** (research + requirements) from the `/work "prompt"` flow | `status_changed to=researching` → `artifact_added kind=research …` → `status_changed to=requirements` → `artifact_added kind=requirements …`; if epic-bound, `phase_done phase=requirements` |
   | `planning` (from 📝 Requirements) | Execute **`/planv0 --work <id>`** logic | `status_changed to=planning` → `artifact_added kind=plan …` (per plan file); if epic-bound, `phase_done phase=planning` |
   | `implementation` (from 🎨 Planning / 🔄 In Implementation) | Execute **`/implement_plan $WD/plans/master.md`** logic; when implementation is complete, run **`/commit --force`** (auto-commit only — no push/PR) | `status_changed to=implementation` → (on completion) `status_changed to=completed`; if epic-bound, `phase_done phase=implementation` |
   | `validation` | Run the plan's tests / validation steps | if epic-bound, `phase_done phase=validation`; `status_changed to=completed` (or `to=blocked` on failure) |

5. **Record and STOP.**
   - **Epic-bound:** append `phase_done phase=<target>` (with a `note=` describing what settled), run
     `wrender.sh "$WD"`, and STOP. Output: `✅ {repo} settled {target}. Conductor will advance the
     barrier.` Do **not** run `/epic sync`, do **not** prompt the user, do **not** start the next phase.
   - **Standalone:** append the `status_changed` for the new status (with a `note=`), run `wrender.sh
     "$WD"`, and STOP. The loop's next invocation continues from the new status.
   - **On a blocker** (failed validation, missing plan, unresolved dependency): append
     `status_changed to=blocked note="<the blocker>"`, run `wrender.sh "$WD"`, output one line naming
     it, and STOP (do not retry in a tight loop).

#### How the two loops cooperate

```
solution root (conductor loop):   /loop /epic board <epic-id>     # sync relays + recompute barrier
each repo (worker loop):          /loop /work <id> auto           # execute current barrier phase, stop
```

The worker reads the epic's `**Epic Phase**` (barrier, generated by `epic-board.sh`) and acts only up
to it; the conductor folds every repo's `phase_done` events (→ `Epic Phase Done`) + open relays and
opens the barrier when all repos settle with zero open relays. Neither loop mutates the other's
authority — workers own `phase_done` + relay events; the conductor owns the derived `Epic Phase` +
relay delivery. A worker at the barrier idles cheaply (step 3 STOP) until the conductor advances the
barrier, at which point the next worker iteration does the new phase.

**Exception recap:** `auto` removes the per-phase review gate — each invocation executes one phase and
stops, so the item advances without manual relaying. The default `/work <id>` keeps the review gates.

---

### When user runs: `/work --epic <id>`

**Promote an existing work item to a cross-repo epic.** Use this when you started with a normal `/work "prompt"` and later discover the feature needs changes in other repos.

1. Resolve `<id>` → `$WD`. Read `$WD/manifest.md` → original request and current status; read
   `$WD/work.jsonl` for the precise `created` metadata.
2. If the item already carries an `**Epic**` link → "This work item is already linked to {epic-id}."
3. Generate title from the Original Request (3-8 words).
4. Mint the epic id `epic-<YYMMDDHHMM>-<slug>` (see **ID Format**) — **no scan, no counter**. Reuse the
   work item's slug (the part after the timestamp in `<id>`) so the epic and its primary work item share
   a slug for traceability:
   ```bash
   SLUG="<work item slug>"
   EPIC_ID="epic-$(date +%y%m%d%H%M)-$SLUG"
   ```
5. Determine this repo's name from the current directory (basename of pwd).
6. Create the epic under `../docs/epics/$EPIC_ID/` per the `/epic` command (a thin authored brief is
   fine; its Tracked-Repos/barrier state is **folded** by `scripts/epic-board.sh`, never hand-synced).
7. Link the work item back to the epic by appending an event (not by editing the manifest):
   ```bash
   scripts/wlog.sh "$WD" meta_changed epic=$EPIC_ID note="promoted to cross-repo epic"
   scripts/wrender.sh "$WD"
   ```
8. Create `$WD/epic/context.md` (authored prose) with the epic id, title, and cross-repo guidance
   (same as the `/epic` command's context template), and ensure relay folders exist:
   ```bash
   mkdir -p "$WD"/epic "$WD"/relays/outbound "$WD"/relays/inbound
   ```
9. Output:
   ```
   Created {epic-id}: {Title}
   Linked to {work-id} in {this-repo}

   To relay findings to other repos, author relays/outbound/to-{repo}--{slug}.md and append a
   relay_sent event; the target repo reads it and runs scripts/epic-board.sh / /epic to roll up.
   ```

> **Note**: There is no `/work --sync`. Cross-repo state is never pushed by a child — the epic rollup
> is **derived** from child `work.jsonl` logs by `scripts/epic-board.sh` (read-only). Children write
> relays + `phase_done` events and STOP; the conductor (`/epic` at the monorepo root) folds and
> advances. Any upward wishlist reflection is likewise a read-only fold, never a hand-edited table.

---

### When user runs: `/work show <id>`

You MUST:
1. Resolve `<id>` → `$WD` and read `$WD/manifest.md` (the rendered view). If it looks stale, run
   `scripts/wrender.sh "$WD"` first to re-fold from `work.jsonl`.
2. Display formatted work item details
3. Show linked artifacts and their status (from the manifest's Artifacts section)
4. List related journal sessions

### When user runs: `/work list`

You MUST:
1. Regenerate + read the registry: `scripts/windex.sh docs/work` folds every item's generated
   `manifest.md` into `docs/work/index.md` (a table of id, title, status, epic, phase, updated).
   Read that generated `index.md` (or glob `ls docs/work/*/` directly). Never hand-edit `index.md`.
2. Display the table (windex already sorts by Last Updated desc)
3. Highlight active work items (In Implementation status)

### When user runs: `/work update <id> --status NEW_STATUS`

You MUST:
1. Resolve `<id>` → `$WD`. Map `NEW_STATUS` to a status key
   (`proposed|researching|requirements|planning|implementation|completed|blocked|on_hold|cancelled`).
2. Append the event and regenerate — never hand-edit the Status line or Change Log:
   ```bash
   scripts/wlog.sh "$WD" status_changed to=<status-key> note="<optional reason>"
   scripts/wrender.sh "$WD"
   ```
   The renderer updates the badge, the change log, and Last Updated.

## The manifest is GENERATED — do not template it by hand

`manifest.md` is a **pure projection** of `$WD/work.jsonl`, produced by `scripts/wrender.sh`. You never
write or template it. The renderer owns every derived field — there is no manifest template to fill in.
For reference, the generated shape is:

```
# Work Item: <title>
<!-- GENERATED by scripts/wrender.sh — DO NOT EDIT BY HAND. -->
**ID** · **Status** · **Created** · **Last Updated** · **Owner** · **Epic** · **Wishlist**
**Epic Phase Done** · **Priority** · **Estimated Effort**
## Artifacts          (folded from artifact_added events)
## Open Relays        (relay_sent/received minus relay_resolved, by direction+slug; synced ✓)
## Upstream Messages  (all relay_received)
## Change Log         (every event in seq order, with its note)
```

The mapping from events to manifest:

| Manifest field / section | Driven by event(s) |
|---|---|
| `**Status**` badge | last `status_changed to=…` (`wrender.sh` owns the emoji vocabulary) |
| `**Created**` / `**Last Updated**` | `created` `ts` / latest event `ts` |
| `**Owner**` / `**Priority**` / `**Estimated Effort**` | `created` + `meta_changed` |
| `**Epic**` / `**Wishlist**` | `created`/`meta_changed` `epic=` / `wishlist=` |
| `**Epic Phase Done**` | last `phase_done` `phase=` |
| Workflow Progress checkboxes | `status_changed` + `phase_done` history |
| Artifacts | `artifact_added` events |
| Open Relays / Upstream Messages | `relay_sent`/`relay_received`/`relay_synced`/`relay_resolved` |
| Change Log | every event, in `seq` order, with its `note` |

> **Never** hand-edit any of the above. To change state, append the matching `wlog.sh` event and run
> `scripts/wrender.sh "$WD"`. The single place LLM narrative enters the log is the `note=`/`body=` field
> on an event — the renderer places it verbatim.

The original prompt is captured on the `created` event; artifact *content* (research, requirements,
issues, plans, `implementation/status.md`, `epic/context.md`, relay bodies) stays markdown prose and is
authored exactly as before — only *state* is structured.

## Status Values

- 🎯 **Proposed**: Work item created, needs research
- 📚 **Researching**: Research in progress
- 📝 **Requirements**: Gathering/documenting requirements
- 🎨 **Planning**: Creating implementation plan
- 🔄 **In Implementation**: Active development
- ✅ **Completed**: Work finished and deployed
- 🔴 **Blocked**: Waiting on dependencies
- ⏸️ **On Hold**: Paused for later
- ❌ **Cancelled**: Will not be implemented

## Work Registry — generated, not hand-maintained

`docs/work/index.md` is **generated**, never hand-edited. Regenerate it with `scripts/windex.sh docs/work`
(or `scripts/windex.sh repos/<repo>/docs/work` for a child repo) — it folds every item's generated
`manifest.md` into a roll-up table (id, title, status, epic, Epic Phase Done, updated), sorted by Last
Updated. There are **no** rows to move on a status change — status lives in each item's `work.jsonl`,
flows into the manifest via `wrender.sh`, and into the index via `windex.sh`.

## Tools Available

- **Bash**: Run `scripts/wlog.sh` (append events), `scripts/wrender.sh` (regenerate manifest),
  `scripts/epic-board.sh` (read-only epic rollup), and `date` (mint ids)
- **Read**: Read generated manifests, `work.jsonl`, and prose artifacts
- **Write**: Create prose artifact content (research/requirements/issues/plans/relay bodies/epic context)
- **Edit**: Edit prose artifacts (NEVER `manifest.md` — that is generated)
- **Glob**: Resolve ids and enumerate the registry
- **Grep**: Search work content (if needed)

## Integration with Other Commands

### Automatic Integration (Primary Workflow)

When `/work "prompt"` is used, it **automatically**:
1. Creates work item folder: `$WD` (`docs/work/<id>/`)
2. Appends a `created` event and renders `$WD/manifest.md` via `scripts/wrender.sh`
3. Creates research folder and document: `$WD/research/0001-*.md`
4. Creates requirements folder and document: `$WD/requirements/0001-*.md`
5. Appends `artifact_added`/`status_changed` events and re-renders (artifact links derived, not hand-written)
6. Returns control to user for review

**All artifacts organized under one folder**: `$WD`

### Additional Research (Manual Trigger)

User runs `/research --work <id> "research topic"` to add more research. The research command MUST:
1. **Find next research number** in `$WD/research/`
2. **Create new research document** - `$WD/research/NNNN-{slug}-research.md`
3. **Append an event** - `scripts/wlog.sh "$WD" artifact_added kind=research path=research/NNNN-{slug}-research.md title="…"` then `scripts/wrender.sh "$WD"` (do NOT hand-edit the Artifacts section)
4. **Run research agents** as needed

### Additional Requirements (Manual Trigger)

User runs `/new_req --work <id> "requirements topic"` to add more requirements. The new_req command MUST:
1. **Find next requirements number** in `$WD/requirements/`
2. **Create new requirements document** - `$WD/requirements/NNNN-{slug}-req.md`
3. **Append an event** - `scripts/wlog.sh "$WD" artifact_added kind=requirements path=requirements/NNNN-{slug}-req.md title="…"` then `scripts/wrender.sh "$WD"`
4. **Run validation agents** as needed

### Additional Issues (Manual Trigger)

User runs `/new_issue --work <id> "issue description"` to track issues. The new_issue command MUST:
1. **Find next issue number** in `$WD/issues/`
2. **Create new issue document** - `$WD/issues/NNNN-{slug}-issue.md`
3. **Append an event** - `scripts/wlog.sh "$WD" artifact_added kind=issue path=issues/NNNN-{slug}-issue.md title="…"` then `scripts/wrender.sh "$WD"`

### Manual Planning Trigger

User manually runs `/planv0 --work <id>` when ready. The planv0 command MUST:
1. **Read Work Manifest** - `$WD/manifest.md`
   - Get context, title, original request
2. **Read ALL Research Documents** - `$WD/research/*.md`
   - Understand problem space from all research
   - Review technology options analyzed
   - Consider architectural approaches explored
3. **Read ALL Requirements Documents** - `$WD/requirements/*.md`
   - Extract functional requirements from all docs
   - Extract non-functional requirements from all docs
   - Review acceptance criteria
   - Understand constraints
4. **Create Implementation Plan** - Based on ALL research + ALL requirements
   - Create `$WD/plans/` folder
   - Master plan: `$WD/plans/master.md`
   - Phase plans: `$WD/plans/phase-N.md`
   - Link back to research and requirements (relative paths)
   - Address all requirements from all documents with traceability
5. **Record state via events** (never hand-edit the manifest):
   ```bash
   scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/master.md title="Master Plan"
   scripts/wlog.sh "$WD" status_changed to=planning
   scripts/wrender.sh "$WD"
   ```

### Manual Implementation Trigger

User manually runs `/implement_plan <plan-path>` where <plan-path> is the path to the master plan file.

**Example**: `/implement_plan docs/work/<id>/plans/master.md`

The implement_plan command MUST:
1. **Read the plan file** provided as parameter
2. **Extract work id** from plan content (should contain `Work Item: <id>`) and resolve to `$WD`
3. **Read work manifest** - `$WD/manifest.md`
4. **Read all plan documents** in `$WD/plans/`
5. **Execute implementation** according to plan
6. **Create implementation status** - `$WD/implementation/status.md` (prose)
7. **Record progress via events** - append `scripts/wlog.sh "$WD" status_changed to=implementation`,
   `scripts/wlog.sh "$WD" artifact_added kind=implementation path=implementation/status.md title="Implementation Status"`,
   and on completion `scripts/wlog.sh "$WD" status_changed to=completed`; run `scripts/wrender.sh "$WD"`
   after each. Never hand-edit the manifest (no --work parameter needed — the id comes from the plan).

### Important Notes

- Research and requirements are **automatic** (triggered by `/work "prompt"`)
- Planning is **manual** (triggered by `/planv0 --work <id>`)
- Implementation is **manual** (triggered by `/implement_plan <plan-path>`)
- User reviews and provides feedback between each phase

## Examples

### Example 1: Automatic Research + Requirements

```bash
# User provides natural language prompt
/work "I want to add OAuth social login with Google and GitHub"

# System automatically (WORK_ID e.g. work-2607010322-oauth-social-login):
# 1. Mints WORK_ID = work-$(date +%y%m%d%H%M)-oauth-social-login; mkdir -p "$WD"/{research,requirements,issues,plans,relays/outbound,relays/inbound}
# 2. wlog.sh "$WD" created title="OAuth Social Login Integration" slug=oauth-social-login … ; wrender.sh "$WD"
# 3. Runs research (explores OAuth patterns etc.)
# 4. Creates "$WD"/research/0001-oauth-social-login-research.md
# 5. wlog.sh "$WD" status_changed to=researching; artifact_added kind=research path=…; status_changed to=requirements; wrender.sh "$WD"
# 6. Creates requirements based on research
# 7. Creates "$WD"/requirements/0001-oauth-requirements.md
# 8. wlog.sh "$WD" artifact_added kind=requirements path=… ; wrender.sh "$WD"  (manifest is GENERATED — never hand-edited)
# 9. Returns to user

# Output:
# ✅ Created work-2607010322-oauth-social-login: OAuth Social Login Integration
# 📋 Original Request: I want to add OAuth social login with Google and GitHub
# 🔍 Starting automatic research and requirements gathering...
# [research happens]
# ✅ Research completed: $WD/research/0001-oauth-social-login-research.md
# ✅ Requirements documented: $WD/requirements/0001-oauth-requirements.md
#
# 📊 Work Item Status: 📝 Requirements (Ready for Planning)
#
# All artifacts are in: $WD
# You can add more research or requirements with:
#   /research --work <id> "Additional research topic"
#   /new_req --work <id> "Additional requirements"
#
# When ready, run: /planv0 --work <id>
```

### Example 2: Adding More Research and Requirements

```bash
# After initial research, user realizes they need more investigation
/research --work <id> "Apple Sign-In integration details"
# Creates: $WD/research/0002-apple-signin-integration-research.md
# Records:  wlog.sh "$WD" artifact_added kind=research path=… ; wrender.sh "$WD"

# Add security-specific requirements
/new_req --work <id> "Security and compliance requirements"
# Creates: $WD/requirements/0002-security-compliance-req.md
# Records:  wlog.sh "$WD" artifact_added kind=requirements path=… ; wrender.sh "$WD"

# Add performance requirements
/new_req --work <id> "Performance requirements"
# Creates: $WD/requirements/0003-performance-req.md
# Records:  wlog.sh "$WD" artifact_added kind=requirements path=… ; wrender.sh "$WD"
```

### Example 3: Review and Plan

```bash
# User reviews all research and requirements, confirms ready to plan

/planv0 --work <id>
# Reads: $WD/manifest.md
# Reads ALL: $WD/research/*.md (all 2 research docs)
# Reads ALL: $WD/requirements/*.md (all 3 requirements docs)
# Creates: $WD/plans/master.md
# Creates: $WD/plans/phase-1.md (if multi-phase)
# Records:  wlog.sh "$WD" artifact_added kind=plan path=plans/master.md … ; status_changed to=planning ; wrender.sh "$WD"
```

### Example 4: View Work Status

```bash
# Show work details
/work show <id>
# → Displays the generated manifest with all artifacts

# List all work
/work list
# → Enumerates docs/work/*/ and shows a table from each manifest

# Update status (appends an event, regenerates the manifest)
/work update <id> --status "Blocked"
# → wlog.sh "$WD" status_changed to=blocked note="…" ; wrender.sh "$WD"
```
