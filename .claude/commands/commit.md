# Commit Changes

You are tasked with creating git commits for the changes made during this session using the Conventional Commits specification.

## Invocation

```bash
/commit                # Interactive: plan commits, present plan, ask before executing
/commit --force        # Non-interactive: plan + execute immediately, NO confirmation prompt
```

`--force` (alias: `auto`) is for **unattended / autonomous use** (e.g. when `/work … auto`
runs `/commit --force` after an implementation phase). In force mode:

- **Skip the "Shall I proceed?" confirmation** in step 4 — go straight from planning (step 3) to
  execution (step 5).
- The optional pre-commit code-reviewer check (step 2) becomes **non-blocking**: if it surfaces
  critical issues, record them in the commit body / session output but still commit (the loop
  surfaces them rather than halting on a prompt that no human will answer).
- Everything else — Conventional Commits format, the no-attribution
  rule — is unchanged. **Force mode still does NOT push or open a PR**; it only commits.

## Process:

> **Single-repo scope.** The child service repos are independent clones — to commit changes in a
> service repo, run `/commit` inside `repos/<name>`. In the solution root, `/commit` only touches the
> solution's own files (`repos/` is git-ignored). `/commit` always operates on the **current repo
> only**.

1. **Enumerate changed files deterministically:**
   - **Run `git status --porcelain`** to get the authoritative list of changed/added/deleted files. This is the source of truth for *what* changed — do not reconstruct the file list from conversation memory.
   - You may use `git diff --stat` for a quick scale check; avoid full `git diff` unless you genuinely need to inspect content (it pollutes context).
   - Consider whether changes should be one commit or multiple logical commits.

   **What is deterministic vs. your judgment:**
   - **Deterministic (from git)**: the set of changed files. Always derive this from `git status --porcelain`.
   - **Judgment (yours)**: how to *group* files into logical commits, the commit type/scope, and the message wording. Use your session context — what the changes accomplish and why — to write meaningful messages, not to enumerate files.

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
   - Identify which files belong together
   - Determine the appropriate commit type and scope
   - Draft commit messages following the format: type[optional scope]: description
   - Add body and footers if needed for complex changes
   - Mark breaking changes with exclamation mark or BREAKING CHANGE footer

4. **Present your plan to the user:**
   - List the files you plan to add for each commit
   - Show the commit message(s) you'll use in Conventional Commits format
   - Include quality check summary if performed
   - Ask: "I plan to create [N] commit(s) with these changes. Shall I proceed?"
   - **In `--force` mode, skip this confirmation entirely** — proceed directly to step 5.

5. **Execute upon confirmation using subagent:**

   **IMPORTANT**: Delegate commit execution to a subagent to reduce main context token usage.

   Use the Task tool with `subagent_type="git-workflow-manager"`:

   ```
   Task tool parameters:
   - subagent_type: "git-workflow-manager"
   - description: "Execute planned git commits"
   - prompt: |
       Execute the following git commits in the current repo. Return only a brief summary.

       Repo root: [absolute path]

       ## Commit(s):
       [For each planned commit:]
       Files: [list of files to add]
       Message: [commit message]

       ## Instructions:
       1. For each commit: git add files, git commit
       2. Run: git log --oneline -n [N] to show results

       Return format:
       - List of commits created (hash + message)
       - Any errors encountered
       - Suggested push command
   ```

   **After subagent completes:**
   - Display the commit summary returned by the subagent
   - Suggest: `git push`

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

- **Enumerate files from `git status --porcelain`** - this is the authoritative changed-file list; don't reconstruct it from memory
- **Avoid full git diff** - it pollutes context unnecessarily (a `git diff --stat` is fine)
- **Use session context for judgment** - what each change accomplishes and why, to write good messages and group commits
- Group related changes together
- Keep commits focused and atomic when possible
- The user trusts your judgment - they asked you to commit
- Use appropriate commit types and scopes for semantic versioning
- **Never hand-edit `manifest.md` or work-item change logs as part of committing** - the manifest is generated by `scripts/wrender.sh` from `work.jsonl`; commit those generated/appended files as they stand

## File Enumeration vs. Judgment:

**Deterministic (derive from git, never from memory)**:
- The set of changed/added/deleted files → `git status --porcelain`
- Inspect content only when needed → `git diff --stat`, or a scoped `git diff <path>` as a last resort (pollutes context)

**Your judgment (use session context)**:
- How to group files into logical, atomic commits
- The commit type, scope, and message wording
- Why the changes were made — captured in the message body
