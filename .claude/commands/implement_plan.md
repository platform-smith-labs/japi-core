# Implement Plan

You are tasked with implementing an approved technical plan. These plans contain phases with specific changes and success criteria.

## Command Usage

```bash
/implement_plan <path-to-plan-file>
```

**Examples**:
- `/implement_plan docs/work/work-0001/plans/master.md`
- `/implement_plan docs/work/work-0002/plans/phase-1.md`

## Getting Started

When given a plan path:

1. **Read the plan file** completely
   - Check for any existing checkmarks (- [x])
   - **Extract work item reference** if present (look for `Work Item: work-NNNN`)

2. **If work item detected** (e.g., `Work Item: work-0001`):
   - Read work manifest: `docs/work/work-0001/manifest.md`
   - Read ALL research documents: `docs/work/work-0001/research/*.md` (for context)
   - Read ALL requirements documents: `docs/work/work-0001/requirements/*.md` (for acceptance criteria)
   - Read ALL issues: `docs/work/work-0001/issues/*.md` (if any)
   - Read all plan files: `docs/work/work-0001/plans/*.md`
   - Prepare to update manifest with progress

3. **Read all files mentioned** in the plan
   - **Read files fully** - never use limit/offset parameters, you need complete context

4. **Create a todo list** to track your progress

5. **Start implementing** if you understand what needs to be done

If no plan path provided, ask for one.

## Implementation Philosophy

Plans are carefully designed, but reality can be messy. Your job is to:

- Follow the plan's intent while adapting to what you find
- Implement each phase fully before moving to the next
- Verify your work makes sense in the broader codebase context
- Update checkboxes in the plan as you complete sections
- **NEW**: Leverage domain specialists for implementation
- **NEW**: Use quality gates to ensure high standards

When things don't match the plan exactly, think about why and communicate clearly. The plan is your guide, but your judgment matters too.

## Enhanced Implementation Workflow (NEW)

For each phase, follow this enhanced process:

### Phase Start: Domain Specialist Assignment

1. **Detect phase type** by analyzing the work:
   - Backend/API work → **backend-developer**
   - Frontend/UI work → **frontend-developer**
   - Database work → **postgres-pro** or **database-administrator**
   - API design → **api-designer**
   - Full-stack → **fullstack-developer**
   - CLI tools → **cli-developer**
   - Mobile → **mobile-app-developer**
   - Infrastructure → **platform-engineer** or **devops-engineer**

2. **Assign primary specialist** (optional but recommended for complex phases):
   - Use Task tool with appropriate subagent_type
   - Provide phase context and implementation goals
   - Let specialist handle implementation details

3. **Support specialists** as needed during implementation:
   - **debugger** - When hitting issues
   - **refactoring-specialist** - When refactoring existing code
   - **performance-engineer** - For performance-critical sections

### Phase Completion: Quality Gates

After implementing each phase, run quality validation:

1. **code-reviewer** - Code quality check:
   ```
   Use Task tool with subagent_type="code-reviewer":

   "Review the code changes for [phase name]:

   Files changed: [list files]

   Check for:
   - Code quality and best practices
   - Security vulnerabilities
   - Performance issues
   - Maintainability concerns

   Focus on critical issues only."
   ```

2. **Fix critical issues** identified by code-reviewer before proceeding

3. **Run automated checks**:
   - Tests pass: `pnpm test`
   - Types check: `pnpm run typecheck`
   - Linting passes: `pnpm lint`
   - Build succeeds: `pnpm build`

4. **Update plan checkboxes** - Mark phase as complete

5a. **Update Phase Plan File Status**:
   - Change the phase file's `**Status**` field from "Ready for Implementation" to "✅ Completed"
   - Example: In `docs/work/work-NNNN/plans/phase-1.md`, change `**Status**: Ready for Implementation` to `**Status**: ✅ Completed`

5b. **Update Master Plan Phase Status**:
   - In `master.md`, update the inline phase listing status (e.g., `Status: ⏳ Not Started` → `Status: ✅ Completed`)
   - Update the progress tracking table row with status, start date, and completion date
   - If ALL phases are now complete, update master plan's top-level `**Status**` to `✅ Completed`

5c. **Update Work Manifest** (if work item detected):
   - Update `docs/work/work-NNNN/manifest.md`:
     - First phase starts: status `🎨 Planning → 🔄 In Implementation`, check `[x] Implementation` in workflow progress
     - All phases complete: status `🔄 In Implementation → ✅ Completed`, check `[x] Validation` in workflow progress
     - Add change log entry for each phase completion
   - Create/update implementation status: `docs/work/work-NNNN/implementation/status.md`
     - Track phase completion
     - Document decisions made
     - Note any blockers or deviations

5d. **Update Work Index** (if work item detected and ALL phases complete):
   - Update `docs/work/index.md`:
     - Move the work item row from "Active Work Items" to "Completed Work Items"
     - Update status to `✅ Completed`
     - Update Artifacts column to reflect all artifact types (e.g., `R, Req, P, I`)

6. **Proceed to next phase** only if quality gates pass

If you encounter a mismatch:

- STOP and think deeply about why the plan can't be followed
- Present the issue clearly:

  ```
  Issue in Phase [N]:
  Expected: [what the plan says]
  Found: [actual situation]
  Why this matters: [explanation]

  How should I proceed?
  ```

## Verification Approach

After implementing a phase:

- Run the success criteria checks (usually `make check test` covers everything)
- Fix any issues before proceeding
- Update your progress in both the plan and your todos
- Check off completed items in the plan file itself using Edit

Don't let verification interrupt your flow - batch it at natural stopping points.

## If You Get Stuck

When something isn't working as expected:

- First, make sure you've read and understood all the relevant code
- Consider if the codebase has evolved since the plan was written
- Present the mismatch clearly and ask for guidance

Use sub-tasks sparingly - mainly for targeted debugging or exploring unfamiliar territory.

## Resuming Work

If the plan has existing checkmarks:

- Trust that completed work is done
- Pick up from the first unchecked item
- Verify previous work only if something seems off

Remember: You're implementing a solution, not just checking boxes. Keep the end goal in mind and maintain forward momentum.
