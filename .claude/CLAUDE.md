# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

This is a **Claude Code Workflow Template** providing structured commands, agents, and workflows for software development. Copy `.claude/` to your projects to enable intelligent development workflows.

## 🤖 Proactive Workflow Recommendations

**IMPORTANT**: When a user makes requests, proactively recommend and use the appropriate workflow. Don't wait to be asked.

### When to Automatically Use `/work` (Most Common)

**Trigger `/work` for any of these user requests:**

✅ "Add [feature]" / "Implement [feature]" / "Build [feature]"
✅ "I want to..." / "I need to..." / "Can you help me..."
✅ "Fix [bug/issue]" (if non-trivial, requires investigation)
✅ "Improve [aspect]" / "Optimize [component]"
✅ "Integrate [service/library]"
✅ "Refactor [component]" (if substantial)

**Response pattern:**
```
I'll help you [build/add/fix] [feature]. Let me create a work item to organize this properly.

Creating work item with automatic research and requirements...
[Execute: /work "user's description"]
```

**When NOT to use /work:**
- Simple questions ("How does X work?") → Use `/research_codebase`
- Trivial fixes (typos, formatting) → Just do it
- Already working within a work item → Use work item context

### When to Use Standalone Commands

**Use `/research_codebase` for:**
- "How does [component] work?"
- "Where is [feature] implemented?"
- "Find all [pattern] in the codebase"
- "What files handle [responsibility]?"

**Use `/research` standalone when:**
- Quick investigation needed (no work item yet)
- User explicitly says "just research this"
- Exploring options before committing to work item

**Use `/planv0` standalone when:**
- User has a clear plan and wants to document it
- Simple feature without needing work item overhead

### When to Use Incremental Planning

**Automatically suggest incremental planning when:**
- User says "also need to..." during implementation
- User discovers new requirement mid-implementation
- User says "wait, we also need [feature]"

**Response pattern:**
```
I notice this is a new requirement that fits into the existing work item.
Let me add this as a new phase to the current plan.

[Execute: /planv0 --work work-NNNN "new requirement"]
```

### When to Use `/commit`

**Automatically use `/commit` when:**
- User says "commit this" / "commit the changes"
- You've completed a substantial piece of work
- User asks "what did we do?" (journal first, then offer commit)

**Don't auto-commit unless:**
- User explicitly requests it
- You've asked and received confirmation

### Decision Tree

```
User Request
│
├─ Question/Investigation?
│  └─ Use /research_codebase
│
├─ New Feature/Bug/Improvement?
│  ├─ Trivial (1-2 line fix)?
│  │  └─ Just do it
│  └─ Non-trivial?
│     └─ Use /work "description"
│        (auto creates research + requirements)
│
├─ Already in work item context?
│  ├─ New requirement discovered?
│  │  └─ Use /planv0 --work work-NNNN "new thing"
│  │     (incremental planning)
│  └─ Continue with workflow
│
└─ Document session?
   └─ Use /journal

## Core Workflow System

### Unified Work Item Workflow (Recommended)

The primary workflow centers around **work items** that group related artifacts.

**You should automatically initiate this workflow** when users request features, fixes, or improvements.

### Automatic Workflow Execution

**User says:** "Add OAuth social login to the app"

**You should:**
1. Recognize this as a feature request (non-trivial)
2. Respond: "I'll create a work item for OAuth social login with automatic research and requirements."
3. Execute: `/work "Add OAuth social login to the app"`
4. After completion: "✅ Created work-0001. I've researched existing authentication patterns and documented requirements. Review docs/work/work-0001/ and let me know when you're ready for the implementation plan."

### Manual Workflow Reference

```bash
# 1. Start with a natural language description
/work "Add OAuth social login to the app"
# → Creates work-NNNN with automatic research + requirements

# 2. Review research and requirements in docs/work/work-NNNN/

# 3. Create implementation plan when ready
/planv0 --work work-NNNN
# → Creates master.md + phase plans with agent recommendations

# 4. Implement phases
/implement_plan docs/work/work-NNNN/plans/master.md
# → Uses domain specialists + quality gates

# 5. Commit changes
/commit
# → Creates conventional commits with optional code review
```

### Work Item Structure

All artifacts are organized under `docs/work/work-NNNN/`:
- `manifest.md` - Work item metadata and artifact index
- `research/NNNN-*.md` - Research documents
- `requirements/NNNN-*.md` - Requirements specifications
- `issues/NNNN-*.md` - Issue tickets
- `plans/master.md` + `phase-N.md` - Implementation plans
- `implementation/status.md` - Implementation progress

### Standalone Mode (Optional)

All commands support standalone usage without work items:
```bash
/research "How does authentication work?"      # → docs/research/
/new_req "API rate limiting requirements"      # → docs/requirements/
/planv0 "Implement dark mode"                   # → docs/plans/
```

## Custom Commands

### Work Management
- `/work "description"` - Create work item with auto research + requirements
- `/work show work-NNNN` - Display work item details
- `/work list` - List all work items
- `/work update work-NNNN --status X` - Update status

### Research & Requirements
- `/research [--work work-NNNN] "topic"` - Conduct thorough research
  - Uses codebase-locator, codebase-analyzer, domain experts
  - Creates numbered research documents
- `/new_req [--work work-NNNN] "requirements"` - Document requirements
  - Uses ux-researcher, architect-reviewer, qa-expert for validation
- `/new_issue [--work work-NNNN] "issue"` - Create issue tickets
  - Uses debugger, performance-engineer, security-engineer based on type

### Planning & Implementation
- `/planv0 [--work work-NNNN]` - Create implementation plans
  - Initial planning: Creates master + phase plans
  - Incremental planning: Adds phases to existing plans (phase-2.1, phase-3.1)
  - Uses architect-reviewer, domain specialists, qa-expert
  - **MANDATORY**: Populates specific agent recommendations in all plans
- `/implement_plan <plan-path>` - Execute implementation
  - Assigns domain specialists per phase
  - Runs code-reviewer quality gates
  - Updates work manifest automatically

### Documentation
- `/commit` - Create conventional commits (with optional code-reviewer)
- `/journal [session-name]` - Document development session
- `/learn [topic]` - Capture concise learnings

### Codebase Research
- `/research_codebase` - Deep codebase exploration
  - Uses codebase-locator, codebase-analyzer, codebase-pattern-finder
  - Generates comprehensive research documents with file:line references

## Specialized Agents

The `.claude/agents/` directory contains 80+ specialized agents automatically used by commands:

### Codebase Exploration
- **codebase-locator** - Find files and components
- **codebase-analyzer** - Understand implementations
- **codebase-pattern-finder** - Find similar code examples

### Domain Specialists
- **backend-developer**, **frontend-developer**, **fullstack-developer**
- **postgres-pro**, **database-administrator**, **database-optimizer**
- **react-specialist**, **typescript-pro**, **javascript-pro**
- **api-designer**, **websocket-engineer**, **payment-integration**

### Quality & Security
- **code-reviewer** - Code quality, security, best practices (MANDATORY in all plans)
- **security-engineer**, **security-auditor**, **penetration-tester**
- **qa-expert**, **test-automator**, **accessibility-tester**

### Architecture & DevOps
- **architect-reviewer** - System design validation
- **platform-engineer**, **devops-engineer**, **sre-engineer**
- **cloud-architect**, **kubernetes-specialist**, **terraform-engineer**

### Others
- **performance-engineer**, **ui-designer**, **debugger**, **refactoring-specialist**

See `.claude/agents/` for complete list with detailed capabilities.

## Incremental Planning

When you discover new requirements during implementation:

```bash
# Add a new phase to existing plan
/planv0 --work work-NNNN "Add caching layer for performance"

# Agent will:
# 1. Analyze existing phases (1, 2, 3)
# 2. Intelligently determine placement (e.g., after phase-2)
# 3. Create phase-2.1.md (decimal numbering - no renumbering!)
# 4. Update only affected downstream phases
# 5. Update master.md with new phase entry
```

**Key principles**:
- Decimal numbering (phase-2.1, phase-3.2) avoids renumbering existing phases
- Intelligent placement using architect-reviewer + domain specialists
- Minimal updates - only touch genuinely affected phases
- Conservative approach - when in doubt, don't update

## Agent Recommendations in Plans

**All plans MUST have specific agent recommendations** (enforced by validation):

✅ **Good** (specific, actionable):
```markdown
**Agents for This Phase**:
- **backend-developer** - Implement REST API endpoints and business logic
- **postgres-pro** - Validate database schema and RLS policies
- **code-reviewer** - Final quality check before phase completion
```

❌ **Bad** (placeholders, will fail validation):
```markdown
**Agents**:
- **[agent-name]** - [When to use]
- Other agents as needed
```

The `planv0` command validates and ensures NO placeholders remain.

## File Numbering Conventions

All documents use sequential numbering within their scope:
- **Work items**: `work-0001`, `work-0002` (global across all work)
- **Research**: `0001-topic-research.md`, `0002-topic-research.md` (per work item or global)
- **Requirements**: `0001-feature-req.md`, `0002-feature-req.md` (per work item or global)
- **Issues**: `0001-bug-issue.md`, `0002-task-issue.md` (per work item or global)
- **Plans**: `phase-1.md`, `phase-2.md`, `phase-2.1.md` (per work item)

## Status Values for Work Items

- 🎯 **Proposed** - Work item created, needs research
- 📚 **Researching** - Research in progress
- 📝 **Requirements** - Gathering/documenting requirements
- 🎨 **Planning** - Creating implementation plan
- 🔄 **In Implementation** - Active development
- ✅ **Completed** - Work finished and deployed
- 🔴 **Blocked** - Waiting on dependencies
- ⏸️ **On Hold** - Paused for later
- ❌ **Cancelled** - Will not be implemented

## Enabled Plugins

From `.claude/settings.json`:
- `security-guidance@claude-plugins-official` - Security best practices
- `code-simplifier@claude-plugins-official` - Code simplification
- `example-skills@anthropic-agent-skills` - Example skill templates

## Best Practices

1. **Start with `/work`** for any non-trivial feature or bug
2. **Let automation work** - research and requirements are auto-generated
3. **Review before planning** - verify research/requirements are complete
4. **Use incremental planning** - add phases as you discover new needs
5. **Leverage specialists** - plans include specific agent recommendations
6. **Quality gates matter** - code-reviewer is mandatory before phase completion
7. **Document as you go** - use `/journal` and `/learn` during implementation

## 🎯 Proactive Behavior Rules

**As Claude Code, you should:**

1. **Recognize patterns and suggest workflows** - Don't wait for users to know slash commands exist
   - User: "I want to add OAuth login"
   - You: "I'll create a work item for OAuth login integration with automatic research and requirements. [Trigger /work]"

2. **Explain what you're doing** - Be transparent about workflow automation
   - "Creating work item work-0001 with automatic research..."
   - "I'll research the codebase first using specialized agents..."
   - "Adding this as phase-2.1 to the existing plan..."

3. **Offer next steps** - Guide users through the workflow
   - After /work completes: "Review the research and requirements in docs/work/work-0001/. When ready, I can create the implementation plan."
   - After /planv0 completes: "Review the plan. When ready, run /implement_plan to start implementation."

4. **Use work item context** - Once a work item is created, stay in that context
   - Track which work item is active
   - Suggest adding research/requirements to it
   - Use incremental planning for new discoveries

5. **Validate before executing** - For destructive or significant actions
   - Before /commit: Show what will be committed
   - Before incremental planning: Explain where the new phase will go
   - Get user confirmation for major steps

## 🚫 What NOT to Do

**Don't:**
- Ask "Would you like me to create a work item?" if request clearly needs one → Just do it
- Mention slash commands by name unless user needs to run them manually → Use them transparently
- Create work items for trivial tasks (typos, simple questions)
- Auto-commit without explicit user request
- Force the work item workflow for simple investigations

## Directory Structure for Documentation

When using this template in a code project:

```
docs/
├── work/                    # Work items (created by /work)
│   └── work-NNNN/
│       ├── manifest.md
│       ├── research/
│       ├── requirements/
│       ├── issues/
│       ├── plans/
│       └── implementation/
├── research/                # Standalone research (optional)
├── requirements/            # Standalone requirements (optional)
├── issues/                  # Standalone issues (optional)
├── plans/                   # Standalone plans (optional)
├── journal/                 # Session journals
└── learnings/               # Learning notes
```
