# Implement Plan

You are tasked with implementing an approved technical plan. These plans contain phases with specific changes and success criteria.

> **Work-item state is an append-only event log.** Never hand-edit `manifest.md` — it is generated. Record work-item state by appending events with `scripts/wlog.sh "$WD" …` and regenerating the manifest with `scripts/wrender.sh "$WD"`. Plan checkboxes (`plans/phase-N.md`, `plans/master.md`) remain plan content and stay the authoring surface for plan progress. See [docs/dev/decisions/append-only-work-event-log.md](../../docs/dev/decisions/append-only-work-event-log.md).
>
> Throughout this command, `"$WD"` is the work item directory extracted from the plan path (e.g. `docs/work/work-NNNN-MMDDHHMM-slug/`).

## Command Usage

```bash
/implement_plan <path-to-plan-file>
```

**Examples**:
- `/implement_plan docs/work/work-0001/plans/master.md` (legacy form)
- `/implement_plan docs/work/work-0027-05071523-runtime-sessions-count/plans/phase-1.md` (new form)

The plan path may reference a work item in either the legacy form (`docs/work/work-NNNN/`) or the new conflict-resistant form (`docs/work/work-NNNN-MMDDHHMM-slug/`). The path is used as-is — no ID resolution is needed since the user provides the full directory path.

In the prose below, `work-NNNN` is shorthand for the work item directory name extracted from the plan path (use it verbatim when constructing other paths under that directory).

## Getting Started

When given a plan path:

1. **Read the plan file** completely
   - Check for any existing checkmarks (- [x])
   - **Extract work item reference** if present (look for `Work Item: work-NNNN` — note that the value may be the short ID, the full new-format ID, or absent)

2. **If work item detected** (use the directory name extracted from the plan path — this is `"$WD"`):
   - Load **just-in-time**, not everything up front. Read the plan you were given (`master.md` or a `phase-N.md`) and only the artifacts the **current phase** references.
   - When a phase cites a specific research/requirements/issue document, read **that** document then — not the whole `research/`, `requirements/`, or `issues/` folder.
   - Do **not** read `manifest.md` to drive state — it is a generated projection of `work.jsonl`. Work-item state is recorded by appending events (see Phase Completion below).

3. **Read the files the current phase references** (in the plan and in the artifacts it cites)
   - **Read files fully** - never use limit/offset parameters, you need complete context for the files you do open

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

4. **Update plan checkboxes** — Mark the phase complete in the plan files (these are plan content, the authoring surface for plan progress):
   - **Phase plan file**: change the phase file's `**Status**` field, e.g. in `plans/phase-1.md` change `**Status**: Ready for Implementation` → `**Status**: ✅ Completed`.
   - **Master plan**: in `plans/master.md`, update the inline phase listing status (e.g. `Status: ⏳ Not Started` → `Status: ✅ Completed`) and the progress-tracking table row (status, start/completion dates). If ALL phases are now complete, update master plan's top-level `**Status**` to `✅ Completed`.

5. **Record work-item state in the event log** (if work item detected). This is the work item's state of record — append events, then regenerate the manifest. Do **not** hand-edit `manifest.md`, the change log, the workflow checkboxes, or `docs/work/index.md`; all of those are derived (the manifest from `work.jsonl`; the index from `ls docs/work/*/`).

   - **When implementation starts** (first phase begins):
     ```bash
     scripts/wlog.sh "$WD" status_changed to=implementation
     scripts/wrender.sh "$WD"
     ```

   - **On each phase completion** (append a `phase_done` for the `implementation` phase; the optional `note=` is your one-line narrative for the change log):
     ```bash
     scripts/wlog.sh "$WD" phase_done phase=implementation note="<what this phase delivered>"
     scripts/wrender.sh "$WD"
     ```

   - **When validation passes** (all phases done and quality gates green):
     ```bash
     scripts/wlog.sh "$WD" status_changed to=completed
     scripts/wlog.sh "$WD" phase_done phase=validation note="<validation summary>"
     scripts/wrender.sh "$WD"
     ```

   - **Optional implementation notes**: if you keep a prose `implementation/status.md` (decisions, blockers, deviations), register it once as an artifact — don't treat it as state of record:
     ```bash
     scripts/wlog.sh "$WD" artifact_added kind=implementation path=implementation/status.md title="Implementation status"
     scripts/wrender.sh "$WD"
     ```

   Always follow any `wlog.sh` append(s) with `scripts/wrender.sh "$WD"` so the generated `manifest.md` reflects the new events.

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
