# planv0 — Plan-File Templates

These are the plan-file templates referenced by `/planv0` (Template A: single-file plan; Template B: master plan for multi-file plans; Template C: individual phase plan). Read this file when you author the plan files; the CORE command (`.claude/commands/planv0.md`) points here on demand rather than inlining these.

**Template A: Single-File Plan Structure**

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
- [ ] Phase plan status updated to ✅ Completed (plan-internal status)
- [ ] Master plan phase listing and progress table updated (plan-internal status)
- [ ] Work-item state recorded via event — append `phase_done`/`status_changed` with `scripts/wlog.sh` and run `scripts/wrender.sh "$WD"`; never hand-edit `manifest.md`

## Next Phase

After completing this phase, proceed to:
- **Phase [N+1]**: [phase-[N+1].md](./phase-[N+1].md)

---

**Document ID**: NNNN-phase-N
**Master Plan**: NNNN
**Previous Phase**: NNNN-phase-[N-1] | **Next Phase**: NNNN-phase-[N+1]
````
