# planv0 — Tools, Agents & Planning Guidance

Reference material for `/planv0`: the agent catalog + selection guidance, and the tail planning-guidance sections. The CORE command (`.claude/commands/planv0.md`) points here on demand when choosing per-phase agents or applying planning best practices.

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
