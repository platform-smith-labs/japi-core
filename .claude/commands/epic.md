# Cross-Repo Epic Command

**Purpose**: Coordinate multi-repo features from the parent directory. Creates "epics" that track work items across child repos, with file-based relay for cross-repo communication.

**This command is designed for the parent directory.** Child repos use `/work --epic` and `/work --sync` instead.

## Command Usage

```bash
/epic {repo} "prompt"                   # Create new epic with primary repo
/epic sync [epic-NNNN]                  # Full cross-repo sync for all tracked repos
/epic status [epic-NNNN]               # Dashboard of all repos' status
/epic show [epic-NNNN]                 # Detailed epic manifest view
/epic list                              # List all epics
```

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

1. **Next Epic ID**: Use Glob to find `docs/epics/epic-*/manifest.md`. Extract highest NNNN, increment by 1. Format as `epic-NNNN` (zero-padded to 4 digits). If no epics exist, start with `epic-0001`.

2. **Next Work ID in target repo**: Use Glob to find `{repo}/docs/work/work-*/manifest.md`. Extract highest NNNN, increment by 1. If no work items exist, start with `work-0001`.

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
| epic-0001 | {Title} | {repo} | {N} | Active | {date} |

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
