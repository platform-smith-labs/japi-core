# New Requirement

Create detailed requirements documents for complex features and specifications within a work item context.

## Work Item Context (Optional)

The `--work work-NNNN` parameter is **OPTIONAL** but recommended for organized requirement tracking.

**When `--work work-NNNN` is provided**:
1. Read work manifest: `docs/work/work-NNNN/manifest.md`
2. Read existing research documents in `docs/work/work-NNNN/research/` for context
3. Read existing requirements in `docs/work/work-NNNN/requirements/` to avoid duplication
4. Plan to create requirements in `docs/work/work-NNNN/requirements/NNNN-*.md`
5. Plan to update work manifest when complete

**When `--work` is NOT provided (Standalone Mode)**:
1. Create requirements in `docs/requirements/NNNN-*.md`
2. Use global numbering (check all files in `docs/requirements/`)
3. No manifest updates
4. Self-contained requirements document

## Initial Response

When this command is invoked, respond with:

```
I'll help you create a new requirements document. What type of requirements are you defining?

1. Feature specification - Detailed functional requirements for a new feature
2. System requirements - Technical specifications for system behavior
3. API specification - Requirements for new API endpoints or services
4. Integration requirements - Requirements for connecting systems

Please describe the feature or system you need requirements for.
```

Then wait for the user's input.

## NUMBERING SCHEME

### Format: NNNN-descriptive-name-req.md

- **NNNN**: Sequential number (0001, 0002, etc.)
- **descriptive-name**: Kebab-case description of the requirement
- **req**: Type suffix

### Auto-Numbering Process:

**With work item (`--work work-NNNN`)**:
1. Use Glob to find files in `docs/work/work-NNNN/requirements/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-req.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits

**Standalone (no `--work`)**:
1. Use Glob to find files in `docs/requirements/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-req.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits

## Requirements Structure

**Location**:
- **With work item**: `docs/work/work-NNNN/requirements/NNNN-*.md`
- **Standalone**: `docs/requirements/NNNN-*.md`

Structure:

```markdown
# [Feature/Component] - Requirements

**Document ID**: NNNN
**Work Item**: work-NNNN (if applicable)
**Created**: [Current Date]
**Type**: Requirements
**Status**: [Draft/Ready for Planning/Approved]

## Overview

[Brief description of the feature and its purpose]

## Functional Requirements

### [Requirement Category]

- **REQ-001**: [Specific requirement statement]
- **REQ-002**: [Another requirement]

### [Another Category]

- **REQ-003**: [Requirement with priority]

## Non-Functional Requirements

### Performance

- **NFR-001**: [Performance requirement]

### Security

- **NFR-002**: [Security requirement]

### Usability

- **NFR-003**: [Usability requirement]

## User Stories

### [User Type]

- As a [user type], I want [goal] so that [benefit]
- As a [user type], I want [goal] so that [benefit]

## Acceptance Criteria

### [Feature Area]

- [ ] Given [context], when [action], then [outcome]
- [ ] Given [context], when [action], then [outcome]

## Constraints and Assumptions

### Technical Constraints

- [Constraint 1]
- [Constraint 2]

### Business Constraints

- [Constraint 1]
- [Constraint 2]

### Assumptions

- [Assumption 1]
- [Assumption 2]

## References

- **Work Item**: `work-NNNN`
- **Related Issues**: `../issues/NNNN-[name]-issue.md`
- **Research**: `../research/NNNN-[name]-research.md`
- **Implementation Plan**: [Will be linked when plan is created]
- **External Specifications**: [links]

## Approval

- [ ] Business stakeholder review
- [ ] Technical review
- [ ] Security review (if applicable)

---

**Document ID**: NNNN  
**Cross-References**: Auto-updated by workflow commands
```

## Naming Convention

- **Format**: `NNNN-descriptive-name-req.md`
- **Sequential Numbering**: Auto-increment from highest existing number within the work item
- **Descriptive Names**: Use kebab-case, be specific but concise
- **Type Suffix**: Always end with `-req.md`

Examples:

**Work Item work-0001**:
- `docs/work/work-0001/requirements/0001-multi-tenant-user-management-req.md`
- `docs/work/work-0001/requirements/0002-api-rate-limiting-req.md`
- `docs/work/work-0001/requirements/0003-oauth-integration-req.md`

## Examples

### Feature Specification

```markdown
# Multi-Tenant User Management Requirements

## Overview

System to support multiple organizations with isolated user data and permissions.

## Functional Requirements

### Organization Management

- **REQ-001**: System must support creating new organizations
- **REQ-002**: Each organization must have isolated user data
- **REQ-003**: Organization admins can manage their users

### User Authentication

- **REQ-004**: Users can belong to multiple organizations
- **REQ-005**: Authentication must include organization context
```

### API Specification

```markdown
# REST API Rate Limiting Requirements

## Overview

Implement rate limiting for API endpoints to prevent abuse and ensure fair usage.

## Functional Requirements

### Rate Limiting Rules

- **REQ-001**: API must enforce 100 requests per minute per user
- **REQ-002**: Different limits for different endpoint categories
- **REQ-003**: Premium users get higher rate limits

## Non-Functional Requirements

### Performance

- **NFR-001**: Rate limiting must add <10ms latency
- **NFR-002**: Must handle 10,000 concurrent users
```

## Process

1. **Gather Information**:
   - Understand the feature scope and complexity
   - Identify stakeholders and users
   - Define success criteria

2. **Create Requirements**:
   - Structure functional and non-functional requirements
   - Include user stories and acceptance criteria
   - Define constraints and assumptions

3. **Validate Requirements** (Enhancement - NEW):

   After drafting requirements, validate with specialists:

   a. **ux-researcher** - User stories and acceptance criteria:
      ```
      Use Task tool with subagent_type="ux-researcher":

      "Review these user stories and acceptance criteria:

      [User stories section]

      Validate:
      - Are user needs clearly captured?
      - Any missing user scenarios?
      - Are acceptance criteria testable?
      - Suggest UX improvements"
      ```

   b. **architect-reviewer** - Technical feasibility:
      ```
      Use Task tool with subagent_type="architect-reviewer":

      "Review technical feasibility of these requirements:

      [Requirements summary]

      Assess:
      - Architectural impact and feasibility
      - Technical constraints
      - Potential alternatives
      - Integration considerations"
      ```

   c. **qa-expert** - Testability review:
      ```
      Use Task tool with subagent_type="qa-expert":

      "Review testability of these requirements:

      [Acceptance criteria section]

      Check:
      - Are acceptance criteria testable?
      - Missing edge cases?
      - Suggested test scenarios?"
      ```

   d. **Incorporate feedback** into requirements document

4. **Save and Link**:
   - **With work item**: Save to `docs/work/work-NNNN/requirements/NNNN-*.md`
   - **Standalone**: Save to `docs/requirements/NNNN-*.md`
   - Link to any existing research documents

5. **Update Work Manifest (If Using Work Item)**:

   **If `--work work-NNNN` was provided**:
   - Read work manifest: `docs/work/work-NNNN/manifest.md`
   - Add requirements document to `## Artifacts > ### Requirements` section
     - Format: `- [NNNN: {Title}](./requirements/NNNN-{slug}-req.md) ({date})`
   - Add change log entry: `{date}: Added requirements document NNNN`
   - Save updated manifest

   **If standalone**:
   - Skip manifest updates
   - Requirements document is self-contained

6. **Next Steps**:
   - **If work item mode**: Note that manifest was updated
   - **If standalone**: Note the requirements document location
   - Suggest adding more requirements with `/new_req` (with or without --work)
   - Suggest using `/research` for additional technical investigation
   - Suggest using `/planv0` for implementation planning

This command focuses specifically on creating comprehensive requirements for complex features and system specifications.
