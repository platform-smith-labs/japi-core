# Research Command

Conduct thorough research and create research documents for analysis, investigation, and understanding within a work item context.

> **Work-item state is an append-only event log** — register artifacts with `scripts/wlog.sh "$WD" artifact_added ...` then `scripts/wrender.sh "$WD"`; never hand-edit `manifest.md`. See [docs/dev/decisions/append-only-work-event-log.md](../../docs/dev/decisions/append-only-work-event-log.md).

## Resolving `--work` IDs

When the user passes `--work <id>` (e.g., `--work work-0027` or `--work work-2607010322-dark-mode`), the value is a **short reference**, never an index to compute. The directory may be the legacy short form (`work-NNNN`), the legacy slug form (`work-NNNN-MMDDHHMM-slug`), or the event-log form (`work-<YYMMDDHHMM>-<slug>`). Before any file read/write, **resolve the reference to the real directory by glob — never by arithmetic**:

1. **Try exact match** — Glob `docs/work/{arg}/` (or `repos/<repo>/docs/work/{arg}/`). If found, use it as the work item directory.
2. **Else glob with dash suffix** — Glob `docs/work/{arg}-*/` (matches the slug-suffixed forms).
3. **Else glob by slug fragment** — Glob `docs/work/*{arg}*/` for a bare slug reference.
4. **If exactly one match**, use that directory throughout the command. If zero, error: "Work item {arg} not found in docs/work/." If multiple, error and list the matches.

Throughout the rest of this document, `work-NNNN` / `$WD` is shorthand for **the resolved work item directory** — substitute the actual resolved path when constructing commands. State for that item lives in `$WD/work.jsonl`; `$WD/manifest.md` is a generated projection of it.

## PART I - CONTEXT GATHERING

### Work Item Context (Optional)

The `--work work-NNNN` parameter is **OPTIONAL** but recommended for organized knowledge tracking.

**When `--work work-NNNN` is provided**:
1. Resolve the short ID to the actual directory (see "Resolving `--work` IDs" above).
2. Read work manifest: `docs/work/work-NNNN/manifest.md` (a generated view — read it for context, never edit it)
3. Read existing research documents in `docs/work/work-NNNN/research/` for context
4. Determine research focus based on work item and user's research topic
5. Plan to create research in `docs/work/work-NNNN/research/NNNN-*.md`
6. Plan to register the artifact in the event log when complete (see "Register the Artifact" below) — do **not** plan to hand-edit the manifest

**When `--work` is NOT provided** (Standalone Mode):
1. Create research in `docs/research/NNNN-*.md`
2. Use local numbering within `docs/research/` (highest existing NNNN + 1)
3. No event log, no manifest — and do not hand-maintain any global `docs/research/index.md` registry (it is derivable from the directory)
4. Self-contained research document

### If a task/topic/issue is mentioned:

1. Read any issue file if mentioned (e.g., `docs/work/work-NNNN/issues/0001-feature-xyz-issue.md`)
2. Read any requirement file if mentioned (e.g., `docs/work/work-NNNN/requirements/0001-feature-xyz-req.md`)
3. Understand what research is needed based on the task description
4. Identify any previous attempts or related work mentioned

### If no specific task is mentioned:

1. Ask the user what they want to research
2. Get clarification on scope and focus areas

## PART II - RESEARCH PROCESS

Think deeply about the research requirements.

### Step 1: Understand Research Scope

1. Read any linked documents or related files to understand context
2. If insufficient information to conduct research, ask for clarification
3. **Determine next available document number**:
   - Check `docs/work/work-NNNN/research/` for next sequence
   - Find highest NNNN and increment by 1, zero-pad to 4 digits

### Step 2: Conduct Research

**Determine Document Location**:
- **If `--work work-NNNN` provided**: Create in `docs/work/work-NNNN/research/NNNN-*.md`
- **If standalone**: Create in `docs/research/NNNN-*.md`

**Part A: Codebase Discovery** (Foundation)

1. Read `.claude/commands/research_codebase.md` for guidance on effective codebase research
2. Use WebSearch to research external solutions, APIs, or best practices if needed
3. Search the codebase for relevant implementations and patterns using specialized agents:
   - **codebase-locator** to find relevant files
   - **codebase-analyzer** to understand implementations
   - **codebase-pattern-finder** to find similar patterns
4. Examine existing similar features or related code
5. Identify technical constraints and opportunities
6. Be unbiased - document all related files and how systems work today

**Part B: Domain Expert Analysis** (Enhancement - NEW)

After codebase research completes, determine research domain and add expert validation:

7. **Detect research domain** based on topic and findings:
   - Backend/API research → backend-developer, api-designer, postgres-pro
   - Frontend research → frontend-developer, react-specialist, ui-designer
   - Infrastructure → platform-engineer, devops-engineer, cloud-architect
   - Security → security-engineer, security-auditor
   - Performance → performance-engineer, database-optimizer
   - Database → postgres-pro, database-administrator, database-optimizer

8. **Spawn domain expert agents** (1-3 relevant specialists):
   Use Task tool with appropriate subagent_type to get expert analysis of findings

   Example prompts:
   - "Review the backend patterns found: [summary]. Validate approach and suggest improvements."
   - "Analyze the API design discovered: [summary]. Check RESTful conventions and best practices."
   - "Review database schema patterns: [summary]. Assess normalization, indexing, and RLS usage."

9. **Wait for domain expert analysis** to complete

**Part C: Knowledge Synthesis** (Enhancement - NEW)

10. **Synthesize all findings** using coordination agents:
    - **knowledge-synthesizer** - Extract patterns from codebase + expert findings
    - **research-analyst** - Prioritize recommendations
    - **technical-writer** - Enhance documentation clarity (optional)

11. **Document comprehensive findings** in numbered research document:
    - **With work item**: `docs/work/work-NNNN/research/NNNN-[topic]-research.md`
    - **Standalone**: `docs/research/NNNN-[topic]-research.md`
    - Include codebase research findings (Part A)
    - Include domain expert validation (Part B)
    - Include synthesized insights (Part C)
    - Add `Work Item: work-NNNN` to document header (if applicable)

### Step 3: Synthesize Findings

1. Summarize key findings and technical decisions
2. Identify potential implementation approaches
3. Note any risks or concerns discovered
4. Present findings to the user

### Step 4: Register the Artifact in the Event Log (If Using Work Item)

**Do NOT hand-edit `manifest.md`.** Work-item state is an append-only event log (`work.jsonl`); the manifest is regenerated from it. With `$WD` = the resolved work item directory and `<rel-path>` = the artifact path *relative to `$WD`* (e.g. `research/NNNN-{slug}-research.md`):

**If `--work work-NNNN` was provided**:
1. Append an `artifact_added` event:
   ```bash
   scripts/wlog.sh "$WD" artifact_added kind=research path=<rel-path> title="{Title}"
   ```
2. Move the item into the researching state (research → `researching`). Append this only when the item is not already in `researching` or a later phase:
   ```bash
   scripts/wlog.sh "$WD" status_changed to=researching
   ```
3. Regenerate the manifest from the log:
   ```bash
   scripts/wrender.sh "$WD"
   ```

The `artifact_added` event records the registration; `wrender.sh` folds it into the manifest's Artifacts section, Change Log, Status, and Last Updated automatically. Never open `manifest.md` to edit any of these by hand.

**If standalone**:
- No event log and no manifest — the research document is self-contained.
- Do not hand-maintain any global registry/index of `docs/research/`; it is derivable from the directory listing.

### Step 5: Provide Research Summary

1. Present key findings and insights
2. **If work item mode**: Note that the artifact was registered in the event log and the manifest regenerated
3. **If standalone**: Note the research document location
4. Suggest next steps or areas for further investigation
5. **If work item mode**: Suggest adding more research with `/research --work work-NNNN "topic"`
6. **If standalone**: Suggest adding more research with `/research "topic"` or creating a work item with `/work "description"`
7. Offer to create implementation plans if appropriate using `/planv0` (with or without --work)

## DOCUMENT TEMPLATE

Create research documents using this structure:

```markdown
# [Research Topic] - Research Document

**Document ID**: NNNN
**Work Item**: work-NNNN (if applicable)
**Created**: [Current Date]
**Type**: Research
**Status**: [In Progress/Complete]

## Overview

[Brief description of what was researched and why]

## Research Scope

[What was investigated, boundaries, focus areas]

## Key Findings

### Current Implementation

[What exists now with file:line references]

### Technical Patterns

[Patterns discovered in the codebase]

### External Research

[Best practices, libraries, approaches found]

### Constraints and Opportunities

[Technical limitations and possibilities]

## Detailed Analysis

### [Finding Category 1]

- [Specific finding with file:line reference]
- [Another finding]

### [Finding Category 2]

- [Finding with context]
- [Related patterns]

## Domain Expert Analysis (NEW)

### [Domain] Expert Review

**Reviewed By**: [expert agent name]

**Validation**:
- [Expert validation of findings]
- [Pattern confirmation or suggestions]

**Expert Recommendations**:
- [Expert-specific recommendations]
- [Best practices from domain expertise]

## Synthesized Insights (NEW)

**Cross-Cutting Patterns**:
- [Patterns identified across codebase and expert analysis]

**Priority Recommendations**:
1. [High priority recommendation]
2. [Medium priority recommendation]

**Trade-offs Identified**:
- [Key trade-offs to consider]

## Recommendations

[Implementation approaches or next steps based on comprehensive research]

## References

### Codebase Files

- `[file:line]` - [Description of relevance]
- `[file:line]` - [Description of relevance]

### External Sources

- [Link/resource] - [Description]

### Related Documents

- **Previous Research**: [Link to related research if any]
- **Tickets**: [Link to related tickets if any]
- **Requirements**: [Link to related requirements if any]

## Next Steps

[Suggested actions based on research findings]

- Consider using `/planv0` to create implementation strategy
- Additional research areas if needed
- Specific technical decisions to make

---

**Auto-generated cross-references will be added by planning/implementation commands**
```

## NUMBERING SCHEME

### Format: NNNN-descriptive-name-research.md

- **NNNN**: Sequential number (0001, 0002, etc.)
- **descriptive-name**: Kebab-case description of research topic
- **research**: Type suffix

### Auto-Numbering Process:

**With work item (`--work work-NNNN`)**:
1. Use Glob to find files: `docs/work/work-NNNN/research/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-research.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits
5. Create: `docs/work/work-NNNN/research/NNNN-*.md`

**Standalone (no `--work`)**:
1. Use Glob to find files: `docs/research/*.md`
2. Extract numbers from filenames matching pattern `NNNN-*-research.md`
3. Find the highest number and increment by 1
4. Pad with leading zeros to 4 digits
5. Create: `docs/research/NNNN-*.md`

## CROSS-REFERENCING

Research documents are pure research and do NOT auto-create tickets or requirements. They serve as:

- Foundation for planning decisions
- Reference for implementation
- Knowledge base for future work

Other commands will reference research documents in their templates.

## IMPORTANT GUIDELINES

1. **Pure Research Focus**: This command only creates research - no tickets/requirements
2. **Numbered Documentation**: Always use the numbering scheme
3. **Comprehensive Analysis**: Use specialized agents for thorough investigation
4. **Actionable Insights**: Make findings useful for decision-making
5. **Cross-Reference Ready**: Structure for linking from other documents

Think deeply, use TodoWrite to track your research tasks. Focus on thorough investigation and clear documentation that will be valuable for future planning and implementation.
