# Cross-Repo Epic Command

**Purpose**: Coordinate multi-repo features from the parent directory. Creates "epics" that track work items across child repos, with file-based relay for cross-repo communication.

**This command operates on monorepo-root artifacts** (`docs/epics/`, `docs/wishlist/`). It is
normally run from the monorepo root, but is repo-context-free — if invoked from a child repo it
resolves the root via **Monorepo Root Resolution** (below). Child repos can also use `/work --epic`
(promote) and `/work --sync` (sync up) for the work→epic side.

## Command Usage

```bash
/epic {repo} "prompt"                   # Create new epic with primary repo
/epic sync [epic-NNNN]                  # Full cross-repo sync for all tracked repos
/epic status [epic-NNNN]                # Dashboard of all repos' status
/epic show [epic-NNNN]                  # Detailed epic manifest view
/epic list                              # List all epics
/epic next [epic-NNNN]                  # Surface next sub-epic from Sub-Epic Roadmap
/epic update-sub [epic-NNNN] {phase} {status}  # Mark sub-epic row status (e.g. V1.1 Completed)
/epic next-milestone [epic-NNNN | wishlist NNNN]  # After a wishlist-linked epic completes, scaffold the next milestone's epic
```

> **Two "next" commands, two roadmap sources** — keep them distinct:
> - `/epic next` reads a **parent epic's** own `## Sub-Epic Roadmap` (epic-of-epics).
> - `/epic next-milestone` reads a **wishlist item's** milestone roadmap / `## Tracking` table
>   (wishlist → epic), and scaffolds the next milestone's epic. Use this for items like
>   wishlist 0003 that are delivered one epic per milestone (M0→Mn).

## ID Format

Epic IDs follow the format: **`epic-{NNNN}-{MMDDHHMM}-{short-title}`**

- **`NNNN`** — 4-digit zero-padded sequence number (e.g. `0003`).
- **`MMDDHHMM`** — current local time: 2-digit month, day, hour (24h), minute, no separators (e.g. `04251432` for April 25, 14:32).
- **`short-title`** — kebab-case slug derived from the prompt: lowercase, hyphen-separated, 2–5 words, ≤30 chars (e.g. `oauth-social-login`).

**Full example**: `epic-0003-04251432-oauth-social-login`

> ### ⚠️ MANDATORY — Read before generating any ID
>
> **Use this format for ALL new epic and work IDs, unconditionally.** Do not use any other format.
>
> You may observe that existing items in `docs/epics/` and `docs/work/` use a shorter legacy format (`epic-NNNN`, `work-NNNN`). **This is expected. Legacy items are not migrated.** New items still use the new format.
>
> **Do NOT reason as follows** (this is the documented failure mode this rule guards against):
> - ❌ "All existing epics use `epic-NNNN`, so I'll match for consistency."
> - ❌ "The legacy format is simpler, I'll use it."
> - ❌ "Only one example uses the new format, the rest are legacy — I'll match the majority."
>
> The new format is **required** to prevent directory-path conflicts when multiple developers create epics on parallel branches. Consistency with legacy items is **not** a valid reason to skip it.
>
> **Both formats coexist**: `docs/epics/epic-0049/` (legacy) and `docs/epics/epic-0050-05071523-runtime-sessions-count/` (new) live side by side. Never rename or migrate legacy items unless explicitly asked.

**Why this format**: Multiple developers can create epics on independent branches without directory-path conflicts. Even if the `NNNN` sequence collides between branches, the timestamp + slug components keep the full IDs (and therefore directory paths) unique on merge. Throughout this document, `epic-NNNN` is used as shorthand for the full generated ID.

Work IDs created by `/epic` follow the same format: **`work-{NNNN}-{MMDDHHMM}-{short-title}`**.

## Resolving Existing IDs

When a subcommand takes an `[epic-NNNN]` argument (`/epic show`, `/epic sync`, `/epic status`, `/epic next`, `/epic update-sub`), the user typically passes the **short ID** (e.g., `epic-0050`). The actual directory may be either legacy `epic-NNNN/` or new-format `epic-NNNN-MMDDHHMM-slug/`. Resolve before reading/writing:

1. **Try exact match** — `docs/epics/{arg}/manifest.md`. If found, use `docs/epics/{arg}/`.
2. **Else glob with dash suffix** — `docs/epics/{arg}-*/manifest.md` (matches new format).
3. **If exactly one match**, use that directory. If zero, error: "Epic {arg} not found." If multiple, error and list matches.

The same rule applies to **work IDs** referenced from inside the epic's Tracked Repos table — when scanning `{repo}/docs/work/work-NNNN*/manifest.md`, accept both legacy and new-format directories.

Throughout this document, `epic-NNNN` and `work-NNNN` are shorthand for the resolved directory names.

## Behavior

### When user runs: `/epic {repo} "prompt"`

Create a new epic with the specified repo as the primary target.

**Examples**:
- `/epic backend "add audit logging across all services"`
- `/epic worker "implement heartbeat-based liveness tracking"`
- `/epic ui "add token management UI"`

Repo names are resolved via the `## Repo Aliases` table in `CLAUDE.md`. Both aliases and full directory names are accepted. See **Repo Name Resolution** below.

You MUST execute this workflow:

#### Step 1: Determine IDs

Both IDs follow the conflict-resistant format defined in **ID Format** above: `epic-{NNNN}-{MMDDHHMM}-{short-title}` and `work-{NNNN}-{MMDDHHMM}-{short-title}`.

1. **Next Epic ID** — `epic-{NNNN}-{MMDDHHMM}-{short-title}`:
   - **NNNN (sequence)**: Use Glob to find `docs/epics/epic-*/manifest.md`. From each directory name, parse the 4 digits immediately after `epic-`. Take the highest, increment by 1, zero-pad to 4 digits. If no epics exist, start with `0001`.
   - **MMDDHHMM (timestamp)**: Use Bash to run `date +%m%d%H%M` once and reuse the value for both IDs. This keeps the epic and its primary work item time-aligned.
   - **short-title (slug)**: Generate a kebab-case slug from the user's prompt — lowercase, hyphen-separated, 2–5 distinctive words, ≤30 characters. Strip stopwords (e.g. "the", "a", "for"); keep nouns/verbs. Example: `"add audit logging across all services"` → `audit-logging`.
   - **Combine** as `epic-{NNNN}-{MMDDHHMM}-{short-title}`.

2. **Next Work ID in target repo** — `work-{NNNN}-{MMDDHHMM}-{short-title}`:
   - **NNNN**: Use Glob to find `{repo}/docs/work/work-*/manifest.md`. Parse the 4 digits after `work-`, take highest, increment, zero-pad. If none, start with `0001`.
   - **MMDDHHMM**: Reuse the same timestamp captured for the epic.
   - **short-title**: Reuse the epic's slug (or, if this repo's contribution is meaningfully narrower, generate a more specific slug from the same prompt).
   - **Combine** as `work-{NNNN}-{MMDDHHMM}-{short-title}`.

#### Step 2: Create Epic at Parent Level

1. Create directory: `docs/epics/epic-NNNN/`

2. Create `docs/epics/epic-NNNN/manifest.md` using the Epic Manifest Template (see below):
   - Status: Active
   - Primary Repo: {repo}
   - Original Request: {user's prompt}
   - Tracked Repos table: one row for {repo} with work-MMMM, Phase: 🎯 Proposed

3. Create or update `docs/epics/index.md` using the Epic Index Template (see below)

4. **Wishlist linkage (REQUIRED when this epic implements a `docs/wishlist/` item).**
   This epic originates from a wishlist item when the user names a wishlist number, the prompt
   references `docs/wishlist/NNNN_*`, or you scaffolded it from one (see **Wishlist Linkage**
   section below for detection). When it applies, the link MUST be **bidirectional** — the
   wishlist→epic direction matters as much as epic→wishlist:
   - **a.** Set the epic manifest's `**Wishlist**: NNNN — milestone Mx` header field (`Mx` only
     if the wishlist item has a multi-milestone roadmap; otherwise just `NNNN`).
   - **b.** In `docs/wishlist/NNNN_slug/README.md`, add/append a row to its
     `## Tracking (epics / work items)` section linking this epic + the primary work item +
     status. Create that section from the wishlist skill's template if it does not exist yet.
   - **c.** In `docs/wishlist/README.md`, move the item's row from **Open** to **Picked up**
     (Epic/Work ID = this epic). For a multi-milestone item, note the milestone and that the
     others are pending; if the item is already under **Picked up** (an earlier milestone
     scaffolded a prior epic), append this milestone's epic to its Epic/Work ID cell instead of
     duplicating the row.
   If no wishlist item is involved, omit the `**Wishlist**:` field and skip this step.

#### Step 3: Create Work Item in Target Repo

1. Create directory: `{repo}/docs/work/work-MMMM/`
2. Create subdirectories: `research/`, `requirements/`, `plans/`, `epic/`, `upstream/`

3. Create `{repo}/docs/work/work-MMMM/manifest.md`:
   - Use the standard work manifest template from `/work` command
   - Add `**Epic**: epic-NNNN` in the header (after Owner line)
   - If the epic has a `**Wishlist**:` field, carry the same `**Wishlist**: NNNN — milestone Mx`
     line into the work manifest header (right after the `**Epic**:` line)
   - Status: 🎯 Proposed
   - Original Request: {user's prompt}
   - Include the `## Upstream Messages` section (empty)

4. Create `{repo}/docs/work/work-MMMM/epic/context.md`:
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
      create a file in `upstream/` for each target repo:
      - Filename: `to-{target-repo}--{descriptive-slug}.md`
      - Content: What the target repo needs to do, any interface contracts (API endpoints,
        message formats, DB schemas), and why.

   2. **Check for incoming messages**: Before starting research/planning, read all
      `upstream/from-*` files for context from other repos.

   3. **After writing upstream files**: Tell the user to run `/work --sync epic-NNNN`
      in the target repo to deliver the messages.
   ```

5. Update `{repo}/docs/work/index.md`:
   - Read existing index, add new work item entry
   - If index doesn't exist, create it using the standard work index template

#### Step 4: Output

```
Created epic-NNNN: {Title}
  Primary repo: {repo} → work-MMMM

Next: Open a Claude session in {repo}/ and run:
  /work work-MMMM
```

---

### When user runs: `/epic sync [epic-NNNN]`

Full cross-repo synchronization from the parent directory. If no epic ID is provided, sync the most recently updated active epic.

#### Step 1: Load Epic

1. Read `docs/epics/epic-NNNN/manifest.md`
2. Extract all tracked repos and their linked work item IDs
3. If epic not found: "Epic epic-NNNN not found. Run `/epic list` to see available epics."

#### Step 2: Scan Each Tracked Repo

For each repo in the Tracked Repos table:

1. **Read work item status**: Read `{repo}/docs/work/work-MMMM/manifest.md`
   - Extract current status, phase, latest change log entry

2. **Scan for outgoing relay files**: Glob `{repo}/docs/work/work-MMMM/upstream/to-*`
   - For each `to-{target}--{slug}.md` found:

     **If target repo is NOT yet tracked in the epic**:
     - Determine next work ID in the target repo
     - Create `{target}/docs/work/work-PPPP/manifest.md` (Proposed, Epic: epic-NNNN)
     - Create `{target}/docs/work/work-PPPP/epic/context.md` with:
       - Epic context (title, original request)
       - Summary of what other tracked repos are doing
       - Content from the relay file
     - Create `{target}/docs/work/work-PPPP/upstream/from-{source}--{slug}.md` (copy relay content)
     - Update `{target}/docs/work/index.md`
     - Add target repo to epic manifest's Tracked Repos table

     **If target repo IS tracked and has a work item**:
     - Copy to `{target}/docs/work/work-PPPP/upstream/from-{source}--{slug}.md`
     - Append to target's manifest `## Upstream Messages`:
       ```
       - [{YYYY-MM-DD}] from {source}: [{slug}](./upstream/from-{source}--{slug}.md)
       ```

     **In both cases**:
     - Delete the `to-` file from source repo (it's been delivered)
     - Add entry to epic's Relay Log table

#### Step 3: Update Epic Manifest

- Update each repo's Phase and Status in the Tracked Repos table
- Update Last Synced timestamp
- Add change log entries for all actions taken
- **Reflect status to the wishlist (if the epic has a `**Wishlist**:` field).** Update the
  corresponding row in `docs/wishlist/NNNN_slug/README.md`'s `## Tracking (epics / work items)`
  section to the epic's current status (e.g. ✅ done/GO, 🔄 in progress, 🔴 blocked). When the
  epic reaches a terminal state, ensure the item's registry row in `docs/wishlist/README.md` is
  under **Picked up** with this epic recorded. Keep the wishlist back-link current — it is the
  single place a future session looks to see which milestones of a wishlist item are done.

#### Step 4: Present Summary

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

### When user runs: `/epic status [epic-NNNN]`

Quick dashboard. If no epic ID, show the most recently updated active epic.

1. Read `docs/epics/epic-NNNN/manifest.md`
2. For each tracked repo, read the linked work item's manifest to get current status
3. Scan for pending relay files (`upstream/to-*`) across all tracked repos
4. Present dashboard:

```
Epic epic-NNNN: {Title}
Status: Active  |  Repos: {N}  |  Last Synced: {date}

| Repo         | Work Item  | Phase          | Status            |
|-------------|-----------|----------------|-------------------|
| repo-a       | work-MMMM | 📝 Requirements | Done              |
| repo-b       | work-PPPP | 📚 Researching  | In progress       |
| repo-c       | work-QQQQ | 🎯 Proposed     | Ready to start    |

Pending Relays: {N}
{list of to-* files waiting to be delivered}

Recommended:
  1. {highest priority action}
  2. {next action}
```

---

### When user runs: `/epic show [epic-NNNN]`

Detailed view:
1. Read and display the full `docs/epics/epic-NNNN/manifest.md`
2. Include the full Relay Log and Change Log

---

### When user runs: `/epic list`

1. Read `docs/epics/index.md`
2. If it doesn't exist, create it and report "No epics yet."
3. Display the table of all epics

---

### When user runs: `/epic next [epic-NNNN]`

**Surface the next sub-epic to start within a parent epic's roadmap.** Use this for parent epics that spawn a sequenced set of child sub-epics (typically a vision/research epic with a phased V1/V2/V3 implementation plan).

If no epic ID is provided, use the most recently updated active epic.

#### Step 1: Load Epic + Roadmap

1. Read `docs/epics/epic-NNNN/manifest.md`
2. Look for a `## Sub-Epic Roadmap` section. If absent → "Epic epic-NNNN has no Sub-Epic Roadmap. This command only applies to parent epics that orchestrate a sequence of child sub-epics."
3. Parse the roadmap table. Expected columns: `#`, `Phase`, `Sub-Epic`, `Repos`, `Effort`, `Status`, `Sub-Epic ID`.

#### Step 2: Pick the Next Row

Scan rows in order. The "next" row is the **first row** matching one of:
- Status = `🎯 Pending scaffold` — sub-epic not yet created
- Status = `🔄 Active` — sub-epic created, work in progress (resume)
- Status = `🔴 Blocked` — surface for user attention

Skip rows where Status = `✅ Completed`, `❌ Cancelled`, or `⏸️ On Hold`.

If all rows are Completed → "All sub-epics complete for epic-NNNN. Roadmap exhausted."

#### Step 3: Output Action

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

(The `--parent` / `--epic-from-parent` flags are documented in the parent-epic linkage section below; if the harness doesn't yet support them, the manual form is `/epic {repo} "..."` followed by editing the new epic's `Related to:` to point at the parent and adding the new sub-epic ID into the parent's roadmap row.)

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

### When user runs: `/epic update-sub [epic-NNNN] {Phase} {Status}`

**Update a single row of a parent epic's Sub-Epic Roadmap.** Used to flip status as sub-epics scaffold, activate, or complete.

If `epic-NNNN` is omitted, use the most recently updated active epic.

#### Step 1: Validate

1. Read `docs/epics/epic-NNNN/manifest.md`
2. Find row in `## Sub-Epic Roadmap` matching `Phase` (e.g. `V1.1`, `V1.2`)
3. If row not found → "Phase {Phase} not in epic-NNNN's Sub-Epic Roadmap. Run `/epic show epic-NNNN`."
4. Validate `Status` is one of: `🎯 Pending scaffold`, `🔄 Active`, `🔴 Blocked`, `⏸️ On Hold`, `✅ Completed`, `❌ Cancelled`.

#### Step 2: Update

1. Set the row's `Status` column to the new value
2. If a sub-epic ID is provided as a 4th argument (`/epic update-sub epic-NNNN V1.1 Active epic-0027-04251432-token-rotation`), set the `Sub-Epic ID` column to it
3. Append change-log entry: `- {YYYY-MM-DD}: Sub-Epic Roadmap row {Phase} → {Status}{ (linked to epic-PPPP)}`
4. Update `Last Updated` timestamp

#### Step 3: Output

```
Updated epic-NNNN Sub-Epic Roadmap:
  {Phase} → {Status}{ · Sub-Epic ID: epic-PPPP if provided}

Run /epic next epic-NNNN to see what's up next.
```

---

### When user runs: `/epic next-milestone [epic-NNNN | wishlist NNNN]`

**Scaffold the next milestone's epic for a wishlist item, after the current milestone's epic
completes.** This is the wishlist→epic "kick off the next one" command. It is how a
multi-milestone wishlist item (e.g. wishlist 0003: M0→M4) advances one epic at a time, keeping
the back-link current at every step.

#### Step 1: Resolve the wishlist item + current milestone

- **Arg is `epic-NNNN`** (or omitted → most recently updated **wishlist-linked** epic): read the
  epic manifest's `**Wishlist**: NNNN — milestone Mx` field. If the epic has no `**Wishlist**:`
  field → "Epic epic-NNNN is not linked to a wishlist item; `/epic next-milestone` only applies to
  wishlist-derived epics."
- **Arg is `wishlist NNNN`**: use that item directly; the "current milestone" is the latest one
  with a scaffolded epic in its `## Tracking` table.
- Resolve the wishlist item dir at the monorepo root: `docs/wishlist/NNNN_slug/` (see **Monorepo
  Root Resolution**). Read its `README.md` `## Tracking (epics / work items)` table and, if present,
  its `roadmap.md` for the full milestone sequence (M0→Mn) + each milestone's primary repo.

#### Step 2: Guard + pick the next milestone

1. **Completion guard**: confirm the current milestone's epic is **Completed** (its Tracking-row
   status is ✅, or its epic manifest Status is Completed). If not → "Current milestone Mx
   (epic-NNNN) is not yet complete. Finish it (or pass `--force`) before scaffolding the next."
2. **Next milestone** = the first milestone after Mx in the roadmap whose Tracking row is absent or
   not yet scaffolded. If none remain → "All milestones of wishlist NNNN are scaffolded/complete —
   roadmap exhausted." and stop.
3. Determine the next milestone's **primary repo** from the roadmap row (e.g. roadmap.md's
   `[repo]` tag). If ambiguous, ask the user which repo is primary.

#### Step 3: Scaffold the next epic (reuse `/epic {repo} "prompt"`)

Run the **Step 1–3 creation flow** of `/epic {primary-repo} "{next-milestone title}"` with:
- the new epic manifest's `**Wishlist**: NNNN — milestone M(x+1)` field set;
- the **wishlist linkage** (Step 2.4) applied — add a `## Tracking` row for the new epic
  (status 🔄 Active) and keep the registry row under **Picked up**;
- the new epic's `## Original Request` seeded from the wishlist item's milestone description.

#### Step 4: Output

```
Advanced wishlist NNNN: Mx (epic-AAAA ✅) → M(x+1) (epic-BBBB, new)

Created epic-BBBB: {next-milestone title}
  Primary repo: {repo} → work-MMMM
  Wishlist back-link updated (Tracking row + Picked up registry).

Next: open a session in {repo}/ and run:
  /work work-MMMM
```

---

## Monorepo Root Resolution

`/epic` operates on monorepo-root artifacts (`docs/epics/`, `docs/wishlist/`). It is normally run
from the monorepo root (the `solution/` parent), but must be **free of repo context** — invocable
from any child repo too. Resolve the root as:

1. If `./docs/epics/` exists in the CWD → the CWD is the monorepo root; use `.`.
2. Else if `../docs/epics/` exists → CWD is a child repo; the monorepo root is `..`. Operate there
   (and child-repo work items are at `{root}/repos/{repo}/docs/work/` — or via the `## Repo
   Aliases` table in the root `CLAUDE.md`).
3. Else → error: "Could not locate the monorepo root (no `docs/epics/` here or in the parent)."

All `docs/epics/...` and `docs/wishlist/...` paths in this document are relative to the resolved
monorepo root. Repo names are always resolved via the root `CLAUDE.md` `## Repo Aliases` table — never hardcoded.

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

```markdown
# Epic: epic-NNNN — {Title}

**Status**: Active
**Created**: {YYYY-MM-DD}
**Last Updated**: {YYYY-MM-DD}
**Primary Repo**: {repo-name}
**Wishlist**: {NNNN — milestone Mx if this epic implements a docs/wishlist/ item; omit this line otherwise}
**Last Synced**: Never

## Original Request

{User's original prompt — exactly as provided}

## Tracked Repos

| Repo | Work Item | Phase | Status | Added |
|------|-----------|-------|--------|-------|
| {repo} | work-MMMM | 🎯 Proposed | Not started | {YYYY-MM-DD} |

## Relay Log

| Date | From | To | Slug | Action |
|------|------|----|------|--------|

## Change Log

- {YYYY-MM-DD}: Epic created, primary repo: {repo} (work-MMMM)
```

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

## Repo Name Resolution

When the user provides a repo name (short alias or full directory name):

1. **Read `CLAUDE.md`** in the current directory (or parent directory if running from a child repo)
2. **Look for a `## Repo Aliases` section** with a table mapping aliases to directory names
3. If the provided name matches an alias → use the mapped directory name
4. If the provided name matches a full directory name directly → use it as-is
5. If no `## Repo Aliases` section exists in `CLAUDE.md`:
   - Error: "No `## Repo Aliases` section found in CLAUDE.md. Add a Repo Aliases table mapping short names to directory names."
6. If the name doesn't match any alias or directory:
   - Error: "Repo '{name}' not found. Check the `## Repo Aliases` table in CLAUDE.md or verify the directory exists."

Always validate that the resolved directory exists: `./{resolved-name}/` (from parent) or `../{resolved-name}/` (from child repo).

## Edge Cases

### No docs/epics/ directory
Create it. Also create `docs/` if needed.

### Target repo has no docs/work/
Create `docs/work/` and `docs/work/index.md` in the target repo.

### Epic ID not provided for sync/status
Use the most recently updated active epic (by Last Updated date in index).

### Relay to self
If a `to-{this-same-repo}--*.md` file is found, ignore it (a repo doesn't relay to itself).

## Wishlist Linkage

Many epics are the act of **picking up** a `docs/wishlist/NNNN_slug/` item (see the `wishlist`
skill). The wishlist defines a linkage protocol — *"when an item is picked up, scaffold it, link
the new ID in its folder's README, and move its row to Picked up"* — and `/epic` is responsible
for executing the **wishlist side** of that link so it is never left stale.

**Detecting a wishlist origin** (any one of):
- the user passes a wishlist number/path (e.g. "scaffold wishlist 0003", "from docs/wishlist/0003_*");
- the epic prompt is quoting/derived from a wishlist item's README;
- you are scaffolding a milestone of a multi-milestone wishlist roadmap.
When unsure whether an epic maps to a wishlist item, ask the user rather than guessing.

**The bidirectional contract** (maintained by `/epic` create — Step 2.4 — and `/epic sync` — Step 3):
- Epic → wishlist: the epic manifest carries `**Wishlist**: NNNN — milestone Mx`.
- Wishlist → epic: the wishlist item's `## Tracking (epics / work items)` table has a row for this
  epic + its work item(s) + status, and the registry (`docs/wishlist/README.md`) lists the item
  under **Picked up** with the epic recorded.
- The work item inherits the `**Wishlist**:` field from its epic (Step 3).

**Multi-milestone items**: one wishlist item may map to several epics (one per milestone, M0→Mn).
The item stays a single registry row under **Picked up**; its `## Tracking` table grows one row
per milestone/epic. Do **not** create a new wishlist item per milestone.

## Integration with Other Commands

- `/work work-NNNN` — Resume a work item created by `/epic` (in child repo)
- `/work --epic work-NNNN` — Promote an existing work item to an epic (in child repo)
- `/work --sync epic-NNNN` — Sync cross-repo state from a child repo (preferred over `/epic sync`)
- `wishlist` skill — Captures deferred items into `docs/wishlist/`; `/epic` picks them up and
  maintains the back-link (see **Wishlist Linkage** above).
