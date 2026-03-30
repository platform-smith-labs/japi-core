# Commit Changes

You are tasked with creating git commits for the changes made during this session using the Conventional Commits specification.

## Process:

1. **Check Session Context First:**
   - **Review the conversation history** - you have full context of what was done
   - **Identify all files you created, edited, or deleted** during this session
   - **Understand the nature of changes** from the work you performed
   - **ONLY if context is unclear or incomplete**, then run minimal git commands:
     - Run `git status` to see current changes
     - Avoid `git diff` unless absolutely necessary (it pollutes context)
   - Consider whether changes should be one commit or multiple logical commits

   **IMPORTANT**: You participated in making these changes, so you already know:
   - What files were modified
   - What the changes accomplish
   - Why the changes were made
   - Use this knowledge instead of running git commands

1a. **Check for Git Submodules:**
   - Check if `.gitmodules` file exists to detect submodules
   - If submodules exist, check each submodule for uncommitted changes:
     ```bash
     # For each submodule in apps/ or other locations
     cd <submodule-path> && git status
     ```
   - Identify which submodules have changes that need to be committed
   - Plan to commit submodule changes BEFORE updating parent repository references

   **Submodule Commit Strategy**:
   - Commit changes within each submodule first
   - Then commit the parent repository to update submodule references
   - This ensures the parent always points to valid commits in submodules

2. **Pre-Commit Quality Check** (Enhancement - NEW - Optional):

   For significant code changes, run quick quality validation:

   a. **Determine if quality check is needed**:
      - Code changes (not just docs/config) → Yes
      - Documentation/markdown only → No
      - Configuration only → No

   b. **If needed, run code-reviewer quick check**:
      ```
      Use Task tool with subagent_type="code-reviewer":

      "Quick pre-commit review of changes:

      Files changed: [list code files]

      Check CRITICAL ISSUES ONLY:
      - Security vulnerabilities
      - Obvious bugs
      - Breaking changes not marked

      Skip: style issues, minor optimizations, documentation"
      ```

   c. **If critical issues found**:
      - Report findings to user
      - Offer to fix before committing
      - User decides whether to fix or commit anyway

   d. **If no critical issues**, proceed to commit planning

3. **Plan your commit(s) using Conventional Commits format:**
   - **For Submodules**: Plan commits for each submodule separately
     - Each submodule gets its own commit with appropriate message
     - Use the submodule's context to write meaningful commit messages
   - **For Parent Repository**: Plan commits for parent repo changes
     - If only updating submodule references, use: `chore: update submodule references`
     - If there are other changes, group them logically
   - Identify which files belong together
   - Determine the appropriate commit type and scope
   - Draft commit messages following the format: type[optional scope]: description
   - Add body and footers if needed for complex changes
   - Mark breaking changes with exclamation mark or BREAKING CHANGE footer

4. **Present your plan to the user:**
   - **Show submodule commits first** (if any):
     - List each submodule with changes
     - Show the commit message for each submodule
     - List files to be committed in each submodule
   - **Then show parent repository commits**:
     - List the files you plan to add for each commit
     - Show the commit message(s) you'll use in Conventional Commits format
   - Include quality check summary if performed
   - Ask: "I plan to create [N] commit(s) ([X] in submodules, [Y] in parent repo) with these changes. Shall I proceed?"

5. **Execute upon confirmation using subagent:**

   **IMPORTANT**: Delegate commit execution to a subagent to reduce main context token usage.

   Use the Task tool with `subagent_type="git-workflow-manager"`:

   ```
   Task tool parameters:
   - subagent_type: "git-workflow-manager"
   - description: "Execute planned git commits"
   - prompt: |
       Execute the following git commits. Return only a brief summary.

       Project root: [absolute path]

       ## Submodule Commits (execute in order):
       [For each submodule with changes:]
       ### [submodule-name] ([path])
       Files: [list of files to add]
       Message: [commit message]

       ## Parent Repository Commit:
       Files: [list of files to add, including submodule refs]
       Message: [commit message]

       ## Instructions:
       1. For each submodule: cd to path, git add files, git commit
       2. Return to project root
       3. Add parent files + submodule references
       4. Commit parent repository
       5. Run: git log --oneline -n [N] to show results

       Return format:
       - List of commits created (hash + message)
       - Any errors encountered
       - Suggested push command
   ```

   **After subagent completes:**
   - Display the commit summary returned by the subagent
   - Suggest: `git push --recurse-submodules=on-demand`

## Conventional Commits Format:

Format:
type[optional scope]: description

[optional body]

[optional footer(s)]

### Common Types:
- feat: - New feature (MINOR version)
- fix: - Bug fix (PATCH version)
- docs: - Documentation changes
- style: - Code style changes (formatting, etc.)
- refactor: - Code refactoring
- perf: - Performance improvements
- test: - Adding or updating tests
- build: - Build system or dependencies
- ci: - CI/CD configuration
- chore: - Other maintenance tasks

### Examples:
- feat(auth): add OAuth2 integration
- fix: resolve memory leak in data processor
- docs: update API documentation
- feat!: remove deprecated user endpoints (breaking change)
- refactor(ui): modernize component structure

### Breaking Changes:
- Add exclamation mark after type/scope: feat!: or feat(api)!:
- Or use footer: BREAKING CHANGE: description of breaking change

## Important:

- **NEVER add co-author information or Claude attribution**
- Commits should be authored solely by the user
- Do not include any "Generated with Claude" messages
- Do not add "Co-Authored-By" lines
- Write commit messages as if the user wrote them
- Follow Conventional Commits specification strictly

## Remember:

- **You have full context** - you participated in making these changes
- **Avoid git diff** - it pollutes context unnecessarily
- **Use session memory** - recall what you created, edited, or deleted
- Group related changes together
- Keep commits focused and atomic when possible
- The user trusts your judgment - they asked you to commit
- Use appropriate commit types and scopes for semantic versioning

## Session Context Priority:

**Primary Source (use first)**:
- Your memory of files you worked with during this session
- Your understanding of what each change accomplishes
- The conversation history showing what was requested and completed

**Fallback (only if needed)**:
- `git status` to confirm file list if uncertain
- `git diff` ONLY as absolute last resort (pollutes context)

## Submodule Workflow Example:

When working with a monorepo that has submodules:

1. **Detect submodules**:
   ```bash
   # Check if .gitmodules exists
   test -f .gitmodules && echo "Submodules detected" || echo "No submodules"

   # List submodules
   git submodule status
   ```

2. **Check each submodule for changes**:
   ```bash
   cd apps/ps-portal-api && git status
   cd ../ps-portal-app && git status
   cd ../ps-web && git status
   cd ../..
   ```

3. **Commit in submodules first**:
   ```bash
   # In submodule 1
   cd apps/ps-portal-api
   git add src/auth.ts
   git commit -m "feat: add OAuth2 authentication"
   cd ../..

   # In submodule 2
   cd apps/ps-web
   git add components/Login.tsx
   git commit -m "feat: add login component"
   cd ../..
   ```

4. **Update parent repository**:
   ```bash
   # Parent repo now sees submodules have new commits
   git status
   # Shows: modified: apps/ps-portal-api (new commits)

   # Add submodule references
   git add apps/ps-portal-api apps/ps-web
   git commit -m "chore: update submodules with authentication features"
   ```

5. **Push everything**:
   ```bash
   # Push submodules and parent in one command
   git push --recurse-submodules=on-demand
   ```
