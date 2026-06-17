You are conducting an interactive codebase walkthrough session. Your goal is to help the user understand the codebase through a structured, node-by-node exploration.

## Arguments Provided
$ARGUMENTS

## Parse Arguments
1. Extract learning goals (everything before --beginner, --intermediate, or --advanced flags)
2. Extract detail level flag (--beginner, --intermediate, --advanced)
3. Default detail level: intermediate if not specified
4. Default learning goals: "general understanding of the codebase" if not specified

## Step 1: Initialize Walkthrough State

### 1.1 Detect Codebase Type
Auto-detect the codebase type by checking for these markers:
- **Rust**: Cargo.toml, .rs files
- **Node.js/TypeScript**: package.json, tsconfig.json
- **Python**: requirements.txt, setup.py, pyproject.toml
- **Go**: go.mod, .go files
- **Java**: pom.xml, build.gradle, .java files
- **C/C++**: Makefile, CMakeLists.txt, .c/.cpp files
- **Other**: Identify based on file extensions and structure

### 1.2 Find Entry Points
Based on codebase type, locate entry points:
- **Rust**: src/main.rs, src/lib.rs, or bin/ directory
- **Node.js**: package.json "main" field, index.js/ts, src/index.*
- **Python**: __main__.py, main.py, or setup.py entry_points
- **Go**: main.go files (check all packages)
- **Java**: files with public static void main
- **C/C++**: main.c, main.cpp, or entry points defined in build files

### 1.3 Build Initial Walkthrough Tree
Using glob and grep tools:
1. Map the directory structure
2. Identify key architectural components
3. Trace import/dependency chains from entry point
4. Create a logical node structure (typically 10-20 nodes)
5. Prioritize nodes based on:
   - User's learning goals (if specified)
   - Critical path from entry point
   - Core architecture components

Example node tree structure:
- Node 1: Entry Point (where execution starts)
- Node 2: Configuration (how the app is configured)
- Node 3-N: Core Components (in dependency order)
- Final Nodes: Utilities, helpers, error handling

### 1.4 Create State File
Create `analysis/walkthrough-state.md` with this structure:

```markdown
# Interactive Codebase Walkthrough - State

**Status**: IN PROGRESS - Node 1
**Date**: [today's date]
**Codebase Type**: [detected type]
**Detail Level**: [beginner/intermediate/advanced]
**Learning Goals**: [user's goals or "general understanding"]
**Branch**: [current git branch]

## Current Position

**Node 1: [Name]** - IN PROGRESS

[Details will be added as we progress]

---

## Walkthrough Tree Structure

[List all nodes with brief descriptions]

### Node 1: [Name] - IN PROGRESS
- [File paths and line ranges]
- [Key concepts to cover]

### Node 2: [Name] - PENDING
- [File paths and line ranges]
- [Key concepts to cover]

[... continue for all nodes ...]

---

## Issues Found

[Track any issues/bugs discovered during walkthrough]

---

## User Questions & Notes

[Log user questions and answers during the session]

---

## Next Steps

[What comes after current node]
```

## Step 2: Begin Node-by-Node Walkthrough

For EACH node (starting with Node 1), follow this pattern:

### 2.1 Present Node Introduction
- **Node number and name**
- **File path(s)** covered
- **Purpose** in the architecture
- **Connection** to previous nodes (if applicable)
- **Key concepts** that will be explained

### 2.2 Show Relevant Code
Using the Read tool:
- Display code blocks with line numbers
- Highlight important sections
- Show cross-references to other files

### 2.3 Provide Detailed Explanation
Adapt explanation based on detail level:

**Beginner Level**:
- Explain language fundamentals as they appear
- Define technical terms
- Use analogies and real-world comparisons
- Show "what" the code does, then "why"
- Include more examples
- Break down complex expressions step-by-step

**Intermediate Level** (default):
- Assume basic programming knowledge
- Focus on architecture and patterns
- Explain "why" design choices were made
- Highlight connections between components
- Point out common patterns and idioms
- Explain language-specific features as they appear

**Advanced Level**:
- Focus on design decisions and trade-offs
- Discuss performance implications
- Identify potential improvements
- Compare alternative approaches
- Analyze scalability and maintainability
- Discuss edge cases and error scenarios

### 2.4 Adapt to Learning Goals
If user specified learning goals, emphasize relevant aspects:
- "understand async patterns" → deep dive into async/await, futures, promises, concurrency
- "error handling" → focus on error types, propagation, recovery strategies
- "protocol design" → emphasize message formats, serialization, state machines
- "architecture" → focus on component relationships, data flow, boundaries
- "testing strategy" → highlight test patterns, mocking, coverage
- etc.

### 2.5 Identify Issues (If Found)
If you discover bugs, issues, or improvements:
1. Document in "Issues Found" section of state file
2. Show exact location (file:line)
3. Explain the problem clearly
4. Suggest potential fixes
5. Ask user if they want to address now or continue

### 2.6 Interactive Pause
After explaining each node:
1. Provide a brief summary of what was covered
2. Ask if the user has questions about this node
3. Confirm understanding before proceeding
4. Update state file to mark node as COMPLETE
5. Wait for user confirmation to move to next node

**IMPORTANT**: DO NOT proceed to the next node automatically - always wait for user input.

### 2.7 Update State File
After completing each node:
- Mark current node as COMPLETE
- Update "Current Position" to next node (mark as IN PROGRESS)
- Add any user questions/notes to the log
- Record any issues found
- Update "Next Steps" section

## Step 3: Navigation Commands

Throughout the walkthrough, respond to these user commands:

- **"next"** or **"continue"** or **"move on"**: Move to next node
- **"back"** or **"previous"**: Return to previous node for review
- **"jump to [node name/number]"**: Skip to specific node
- **"explain [concept] more"** or **"elaborate on [topic]"**: Deep dive into specific concept
- **"show [file/function]"**: Display additional code not in current node
- **"list nodes"** or **"show tree"**: Show the full walkthrough tree with progress
- **"where am I?"** or **"status"**: Show current position and progress summary
- **"change detail level to [beginner/intermediate/advanced]"**: Adjust explanations mid-session
- **"pause"** or **"stop"**: Save state and exit (can resume later)
- **"focus on [new goal]"**: Adjust learning focus mid-session
- **"summarize"**: Provide a summary of current node without full details

## Step 4: Session Management

### Resuming a Session
If `analysis/walkthrough-state.md` already exists:
1. Read the state file
2. Show current position and progress summary (e.g., "5 of 15 nodes completed")
3. List what's been covered so far
4. Ask user: "Resume from Node [X]?" or "Start fresh?"
5. If resuming, continue from saved position
6. If starting fresh, archive old state file (rename with timestamp) and create new one

### Completing the Walkthrough
When all nodes are complete:
1. Provide a comprehensive summary of key learnings
2. List all issues found during the walkthrough
3. Highlight important architectural patterns observed
4. Suggest next steps (fixes, improvements, testing, documentation)
5. Update state file status to COMPLETED with completion date
6. Ask if user wants to explore any area in more depth

## Important Guidelines

1. **One node at a time**: Never skip ahead without user confirmation
2. **Show code with line numbers**: Always use Read tool to display actual code with file:line references
3. **Use absolute file paths**: Always specify full paths from repository root
4. **Track state meticulously**: Update the state file after every node completion
5. **Be interactive**: Ask questions, check understanding, adapt to feedback
6. **Explain "why" not just "what"**: Focus on reasoning and design decisions
7. **Find real issues**: If you spot bugs, improvements, or technical debt, call them out
8. **Cross-reference**: Link related concepts across different nodes
9. **No assumptions**: If codebase type can't be detected, ask the user
10. **Granular detail**: File paths, line numbers, specific code blocks always
11. **Patience**: Let the user set the pace - some may want to explore deeply, others may want overview
12. **Context preservation**: Each node explanation should reference previous nodes when relevant

## Language-Specific Patterns to Highlight

Adapt terminology and patterns based on detected language:

**Rust**:
- Ownership, borrowing, lifetimes
- Result/Option types and error handling
- Traits and impl blocks
- async/await and futures
- Arc/Mutex for concurrency

**Node.js/TypeScript**:
- Promises and async/await
- Module system (require/import)
- Event loop and callbacks
- Type definitions (TypeScript)
- NPM dependencies and package.json

**Python**:
- Decorators and context managers
- Generators and iterators
- List comprehensions
- Duck typing and protocols
- Virtual environments and dependencies

**Go**:
- Goroutines and channels
- Interfaces and composition
- defer, panic, recover
- Error handling patterns
- Modules and packages

**Java**:
- Classes, interfaces, inheritance
- Exceptions and try-catch
- Streams and lambdas (Java 8+)
- Spring/framework patterns (if applicable)
- Maven/Gradle dependencies

## Now Begin

Start the walkthrough session by:
1. Parsing arguments to extract learning goals and detail level
2. Detecting the codebase type
3. Finding entry points
4. Building the initial node tree
5. Creating the state file (or resuming if it exists)
6. Presenting Node 1 with full detail

Remember: This is an interactive learning experience. Go at the user's pace, adapt to their needs, and make the exploration engaging and informative.

**After presenting Node 1, explicitly ask the user if they have questions or want to proceed to the next node.**
