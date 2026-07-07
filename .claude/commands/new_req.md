# New Requirement

Create detailed requirements documents for complex features and specifications within a work item context.

> **Work-item state is an append-only event log** — register artifacts with `scripts/wlog.sh "$WD" artifact_added ...` then `scripts/wrender.sh "$WD"`; never hand-edit `manifest.md`. See [docs/dev/decisions/append-only-work-event-log.md](../../docs/dev/decisions/append-only-work-event-log.md).

## 🚧 Repo isolation — no cross-repo reads or edits (MANDATORY)

This skill runs inside a **single repo** that in the PlatformSmith product has **no filesystem access to sibling repos**. Enforce it here and in every sub-agent this skill spawns:

- **Never** `Read`/`Grep`/`Glob`/`Edit` any file **outside this repo** (another repo's working tree). Your world is **this repo only** — scope every research sub-agent to this repo too.
- **Cross-repo knowledge** comes *only* from the local **folded KB** at `docs/kb/peers/<repo>/` (start at `docs/kb/index.md`) — the sole cross-repo research surface. Reading your own `docs/kb/peers/**` is allowed.
- If the KB is unclear on a **system-critical** fact, is a gap / `UNKNOWN`, or is contradicted by observed behavior → emit an A2A **relay** (the live ask-a-peer A2A channel — not a local script). Do **not** relay for routine confirmation.
- **Cross-repo edits are never allowed.** If a cross-repo read seems unavoidable, **stop and ask the human**. See [docs/dev/decisions/repo-isolation-kb-first-cross-repo.md](../../docs/dev/decisions/repo-isolation-kb-first-cross-repo.md).

## Resolving `--work` IDs

When the user passes `--work <id>` (e.g., `--work work-0027` or `--work work-2607010322-dark-mode`), the value is a **short reference**, never an index to compute. The directory may be the legacy short form (`work-NNNN`), the legacy slug form (`work-NNNN-MMDDHHMM-slug`), or the event-log form (`work-<YYMMDDHHMM>-<slug>`). Before any file read/write, **resolve the reference to the real directory by glob — never by arithmetic**:

1. **Try exact match** — Glob `docs/work/{arg}/`. If found, use it.
2. **Else glob with dash suffix** — Glob `docs/work/{arg}-*/` (matches the slug-suffixed forms).
3. **Else glob by slug fragment** — Glob `docs/work/*{arg}*/` for a bare slug reference.
4. **If exactly one match**, use that directory. If zero, error: "Work item {arg} not found." If multiple, error and list matches.

Throughout the rest of this document, `work-NNNN` / `$WD` is shorthand for the resolved work item directory. State for that item lives in `$WD/work.jsonl`; `$WD/manifest.md` is a generated projection of it.

## Work Item Context (Optional)

The `--work work-NNNN` parameter is **OPTIONAL** but recommended for organized requirement tracking.

**When `--work work-NNNN` is provided**:
1. Resolve the short ID to the actual directory (see above).
2. Read work manifest: `docs/work/work-NNNN/manifest.md` (a generated view — read for context, never edit it)
3. Read existing research documents in `docs/work/work-NNNN/research/` for context
4. Read existing requirements in `docs/work/work-NNNN/requirements/` to avoid duplication
5. Plan to create requirements in `docs/work/work-NNNN/requirements/NNNN-*.md`
6. Plan to register the artifact in the event log when complete (see "Register the Artifact in the Event Log" below) — do **not** plan to hand-edit the manifest

**When `--work` is NOT provided (Standalone Mode)**:
1. Create requirements in `docs/requirements/NNNN-*.md`
2. Use local numbering within `docs/requirements/` (highest existing NNNN + 1)
3. No event log, no manifest — and do not hand-maintain any global `docs/requirements/index.md` registry (it is derivable from the directory)
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

5. **Register the Artifact in the Event Log (If Using Work Item)**:

   **Do NOT hand-edit `manifest.md`.** Work-item state is an append-only event log (`work.jsonl`); the manifest is regenerated from it. With `$WD` = the resolved work item directory and `<rel-path>` = the artifact path *relative to `$WD`* (e.g. `requirements/NNNN-{slug}-req.md`):

   **If `--work work-NNNN` was provided**:
   - Append an `artifact_added` event:
     ```bash
     scripts/wlog.sh "$WD" artifact_added kind=requirements path=<rel-path> title="{Title}"
     ```
   - Move the item into the requirements state. Append this only when the item is not already in `requirements` or a later phase:
     ```bash
     scripts/wlog.sh "$WD" status_changed to=requirements
     ```
   - Regenerate the manifest from the log:
     ```bash
     scripts/wrender.sh "$WD"
     ```
   - `wrender.sh` folds these events into the manifest's Artifacts section, Change Log, Status, and Last Updated automatically. Never open `manifest.md` to edit any of these by hand.

   **If standalone**:
   - No event log and no manifest — the requirements document is self-contained.
   - Do not hand-maintain any global registry/index of `docs/requirements/`; it is derivable from the directory listing.

6. **Next Steps**:
   - **If work item mode**: Note that the artifact was registered in the event log and the manifest regenerated
   - **If standalone**: Note the requirements document location
   - Suggest adding more requirements with `/new_req` (with or without --work)
   - Suggest using `/research` for additional technical investigation
   - Suggest using `/planv0` for implementation planning

This command focuses specifically on creating comprehensive requirements for complex features and system specifications.
