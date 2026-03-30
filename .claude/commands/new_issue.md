# New Issue

Create tickets for bugs, fixes, and small tasks within a work item context.

## Work Item Context (Optional)

The `--work work-NNNN` parameter is **OPTIONAL** but recommended for organized issue tracking.

**When `--work work-NNNN` is provided**:
1. Read work manifest: `docs/work/work-NNNN/manifest.md`
2. Read existing issues in `docs/work/work-NNNN/issues/` to avoid duplication
3. Plan to create issue in `docs/work/work-NNNN/issues/NNNN-*.md`
4. Plan to update work manifest when complete

**When `--work` is NOT provided (Standalone Mode)**:
1. Create issue in `docs/issues/NNNN-*.md`
2. Use global numbering (check all files in `docs/issues/`)
3. No manifest updates
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

4. **Update Work Manifest (If Using Work Item)**:

   **If `--work work-NNNN` was provided**:
   - Read work manifest: `docs/work/work-NNNN/manifest.md`
   - Add issue document to `## Artifacts > ### Issues` section
     - Format: `- [NNNN: {Title}](./issues/NNNN-{slug}-issue.md) ({date})`
   - Add change log entry: `{date}: Added issue NNNN`
   - Save updated manifest

   **If standalone**:
   - Skip manifest updates
   - Issue document is self-contained

5. **Next Steps**:
   - **If work item mode**: Note that manifest was updated with new issue
   - **If standalone**: Note the issue document location
   - Suggest adding more issues with `/new_issue` (with or without --work)
   - Suggest using `/research` if investigation needed
   - Suggest using `/planv0` for complex issues
   - Link to related requirements if this grows in scope

This command focuses specifically on creating well-structured tickets for bugs, fixes, and small features.
