# Cross-Repo Epic Command

**Purpose**: Coordinate multi-repo features from the parent directory. Creates "epics" that track work items across child repos, with file-based relay for cross-repo communication.

**This command is designed for the parent directory.** Child repos use `/work --epic` and `/work --sync` instead.

## Command Usage

```bash
/epic {repo} "prompt"                   # Create new epic with primary repo
/epic sync [epic-NNNN]                  # Full cross-repo sync for all tracked repos
/epic status [epic-NNNN]                # Dashboard of all repos' status
/epic show [epic-NNNN]                  # Detailed epic manifest view
/epic list                              # List all epics
/epic next [epic-NNNN]                  # Surface next sub-epic from Sub-Epic Roadmap
/epic update-sub [epic-NNNN] {phase} {status}  # Mark sub-epic row status (e.g. V1.1 Completed)
```

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

#### Step 3: Create Work Item in Target Repo

1. Create directory: `{repo}/docs/work/work-MMMM/`
2. Create subdirectories: `research/`, `requirements/`, `plans/`, `epic/`, `upstream/`

3. Create `{repo}/docs/work/work-MMMM/manifest.md`:
   - Use the standard work manifest template from `/work` command
   - Add `**Epic**: epic-NNNN` in the header (after Owner line)
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

## Integration with Other Commands

- `/work work-NNNN` — Resume a work item created by `/epic` (in child repo)
- `/work --epic work-NNNN` — Promote an existing work item to an epic (in child repo)
- `/work --sync epic-NNNN` — Sync cross-repo state from a child repo (preferred over `/epic sync`)
