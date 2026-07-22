# Work Item Management Command

**Purpose**: Create and manage unified work items that group related research, requirements, plans, and implementation artifacts.

## 🚧 Repo isolation — no cross-repo reads or edits (MANDATORY)

A work item belongs to **one repo**, which in the PlatformSmith product runs in its own container with **no filesystem access to sibling repos**. Enforce that in this command and every sub-agent it spawns:

- **Never** `Read`/`Grep`/`Glob`/`Edit` any file **outside this repo** (another repo's working tree). The work item's world is **this repo only**.
- **Cross-repo knowledge** comes *only* from the local **folded KB** at `docs/kb/peers/<repo>/` (start at `docs/kb/index.md`) — the sole cross-repo research surface. Reading your own `docs/kb/peers/**` is allowed.
- If the KB is unclear on a **system-critical** fact, is a gap / `UNKNOWN`, or is contradicted by observed behavior → emit an A2A **relay** (the live ask-a-peer A2A channel — not a local script). Do **not** relay for routine confirmation. Cross-repo requests to *change* something also go via relay — never by editing the other repo.
- **Cross-repo edits are never allowed.** If a cross-repo read seems unavoidable, **stop and ask the human**. See [docs/dev/decisions/repo-isolation-kb-first-cross-repo.md](../../docs/dev/decisions/repo-isolation-kb-first-cross-repo.md).

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
/work "Natural language prompt"       # Auto-create STANDALONE work + research + requirements
/work --parent-work <parent-id> --parent-project <repo> "prompt"
                                      # Create a CHILD work item under a standalone parent
                                      # (both -- flags present, or both absent)
/work <id>                            # Resume (review-gated: prints the next command, hands back to you)
/work <id> auto                       # Autonomous: EXECUTE one phase, append events, STOP (loop-friendly)
/work show <id>                       # Show work item details
/work list                            # List all work items
/work update <id> --status X          # Update work status (appends status_changed)
```

> **`/work --epic` is REMOVED.** Parenthood is implicit — a standalone item *becomes* a parent the
> moment its first child is created (by `/conduct <parent> scaffold <repo> "…"` or by
> `--parent-work` above). There is no promotion ceremony. Legacy epic-bound items are frozen on the
> old model (see **Conductor-aware work** below).

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

The workflow has three nesting tiers, **each parent optional** — and the middle tier is itself a
work item (the epic *entity* is retired; see `/conduct` +
`docs/dev/decisions/parent-child-work-items-and-conduct.md`):

```
wishlist item     (a deferred idea; may spawn 0..N parents, one per milestone)  ← docs/wishlist/ (monorepo root)
   └─ PARENT work (standalone; cross-repo conductor seat; owns the board)       ← docs/work/ (usually solution)
        └─ CHILD work (single-repo execution; declares parent= at creation)     ← {repo}/docs/work/
```

- A **work item** is **standalone** (`/work "prompt"`) or a **child**
  (`--parent-work <parent-id> --parent-project <repo>` — both flags or neither). Nesting is
  **N-level** (2026-07-06 relaxation): a child may itself parent children (program → sub-effort →
  per-repo strand). Two rules: the `parent=` chain must terminate at a standalone root with **no
  cycles** (validated at create), and a mid-level node obeys the **settling rule** — it may not
  settle its final/validation `phase_done` toward its parent while its own children board is
  incomplete.
- A standalone item **becomes a parent implicitly** when its first child is created. The parent's
  children board is derived by `scripts/conduct-board.sh` from the children's `parent=`
  declarations — never hand-maintained.
- A **wishlist item** may map to **0..N parents** over time (one per milestone — sequenced sibling
  parents, driven by `/conduct <parent> next`).
- **Legacy**: items with an `epic=` link (and everything under `docs/epics/`) stay on the frozen
  `/epic` model — never migrate them.

**Linkage fields** are carried as event metadata (the `created`/`meta_changed` `parent=`,
`parent_project=`, `epic=` (legacy) and `wishlist=` keys), rendered into the manifest header by
`wrender.sh` — never hand-edited:
- work manifest header: `**Parent**` (child items), `**Epic**` (legacy), `**Wishlist**` lines

**Upward status sync is derived, not pushed.** Each work item's state lives in its own `work.jsonl`;
the parent rollup is **folded from child logs** by `scripts/conduct-board.sh` (read-only; legacy
epics: `scripts/epic-board.sh`). There is **no** write-back into parent or wishlist tables. Run
`scripts/conduct-board.sh <parent-id>` (or `/conduct`) at the monorepo root for the rolled-up view.

**Two logs per child, one writer each** (writer partition — see the decision doc): you (the
worker/child side) append **only to `work.jsonl`** via `scripts/wlog.sh`. The sibling file
**`relays.jsonl` is the conductor's delivery log** (`relay_received`/`relay_synced`, written via
`scripts/rlog.sh` by `/conduct sync` only) — **never write to it**. `wrender.sh` folds both into
your manifest's Open Relays / Upstream Messages. See **Conductor-aware work** below.

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
       [parent=<parent-work-id> parent_project=<parent-repo>] [wishlist=<n>] \
       [priority=<P>] [effort=<S|M|L>]
     scripts/wrender.sh "$WD"
     ```
     Omit `parent=`/`parent_project=` for a standalone item. Include **both** (never one) when the
     user passed `--parent-work` + `--parent-project` — and first **validate the parent's ancestry
     chain**: resolve it (glob under `docs/work/`; a parent that lives in **another repo** is resolved
     by the conductor via A2A/DB, never by reading a sibling repo's tree), then walk its
     `parent=` links upward — the chain must terminate at a standalone root, with no cycles and
     without the item being created (N-level nesting is allowed; the parent may itself be a
     child). Error on any violation. `epic=` is legacy-only — never set it on new items.
   - **Always pass `request=`** with the user's original prompt verbatim — `wrender.sh` surfaces it as
     the `## Original Request` section (load-bearing context for git-resume). `wlog.sh` JSON-encodes it
     safely, so quotes/newlines in the prompt are fine. Do **not** hand-write a manifest — `wrender.sh`
     generates it, with a "DO NOT EDIT BY HAND" banner.

4. **Registry is generated — do not hand-maintain an index**
   - There is **no** `docs/work/index.md` row to edit. Regenerate the roll-up with
     `scripts/windex.sh docs/work` — it folds every item's generated `manifest.md` into `index.md`.
     Never hand-edit `index.md`.

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
   - **MUST satisfy provenance Rules A–C** (full rules: `/research` **Provenance & relays**; law:
     [research-provenance-and-relay-first](../../docs/dev/decisions/research-provenance-and-relay-first.md)):
     every cross-repo claim tagged `[CODE <path:line>]` / `[KB@<fold-ref>]` / `[RELAY <slug>]` /
     `[UNKNOWN]`; **KB Vintage** table when `docs/kb/peers/**` was consulted; **Relay Candidates**
     section (parent-bound: relays written to `$WD/relays/outbound/` + `relay_sent` events; standalone:
     drafts stay in the doc)
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
   - Cross-repo claims cited from research **inherit their provenance tags** (Rule E consume side —
     see `/new_req`); a requirement resting on a system-critical `[UNKNOWN]` must say so explicitly
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
   and proceed to step 4. **Reply relays fold back (Rule D)**: when an open inbound relay answers one
   of this item's research `[UNKNOWN]`s, fold the answers into a **new numbered research addendum**
   (`research/NNNN-*.md`, registered via `artifact_added`), upgrading `[UNKNOWN]` → `[RELAY <slug>]`,
   before appending `relay_resolved`. See
   [docs/dev/decisions/research-provenance-and-relay-first.md](../../docs/dev/decisions/research-provenance-and-relay-first.md).
4. Check status (the rendered badge, derived from the last `status_changed`) and continue:
   - **🎯 Proposed** → Start Phase 2 (Research). Use `epic/` and inbound relays if present.
   - **📚 Researching** → Check what research exists in `research/`. If incomplete, continue. If complete, move to Phase 3 (Requirements).
   - **📝 Requirements** → Check what requirements exist. If complete, prompt: "Requirements ready. Run `/planv0 --work <id>` to create implementation plan."
   - **🎨 Planning** → "Planning phase. Run `/planv0 --work <id>` to create or review the plan."
   - **🔄 In Implementation** → "Implementation in progress. Run `/implement_plan $WD/plans/master.md` to continue."
   - **✅ Completed** → "This work item is already completed."
   - **🔴 Blocked** → Display blockers (from the latest `status_changed to=blocked` note), suggest resolution.

**Conductor-aware work (barrier-synchronized conductor model — parent-bound or legacy epic-bound)**:
When the work item has a `**Parent**` link (new model) or an `**Epic**` link (legacy, frozen), this
repo is one strand of a **barrier-synchronized** run. The rules below apply identically to both;
the translation table:

| | Parent-bound (new) | Epic-bound (legacy) |
|---|---|---|
| Conductor | `/conduct <parent-id>` at the monorepo root | `/epic` at the monorepo root |
| Barrier signal you READ | the **latest `barrier_advanced` event in YOUR OWN `$WD/relays.jsonl`** — the conductor PUSHES it there on every `conduct-board.sh --write` (repo isolation: you can NEVER read the parent manifest). It carries `phase` + `state` (`open`\|`held`\|`complete`). No `barrier_advanced` yet ⇒ the conductor has not synced since you became active — treat the barrier as the kickoff phase (`requirements`) and ask the human to run `/conduct sync` | epic manifest's `**Epic Phase**` |
| Delivery + barrier events (`relay_received`/`relay_synced`/`barrier_advanced`) | conductor-owned **`relays.jsonl`** (via `rlog.sh`) — **you never write it** | conductor writes them into `work.jsonl` |
| Your events (everything else, incl. `relay_sent`/`relay_resolved`/`escalated`) | your `work.jsonl` via `wlog.sh` | same |

Honor these rules in EVERY phase (requirements, planning, implementation), not just research. Relay
**messages** are immutable files under direction-named folders; relay **lifecycle** is events.
Resolution is an **event, never a file move/delete**.

1. **Read inbound first.** Read the barrier from the **latest `barrier_advanced` event in your own
   `$WD/relays.jsonl`** (the conductor-pushed barrier — your ONLY barrier source; parent-bound items
   NEVER resolve or read the parent's directory), all `epic/` files, and every **open inbound** relay
   file (`relays/inbound/from-*.md`) whose `relay_received` has no matching `relay_resolved`. Read it
   with, e.g., `jq -rs 'map(select(.type=="barrier_advanced"))|last' "$WD/relays.jsonl"` → `phase` +
   `state`. If `state=held`, do **not** start that phase (only process open inbound relays). No
   `barrier_advanced` yet ⇒ the conductor has not synced since you became active — act only up to the
   phase you have already settled, and ask the human to run `/conduct <parent-id> sync`.
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
   Then tell the user: *"{this-repo} has settled {phase}. Run `scripts/conduct-board.sh <parent-id>`
   (or `/conduct <parent-id> board`; legacy epics: `scripts/epic-board.sh` / `/epic`) at the monorepo
   root — the conductor folds the child logs and advances when all repos settle."* The global barrier
   means no repo proceeds until every repo settles this phase. The `**Epic Phase Done**` line in your
   manifest is **rendered** from your last `phase_done` event — never hand-edit it, and never
   hand-edit the parent's board cells (folded by `scripts/conduct-board.sh`).
4b. **Escalate instead of spinning.** If you exhaust bounded attempts on the same failure (same
   signature ~3×), or genuinely need a human decision, do NOT loop and do NOT just go blocked —
   append the terminal-until-human signal and STOP:
   ```bash
   scripts/wlog.sh "$WD" escalated note="<what failed, the repeated signature, what decision is needed>"
   scripts/wrender.sh "$WD"
   ```
   🚨 Escalated ≠ 🔴 Blocked: *Blocked* = waiting on something expected to resolve (an open relay, a
   dependency) — the conductor keeps you in play. *Escalated* = out of play until a human decides —
   the board excludes you from the barrier, emits no run-command for you, and the run cannot
   complete until a human resumes you (`status_changed to=<phase status>`) or cancels.
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
2. **Barrier-safe for conductor-bound work.** Never advance past the current barrier — parent-bound:
   the latest `barrier_advanced` in your own `relays.jsonl`; legacy epic-bound: the epic's
   `**Epic Phase**`. Do the barrier phase, append `phase_done phase=<target>`, STOP. The conductor
   (`/conduct board`/`/conduct sync`; legacy `/epic …` — at the monorepo root) is the **only** thing
   that opens the barrier — `auto` mode **never** runs sync, never edits the parent/epic manifest,
   never writes its own `relays.jsonl`, and never hand-edits its own `Epic Phase Done` (that line is
   rendered from the `phase_done` event).
3. **Loop-safe / idempotent.** If there is nothing actionable this iteration (at the barrier waiting on
   other repos, blocked, or completed), **report it in one line and STOP without appending any event.**
   Re-running must not create duplicate work or re-do a finished phase.

#### Algorithm

1. **Read state.** Resolve `<id>` → `$WD` and read `$WD/manifest.md` (the generated view; fold detail
   from `$WD/work.jsonl` if needed). Note `Status`, `**Parent**`/`**Epic**`, `**Epic Phase Done**`.
   If `**Parent**: <parent-id> @ <repo>` is present, read the barrier from the **latest
   `barrier_advanced` event in your own `$WD/relays.jsonl`** — the conductor PUSHES it there on every
   `conduct-board.sh --write` (repo isolation forbids reading the parent manifest):
   ```bash
   jq -rs 'map(select(.type=="barrier_advanced"))|last // {} | "\(.phase // "") \(.state // "")"' "$WD/relays.jsonl"
   ```
   giving the barrier `phase` + `state` (`open`|`held`|`complete`). If NO `barrier_advanced` exists yet,
   the conductor has not synced since you became active ⇒ treat the barrier as the kickoff phase
   (`requirements`, `state=open`) and proceed (a brand-new child's first phase never needs a push).
   Legacy `**Epic**` items read the epic manifest's `**Epic Phase**` instead. Read `epic/context.md`
   if present. If `Status` is 🚨 Escalated: output one line ("escalated — awaiting human decision:
   <note>") and **STOP** (no event) — only a human `status_changed` resumes an escalated item.

2. **Open inbound relays are the highest-priority unit of work (epic-bound).** If the manifest's
   **Open Relays** lists any **open inbound** relay (a `relay_received` with no matching
   `relay_resolved` for `direction=inbound`+`slug`), process them per **Epic-aware work** above
   (validate → act → reply relay if needed → append `relay_resolved` + `wrender.sh`), and **STOP** —
   relays are this invocation's one phase-unit. Do not also advance a phase in the same run.
   A reply relay answering a research `[UNKNOWN]` follows **Rule D**: fold the answers into a new
   numbered research addendum (upgrade `[UNKNOWN]` → `[RELAY <slug>]`, register via `artifact_added`)
   before resolving.

3. **Pick the target phase (exactly one).**
   - **Parent-bound:** use the barrier `phase` + `state` read from your own `relays.jsonl` in step 1
     (compare phase ordinals `requirements(1) < planning(2) < implementation(3) < validation(4)`).
     - If barrier `state=complete` → the run is finished. Output `🎉 {repo} — run complete
       (barrier: complete).` and **STOP** (no event).
     - If barrier `state=held` → nobody starts the barrier phase yet (open relays / escalations
       upstream). Output `⏳ {repo} — barrier {phase} HELD; waiting on conductor.` and **STOP** (no
       event). (Open **inbound** relays were already handled in step 2, which STOPs before here.)
     - Else (`state=open`): if `ord(Epic Phase Done) ≥ ord(barrier phase)` → this repo is **at the
       barrier**. Output `✅ {repo} settled @ {Epic Phase Done}; waiting on conductor/other repos.`
       and **STOP** (no event appended). Otherwise **target = the barrier phase**. Never pick a phase
       beyond it.
   - **Epic-bound (legacy):** same ordinal comparison against the epic manifest's `**Epic Phase**`
     (read across to the epic manifest, which legacy epics still permit).
   - **Standalone:** target = the next pending phase implied by `Status` (see mapping below).

4. **Execute the target phase — and only that phase — to completion.** Record via events, then STOP:

   | Target phase | Action (run the command's logic inline) | Events appended (then `wrender.sh "$WD"`) |
   |---|---|---|
   | `requirements` (from 🎯 Proposed / 📚 Researching) | Run **Phase 2 + Phase 3** (research + requirements) from the `/work "prompt"` flow; the research doc MUST satisfy provenance Rules A–C (tags, KB Vintage table, Relay Candidates — parent-bound: write relays + `relay_sent`) | `status_changed to=researching` → `artifact_added kind=research …` → `status_changed to=requirements` → `artifact_added kind=requirements …`; plus any `relay_sent` from Rule C; if epic-bound, `phase_done phase=requirements` |
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
solution root (conductor loop):   /loop /conduct <parent-id> board   # sync relays + recompute barrier
                                  (legacy epics: /loop /epic board <epic-id>)
each repo (worker loop):          /loop /work <id> auto              # execute current barrier phase, stop
```

The worker reads the barrier — parent-bound: the latest `barrier_advanced` event the conductor PUSHED
into its own `relays.jsonl` (by `conduct-board.sh --write`); legacy epic-bound: the epic's
`**Epic Phase**` — and acts only up to it; the conductor folds every repo's `phase_done` events
(→ `Epic Phase Done`) + open relays and opens the barrier when all repos settle with zero open relays.
Neither loop mutates the other's authority — workers own `phase_done` + relay events; the conductor
owns the derived barrier + relay delivery + the `barrier_advanced` push. A worker at the barrier idles
cheaply (step 3 STOP) until the conductor advances the barrier (a new `barrier_advanced` in its own
tree), at which point the next worker iteration does the new phase. **Worst case is one sync-cycle of
latency, never a deadlock** — the conductor cannot write the child's `phase_done` (single-writer), and
now the child never has to reach across repos for the barrier. See
[docs/dev/decisions/conductor-pushes-barrier-into-child-territory.md](../../docs/dev/decisions/conductor-pushes-barrier-into-child-territory.md).

**Exception recap:** `auto` removes the per-phase review gate — each invocation executes one phase and
stops, so the item advances without manual relaying. The default `/work <id>` keeps the review gates.

---

### When user runs: `/work --parent-work <parent-id> --parent-project <repo> "prompt"`

**Create a CHILD work item under a standalone parent.** Both `--` flags must be present, or both
absent (absent = standalone item, the normal flow). This is the child-repo-side equivalent of
`/conduct <parent> scaffold` — use whichever context you're in. (`/work --epic` is **removed**;
under the parent/child model there is no promotion — a standalone item becomes a parent when its
first child is created.)

1. **Resolve + validate the parent** (before creating anything): resolve `<parent-id>` by glob
   under `docs/work/` — a parent that lives in **another repo** is resolved by the conductor via
   A2A/DB, never by reading a sibling repo's tree; then **validate the ancestry chain**: walk the
   parent's own `parent=` links upward — the chain must terminate at a **standalone root**, must
   not revisit any node (**no cycles**), and must not include the item being created. Error on any
   violation. If not found / ambiguous: error with matches. (N-level nesting is allowed — the
   parent may itself be a child.)
2. Mint the child id + scaffold exactly as the standalone Phase 1 flow, with the parent keys on the
   `created` event:
   ```bash
   scripts/wlog.sh "$WD" created title="<title>" slug="$SLUG" kind=work repo=<this-repo> \
     owner=<owner> request="<prompt>" parent=<parent-id> parent_project=<parent-repo-alias>
   scripts/wrender.sh "$WD"
   ```
3. **Inherit scaffold rules.** Walk the ancestry chain root→parent; for each ancestor whose work
   dir contains a **`scaffold-rules.md`**, copy its rules into this child's `context.md` (root's
   first, nearest ancestor's last), noting the source (`inherited from
   <ancestor-id>/scaffold-rules.md`). These rules bind this item's research/implementation (e.g.
   the epistemics rule: prior research is a pointer, never gospel — independent research first,
   then critical match).
4. No parent-side registration is needed — the parent's board **discovers** this child on its next
   fold (`scripts/conduct-board.sh` scans for `parent=` declarations). Optionally proceed with the
   automatic research+requirements phases as in the standalone flow, honoring the parent's barrier.

> **Note**: There is no `/work --sync`. Cross-repo state is never pushed by a child — the parent
> rollup is **derived** from child logs by `scripts/conduct-board.sh` (read-only; legacy epics:
> `scripts/epic-board.sh`). Children write relays + `phase_done` events and STOP; the conductor
> (`/conduct` at the monorepo root) delivers + folds + advances. Any upward wishlist reflection is
> likewise a read-only fold, never a hand-edited table.

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
| `**Status**` badge | last `status_changed to=…` — or 🚨 Escalated if a later `escalated` event exists (`wrender.sh` owns the emoji vocabulary) |
| `**Created**` / `**Last Updated**` | `created` `ts` / latest event `ts` |
| `**Owner**` / `**Priority**` / `**Estimated Effort**` | `created` + `meta_changed` |
| `**Parent**` / `**Epic**` (legacy) / `**Wishlist**` | `created`/`meta_changed` `parent=`+`parent_project=` / `epic=` / `wishlist=` |
| `**Epic Phase Done**` | last `phase_done` `phase=` |
| Children board (parents only, between BOARD anchors) | written by `scripts/conduct-board.sh --write`; `wrender.sh` preserves it verbatim |
| Workflow Progress checkboxes | `status_changed` + `phase_done` history |
| Artifacts | `artifact_added` events |
| Open Relays / Upstream Messages | `relay_sent`/`relay_resolved` (your `work.jsonl`) **+** `relay_received`/`relay_synced` (conductor's `relays.jsonl`) — folded together |
| Change Log | every `work.jsonl` event, in `seq` order, with its `note` (relays.jsonl has its own seq space and is not in the Change Log) |

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
- 🔴 **Blocked**: Waiting on dependencies **expected to resolve** (open relay, upstream work) — the
  conductor keeps the item in play
- 🚨 **Escalated**: Bounded attempts exhausted / human decision required — **out of play until a
  human acts** (rendered from an `escalated` event; resume with a later `status_changed`). The
  conductor excludes it from the barrier and the run cannot complete while any child is escalated
- ⏸️ **On Hold**: Paused for later
- ❌ **Cancelled**: Will not be implemented

## Work Registry — generated, not hand-maintained

`docs/work/index.md` is **generated**, never hand-edited. Regenerate it with `scripts/windex.sh docs/work`
— it folds every item's generated
`manifest.md` into a roll-up table (id, title, status, epic, Epic Phase Done, updated), sorted by Last
Updated. There are **no** rows to move on a status change — status lives in each item's `work.jsonl`,
flows into the manifest via `wrender.sh`, and into the index via `windex.sh`.

## Tools Available

- **Bash**: Run `scripts/wlog.sh` (append events — your ONLY writer; never touch `relays.jsonl`,
  that's the conductor's via `scripts/rlog.sh`), `scripts/wrender.sh` (regenerate manifest),
  `scripts/conduct-board.sh` (read-only parent rollup; legacy: `scripts/epic-board.sh`), and
  `date` (mint ids)
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
