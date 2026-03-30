# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

**japi-core** is a type-safe Go framework/library for building REST APIs with PostgreSQL. It is consumed by other Go services via `go get github.com/platform-smith-labs/japi-core`.

The `.claude/` directory in this repo is a reusable Claude Code workflow template. Copy `.claude/` to other projects to enable intelligent development workflows.

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

## Build & Testing Conventions

japi-core is a **library** — there is no server to run or Docker container to manage.

### Building

```bash
# Verify all packages compile
go build ./...
```

### Testing

```bash
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# Specific package
go test ./handler/...
go test ./db/...
```

### Test Patterns

Tests use `net/http/httptest` and real in-memory state — no mocking of the framework internals. See `handler/registry_test.go` and `handler/context_test.go` for examples.

## API Model Design Guidelines

### Reuse Models Across CRUD Operations

**Always reuse the same model struct for both input (request body) and output (response) across CRUD operations.** Do not create separate `CreateXRequest` / `UpdateXRequest` types unless the shapes genuinely differ in a way that cannot be reconciled.

**Why**: Separate request/response models cause duplication, drift, and unnecessary cognitive overhead. One model is easier to maintain and test.

**How to design a reusable model**:
- Use `*string` (not `string`) for optional nullable fields — `nil` maps cleanly to SQL `NULL` and JSON `null`
- Use `*time.Time` (not `handler.Nullable[time.Time]`) for nullable timestamps
- Never use `handler.Nullable[T]` in model structs — it has no exported fields and produces `{}` when JSON-marshalled. It is only for `HandlerContext` fields (params, body, auth)
- Server-generated fields (`uuid`, `created_at`, `updated_at`, `is_archived`) use `omitempty` so they are absent on input and present on output
- Validate required fields via struct tags (`validate:"required,min=1"`) — `typed.ParseBody` enforces these automatically

**Full update vs partial update**: Prefer full updates (all mutable fields required on every write). This keeps the SQL simple (no `COALESCE`/`CASE` patterns) and the model shape unambiguous.

**When a separate request type IS justified**:
- The request needs a field that must never appear in the response (e.g. `password` in `CreateUserRequest` — use embedding via `UserWithPassword` instead of a standalone model)
- Two operations have structurally incompatible shapes that cannot share a single struct

## Functional Programming Guidelines

This codebase follows **functional programming principles** where practical in Go. japi-core achieves a **4.5/5 FP rating** (top 10% of Go codebases).

### Core Principles

1. **Pure Functions**: Separate business logic (pure) from I/O (impure)
   - Pure functions: deterministic, no side effects, testable
   - Impure functions: database queries, HTTP requests, logging
   - Keep them separated and clearly labeled

2. **Immutability**: Use value receivers, unexported fields, no setters
   - Prefer value types over pointers
   - Unexported fields prevent external mutation
   - No setter methods on structs

3. **Function Composition**: Chain middleware, compose small functions
   - Middleware as higher-order functions
   - Right-to-left composition in `MakeHandler`
   - Small functions (10-20 lines) composed into larger logic

4. **Type Safety**: Leverage generics for compile-time guarantees
   - Generic handlers: `Handler[ParamTypeT, BodyTypeT, ResponseBodyT]`
   - Generic queries: `QueryOne[T]`, `QueryMany[T]`
   - Nullable monad: `Nullable[T]` for type-safe optionals

5. **Nullable Monad**: Use `Nullable[T]` instead of `*T` for optionals
   - `NewNullable(value)` for present values
   - `Nil[T]()` for absent values
   - Methods: `Value() (T, error)`, `TryValue() (T, bool)`, `ValueOr(default)`, `ValueOrDefault()`, `HasValue() bool`

6. **Explicit Dependencies**: Inject via constructors, not globals
   - No global mutable state
   - Dependencies passed to constructors
   - Use closures for dependency injection in middleware

### Patterns to Follow

✅ **DO**:
- Use higher-order functions for middleware
- Compose small pure functions (10-20 lines each)
- Use `Nullable[T]` for optional values instead of `*T`
- Inject dependencies via constructors and closures
- Separate pure calculation from I/O side effects
- Use generics for type-safe reusable functions
- Load configuration once at startup (immutable)
- Use value receivers for immutable types
- Return new values instead of mutating parameters

❌ **DON'T**:
- Use global mutable state (package-level vars modified at runtime)
- Mix side effects (logging, I/O) with business logic
- Use `*T` pointers for optional values (use `Nullable[T]`)
- Mutate shared data structures (maps, slices)
- Write 100+ line functions (break into composable pieces)
- Lazy-load config in middleware (race conditions)
- Use `init()` for side effects (load in `main()` instead)

### Code Examples

#### Higher-Order Functions (Middleware)

```go
// Middleware is a higher-order function: (Handler -> Handler)
func RequireAuth[P, B, R any](next handler.Handler[P, B, R]) handler.Handler[P, B, R] {
    return func(ctx handler.HandlerContext[P, B], w http.ResponseWriter, r *http.Request) (R, error) {
        // Validate JWT
        token := extractToken(r)
        if !validateToken(token) {
            var zero R
            return zero, errors.New("unauthorized")
        }
        // Call next handler (composition)
        return next(ctx, w, r)
    }
}
```

#### Nullable Monad (Type-Safe Optionals)

```go
// ❌ BAD: Pointer (can panic)
var userID *uuid.UUID
if userID != nil {
    fmt.Println(*userID)  // Risky
}

// ✅ GOOD: Nullable monad (safe)
var userID handler.Nullable[uuid.UUID]
fmt.Println(userID.ValueOr(uuid.Nil))  // Safe default

// Create with a value
populated := handler.NewNullable(someUUID)

// Empty Nullable
empty := handler.Nil[uuid.UUID]()

// Pattern matching
if id, ok := userID.TryValue(); ok {
    fmt.Println(id)  // Type-safe
}

// Check presence
if userID.HasValue() {
    id, _ := userID.Value() // safe after HasValue() check
}
```

#### Pure vs Impure Separation

```go
// ✅ PURE: Business logic (no side effects)
func calculateTotal(items []Item) float64 {
    total := 0.0
    for _, item := range items {
        total += item.Price
    }
    return total  // Deterministic
}

// ⚠️ IMPURE: I/O wrapper (explicit effects)
func calculateTotalWithLogging(logger *slog.Logger, items []Item) float64 {
    total := calculateTotal(items)  // Call pure function
    logger.Info("Total calculated", "amount", total)  // Side effect
    return total
}
```

#### Dependency Injection (Not Global State)

```go
// ❌ BAD: Global mutable state
var db *sql.DB
var config Config

func init() {
    config = loadConfig()  // Side effect
    db = connectDB(config)
}

// ✅ GOOD: Explicit injection
type Service struct {
    db     *sql.DB
    config Config
}

func NewService(db *sql.DB, config Config) *Service {
    return &Service{db: db, config: config}
}

func main() {
    config := loadConfig()
    db := connectDB(config)
    service := NewService(db, config)  // Explicit
}
```

#### Small Composable Functions

```go
// ❌ BAD: 100+ line function
func HandleRequest(ctx HandlerContext) error {
    // 30 lines: parse
    // 40 lines: validate
    // 50 lines: execute
    // 30 lines: format response
}

// ✅ GOOD: Composed from small pure functions
func HandleRequest(ctx HandlerContext) error {
    request := parseRequest(ctx)        // Pure, 10 lines
    validated := validate(request)      // Pure, 15 lines
    result := execute(ctx, validated)   // Impure, 20 lines
    return formatResponse(result)       // Pure, 10 lines
}
```

### Additional Resources

**Comprehensive FP Analysis**:
- See `docs/work/work-0001/research/0003-functional-programming-paradigms.md` for:
  - Current FP patterns in japi-core (with file references)
  - Anti-patterns to avoid (with examples)
  - Detailed implementation guidelines
  - Haskell equivalents for comparison

**japi-core FP Patterns** (already implemented):
- Higher-order functions: `middleware/typed/*.go`
- Function composition: `handler/types.go:94-128`
- Option monad: `handler/nullable.go`
- Bracket pattern: `db/query.go:20-51`
- Pure functions: `db/query.go:53-94`

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
