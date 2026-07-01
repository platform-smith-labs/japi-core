# Implementation Plan

You are tasked with creating implementation plans. You can adapt from quick planning for simple tickets to comprehensive interactive planning for complex features. Be skeptical, thorough, and work collaboratively with the user to produce high-quality technical specifications.

> ## ⚠️ Work-item state is an append-only event log
>
> **Never hand-edit `manifest.md`.** A work item's state lives in `<WD>/work.jsonl`; the manifest is a
> GENERATED view. Record state by appending an event with `scripts/wlog.sh` and regenerating with
> `scripts/wrender.sh "$WD"` — never by editing the manifest's Status line, Workflow Progress checkbox,
> Artifacts list, or Change Log. **Plan files (`plans/master.md`, `plans/phase-N.md`) remain authored
> markdown prose** — keep writing them exactly as before; only the work-item *state bookkeeping* moves
> to the event log. See [docs/dev/decisions/append-only-work-event-log.md](../../docs/dev/decisions/append-only-work-event-log.md).
>
> The events this command appends:
> - `scripts/wlog.sh "$WD" status_changed to=planning [note="..."]` — when planning starts
> - `scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/master.md title="Master plan"` — per plan file
> - `scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/phase-1.md title="Phase 1 — ..."` — per phase file
> - `scripts/wlog.sh "$WD" phase_done phase=planning [note="..."]` — when the plan is complete
>
> ALWAYS follow any `wlog.sh` append(s) with `scripts/wrender.sh "$WD"` to regenerate the manifest.

## Resolving `--work` IDs

When the user passes `--work <id>` (e.g., `--work work-2607010322-dark-mode`, or just a slug fragment),
resolve it to the work item directory by **glob, not arithmetic**:

- Glob `docs/work/*<id-or-slug>*/` (e.g. `docs/work/*dark-mode*/` or `docs/work/*2607010322*/`).
- **If exactly one match**, use that directory as `$WD` throughout the command. If zero, error: "Work
  item {arg} not found." If multiple, error and list the matches so the user can disambiguate.

Never scan-max-increment or compute a sequential number. Throughout the rest of this document, `$WD`
(equivalently `docs/work/work-NNNN/` in older examples) is shorthand for the resolved work item
directory — substitute the actual resolved path when constructing file paths.

## 🧰 Available Tools & Agents Reference

**Agent catalog + selection guidance:** read `.claude/reference/planv0-guidance.md` when choosing per-phase agents (it holds the full Research/Architecture/Domain/Security/Testing/Specialized agent catalog, the Skills list, Context-Window and Parallel-Execution best practices, and — in its tail sections — the Important Guidelines, Success Criteria Guidelines, Agent Selection Decision Tree, Common Patterns, and Sub-task Spawning Best Practices). Do not inline it here.

---

## Initial Response

When this command is invoked, determine the mode and check for existing plans:

### Mode Detection

1. **Work Item Mode** (`--work work-NNNN` provided):
   - Plans are associated with a work item
   - Create plans in `docs/work/work-NNNN/plans/`
   - Record plan artifacts and status as `work.jsonl` events (`wlog.sh` + `wrender.sh`), never by hand-editing the manifest
   - Load research and requirements from work item context

2. **Standalone Mode** (no `--work` parameter):
   - Plans are created in `docs/plans/NNNN-*.md`
   - Self-contained planning without work item integration
   - User provides context directly or via referenced documents
   - No manifest updates

### Existing Plan Detection

**Before proceeding, check if plans already exist:**

**For work item mode**:
1. Check if `docs/work/work-NNNN/plans/` exists
2. Use Glob to find: `docs/work/work-NNNN/plans/phase-*.md`
3. If plans exist → **Incremental Planning Mode** (see Incremental Planning Workflow below)
4. If no plans exist → **Initial Planning Mode** (continue with standard workflow)

**For standalone mode**:
1. Check if `docs/plans/` exists
2. Use Glob to find: `docs/plans/NNNN-*-phase-*.md`
3. If plans exist → **Incremental Planning Mode**
4. If no plans exist → **Initial Planning Mode**

## Incremental Planning Workflow (NEW)

**When existing plans are detected**, follow this intelligent incremental workflow:

### Step 1: Load Existing Plans and Context

1. **Read ALL existing plan files**:
   - Read master plan fully
   - Read ALL phase plans fully (phase-1.md, phase-2.md, etc.)
   - Extract:
     - Current phase structure and dependencies
     - What each phase accomplishes
     - Success criteria for each phase
     - Implementation status (check for completed checkboxes)

2. **Load work item context** (if work item mode):
   - Read manifest for overall goals
   - Read all research documents
   - Read all requirements documents

3. **Understand the new requirement**:
   - Get description from user (the argument after `--work work-NNNN`)
   - If not provided, ask: "What new feature or requirement do you want to add to the plan?"

### Step 2: Intelligent Phase Placement Analysis

**Use specialized agents to determine optimal placement** (run in PARALLEL):

1. **architect-reviewer** - Analyze architectural dependencies:
   ```
   "Analyze where this new requirement fits in the existing plan:

   **New Requirement**: [User's description]

   **Existing Phases**:
   - Phase 1: [Summary from phase-1.md]
   - Phase 2: [Summary from phase-2.md]
   - Phase 3: [Summary from phase-3.md]

   Determine:
   1. What existing phases does this depend on? (must come AFTER)
   2. What existing phases depend on this? (must come BEFORE)
   3. Suggested placement: After phase-X
   4. Rationale for placement"
   ```

2. **Domain specialist** (backend/frontend/fullstack based on requirement):
   ```
   "Review this new requirement for technical dependencies:

   **New Requirement**: [Description]
   **Existing Implementation Plan**: [Summary]

   Identify:
   1. Technical prerequisites from existing phases
   2. What this new phase enables for future phases
   3. Any conflicts or integration points with existing phases"
   ```

### Step 3: Present Intelligent Recommendation

After analysis, present to user:

```
✓ Analyzed existing plan with phases: 1, 2, 3

**New Requirement**: [User's description]

**Recommended Placement**: After Phase 2 (as phase-2.1)

**Rationale**:
- Depends on: Phase 2 ([specific deliverable])
- Enables: Phase 3 ([specific dependency])
- Logical fit: [architectural reasoning]

**Phase Numbering**:
- New phase: phase-2.1.md
- Existing phases remain unchanged (no renumbering)

**Impact Analysis**:
- Phase 1: ✓ No changes needed
- Phase 2: ✓ No changes needed
- Phase 3: ⚠️ Needs update (depends on new phase-2.1)
- Master Plan: ⚠️ Will be updated with new phase entry

Proceed with this placement? (y/n)
Or specify different placement: "after phase-X" or "end"
```

### Step 4: Create New Phase Plan

1. **Determine phase number**:
   - If inserting: Use decimal notation (phase-2.1, phase-3.2, etc.)
   - If appending: Use next integer (phase-4, phase-5, etc.)

2. **Create detailed phase plan**:
   - File: `docs/work/work-NNNN/plans/phase-X.Y.md` or `phase-N.md`
   - Follow standard phase plan template
   - Include:
     - Clear prerequisites from previous phases
     - Detailed implementation steps
     - Success criteria
     - References to research/requirements as needed

3. **Run domain validation** (same as initial planning):
   - **code-reviewer** - Code quality review
   - Domain specialists as appropriate
   - Incorporate feedback

### Step 5: Intelligent Impact Analysis and Updates

**Analyze impact on subsequent phases**:

1. **For each phase after the new one**:
   - Spawn **codebase-analyzer** or domain specialist to assess:
     ```
     "Analyze if this existing phase needs updates given the new phase:

     **New Phase**: phase-2.1 - [Summary]
     **Existing Phase**: phase-3 - [Current content]

     Determine:
     1. Does phase-3 depend on deliverables from new phase-2.1?
     2. Does phase-3 need to reference or build upon phase-2.1?
     3. Are there conflicts or redundancies to resolve?
     4. Specific changes needed (or 'no changes needed')

     Be conservative - only suggest changes if truly necessary."
     ```

2. **Update only affected phases**:
   - If agent says "no changes needed" → Skip
   - If agent identifies impacts → Update the phase file:
     - Add prerequisites from new phase
     - Update implementation steps if dependent
     - Adjust success criteria if needed
     - Add note: `<!-- Updated YYYY-MM-DD: Added dependency on phase-X.Y -->`

### Step 6: Update Master Plan

1. **Read current master plan**
2. **Add new phase entry** in correct position:
   ```markdown
   2. **[Phase 2 Name]** → [./phase-2.md](./phase-2.md)
      - Status: ✅ Complete
      - Summary: ...

   2.1. **[New Phase Name]** → [./phase-2.1.md](./phase-2.1.md)  <!-- NEW -->
      - Status: ⏳ Not Started
      - Summary: [1-2 sentence summary]
      - Key deliverables: [bullet list]
      - **Added**: YYYY-MM-DD (during implementation of phase-1)

   3. **[Phase 3 Name]** → [./phase-3.md](./phase-3.md)
      - Status: ⏳ Not Started
      - Summary: ...
      - **Updated**: YYYY-MM-DD (dependency on phase-2.1 added)  <!-- If updated -->
   ```

3. **Update phase dependencies section** if present
4. **Update progress tracking table** with new row
5. **Add to change log**: `{date}: Added phase-2.1 for [requirement]`

### Step 7: Register the New Phase Artifact (If Work Item Mode)

Record the new phase plan as an event — do **not** hand-edit the manifest's Artifacts list or Change
Log (both are generated from the log). Append one `artifact_added` event for the new phase file and
regenerate:

```bash
scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/phase-2.1.md title="Phase 2.1 — {New Title}" note="incremental: added for {requirement}"
scripts/wrender.sh "$WD"
```

(The brief "added for X" rationale rides on the event's `note=` — it appears in the generated Change
Log. The master plan's own phase list and change-log section, edited in Step 6, are plan-internal
authored prose and stay as-is.)

### Step 8: Confirm to User

```
✅ Phase plan added successfully

**New Phase Created**:
- Phase 2.1: [Title] (docs/work/work-NNNN/plans/phase-2.1.md)

**Updated Plans**:
- Master Plan: Added phase-2.1 entry
- Phase 3: Updated prerequisites to include phase-2.1 deliverables

**No Changes Needed**:
- Phase 1: Independent of new phase
- Phase 2: New phase builds on this phase

**Next Steps**:
- Review the new phase plan: docs/work/work-NNNN/plans/phase-2.1.md
- Review updated phases for accuracy
- Continue implementation: /implement_plan docs/work/work-NNNN/plans/phase-2.1.md
```

---

## Initial Planning Workflow (No Existing Plans)

### Step 1: Load Context

**If `--work work-NNNN` is provided (Work Item Mode)**:

1. **Read Work Manifest**:
   - File: `docs/work/work-NNNN/manifest.md`
   - Extract:
     - Work title and original user request
     - Current status (should be "📝 Requirements")
     - Links to research and requirements artifacts

2. **Read ALL Research Documents**:
   - Folder: `docs/work/work-NNNN/research/`
   - Use Glob to find all files: `docs/work/work-NNNN/research/*.md`
   - Read EACH research document FULLY
   - Extract from all documents:
     - Problem understanding
     - Technology options analyzed
     - Recommended approaches
     - Architectural considerations
     - Existing codebase patterns discovered
   - **IMPORTANT**: Consider insights from ALL research documents when planning

3. **Read ALL Requirements Documents**:
   - Folder: `docs/work/work-NNNN/requirements/`
   - Use Glob to find all files: `docs/work/work-NNNN/requirements/*.md`
   - Read EACH requirements document FULLY
   - Extract from all documents:
     - Functional requirements
     - Non-functional requirements
     - User stories / use cases
     - Acceptance criteria
     - Constraints and assumptions
     - Dependencies
   - **IMPORTANT**: Address requirements from ALL documents in the plan

4. **Read Upstream Messages (if epic-linked)**:
   - Check if the manifest header shows an `**Epic**: epic-<YYMMDDHHMM>-<slug>` line
   - If yes, read the inbound relay messages at `$WD/relays/inbound/from-*.md` (their open/resolved lifecycle is in `work.jsonl`; the generated manifest's **Open Relays** + **Upstream Messages** sections summarize them)
   - Read EACH inbound relay for cross-repo constraints and interface contracts
   - These messages contain requirements from OTHER repos (API contracts, message formats, DB schemas)
   - **IMPORTANT**: Incorporate upstream constraints into the plan. Add an "## Upstream Dependencies" section to `master.md` if any upstream messages exist, listing:
     - What other repos expect from this repo
     - Interface contracts that must be honored
     - Constraints on implementation approach

**If `--work` is NOT provided (Standalone Mode)**:

1. **Gather Context from User**:
   - Ask: "What are you planning to implement?"
   - Ask: "Do you have any research or requirements documents I should reference?"
   - If user provides file paths, read them fully
   - Get clarification on goals, constraints, and acceptance criteria

2. **Optional: Reference Standalone Documents**:
   - User may point to `docs/research/*.md` files
   - User may point to `docs/requirements/*.md` files
   - Read any referenced documents fully

### Step 2: Create Implementation Plan

Based on research + requirements:

1. **Assess Complexity** (using multi-file enforcement rules below)
   - Count implementation phases needed
   - Estimate total plan size
   - Determine if multi-file structure required

2. **Create Plan Files**:

   **With work item (`--work work-NNNN`)**:
   - **Create plans folder**: `docs/work/work-NNNN/plans/`
   - **Single file**: `docs/work/work-NNNN/plans/master.md` (if simple)
   - **Multi-file**:
     - `docs/work/work-NNNN/plans/master.md` (overview only, <200 lines)
     - `docs/work/work-NNNN/plans/phase-1.md` (detailed implementation)
     - `docs/work/work-NNNN/plans/phase-2.md` (detailed implementation)
     - `docs/work/work-NNNN/plans/phase-N.md` (etc.)

   **Standalone (no `--work`)**:
   - **Single file**: `docs/plans/NNNN-descriptive-name-plan.md` (if simple)
   - **Multi-file**:
     - `docs/plans/NNNN-descriptive-name-master-plan.md` (overview)
     - `docs/plans/NNNN-descriptive-name-phase-1-plan.md`
     - `docs/plans/NNNN-descriptive-name-phase-2-plan.md`
     - Use global numbering (check all files in `docs/plans/`)

3. **Plan Content MUST**:

   **With work item**:
   - Reference the work item: `Work Item: work-NNNN`
   - Link to research: `[Research Document](../research/NNNN-*.md)` (relative path)
   - Link to requirements: `[Requirements Document](../requirements/NNNN-*.md)` (relative path)
   - Address ALL requirements from requirements document
   - Incorporate recommendations from research document
   - Include traceability (requirements → plan sections)

   **Standalone**:
   - No work item reference needed
   - Link to any referenced research/requirements (if provided)
   - Address all goals and acceptance criteria gathered from user
   - Include clear scope and objectives

### Step 3: Record Plan State as Events (If Using Work Item)

**If `--work work-NNNN` was provided** — append events to `work.jsonl` and regenerate the manifest.
**Never hand-edit `manifest.md`** (no Status line, no Workflow Progress checkbox, no Artifacts list,
no Change Log, no Last Updated — all of these are generated by `wrender.sh` from the events below).

1. **Transition status to planning**:
   ```bash
   scripts/wlog.sh "$WD" status_changed to=planning
   ```

2. **Register each plan file** (one `artifact_added` event per file — master + every phase):
   ```bash
   scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/master.md  title="Master plan"
   scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/phase-1.md title="Phase 1 — {Title}"
   scripts/wlog.sh "$WD" artifact_added kind=plan path=plans/phase-2.md title="Phase 2 — {Title}"
   # …one per phase file…
   ```

3. **Mark the planning phase done** (the epic-barrier signal that the plan is complete). A brief
   one-line rationale can ride on `note=`; it surfaces in the generated Change Log:
   ```bash
   scripts/wlog.sh "$WD" phase_done phase=planning note="implementation plan created"
   ```

4. **Regenerate the manifest** (always, after the appends above):
   ```bash
   scripts/wrender.sh "$WD"
   ```

The generated manifest will reflect the 🎨 Planning status, the registered plan artifacts, and the
change-log entries — projected from the events, with no hand-editing.

**If standalone**:
- No work item, no event log — plans are self-contained
- Plan-internal status (master/phase markdown) is the only progress surface

### Step 4: Confirm to User

**If work item mode**:
```
✅ Implementation plan created for work-NNNN

Plans created:
- Master Plan: docs/work/work-NNNN/plans/master.md
- Phase 1: {Title} (docs/work/work-NNNN/plans/phase-1.md)
- Phase 2: {Title} (docs/work/work-NNNN/plans/phase-2.md)
...

📊 Work Item Status: 🎨 Planning → Ready for Implementation

All artifacts are in: docs/work/work-NNNN/

Please review the implementation plan.
When ready to implement, run: /implement_plan docs/work/work-NNNN/plans/master.md
```

**If standalone mode**:
```
✅ Implementation plan created

Plans created:
- Master Plan: docs/plans/NNNN-{name}-master-plan.md
- Phase 1: {Title} (docs/plans/NNNN-{name}-phase-1-plan.md)
- Phase 2: {Title} (docs/plans/NNNN-{name}-phase-2-plan.md)
...

Please review the implementation plan.
When ready to implement, run: /implement_plan docs/plans/NNNN-{name}-master-plan.md
```

## ⚠️ CRITICAL REQUIREMENT: Multi-File Plan Enforcement

**BEFORE YOU START PLANNING**, assess complexity:

- **If 4+ phases OR estimated >400 lines**: You MUST create SEPARATE FILES
  - 1 master plan file (overview, architecture, phase list)
  - N phase files (one per phase, detailed implementation)

- **If 1-3 simple phases AND <400 lines**: Single file is acceptable

**DO NOT create a single large file for complex plans. This is a hard requirement.**

Files to create for multi-file plans:

```
docs/work/work-NNNN/plans/master.md              (Master - overview only)
docs/work/work-NNNN/plans/phase-1.md             (Phase 1 details)
docs/work/work-NNNN/plans/phase-2.md             (Phase 2 details)
docs/work/work-NNNN/plans/phase-N.md             (Phase N details)
```

## Process Steps

### Step 1: Context Gathering & Initial Analysis

1. **Read all work item artifacts immediately and FULLY**:
   - Work manifest: `docs/work/work-NNNN/manifest.md`
   - All research documents in `docs/work/work-NNNN/research/`
   - All requirement documents in `docs/work/work-NNNN/requirements/`
   - Any issue files in `docs/work/work-NNNN/issues/`
   - Any existing plans in `docs/work/work-NNNN/plans/`
   - **IMPORTANT**: Use the Read tool WITHOUT limit/offset parameters to read entire files
   - **CRITICAL**: DO NOT spawn sub-tasks before reading these files yourself in the main context
   - **NEVER** read files partially - if a file is mentioned, read it completely

2. **Analyze Work Item Context**:
   - Review all research findings to understand the problem space
   - Review all requirements to understand what needs to be built
   - Understand the current status and any previous planning attempts
   - Identify gaps or areas needing clarification

3. **Spawn research tasks — CONDITIONAL based on existing research**:

   **If work item has research docs** (docs/work/work-NNNN/research/ is non-empty):
   - The broad discovery has already been done by `/work`. Do NOT re-run broad agents.
   - Instead, assess whether the research docs provide sufficient implementation-level detail:
     * Do they identify the specific files that need modification? (file paths + line numbers)
     * Do they show concrete code patterns to follow for this implementation?
   - **If gaps exist**, spawn ONLY targeted agents (IN PARALLEL):
     - **codebase-pattern-finder** - Find specific implementation patterns to model after (e.g., "find an existing NestJS module with CRUD + guards I can copy for the new feature")
     - **codebase-analyzer** - Deep-dive on specific files/modules identified in research that need line-level understanding for the plan
   - **If research docs are thorough** (have file paths, patterns, architectural context): skip agent research entirely and proceed to Step 4.

   **If standalone mode (no work item) OR research docs are empty/missing**:
   - Run full discovery. **Launch these agents in PARALLEL (single message, multiple Task calls)**:
     - **codebase-locator** - Find all files related to the ticket/task
     - **codebase-analyzer** - Understand how the current implementation works
     - **thoughts-locator** - Find any existing documentation, requirements, or tickets about this feature
     - **codebase-pattern-finder** - Find similar implementations to model after

   These agents will:
   - Find relevant source files, configs, and tests
   - Identify the specific directories to focus on (e.g., apps/, libs/, or specific components)
   - Trace data flow and key functions
   - Return detailed explanations with file:line references

4. **Read key files into main context**:
   - If agents were spawned: read ALL files they identified as relevant, FULLY
   - If skipping agents (thorough research docs): read the specific files mentioned in research docs that you'll need to reference when writing the plan
   - This ensures you have complete understanding before proceeding

5. **Analyze and verify understanding**:
   - Cross-reference the ticket/requirement specifications with actual code
   - Identify any discrepancies or misunderstandings
   - Note assumptions that need verification
   - Determine true scope based on codebase reality

6. **Present informed understanding and focused questions**:

   ```
   Based on the requirements and my research of the codebase, I understand we need to [accurate summary].

   I've found that:
   - [Current implementation detail with file:line reference]
   - [Relevant pattern or constraint discovered]
   - [Potential complexity or edge case identified]

   Questions that my research couldn't answer:
   - [Specific technical question that requires human judgment]
   - [Business logic clarification]
   - [Design preference that affects implementation]
   ```

   Only ask questions that you genuinely cannot answer through code investigation.

### Step 1b: Architecture Review

**For complex features, validate architectural approach before detailed planning:**

1. **Determine if architecture review is needed**:
   - Complex features with multi-component changes → Yes
   - Simple bug fixes or single-file changes → No
   - Decision criteria:
     * New architectural components → Review
     * API endpoint changes → Review
     * Database schema changes → Review
     * Multi-service integration → Review
     * Single file bug fix → Skip

2. **If review needed, spawn architect-reviewer**:

   Use Task tool with subagent_type="architect-reviewer":

   Prompt template:
   ```
   Review the architectural approach for this feature:

   **Feature**: [Brief description from requirement/ticket]
   **Proposed Approach**: [1-2 sentence summary from initial understanding]
   **Affected Components**: [List main components from codebase research]
   **Current Architecture**: [Reference architecture patterns found]

   Please validate:
   1. Does this align with our existing architecture?
   2. Are there better architectural patterns we should use?
   3. What architectural risks or concerns exist?
   4. Any recommended adjustments?

   Focus on: scalability, maintainability, alignment with existing patterns.
   ```

3. **Process architecture feedback**:
   - If concerns raised → Present to user, adjust approach
   - If approved → Document architectural validation
   - If alternatives suggested → Evaluate with user

4. **Document architecture review in plan metadata** (will be added to plan)

### Step 1c: Choose Planning Approach

**Assess complexity and choose approach:**

**Simple Approach** (for straightforward tickets with clear requirements):

- Skip extensive user interaction
- Rely on existing work item research docs (do NOT re-run broad discovery)
- Only spawn targeted agents if research docs lack file-level detail
- Create plan directly and present for approval
- Suitable for: bug fixes, small features, well-defined tasks

**Comprehensive Approach** (for complex features):

- Full interactive process with user feedback
- Targeted research to fill gaps not covered by existing work item research
- For standalone mode: extensive parallel research using multiple agents
- Iterative planning with multiple approval points
- Suitable for: new features, architectural changes, unclear requirements

### Step 2a: Simple Planning Workflow

**For straightforward tasks:**

1. **Leverage existing research** (do NOT re-run broad discovery):
   - Review existing work item research documents (already read in Step 1)
   - Only spawn **codebase-pattern-finder** if you need a specific implementation pattern to model after that wasn't covered in the research docs
   - For standalone mode (no work item): use codebase-locator + codebase-pattern-finder

2. **Create Plan Directly**:
   - Write implementation plan to `docs/work/work-NNNN/plans/master.md`
   - Include phases with clear success criteria
   - Reference existing patterns found during research
   - Include recommended agents/skills for implementation

3. **Present for Approval**:
   - Show plan location and summary
   - Ask for user feedback or approval
   - Make adjustments if needed

### Step 2b: Research & Discovery (Comprehensive Approach)

For complex features, after getting initial clarifications:

1. **If the user corrects any misunderstanding**:
   - DO NOT just accept the correction
   - Spawn new research tasks to verify the correct information
   - Read the specific files/directories they mention
   - Only proceed once you've verified the facts yourself

2. **Create a research todo list** using TodoWrite to track exploration tasks

3. **Spawn targeted research tasks based on what's missing (IN PARALLEL)**:

   **If work item has research docs**: Only fill gaps — do NOT re-run broad discovery.
   Assess what the existing research docs DON'T cover and spawn agents only for those gaps:

   - **codebase-pattern-finder** - If research docs don't include specific code patterns to model after
   - **codebase-analyzer** - If research docs identify components but lack line-level implementation detail needed for planning
   - **web-search-researcher** - If research docs flag external libraries/approaches that need deeper investigation

   Skip these (already covered by `/work` research):
   - ~~codebase-locator~~ — file discovery already done
   - ~~thoughts-locator~~ — doc discovery already done
   - ~~thoughts-analyzer~~ — doc analysis already done

   **If standalone mode (no work item)**: Run full discovery:
   - **codebase-locator** - To find specific files
   - **codebase-analyzer** - To understand implementation details
   - **codebase-pattern-finder** - To find similar features to model after
   - **thoughts-locator** - To find related documentation
   - **thoughts-analyzer** - To extract key insights from documents
   - **web-search-researcher** - To research best practices or libraries

   Each agent knows how to:
   - Find the right files and code patterns
   - Identify conventions and patterns to follow
   - Look for integration points and dependencies
   - Return specific file:line references
   - Find tests and examples

4. **Wait for ALL sub-tasks to complete** before proceeding

5. **Present findings and design options**:

   ```
   Based on my research, here's what I found:

   **Current State:**
   - [Key discovery about existing code]
   - [Pattern or convention to follow]

   **Design Options:**
   1. [Option A] - [pros/cons]
   2. [Option B] - [pros/cons]

   **Open Questions:**
   - [Technical uncertainty]
   - [Design decision needed]

   Which approach aligns best with your vision?
   ```

### Step 3: Plan Complexity Assessment & Structure Development

Once aligned on approach:

1. **Assess plan complexity** to determine if it should be single-file or multi-file:

   **Single-File Plan** (For simple features only):
   - 1-3 phases maximum
   - Each phase is simple and straightforward (5 or fewer file changes)
   - Total estimated plan length under 400 lines
   - Single domain (e.g., only frontend OR only backend)
   - Can be fully understood and implemented in one sitting

   **Multi-File Plan** (REQUIRED for complex features):
   - 4+ phases OR
   - Any individual phase requires 6+ file changes OR
   - Multi-domain implementation (frontend + backend + database, etc.) OR
   - Estimated plan length would exceed 400 lines OR
   - Implementation would take multiple days/sessions

   **CRITICAL**: If you estimate the plan will exceed 400 lines, you MUST use multi-file structure.
   **Default to multi-file for any non-trivial feature.**

2. **Create initial plan outline**:

   **For Single-File Plans:**
   ```
   Here's my proposed plan structure:

   ## Overview
   [1-2 sentence summary]

   ## Implementation Phases:
   1. [Phase name] - [what it accomplishes]
   2. [Phase name] - [what it accomplishes]
   3. [Phase name] - [what it accomplishes]

   Does this phasing make sense? Should I adjust the order or granularity?
   ```

   **For Multi-File Plans:**
   ```
   This is a complex implementation. I propose splitting it into:

   ## Master Plan: docs/work/work-NNNN/plans/master.md
   [Overview and orchestration]

   ## Phase Plans:
   1. Phase 1: [Name] → docs/work/work-NNNN/plans/phase-1.md
      - [What it accomplishes]
      - [Estimated file changes: X files]

   2. Phase 2: [Name] → docs/work/work-NNNN/plans/phase-2.md
      - [What it accomplishes]
      - [Estimated file changes: Y files]

   This splitting makes sense because:
   - [Reason 1: e.g., phases are independently implementable]
   - [Reason 2: e.g., each phase is substantial enough to warrant its own document]

   Does this structure work for you?
   ```

3. **Get feedback on structure** before writing details

### Step 4: Detailed Plan Writing

After structure approval:

1. **Create plan files** in `docs/work/work-NNNN/plans/`

2. **MANDATORY: Determine Agent Recommendations** (BEFORE writing plans):

   **Analyze the plan domains and populate specific agent recommendations:**

   a. **Identify plan domains** from requirements and research:
      - Backend/API work → backend-developer, api-designer, postgres-pro
      - Frontend work → frontend-developer, react-specialist, ui-designer
      - Full-stack → fullstack-developer
      - Database → postgres-pro, database-administrator, database-optimizer
      - Security concerns → security-engineer, security-auditor
      - Performance critical → performance-engineer
      - Infrastructure → platform-engineer, devops-engineer, cloud-architect
      - Testing focus → qa-expert, test-automator
      - UI/UX heavy → ui-designer, accessibility-tester
      - Real-time features → websocket-engineer
      - Mobile → mobile-app-developer
      - Payments → payment-integration

   b. **Create specific agent recommendation list** for this plan:
      ```
      For each phase, determine:
      - Which domain specialists to use during implementation
      - Which validation agents to use before completion
      - When to use code-reviewer (always before finalizing)
      - Any special-purpose agents needed
      ```

   c. **Required skills list**:
      - `/frontend-design` if creating UI components
      - `/commit` for all plans (git commits)
      - `/learn` for capturing discoveries
      - Any other relevant skills

   **CRITICAL**: These MUST be SPECIFIC recommendations, not placeholders like "[agent-name]"!

3. **Write the plan(s)** based on complexity:

   **For Single-File Plans:**
   - Write to `docs/work/work-NNNN/plans/master.md`
   - Include all phases in one document
   - Use the single-file template (see Template A below)

   **For Multi-File Plans (CRITICAL - Follow this process exactly):**

   **IMPORTANT**: You must create separate physical files for each phase. Do NOT put all phases in one file.

   **Step-by-step process:**

   a. **Create the master plan first** (`docs/work/work-NNNN/plans/master.md`):
      - Use Template B (Master Plan structure)
      - Include overview, architecture, and phase list with links
      - Keep it high-level (should be under 200 lines)
      - DO NOT include detailed implementation steps here

   b. **Then create EACH phase as a SEPARATE file**:
      - Phase 1: `docs/work/work-NNNN/plans/phase-1.md`
      - Phase 2: `docs/work/work-NNNN/plans/phase-2.md`
      - Phase N: `docs/work/work-NNNN/plans/phase-N.md`
      - Use Template C for EACH phase file
      - Each file should be 100-300 lines maximum
      - Include detailed implementation ONLY in phase files

   c. **Verify separation**:
      - Master plan contains NO detailed implementation steps
      - Each phase file is independent and complete
      - All cross-references are correct
      - Total: 1 master file + N phase files (N = number of phases)

   d. **MANDATORY: Populate Agent Recommendations** in ALL plan files:
      - Use the specific agent list from Step 2
      - Replace ALL placeholders with actual agent names
      - Include WHY each agent should be used
      - Add agents to both master and phase plans

3. **Write the plan(s) using the templates.**

   **Plan-file templates** (single-file / master / phase-N): read `.claude/reference/planv0-templates.md` when you author the plan files — do not inline them here. That file holds **Template A** (single-file plan → `master.md`), **Template B** (master plan for multi-file plans), and **Template C** (each phase file). Apply Template A for single-file plans; apply Template B for the master plan plus Template C for every phase file in multi-file plans.

---

### Step 4b: Domain-Specific Plan Validation

**After writing initial plan(s), validate with domain specialists (IN PARALLEL):**

**FIRST: Verify plan structure compliance:**
- If you created 4+ phases or multi-domain plan, confirm you created MULTIPLE SEPARATE FILES
- Check master plan is under 200 lines (if multi-file)
- Check each phase file is 100-300 lines
- If you put everything in one file, STOP and split it now

**THEN proceed with domain validation:**

**For Single-File Plans**: Validate the complete plan
**For Multi-File Plans**: Validate master plan for architecture, then validate each phase plan for implementation details

1. **Detect plan domains** by analyzing plan content:
   - Mentions API endpoints → Backend/API domain
   - Mentions database schema → Database domain
   - Mentions React components → Frontend domain
   - Mentions infrastructure → Platform/DevOps domain
   - Multiple domains → Full-stack

2. **Spawn validation agents in PARALLEL** based on domains:

   **For ALL Plans (Always Include)**:
   - **code-reviewer** - Code quality, maintainability, best practices
     * Check: Design patterns, technical debt, performance, maintainability
     * CRITICAL: Run for ALL plans regardless of domain

   **For Backend/API Plans**:
   - **api-designer** - Review API endpoint design
     * Check: RESTful conventions, versioning, error handling
   - **postgres-pro** - Review database changes (this project uses PostgreSQL)
     * Check: normalization, indexing, RLS patterns, migrations
   - **security-engineer** - Security implications
     * Check: auth, authorization, data protection, input validation

   **For Frontend Plans**:
   - **ui-designer** - Visual design and UX patterns
     * Check: Design system consistency, visual hierarchy, user experience
   - **frontend-developer** - Component architecture
     * Check: component patterns, state management, reusability
   - **react-specialist** - React-specific patterns
     * Check: hooks usage, performance, best practices
   - **accessibility-tester** - Accessibility considerations
     * Check: WCAG compliance, keyboard navigation

   **For Full-Stack Plans**:
   - **fullstack-developer** - End-to-end integration
     * Check: data flow, error handling, state synchronization

   Use Task tool with appropriate subagent_type and prompts like:
   ```
   Review the [domain] aspects of this implementation plan:

   [Relevant plan excerpt or file path to read]

   Validate:
   1. Are the patterns and approaches sound?
   2. Any missing considerations?
   3. Specific recommendations for improvement?
   4. Any potential issues or risks?
   ```

3. **Wait for all validation agents** to complete

4. **Synthesize validation results**:
   - Collect all feedback from validators
   - Categorize: Critical issues vs. Recommendations
   - Prioritize feedback

5. **Incorporate feedback into plan**:
   - Critical issues → Update plan immediately
   - Recommendations → Add to plan as enhancement checkboxes
   - Document validation in plan's "Plan Validation" section

### Step 4c: Testing Strategy Enhancement

**Generate comprehensive testing strategy (IN PARALLEL):**

1. **Spawn testing strategy agents in parallel**:
   - **qa-expert** - Overall testing strategy
     * Generate: test types, coverage targets, critical test paths
   - **test-automator** - Test automation approach
     * Generate: framework choices, test structure, CI integration
   - **performance-engineer** - Performance testing (if applicable)
     * Generate: benchmarks, load scenarios, acceptance criteria

2. **Wait for testing agents** to complete

3. **Enhance plan's testing section** with generated strategies:
   - Replace generic testing section with detailed strategies
   - Include specific test files, frameworks, and coverage targets
   - Add both automated and manual verification steps

### Step 4d: Agent Recommendation Validation (NEW - MANDATORY)

**BEFORE finalizing the plan, validate that agent recommendations are properly populated:**

1. **Check ALL plan files** (master + all phases):
   - Search for placeholder patterns: `[agent-name]`, `[domain-agent]`, `[specialist]`
   - Search for generic phrases: "Other skills as needed", "if applicable"
   - Verify each phase has SPECIFIC agent recommendations

2. **Validation checklist**:
   ```
   For Master Plan:
   - [ ] Skills section has specific skills (not placeholders)
   - [ ] Agents section lists actual agent names
   - [ ] Per-phase agent recommendations are specific
   - [ ] Domain-to-agent mapping is populated

   For Each Phase Plan:
   - [ ] Primary implementation agent is named
   - [ ] Specialist agents are specific to phase work
   - [ ] code-reviewer is included
   - [ ] Agent usage scenarios are described
   ```

3. **If placeholders found**:
   - STOP plan finalization
   - Analyze plan content to determine correct agents
   - Populate with specific agents based on:
     - Phase implementation focus (backend/frontend/database/etc.)
     - Technical requirements from requirements docs
     - Research findings on complexity
   - Re-validate

4. **Ensure minimum agent coverage**:
   - Every plan MUST have: code-reviewer
   - Every phase MUST have: at least one domain specialist
   - Complex phases MUST have: multiple specialists

**Example of GOOD agent recommendations** (specific, actionable):
```markdown
**Agents for This Phase**:
- **backend-developer** - Implement REST API endpoints and business logic
- **postgres-pro** - Validate database schema and RLS policies
- **api-designer** - Review endpoint design for RESTful conventions
- **code-reviewer** - Final quality check before phase completion
```

**Example of BAD agent recommendations** (placeholders, generic):
```markdown
**Agents**:
- **[agent-name]** - [When to use]
- **[domain-agent]** - [For validation]
- Other agents as needed
```

### Step 5: Final Validation and Sync

1. **MANDATORY: Final Agent Recommendation Check**:
   - Run Step 4d validation one final time
   - Confirm NO placeholders remain in any plan file
   - Verify all agent recommendations are specific and actionable
   - If any placeholders found → STOP and fix before proceeding

2. **Save and organize the validated plan(s)**:
   - Ensure all plans are saved in the appropriate location
   - Create the directory structure if it doesn't exist
   - For multi-file plans, ensure all phase plans are created
   - Verify agent recommendations are in all files

2. **Present the validated plan location(s)**:

   **For Single-File Plans:**
   ```
   I've created the implementation plan at:
   `docs/work/work-NNNN/plans/master.md`

   Please review it and let me know:
   - Are the phases properly scoped?
   - Are the success criteria specific enough?
   - Any technical details that need adjustment?
   - Missing edge cases or considerations?
   ```

   **For Multi-File Plans:**
   ```
   I've created a multi-file implementation plan:

   **Master Plan**: `docs/work/work-NNNN/plans/master.md`
   - Overview, architecture, and phase orchestration

   **Phase Plans**:
   - Phase 1: `docs/work/work-NNNN/plans/phase-1.md`
   - Phase 2: `docs/work/work-NNNN/plans/phase-2.md`
   - Phase N: `docs/work/work-NNNN/plans/phase-N.md`

   Please review:
   - Master plan: Overall architecture and phase organization
   - Individual phase plans: Implementation details and scope
   - Are phases properly separated and independently implementable?
   - Any phase dependencies that need clarification?
   ```

3. **Iterate based on feedback** - be ready to:
- Add missing phases
- Split or merge phases if needed (for multi-file plans)
- Adjust technical approach
- Clarify success criteria (both automated and manual)
- Add/remove scope items
- After making changes, save the updated plan

4. **Continue refining** until the user is satisfied

## Planning Guidance (Important Guidelines, Success Criteria, Agent Selection, Common Patterns, Sub-task Spawning)

**Planning best-practices reference:** read `.claude/reference/planv0-guidance.md` when applying planning standards — it holds the **Important Guidelines** (skeptical/interactive/thorough/practical, no-open-questions, multi-file & agent-recommendation requirements), the **Success Criteria Guidelines** (automated vs. manual split), the **Agent Selection Decision Tree**, the **Common Patterns** (database changes / new features / refactoring), and the **Sub-task Spawning Best Practices**. Do not inline them here.

## Example Workflows

### Example 1: Initial Planning

```
User: /work "Add dark mode toggle to the application"
Assistant:
1. Creates: work-0001 with research and requirements
2. Returns for user review

User: /planv0 --work work-0001
Assistant:
1. Detects: No existing plans
2. Reads: docs/work/work-0001/manifest.md
3. Reads: All research and requirements
4. Determines: Simple feature, 2-3 phases
5. Creates: docs/work/work-0001/plans/master.md (single file)
6. Validates with: code-reviewer, frontend-developer, react-specialist
7. Includes: /frontend-design skill reference for implementation
8. Appends events (status_changed to=planning, artifact_added per plan file, phase_done phase=planning) + runs wrender.sh to regenerate the manifest
```

### Example 2: Incremental Planning - Adding New Phase

```bash
# After implementing phase-1 of dark mode, user discovers new requirement
User: /planv0 --work work-0001 "Add user preference storage for dark mode in database"

Assistant (Incremental Planning Mode):
1. Detects: Existing plans (phase-1.md, phase-2.md)
2. Reads: All existing phase plans
3. Loads: Work item context and requirements
4. Analyzes placement with architect-reviewer and postgres-pro:
   - New requirement depends on: Phase 1 (UI components)
   - Phase 2 depends on: This new phase (needs stored preferences)
5. Recommends: Insert as phase-1.1 (after UI, before theme system)
6. User confirms placement
7. Creates: docs/work/work-0001/plans/phase-1.1.md
8. Impact analysis determines:
   - Phase 1: No changes (new phase builds on it)
   - Phase 2: Needs update (must read from database)
9. Updates: phase-2.md with new prerequisites
10. Updates: master.md with phase-1.1 entry
11. Appends: artifact_added kind=plan path=plans/phase-1.1.md + runs wrender.sh (manifest regenerated, never hand-edited)

Output:
✅ Phase 1.1 created: User Preference Storage
- New phase: docs/work/work-0001/plans/phase-1.1.md
- Updated: phase-2.md (added database dependency)
- Updated: master.md (added phase-1.1 entry)
```

### Example 3: Complex Multi-Phase Planning

```bash
User: /work "Complete e-commerce platform with payments, inventory, and analytics"
Assistant:
1. Creates: work-0002 with comprehensive research and requirements
2. Returns for user review

User: /planv0 --work work-0002
Assistant:
1. Detects: No existing plans
2. Reads: docs/work/work-0002/manifest.md
3. Reads: All research and requirements documents
4. Determines: Very complex, 10+ phases, multi-domain
5. Creates master plan: docs/work/work-0002/plans/master.md
6. Creates phase plans:
   - docs/work/work-0002/plans/phase-1.md (Database & Core Models)
   - docs/work/work-0002/plans/phase-2.md (Product Catalog API)
   - docs/work/work-0002/plans/phase-3.md (Payment Integration)
   - docs/work/work-0002/plans/phase-4.md (Inventory Management)
   - docs/work/work-0002/plans/phase-5.md (Analytics & Reporting)
   - docs/work/work-0002/plans/phase-6.md (Frontend UI)
7. Validates with: code-reviewer, architect-reviewer, security-engineer,
   postgres-pro, api-designer, fullstack-developer, payment-integration
8. Includes skill references: /frontend-design for UI, /commit for commits
9. Appends events (status_changed to=planning, artifact_added per plan file, phase_done phase=planning) + runs wrender.sh to regenerate the manifest

# Later, during implementation...
User: /planv0 --work work-0002 "Add real-time inventory notifications"

Assistant (Incremental Planning Mode):
1. Detects: Existing phases 1-6
2. Analyzes with architect-reviewer and websocket-engineer
3. Determines: Depends on phase-4 (Inventory), enables phase-5 (Analytics)
4. Recommends: Insert as phase-4.1
5. Creates: phase-4.1.md (WebSocket Infrastructure)
6. Updates: phase-5.md (can use real-time data)
7. Updates: master.md with phase-4.1
```

---

## Key Principles for Incremental Planning

1. **Intelligent Placement**: Use agents to analyze dependencies, don't just ask user
2. **Decimal Numbering**: phase-2.1, phase-3.2 to avoid renumbering
3. **Minimal Updates**: Only update phases that are genuinely affected
4. **Conservative Changes**: When in doubt, don't update unless necessary
5. **Clear Communication**: Explain placement rationale and impacts
6. **Master Plan as Source**: Always keep master.md up to date with phase list
7. **Agent Recommendations**: New phases MUST have specific agent recommendations populated

## Key Principles for Agent Recommendations (NEW)

1. **Always Specific**: Never use placeholders - always name actual agents
2. **Domain-Driven**: Select agents based on the technical domain (backend, frontend, etc.)
3. **Actionable Guidance**: Include WHEN and WHY to use each agent
4. **Minimum Coverage**: code-reviewer + at least one domain specialist
5. **Phase-Appropriate**: Tailor agent recommendations to each phase's specific work
6. **Validate Before Finalizing**: Run validation check to ensure no placeholders remain

Think deeply, use TodoWrite to track your tasks, leverage all available agents and skills, and ensure complete documentation traceability for future reference and resumable workflows.
