# Work Item Management Command

**Purpose**: Create and manage unified work items that group related research, requirements, plans, and implementation artifacts.

## Command Usage

```bash
/work "Natural language prompt"       # Auto-create work + research + requirements
/work show work-NNNN                  # Show work item details
/work list                            # List all work items
/work update work-NNNN --status X     # Update work status
```

## Behavior

### When user runs: `/work "Natural language prompt or problem description"`

This is the **PRIMARY and RECOMMENDED** usage. User provides a freeform description of what they want to build or problem they're facing.

**Examples**:
- `/work "I want to add OAuth social login to the app"`
- `/work "Running into performance issues with database queries"`
- `/work "Want to build a payment integration with Stripe"`

You MUST execute this **3-phase automatic workflow**:

#### Phase 1: Create Work Item

1. **Determine Next Work ID**
   - Use Glob to find existing work manifests: `docs/work/work-*-manifest.md`
   - Extract highest NNNN and increment by 1
   - Format as `work-NNNN` (zero-padded to 4 digits)

2. **Extract Title from Prompt**
   - Analyze the user's prompt
   - Generate concise title (3-8 words)
   - Example: "I want to add OAuth login" → "OAuth Social Login Integration"

3. **Create Work Manifest**
   - File: `docs/work/work-NNNN-manifest.md`
   - Use template below
   - Include the user's original prompt in Description section
   - Status: 🎯 Proposed

4. **Update Work Index**
   - Read `docs/work/index.md`
   - Add new work item entry to table
   - Sort by work ID descending (newest first)

5. **Notify User**
   - Output: "✅ Created work-NNNN: {Generated Title}"
   - Output: "📋 Original Request: {User's prompt}"
   - Output: "🔍 Starting automatic research and requirements gathering..."

#### Phase 2: Automatic Research

**Immediately after creating work item**, you MUST:

1. **Spawn Research Agent**
   - Use Task tool with subagent_type appropriate for the domain
   - Pass the user's prompt as research context
   - Include work ID in the task: `--work work-NNNN`
   - Research should cover:
     - Understanding the problem/requirement
     - Exploring existing codebase for related patterns
     - Investigating best practices and approaches
     - Analyzing technology options if applicable

2. **Create Research Document**
   - Folder: `docs/work/work-NNNN/research/`
   - Filename: `docs/work/work-NNNN/research/0001-{slug}-research.md`
   - Content follows standard research document structure
   - References work item: `Work Item: work-NNNN`
   - **IMPORTANT**: Initial research auto-created, more can be added with `/research --work work-NNNN`

3. **Update Work Manifest**
   - Add research document to Artifacts > Research section: `./research/0001-{slug}-research.md`
   - Update status: 🎯 Proposed → 📚 Researching → 📝 Requirements Ready (after research completes)
   - Mark workflow progress: `[x] Research`
   - Add change log entry

#### Phase 3: Automatic Requirements

**Immediately after research completes**, you MUST:

1. **Spawn Requirements Agent**
   - Use Task tool with appropriate agent (ux-researcher, architect-reviewer, qa-expert)
   - Base requirements on:
     - User's original prompt
     - Research findings (from `docs/work/work-NNNN/research.md`)
     - Existing codebase patterns discovered
   - Include work ID context

2. **Create Requirements Document**
   - Folder: `docs/work/work-NNNN/requirements/`
   - Filename: `docs/work/work-NNNN/requirements/0001-{slug}-req.md`
   - Content follows standard requirements structure:
     - Overview and objectives
     - Functional requirements
     - Non-functional requirements
     - User stories / use cases
     - Acceptance criteria
     - Constraints and assumptions
   - References work item: `Work Item: work-NNNN`
   - References research: Link to relevant research docs
   - **IMPORTANT**: Initial requirements auto-created, more can be added with `/new_req --work work-NNNN`

3. **Update Work Manifest**
   - Add requirements document to Artifacts > Requirements section: `./requirements/0001-{slug}-req.md`
   - Update status: 📝 Requirements (requirements ready for review)
   - Mark workflow progress: `[x] Requirements`
   - Add change log entry

#### Phase 4: Return Control to User

After both research and requirements are complete:

1. **Present Summary**
   - Output: "✅ Research completed: docs/work/work-NNNN/research/0001-{slug}-research.md"
   - Output: "✅ Requirements documented: docs/work/work-NNNN/requirements/0001-{slug}-req.md"
   - Output: ""
   - Output: "📊 Work Item Status: 📝 Requirements (Ready for Planning)"

2. **Request User Review**
   - Output: "Please review the research and requirements documents in docs/work/work-NNNN/"
   - Output: "You can add more research or requirements with:"
   - Output: "  /research --work work-NNNN \"Additional research topic\""
   - Output: "  /new_req --work work-NNNN \"Additional requirements\""
   - Output: ""
   - Output: "When you're ready to proceed, run:"
   - Output: "`/planv0 --work work-NNNN`"
   - Output: ""
   - Output: "This will create an implementation plan based on ALL research and requirements."

3. **Support Iteration**
   - User may ask questions, request changes to research or requirements
   - Update documents based on feedback
   - Only proceed to planning when user explicitly runs `/planv0 --work work-NNNN`

### When user runs: `/work show work-NNNN`

You MUST:
1. Read the manifest file
2. Display formatted work item details
3. Show linked artifacts and their status
4. List related journal sessions

### When user runs: `/work list`

You MUST:
1. Read `docs/work/index.md`
2. Display table of all work items with status
3. Highlight active work items (In Progress status)

### When user runs: `/work update work-NNNN --status NEW_STATUS`

You MUST:
1. Read manifest file
2. Update Status field
3. Add entry to Change Log
4. Save manifest

## Work Manifest Template

```markdown
# Work Item: work-NNNN - {Generated Title}

**Status**: 🎯 Proposed → 📚 Researching → 📝 Requirements
**Created**: {YYYY-MM-DD}
**Last Updated**: {YYYY-MM-DD}
**Owner**: {User/Team}
**Priority**: {TBD - to be determined during research}
**Estimated Effort**: {TBD - to be determined during planning}

## Original Request

{User's original natural language prompt - exactly as provided}

## Description

{Concise description based on research findings - populated after research completes}

## Workflow Progress

- [ ] Research
- [ ] Requirements
- [ ] Planning
- [ ] Implementation
- [ ] Validation
- [ ] Deployment

## Artifacts

### Research
- [0001: Initial Research](./research/0001-{slug}-research.md) (auto-created)
- Add more with `/research --work work-NNNN "topic"`

### Requirements
- [0001: Initial Requirements](./requirements/0001-{slug}-req.md) (auto-created)
- Add more with `/new_req --work work-NNNN "topic"`

### Issues
- Add with `/new_issue --work work-NNNN "issue description"`

### Plans
- [Master Plan](./plans/master.md) (when created)
- Phase plans listed here as created

### Implementation
- [Implementation Status](./implementation/status.md) (when started)

## Journal Sessions

{Auto-populated by /journal command}

## Key Decisions

{Add as work progresses}

## Dependencies

{Add as discovered}

## Change Log

- {YYYY-MM-DD}: Work item created
```

## Status Values

- 🎯 **Proposed**: Work item created, needs research
- 📚 **Researching**: Research in progress
- 📝 **Requirements**: Gathering/documenting requirements
- 🎨 **Planning**: Creating implementation plan
- 🔄 **In Implementation**: Active development
- ✅ **Completed**: Work finished and deployed
- 🔴 **Blocked**: Waiting on dependencies
- ⏸️ **On Hold**: Paused for later
- ❌ **Cancelled**: Will not be implemented

## Work Index Template

Create `docs/work/index.md` with:

```markdown
# Work Items Registry

Last Updated: {Auto-update on each change}

## Active Work Items

| ID | Title | Status | Created | Artifacts |
|----|-------|--------|---------|-----------|
| work-0001 | Example Feature | 🔄 In Implementation | 2026-01-02 | R, Req, P |

## Completed Work Items

{Move here when status = Completed}

## Cancelled/On Hold

{Move here when status = Cancelled/On Hold}

---

**Legend**: R=Research, Req=Requirements, P=Plans, I=Implementation
```

## Tools Available

- **Read**: Read existing manifests and index
- **Write**: Create new manifests and index
- **Edit**: Update existing manifests
- **Glob**: Find existing work items for numbering
- **Grep**: Search work content (if needed)

## Integration with Other Commands

### Automatic Integration (Primary Workflow)

When `/work "prompt"` is used, it **automatically**:
1. Creates work item folder: `docs/work/work-NNNN/`
2. Creates manifest: `docs/work/work-NNNN/manifest.md`
3. Creates research folder and document: `docs/work/work-NNNN/research/0001-*.md`
4. Creates requirements folder and document: `docs/work/work-NNNN/requirements/0001-*.md`
5. Updates manifest with artifact links (relative paths)
6. Returns control to user for review

**All artifacts organized under one folder**: `docs/work/work-NNNN/`

### Additional Research (Manual Trigger)

User runs `/research --work work-NNNN "research topic"` to add more research. The research command MUST:
1. **Find next research number** in `docs/work/work-NNNN/research/`
2. **Create new research document** - `docs/work/work-NNNN/research/NNNN-{slug}-research.md`
3. **Update work manifest** - Add to Research artifacts section
4. **Run research agents** as needed

### Additional Requirements (Manual Trigger)

User runs `/new_req --work work-NNNN "requirements topic"` to add more requirements. The new_req command MUST:
1. **Find next requirements number** in `docs/work/work-NNNN/requirements/`
2. **Create new requirements document** - `docs/work/work-NNNN/requirements/NNNN-{slug}-req.md`
3. **Update work manifest** - Add to Requirements artifacts section
4. **Run validation agents** as needed

### Additional Issues (Manual Trigger)

User runs `/new_issue --work work-NNNN "issue description"` to track issues. The new_issue command MUST:
1. **Find next issue number** in `docs/work/work-NNNN/issues/`
2. **Create new issue document** - `docs/work/work-NNNN/issues/NNNN-{slug}-issue.md`
3. **Update work manifest** - Add to Issues artifacts section

### Manual Planning Trigger

User manually runs `/planv0 --work work-NNNN` when ready. The planv0 command MUST:
1. **Read Work Manifest** - `docs/work/work-NNNN/manifest.md`
   - Get context, title, original request
2. **Read ALL Research Documents** - `docs/work/work-NNNN/research/*.md`
   - Understand problem space from all research
   - Review technology options analyzed
   - Consider architectural approaches explored
3. **Read ALL Requirements Documents** - `docs/work/work-NNNN/requirements/*.md`
   - Extract functional requirements from all docs
   - Extract non-functional requirements from all docs
   - Review acceptance criteria
   - Understand constraints
4. **Create Implementation Plan** - Based on ALL research + ALL requirements
   - Create `docs/work/work-NNNN/plans/` folder
   - Master plan: `docs/work/work-NNNN/plans/master.md`
   - Phase plans: `docs/work/work-NNNN/plans/phase-N.md`
   - Link back to research and requirements (relative paths)
   - Address all requirements from all documents with traceability
5. **Update Work Manifest**
   - Add plan artifacts (relative paths: `./plans/master.md`, etc.)
   - Update status: 📝 Requirements → 🎨 Planning
   - Mark workflow progress

### Manual Implementation Trigger

User manually runs `/implement_plan <plan-path>` where <plan-path> is the path to the master plan file.

**Example**: `/implement_plan docs/work/work-0001/plans/master.md`

The implement_plan command MUST:
1. **Read the plan file** provided as parameter
2. **Extract work ID** from plan content (should contain `Work Item: work-NNNN`)
3. **Read work manifest** - `docs/work/work-NNNN/manifest.md`
4. **Read all plan documents** in `docs/work/work-NNNN/plans/`
5. **Execute implementation** according to plan
6. **Create implementation status** - `docs/work/work-NNNN/implementation/status.md`
7. **Update manifest** with progress automatically (no --work parameter needed!)

### Important Notes

- Research and requirements are **automatic** (triggered by `/work "prompt"`)
- Planning is **manual** (triggered by `/planv0 --work work-NNNN`)
- Implementation is **manual** (triggered by `/implement_plan --work work-NNNN`)
- User reviews and provides feedback between each phase

## Examples

### Example 1: Automatic Research + Requirements

```bash
# User provides natural language prompt
/work "I want to add OAuth social login with Google and GitHub"

# System automatically:
# 1. Creates docs/work/work-0001/ folder structure
# 2. Creates docs/work/work-0001/manifest.md with title "OAuth Social Login Integration"
# 3. Creates docs/work/work-0001/research/ folder
# 4. Runs research (explores OAuth patterns etc.)
# 5. Creates docs/work/work-0001/research/0001-oauth-social-login-research.md
# 6. Creates docs/work/work-0001/requirements/ folder
# 7. Creates requirements based on research
# 8. Creates docs/work/work-0001/requirements/0001-oauth-requirements.md
# 9. Updates manifest.md with artifact links
# 10. Returns to user

# Output:
# ✅ Created work-0001: OAuth Social Login Integration
# 📋 Original Request: I want to add OAuth social login with Google and GitHub
# 🔍 Starting automatic research and requirements gathering...
# [research happens]
# ✅ Research completed: docs/work/work-0001/research/0001-oauth-social-login-research.md
# ✅ Requirements documented: docs/work/work-0001/requirements/0001-oauth-requirements.md
#
# 📊 Work Item Status: 📝 Requirements (Ready for Planning)
#
# All artifacts are in: docs/work/work-0001/
# You can add more research or requirements with:
#   /research --work work-0001 "Additional research topic"
#   /new_req --work work-0001 "Additional requirements"
#
# When ready, run: /planv0 --work work-0001
```

### Example 2: Adding More Research and Requirements

```bash
# After initial research, user realizes they need more investigation
/research --work work-0001 "Apple Sign-In integration details"
# Creates: docs/work/work-0001/research/0002-apple-signin-integration-research.md
# Updates: manifest.md

# Add security-specific requirements
/new_req --work work-0001 "Security and compliance requirements"
# Creates: docs/work/work-0001/requirements/0002-security-compliance-req.md
# Updates: manifest.md

# Add performance requirements
/new_req --work work-0001 "Performance requirements"
# Creates: docs/work/work-0001/requirements/0003-performance-req.md
# Updates: manifest.md
```

### Example 3: Review and Plan

```bash
# User reviews all research and requirements, confirms ready to plan

/planv0 --work work-0001
# Reads: docs/work/work-0001/manifest.md
# Reads ALL: docs/work/work-0001/research/*.md (all 2 research docs)
# Reads ALL: docs/work/work-0001/requirements/*.md (all 3 requirements docs)
# Creates: docs/work/work-0001/plans/master.md
# Creates: docs/work/work-0001/plans/phase-1.md (if multi-phase)
# Updates: docs/work/work-0001/manifest.md (status: Planning)
```

### Example 3: View Work Status

```bash
# Show work details
/work show work-0001
# → Displays manifest contents with all artifacts

# List all work
/work list
# → Shows table of all work items

# Update status manually (if needed)
/work update work-0001 --status "Blocked"
# → Updates manifest status and change log
```
