# Implementation Plan

You are tasked with creating implementation plans. You can adapt from quick planning for simple tickets to comprehensive interactive planning for complex features. Be skeptical, thorough, and work collaboratively with the user to produce high-quality technical specifications.

## 🧰 Available Tools & Agents Reference

**BEFORE YOU START PLANNING**, familiarize yourself with these capabilities to leverage them throughout the planning process:

### 🔍 Research & Discovery Agents

**Codebase Exploration**:
- **codebase-locator** - Find files related to features/tasks
  - Use when: You need to locate all relevant files for a feature
  - Returns: File paths, directory structure insights
- **codebase-analyzer** - Understand how implementations work
  - Use when: You need to understand existing code patterns
  - Returns: Implementation details with file:line references
- **codebase-pattern-finder** - Find similar implementations to model after
  - Use when: Looking for examples of similar features
  - Returns: Concrete code examples and patterns

**Documentation & Context**:
- **thoughts-locator** - Find existing docs, requirements, tickets, decisions
  - Use when: Searching for past research or documentation
  - Returns: Relevant document paths
- **thoughts-analyzer** - Deep dive on documentation topics
  - Use when: Need detailed insights from documents
  - Returns: Extracted key insights and analysis
- **search-specialist** - Advanced information retrieval
  - Use when: Complex searches across diverse sources
  - Returns: Comprehensive search results
- **web-search-researcher** - Research modern topics and external information
  - Use when: Need current information or external context
  - Returns: Web-based research findings

### 🏗️ Architecture & Design Agents

**Architecture Review**:
- **architect-reviewer** - Validate system design and architectural patterns
  - Use when: Complex features with multi-component changes
  - Validates: Scalability, maintainability, alignment with patterns
- **microservices-architect** - Design scalable microservice ecosystems
  - Use when: Multi-service architectures
  - Validates: Service boundaries, communication patterns

**Design Specialists**:
- **api-designer** - Design scalable, developer-friendly APIs
  - Use when: Creating or modifying API endpoints
  - Reviews: REST/GraphQL design, consistency, versioning
- **ui-designer** - Visual design and UX patterns (AGENT)
  - Use when: Need design system validation, visual hierarchy review
  - Reviews: Design consistency, interaction patterns, visual hierarchy

### 💻 Domain-Specific Validation Agents

**Backend**:
- **backend-developer** - Server-side solutions and patterns
- **api-designer** - API endpoint design and conventions
- **postgres-pro** - PostgreSQL optimization and design (this project uses PostgreSQL)
  - Reviews: Schema design, indexing, RLS patterns, migrations
- **database-administrator** - High-availability database systems
- **database-optimizer** - Query optimization and performance tuning

**Frontend**:
- **frontend-developer** - Component architecture and patterns
- **react-specialist** - React 18+ patterns and best practices
  - Reviews: Hooks, performance, server components
- **ui-designer** - Design systems and visual design
  - Reviews: Visual hierarchy, design consistency
- **accessibility-tester** - WCAG compliance and inclusive design
- **typescript-pro** - Advanced TypeScript patterns

**Full-Stack**:
- **fullstack-developer** - End-to-end integration
  - Reviews: Data flow, error handling, state synchronization

### 🔒 Security & Quality Agents

**Security**:
- **security-engineer** - DevSecOps and infrastructure security
  - Reviews: Auth, authorization, data protection, vulnerabilities
- **security-auditor** - Comprehensive security assessments
- **penetration-tester** - Vulnerability assessment and security testing

**Code Quality**:
- **code-reviewer** - Code quality, best practices, design patterns
  - Use for: ALL plans before finalization
  - Reviews: Maintainability, technical debt, performance, best practices
- **refactoring-specialist** - Safe code transformation techniques

### 🧪 Testing & Quality Assurance Agents

- **qa-expert** - Comprehensive testing strategy and quality metrics
  - Generates: Test types, coverage targets, critical test paths
- **test-automator** - Test automation frameworks and CI/CD integration
  - Generates: Framework choices, test structure, automation approach
- **performance-engineer** - Performance testing and optimization
  - Generates: Benchmarks, load scenarios, acceptance criteria
- **accessibility-tester** - Accessibility compliance testing

### 🛠️ Specialized Agents

**Infrastructure & DevOps**:
- **devops-engineer** - CI/CD, containerization, cloud platforms
- **sre-engineer** - Site reliability and operational excellence
- **cloud-architect** - Multi-cloud strategies and architectures
- **kubernetes-specialist** - Container orchestration
- **terraform-engineer** - Infrastructure as code
- **platform-engineer** - Internal developer platforms

**Other Specialists**:
- **data-scientist** - Statistical analysis and ML
- **data-engineer** - Data pipelines and ETL processes
- **payment-integration** - Payment gateways and PCI compliance
- **websocket-engineer** - Real-time communication architectures

### 🎯 Skills (Use During Implementation)

**IMPORTANT**: Reference these skills in your plans for implementation phase:

- **/frontend-design** - Create distinctive, production-grade frontend interfaces (SKILL)
  - Use when: Building UI components, landing pages, dashboards, layouts
  - Generates: Creative, polished code avoiding generic AI aesthetics
- **/commit** - Create proper git commits with conventional commit messages
- **/research** - Deep dive research on specific topics during implementation
- **/learn** - Capture learnings and discoveries from implementation
- **/journal** - Document session progress

### 📊 Context Window Management Strategy

**To manage context efficiently during planning:**

1. **Use sub-agents for research** - Keeps main context clean and focused
2. **Read files FULLY** - Use Read tool without limit/offset for complete understanding
3. **Spawn agents in PARALLEL** - Run independent research tasks simultaneously
4. **Use TodoWrite** - Track planning tasks and maintain focus
5. **Multi-file plans for complex features** - Split into master + phase files (>400 lines)
6. **Reference agents in plans** - Guide future implementation with agent recommendations

### 🚀 Parallel Execution Best Practices

**ALWAYS spawn independent research agents in parallel for maximum efficiency:**

✅ **Good** - Parallel execution (single message with multiple Task calls):
```
Spawn simultaneously:
- codebase-locator (find files)
- codebase-analyzer (understand implementation)
- thoughts-locator (find docs)
- codebase-pattern-finder (find similar code)
```

❌ **Bad** - Sequential execution when tasks are independent:
```
1. Run codebase-locator, wait
2. Run codebase-analyzer, wait
3. Run thoughts-locator, wait
4. Run codebase-pattern-finder, wait
```

**Key Rule**: If tasks don't depend on each other's results, run them in parallel!

---

## Initial Response

When this command is invoked, determine the mode and check for existing plans:

### Mode Detection

1. **Work Item Mode** (`--work work-NNNN` provided):
   - Plans are associated with a work item
   - Create plans in `docs/work/work-NNNN/plans/`
   - Update work manifest with plan artifacts
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

### Step 7: Update Work Manifest (If Work Item Mode)

1. **Add new phase to artifacts list**:
   ```markdown
   ### Plans
   - [Master Plan](./plans/master.md)
   - [Phase 1: {Title}](./plans/phase-1.md)
   - [Phase 2: {Title}](./plans/phase-2.md)
   - [Phase 2.1: {New Title}](./plans/phase-2.1.md) ({date}) <!-- NEW -->
   - [Phase 3: {Title}](./plans/phase-3.md)
   ```

2. **Update change log**: `{date}: Added phase-2.1 to implementation plan`

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
   - Check if manifest has `**Epic**: epic-NNNN`
   - If yes, check for `docs/work/work-NNNN/upstream/from-*.md` files
   - Read EACH upstream message for cross-repo constraints and interface contracts
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

### Step 3: Update Work Manifest (If Using Work Item)

**If `--work work-NNNN` was provided**:

1. **Add Plan Artifacts** to manifest (`docs/work/work-NNNN/manifest.md`):
   - Under `## Artifacts > ### Plans` section
   - List master plan and all phase plans with **relative links**:
     - `[Master Plan](./plans/master.md)`
     - `[Phase 1: {Title}](./plans/phase-1.md)`
     - `[Phase 2: {Title}](./plans/phase-2.md)`
   - Include creation date

2. **Update Status**:
   - Change from: `📝 Requirements`
   - Change to: `🎨 Planning`

3. **Update Workflow Progress**:
   - Mark: `[x] Planning`

4. **Add Change Log Entry**:
   - `{YYYY-MM-DD}: Implementation plan created`

**If standalone**:
- Skip manifest updates
- Plans are self-contained

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

3. **Template A: Single-File Plan Structure**

````markdown
# [Feature/Task Name] Implementation Plan

**Document ID**: NNNN
**Created**: [Current Date]
**Type**: Implementation Plan
**Status**: Ready for Implementation

## Overview

[Brief description of what we're implementing and why]

## Documentation Chain

### Source Documents

- **Work Item**: work-NNNN
- **Issues**: [../issues/NNNN-\*-issue.md](../issues/NNNN-*-issue.md)
- **Requirements**: [../requirements/NNNN-\*-req.md](../requirements/NNNN-*-req.md)
- **Research**: [../research/NNNN-\*-research.md](../research/NNNN-*-research.md)

## Architecture Review

**Reviewed By**: architect-reviewer
**Status**: ✅ Approved / ⚠️ Approved with Recommendations / ❌ Needs Revision

**Key Findings**:
- [Architectural validation finding]
- [Pattern or approach confirmation]

**Recommendations**:
- [Architectural recommendation incorporated]

## Current State Analysis

[What exists now, what's missing, key constraints discovered]

## Desired End State

[A Specification of the desired end state after this plan is complete, and how to verify it]

### Key Discoveries:

- [Important finding with file:line reference]
- [Pattern to follow]
- [Constraint to work within]

## Plan Validation

**Validated By**: [List of domain expert agents used]
**Validation Date**: [timestamp]
**Status**: ✅ Approved / ⚠️ Approved with Recommendations

**Domain Reviews**:
- **Code Quality** (code-reviewer): [Status and key feedback]
- **[Domain]** (reviewed by [agent]): [Status and key feedback]

**Incorporated Recommendations**:
- [x] [Recommendation that was incorporated]
- [ ] [Optional enhancement to consider]

## What We're NOT Doing

[Explicitly list out-of-scope items to prevent scope creep]

## Implementation Approach

[High-level strategy and reasoning]

## Recommended Tools for Implementation

**IMPORTANT**: These recommendations MUST be populated with SPECIFIC agents based on the plan content!

**Skills to Use**:
- `/commit` - For creating proper git commits with conventional commit messages
- [Add specific skills based on plan domains, e.g.:]
  - `/frontend-design` - For creating production-grade UI components (if UI work)
  - `/learn` - To capture learnings and discoveries during implementation

**Agents to Leverage During Implementation**:
- **[PRIMARY-DOMAIN-AGENT]** - Main implementation agent for this plan's domain
  - Examples: backend-developer, frontend-developer, fullstack-developer, etc.
  - Use this agent for complex implementation tasks in phases
- **code-reviewer** - MANDATORY before finalizing each phase
  - Check: code quality, security, performance, maintainability
  - Run after implementing each phase, before marking complete
- **[DOMAIN-SPECIFIC-AGENTS]** - Specialists for specific aspects
  - Examples based on plan:
    - postgres-pro - For database schema and query optimization
    - api-designer - For API endpoint design validation
    - react-specialist - For React component patterns
    - security-engineer - For auth/security validation
    - performance-engineer - For performance-critical sections
    - ui-designer - For UI/UX consistency
    - accessibility-tester - For WCAG compliance
    - qa-expert - For test strategy validation

**When to Use Each Agent**:
1. **During Implementation**: Use domain specialists for complex tasks
2. **Before Completion**: ALWAYS run code-reviewer
3. **For Specific Concerns**: Use specialists (security, performance, etc.)

**Example Agent Usage in Implementation**:
```bash
# Phase 1 implementation
/implement_plan docs/work/work-NNNN/plans/phase-1.md
# Agent will use: backend-developer for implementation
# Then: code-reviewer before marking phase complete

# If database changes:
# Also leverage: postgres-pro for schema validation
```

## Phase 1: [Descriptive Name]

### Overview

[What this phase accomplishes]

### Changes Required:

#### 1. [Component/File Group]

**File**: `path/to/file.ext`
**Changes**: [Summary of changes]

```[language]
// Specific code to add/modify
```
````

### Success Criteria:

#### Automated Verification:

- [ ] Tests pass: `pnpm test`
- [ ] Types check: `pnpm run typecheck`
- [ ] Linting passes: `pnpm lint`
- [ ] Build succeeds: `pnpm build`

#### Manual Verification:

- [ ] Feature works as expected when tested via UI
- [ ] Performance is acceptable under load
- [ ] Edge case handling verified manually
- [ ] No regressions in related features

---

## Phase 2: [Descriptive Name]

[Similar structure with both automated and manual success criteria...]

---

## Testing Strategy

### Unit Tests:

- [What to test]
- [Key edge cases]

### Integration Tests:

- [End-to-end scenarios]

### Manual Testing Steps:

1. [Specific step to verify feature]
2. [Another verification step]
3. [Edge case to test manually]

## Performance Considerations

[Any performance implications or optimizations needed]

## Migration Notes

[If applicable, how to handle existing data/systems]

## References

- **Source Ticket/Requirement**: [Link to originating document]
- **Research**: [Link to research documents]
- **Similar Implementation**: [file:line references]
- **Codebase Patterns**: [Patterns to follow]

## Next Steps

Use `/implement_plan docs/work/work-NNNN/plans/master.md` to begin implementation.

---

**Document ID**: NNNN
**Cross-References**: Auto-updated by workflow commands
````

---

**Template B: Master Plan Structure (for Multi-File Plans)**

````markdown
# [Feature/Task Name] Implementation Plan (Master)

**Document ID**: NNNN
**Created**: [Current Date]
**Type**: Master Implementation Plan
**Status**: Ready for Implementation

## Overview

[Brief description of the overall feature and why it's being implemented]

## Documentation Chain

### Source Documents

- **Work Item**: work-NNNN
- **Issues**: [../issues/NNNN-*-issue.md](../issues/NNNN-*-issue.md)
- **Requirements**: [../requirements/NNNN-*-req.md](../requirements/NNNN-*-req.md)
- **Research**: [../research/NNNN-*-research.md](../research/NNNN-*-research.md)

## Architecture Review

**Reviewed By**: architect-reviewer
**Status**: ✅ Approved / ⚠️ Approved with Recommendations

**Key Architectural Decisions**:
- [Major architectural decision 1]
- [Major architectural decision 2]

## Plan Organization

This is a complex implementation split into multiple phase-specific plans for maintainability:

### Phase Plans:

1. **[Phase 1 Name]** → [./phase-1.md](./phase-1.md)
   - Status: ⏳ Not Started / 🔄 In Progress / ✅ Complete
   - Summary: [1-2 sentence summary]
   - Key deliverables: [bullet list]

2. **[Phase 2 Name]** → [./phase-2.md](./phase-2.md)
   - Status: ⏳ Not Started / 🔄 In Progress / ✅ Complete
   - Summary: [1-2 sentence summary]
   - Key deliverables: [bullet list]

3. **[Phase N Name]** → [./phase-N.md](./phase-N.md)
   - Status: ⏳ Not Started / 🔄 In Progress / ✅ Complete
   - Summary: [1-2 sentence summary]
   - Key deliverables: [bullet list]

## Current State Analysis

[High-level analysis of what exists now and what's missing]

## Desired End State

[Specification of the desired end state after all phases are complete]

## Implementation Strategy

[High-level approach explaining why phases are organized this way]

### Phase Dependencies:

- Phase 1 → Phase 2: [Dependency description]
- Phase 2 → Phase 3: [Dependency description]

## What We're NOT Doing

[Explicitly list out-of-scope items]

## Recommended Tools for Implementation

**Skills to Use**:
- `/frontend-design` - [If applicable: For creating production-grade UI components and layouts]
- `/commit` - For creating proper git commits with conventional messages
- `/learn` - To capture learnings and discoveries during implementation
- `/journal` - To document session progress

**Agents to Leverage by Phase**:

**IMPORTANT**: Populate with SPECIFIC agents based on each phase's work!

- **Phase 1: [Phase Name]**
  - Primary: [domain-specialist] - [Why for this phase]
  - Validation: [specific-validator] - [What to validate]
  - Quality: code-reviewer - Before completion

- **Phase 2: [Phase Name]**
  - Primary: [domain-specialist] - [Why for this phase]
  - Validation: [specific-validator] - [What to validate]
  - Quality: code-reviewer - Before completion

- **All Phases**:
  - **code-reviewer** - MANDATORY before finalizing each phase

**Domain-to-Agent Mapping** (use as reference):
- Backend/API → backend-developer, api-designer, postgres-pro
- Frontend/UI → frontend-developer, react-specialist, ui-designer
- Full-stack → fullstack-developer
- Database → postgres-pro, database-optimizer
- Security → security-engineer
- Performance → performance-engineer
- Testing → qa-expert, test-automator

## Overall Success Criteria

### Automated Verification:
- [ ] All phase-specific tests pass
- [ ] Integration tests pass: `pnpm test:integration`
- [ ] Full build succeeds: `pnpm build`

### Manual Verification:
- [ ] End-to-end feature works as expected
- [ ] Performance meets requirements
- [ ] No regressions in existing functionality

## Risk Management

- **Risk 1**: [Description and mitigation]
- **Risk 2**: [Description and mitigation]

## References

- **Source Ticket/Requirement**: [Link]
- **Research**: [Links to research documents]

## Progress Tracking

| Phase | Status | Started | Completed | Notes |
|-------|--------|---------|-----------|-------|
| Phase 1 | ⏳ | - | - | |
| Phase 2 | ⏳ | - | - | |
| Phase N | ⏳ | - | - | |

---

**Document ID**: NNNN
**Type**: Master Plan
**Phase Plans**: NNNN-phase-1, NNNN-phase-2, ...
````

---

**Template C: Phase Plan Structure (for individual phases in Multi-File Plans)**

````markdown
# [Feature Name] - Phase [N]: [Phase Name]

**Document ID**: NNNN
**Phase**: N of [Total]
**Created**: [Current Date]
**Type**: Phase Implementation Plan
**Status**: Ready for Implementation

## Master Plan

📋 **Master Plan**: [master.md](./master.md)

## Phase Overview

[Detailed description of what this phase accomplishes and why it's a separate phase]

## Prerequisites

**Dependencies from previous phases**:
- [ ] Phase [N-1] completed: [specific deliverable needed]
- [ ] [Other prerequisite]

## Phase Scope

### What This Phase Delivers:
- [Specific deliverable 1]
- [Specific deliverable 2]

### What This Phase Does NOT Include:
- [Out of scope item 1]
- [Out of scope item 2]

## Recommended Tools for This Phase

**IMPORTANT**: Populate with SPECIFIC tools based on this phase's work!

**Skills**:
- `/commit` - For proper git commits
- [Add phase-specific skills]:
  - `/frontend-design` - If building UI components in this phase
  - `/learn` - To capture discoveries specific to this phase

**Agents for This Phase**:

**Primary Implementation**:
- **[domain-specialist]** - Main agent for implementing this phase
  - Example: backend-developer, frontend-developer, postgres-pro
  - Use when: [Specific scenarios in this phase]
  - Responsible for: [What this agent handles]

**Validation & Quality**:
- **code-reviewer** - MANDATORY before completing this phase
  - Check: code quality, security, best practices
  - When: After all implementation, before marking phase complete

**Specialist Support** (use as needed):
- **[specialist-1]** - [When to use in this phase]
  - Example: postgres-pro for database schema validation
- **[specialist-2]** - [When to use in this phase]
  - Example: security-engineer for auth implementation

**Phase-Specific Agent Recommendations**:
Based on this phase's focus on [phase focus], prioritize:
1. [Specific agent] - For [specific task]
2. [Specific agent] - For [specific task]
3. code-reviewer - Final quality check (MANDATORY)

## Implementation Details

### Changes Required

#### 1. [Component/Area Name]

**Files**:
- `path/to/file1.ext`
- `path/to/file2.ext`

**Changes**:
[Detailed description of changes]

```[language]
// Specific code examples
```

**Rationale**: [Why these changes]

#### 2. [Another Component/Area]

[Similar structure...]

## Testing Strategy

### Unit Tests

**Files to create/update**:
- `path/to/test.spec.ext`

**Test cases**:
- [ ] [Test case 1]
- [ ] [Test case 2]

### Integration Tests

[Phase-specific integration tests]

## Success Criteria

### Automated Verification:
- [ ] Phase-specific tests pass: `pnpm test:phase-N`
- [ ] Type checking passes: `pnpm typecheck`
- [ ] Linting passes: `pnpm lint`
- [ ] Build succeeds: `pnpm build`

### Manual Verification:
- [ ] [Phase-specific manual check 1]
- [ ] [Phase-specific manual check 2]

## Phase Completion Checklist

- [ ] All code changes implemented
- [ ] All tests written and passing
- [ ] Code reviewed (use code-reviewer agent)
- [ ] Documentation updated
- [ ] Migration scripts tested (if applicable)
- [ ] Success criteria met
- [ ] Phase plan status updated to ✅ Completed
- [ ] Master plan phase listing and progress table updated
- [ ] Work manifest updated with phase completion

## Next Phase

After completing this phase, proceed to:
- **Phase [N+1]**: [phase-[N+1].md](./phase-[N+1].md)

---

**Document ID**: NNNN-phase-N
**Master Plan**: NNNN
**Previous Phase**: NNNN-phase-[N-1] | **Next Phase**: NNNN-phase-[N+1]
````

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

## Important Guidelines

1. **Be Skeptical**:
- Question vague requirements
- Identify potential issues early
- Ask "why" and "what about"
- Don't assume - verify with code

2. **Be Interactive**:
- Don't write the full plan in one shot
- Get buy-in at each major step
- Allow course corrections
- Work collaboratively

3. **Be Thorough**:
- Read all context files COMPLETELY before planning
- Research actual code patterns using parallel sub-tasks
- Include specific file paths and line numbers
- Write measurable success criteria with clear automated vs manual distinction
- automated steps should use `pnpm` or `make` commands when possible - for example `pnpm lint` instead of individual tool commands

4. **Be Practical**:
- Focus on incremental, testable changes
- Consider migration and rollback
- Think about edge cases
- Include "what we're NOT doing"

5. **Track Progress**:
- Use TodoWrite to track planning tasks
- Update todos as you complete research
- Mark planning tasks complete when done

6. **No Open Questions in Final Plan**:
- If you encounter open questions during planning, STOP
- Research or ask for clarification immediately
- Do NOT write the plan with unresolved questions
- The implementation plan must be complete and actionable
- Every decision must be made before finalizing the plan

7. **ENHANCED: Complete Documentation Chain**:
- Auto-create missing tickets/requirements when given descriptions
- Use sequential numbering for chronological ordering
- Ensure proper cross-referencing between all documents
- Follow Research → Plan → Ready for Implementation workflow

8. **Multi-File Plan Requirements** (CRITICAL):
- **MUST split for 4+ phases**: Plans with 4 or more phases REQUIRE separate files
- **MUST split if >400 lines**: If estimated length exceeds 400 lines, MUST use multi-file
- **Create separate physical files**: Each phase gets its own .md file, NOT sections in one file
- **Master plan stays high-level**: Master plan is overview only, NO detailed implementation
- **Phase files have details**: All implementation details go in individual phase files (100-300 lines each)
- **Enforce file limits**: If any phase file exceeds 300 lines, split that phase into sub-phases
- **Master plan is source of truth**: All phase status tracking happens in master plan
- **Bidirectional links**: Each phase links to master, master links to all phases
- **Consistent naming**: NNNN-feature-name-phase-N-plan.md (where N is phase number)

9. **Agent Recommendation Requirements** (CRITICAL - NEW):
- **MANDATORY population**: ALL plans MUST have specific agent recommendations
- **NO placeholders allowed**: Replace [agent-name], [domain-agent], etc. with actual agents
- **Domain-based selection**: Choose agents based on plan's technical domains
- **Minimum coverage**: Every plan needs code-reviewer + at least one domain specialist
- **Phase-specific guidance**: Each phase plan must specify which agents to use when
- **Validation required**: Run validation check (Step 4d) before finalizing
- **Actionable recommendations**: Include WHY and WHEN to use each agent

9. **Leverage Available Tools**:
- **ALWAYS check the Tools & Agents Reference section** at the beginning of this document
- Use appropriate agents for research, validation, and testing
- Reference relevant skills in plans for implementation guidance
- Spawn independent agents in PARALLEL for efficiency
- Include agent/skill recommendations in plan templates

## Success Criteria Guidelines

**Always separate success criteria into two categories:**

1. **Automated Verification** (can be run by execution agents):
- Commands that can be run: `make test`, `npm run lint`, etc.
- Specific files that should exist
- Code compilation/type checking
- Automated test suites

2. **Manual Verification** (requires human testing):
- UI/UX functionality
- Performance under real conditions
- Edge cases that are hard to automate
- User acceptance criteria

**Format example:**
```markdown
### Success Criteria:

#### Automated Verification:
- [ ] Database migration runs successfully: `make migrate`
- [ ] All unit tests pass: `go test ./...`
- [ ] No linting errors: `golangci-lint run`
- [ ] API endpoint returns 200: `curl localhost:8080/api/new-endpoint`

#### Manual Verification:
- [ ] New feature appears correctly in the UI
- [ ] Performance is acceptable with 1000+ items
- [ ] Error messages are user-friendly
- [ ] Feature works correctly on mobile devices
```

## Agent Selection Decision Tree

**Quick reference for which agent to use:**

```
Finding files? → codebase-locator
Understanding code? → codebase-analyzer
Similar implementations? → codebase-pattern-finder
Finding docs? → thoughts-locator
Doc analysis? → thoughts-analyzer
External research? → web-search-researcher

Architecture review? → architect-reviewer
API design? → api-designer
Database design? → postgres-pro
UI/UX design? → ui-designer
Code quality? → code-reviewer (ALWAYS before finalizing)
Security? → security-engineer

Frontend patterns? → react-specialist, frontend-developer
Backend patterns? → backend-developer
Full-stack? → fullstack-developer

Testing strategy? → qa-expert
Test automation? → test-automator
Performance? → performance-engineer
Accessibility? → accessibility-tester
```

## Common Patterns

### For Database Changes:

- Start with schema/migration
- Add store methods
- Update business logic
- Expose via API
- Update clients

### For New Features:

- Research existing patterns first
- Start with data model
- Build backend logic
- Add API endpoints
- Implement UI last (use /frontend-design skill)

### For Refactoring:

- Document current behavior
- Plan incremental changes
- Maintain backwards compatibility
- Include migration strategy

## Sub-task Spawning Best Practices

When spawning research sub-tasks:

1. **Spawn multiple tasks in PARALLEL** for efficiency
2. **Each task should be focused** on a specific area
3. **Provide detailed instructions** including:
   - Exactly what to search for
   - Which directories to focus on
   - What information to extract
   - Expected output format
4. **Be EXTREMELY specific about directories**:
   - Focus on relevant directories like `apps/`, `libs/`, or specific components
   - Be specific about component locations within your Nx monorepo structure
   - Include the full path context in your prompts
5. **Specify read-only tools** to use
6. **Request specific file:line references** in responses
7. **Wait for all tasks to complete** before synthesizing
8. **Verify sub-task results**:
   - If a sub-task returns unexpected results, spawn follow-up tasks
   - Cross-check findings against the actual codebase
   - Don't accept results that seem incorrect

Example of spawning multiple tasks in PARALLEL:

```
Single message with multiple Task tool calls:
- Task 1: codebase-locator (find database-related files)
- Task 2: codebase-analyzer (understand auth flow)
- Task 3: codebase-pattern-finder (find similar API endpoints)
- Task 4: thoughts-locator (find existing documentation)
```

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
8. Updates manifest with plan artifacts
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
11. Updates: manifest.md with new phase artifact

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
9. Updates manifest with all plan artifacts

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
