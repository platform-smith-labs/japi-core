# New Issue

Create tickets for bugs, fixes, and small tasks within a work item context.

> **Work-item state is an append-only event log** — register artifacts with `scripts/wlog.sh "$WD" artifact_added ...` then `scripts/wrender.sh "$WD"`; never hand-edit `manifest.md`. See [docs/dev/decisions/append-only-work-event-log.md](../../docs/dev/decisions/append-only-work-event-log.md).

## Resolving `--work` IDs

When the user passes `--work <id>` (e.g., `--work work-0027` or `--work work-2607010322-dark-mode`), the value is a **short reference**, never an index to compute. The directory may be the legacy short form (`work-NNNN`), the legacy slug form (`work-NNNN-MMDDHHMM-slug`), or the event-log form (`work-<YYMMDDHHMM>-<slug>`). Before any file read/write, **resolve the reference to the real directory by glob — never by arithmetic**:

1. **Try exact match** — Glob `docs/work/{arg}/` (or `repos/<repo>/docs/work/{arg}/`). If found, use it.
2. **Else glob with dash suffix** — Glob `docs/work/{arg}-*/` (matches the slug-suffixed forms).
3. **Else glob by slug fragment** — Glob `docs/work/*{arg}*/` for a bare slug reference.
4. **If exactly one match**, use that directory. If zero, error: "Work item {arg} not found." If multiple, error and list matches.

Throughout the rest of this document, `work-NNNN` / `$WD` is shorthand for the resolved work item directory. State for that item lives in `$WD/work.jsonl`; `$WD/manifest.md` is a generated projection of it.

## Work Item Context (Optional)

The `--work work-NNNN` parameter is **OPTIONAL** but recommended for organized issue tracking.

**When `--work work-NNNN` is provided**:
1. Resolve the short ID to the actual directory (see above).
2. Read work manifest: `docs/work/work-NNNN/manifest.md` (a generated view — read for context, never edit it)
3. Read existing issues in `docs/work/work-NNNN/issues/` to avoid duplication
4. Plan to create issue in `docs/work/work-NNNN/issues/NNNN-*.md`
5. Plan to register the artifact in the event log when complete (see "Register the Artifact in the Event Log" below) — do **not** plan to hand-edit the manifest

**When `--work` is NOT provided (Standalone Mode)**:
1. Create issue in `docs/issues/NNNN-*.md`
2. Use local numbering within `docs/issues/` (highest existing NNNN + 1)
3. No event log, no manifest — and do not hand-maintain any global `docs/issues/index.md` registry (it is derivable from the directory)
4. Self-contained issue document

## Initial Response

When this command is invoked, respond with:

```
I'll help you create a new issue ticket. What type of issue are you creating?

1. Bug report - Something isn't working correctly
2. Small feature - Simple enhancement or addition
3. Task - Work item that needs to be done
4. Fix - Known issue that needs resolution

Please describe the issue or provide details about what needs to be addressed.
```

Then wait for the user's input.

## NUMBERING SCHEME

### Format: NNNN-descriptive-name-issue.md

- **NNNN**: Sequential number (0001, 0002, etc.)
- **descriptive-name**: Kebab-case description of the issue
- **issue**: Type suffix

### Auto-Numbering Process:

**With work item (`--work work-NNNN`)**:
1. Use Glob to find files in `docs/work/work-NNNN/issues/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-issue.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits

**Standalone (no `--work`)**:
1. Use Glob to find files in `docs/issues/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-issue.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits

## Issue Structure

**Location**:
- **With work item**: `docs/work/work-NNNN/issues/NNNN-*.md`
- **Standalone**: `docs/issues/NNNN-*.md`

Structure:

```markdown
# [Issue Title] - Issue

**Document ID**: NNNN
**Work Item**: work-NNNN (if applicable)
**Created**: [Current Date]
**Type**: Issue
**Status**: [Open/In Progress/Resolved]

## Problem to Solve

[Clear description of the problem from a user perspective]

## Type

- [ ] Bug
- [ ] Small Feature
- [ ] Task
- [ ] Fix

## Acceptance Criteria

- [ ] [Specific, testable criterion]
- [ ] [Another criterion]
- [ ] [Third criterion]

## Context

[Any background information, constraints, or related work]

## References

- **Work Item**: `work-NNNN`
- **Related Requirements**: `../requirements/NNNN-[name]-req.md`
- **Related Research**: `../research/NNNN-[name]-research.md`
- **Implementation Plan**: [Will be linked when plan is created]
- **Codebase References**: `[file:line]`

## Status

- [ ] Research needed
- [ ] Ready for implementation
- [ ] Implementation started
- [ ] Testing
- [ ] Done

## Notes

[Any additional notes, decisions, or considerations]

---

**Document ID**: NNNN  
**Cross-References**: Auto-updated by workflow commands
```

## Naming Convention

- **Format**: `NNNN-descriptive-name-issue.md` in `docs/work/work-NNNN/issues/`
- **Sequential Numbering**: Auto-increment from highest existing number within the work item
- **Descriptive Names**: Use kebab-case, be concise but clear
- **Type Suffix**: Always `-issue.md`

Examples:

**Work Item work-0001**:
- `docs/work/work-0001/issues/0001-login-mobile-safari-fix-issue.md`
- `docs/work/work-0001/issues/0002-session-timeout-issue.md`
- `docs/work/work-0001/issues/0003-dark-mode-toggle-issue.md`

## Examples

### Bug Report

```markdown
# Login Button Not Working on Mobile Safari

## Problem to Solve

Users on Mobile Safari can't log in - the login button doesn't respond to taps.

## Type

- [x] Bug

## Acceptance Criteria

- [ ] Login button responds to taps on Mobile Safari
- [ ] No regressions on other browsers
- [ ] Touch events work properly

## Context

Issue reported by multiple users. Seems to be specific to iOS Safari.
```

### Small Feature

```markdown
# Add Dark Mode Toggle

## Problem to Solve

Users want to switch between light and dark themes for better usability in different lighting conditions.

## Type

- [x] Small Feature

## Acceptance Criteria

- [ ] Toggle appears in user settings
- [ ] Dark mode applies to all components
- [ ] Preference persists across sessions
```

## Process

1. **Gather Information**:
   - Understand the issue type and scope
   - Get specific details about the problem
   - Identify acceptance criteria

2. **Analyze Issue** (Enhancement - NEW):

   After gathering issue details, route to appropriate specialist for analysis:

   a. **Categorize and analyze by type**:

      - **Bug** → **debugger**:
        ```
        Use Task tool with subagent_type="debugger":

        "Analyze this bug report:

        [Issue description]

        Provide:
        - Likely root cause
        - Related code/components to investigate
        - Suggested investigation steps"
        ```

      - **Performance Issue** → **performance-engineer**:
        ```
        Use Task tool with subagent_type="performance-engineer":

        "Analyze this performance issue:

        [Issue description]

        Provide:
        - Potential bottlenecks
        - Profiling recommendations
        - Quick wins vs. deep optimization"
        ```

      - **Security Issue** → **security-engineer**:
        ```
        Use Task tool with subagent_type="security-engineer":

        "Analyze this security concern:

        [Issue description]

        Assess:
        - Severity and impact
        - Immediate mitigation steps
        - Long-term fix approach"
        ```

      - **UI/UX Issue** → **ui-designer**:
        ```
        Use Task tool with subagent_type="ui-designer":

        "Analyze this UI/UX issue:

        [Issue description]

        Provide:
        - UX impact assessment
        - Design considerations
        - Suggested improvements"
        ```

   b. **Assess complexity** using **codebase-analyzer**:
      ```
      Use Task tool with subagent_type="codebase-analyzer":

      "Assess the scope of changes needed for:

      [Issue description with specialist analysis]

      Estimate:
      - Files/components affected
      - Integration complexity
      - Testing requirements
      - Complexity: Simple (1-2h) / Medium (1-2d) / Complex (1+w) / Needs Investigation"
      ```

   c. **Incorporate analysis** into ticket:
      - Add specialist findings to Context section
      - Add complexity estimate
      - Update acceptance criteria based on insights

3. **Create Issue**:
   - Generate descriptive filename
   - Fill out structured template with analysis
   - **With work item**: Save to `docs/work/work-NNNN/issues/NNNN-{slug}-issue.md`
   - **Standalone**: Save to `docs/issues/NNNN-{slug}-issue.md`

4. **Register the Artifact in the Event Log (If Using Work Item)**:

   **Do NOT hand-edit `manifest.md`.** Work-item state is an append-only event log (`work.jsonl`); the manifest is regenerated from it. With `$WD` = the resolved work item directory and `<rel-path>` = the artifact path *relative to `$WD`* (e.g. `issues/NNNN-{slug}-issue.md`):

   **If `--work work-NNNN` was provided**:
   - Append an `artifact_added` event (note `kind=issue`, singular):
     ```bash
     scripts/wlog.sh "$WD" artifact_added kind=issue path=<rel-path> title="{Title}"
     ```
   - Regenerate the manifest from the log:
     ```bash
     scripts/wrender.sh "$WD"
     ```
   - `wrender.sh` folds the event into the manifest's Artifacts section, Change Log, and Last Updated automatically. Creating an issue does **not** itself imply a status transition, so do not append a `status_changed` event unless the user explicitly moves the item's lifecycle. Never open `manifest.md` to edit any of this by hand.

   **If standalone**:
   - No event log and no manifest — the issue document is self-contained.
   - Do not hand-maintain any global registry/index of `docs/issues/`; it is derivable from the directory listing.

5. **Next Steps**:
   - **If work item mode**: Note that the artifact was registered in the event log and the manifest regenerated
   - **If standalone**: Note the issue document location
   - Suggest adding more issues with `/new_issue` (with or without --work)
   - Suggest using `/research` if investigation needed
   - Suggest using `/planv0` for complex issues
   - Link to related requirements if this grows in scope

This command focuses specifically on creating well-structured tickets for bugs, fixes, and small features.
