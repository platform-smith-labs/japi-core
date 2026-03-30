---
description: Create a development journal entry for the current session
argument-hint: [session-name]
allowed-tools: Bash(*), Read(*), Write(*), TodoWrite(*)
---

# Session Journal Generator

Create a comprehensive markdown journal documenting our current development session.

## MANDATORY FIRST STEP - DO NOT SKIP

**YOU MUST RUN THESE COMMANDS FIRST before doing anything else:**

**Step 0**: Verify and navigate to project root directory:
```bash
# Check if we're in the project root (should show docs/ directory)
pwd
ls -d docs 2>/dev/null || echo "ERROR: Not in project root - docs/ directory not found"

# If docs/ not found, navigate to project root
# Find the git root directory (project root is usually the git root)
cd $(git rev-parse --show-toplevel 2>/dev/null || echo ".")

# Verify again
pwd
ls -d docs || echo "CRITICAL ERROR: Cannot find docs/ directory"
```

**Step 1**: Get today's date:
```bash
date +%Y%m%d
```

**Step 2**: Find the maximum existing sequence number for today (use the date from Step 1):
```bash
# IMPORTANT: This command MUST be run from the project root directory
# If you're not in the root, the sequence number will be wrong!
ls docs/journal/ 2>/dev/null | grep "^YYYYMMDD-" | sed 's/^YYYYMMDD-\([0-9]*\).*/\1/' | sort -n | tail -1
```
(Replace YYYYMMDD with actual date from Step 1, e.g., `grep "^20260105-"`)

**Step 3**: Calculate next sequence:
- If Step 2 returned empty (no files for today): use `0001`
- If Step 2 returned a number: add 1 and zero-pad to 4 digits
- Example: if max is `0014`, next is `0015`

**STOP AND RECORD the values before proceeding:**
- TODAY = (from Step 1)
- NEXT_SEQUENCE = (from Step 3)

## Task Requirements

1. **Run the sequence command above FIRST** - This gives you TODAY and NEXT_SEQUENCE values
2. **Generate Filename**: Use pattern `{TODAY}-{NEXT_SEQUENCE}-{session-name}.md`
   - Use the TODAY value from the command output
   - Use the NEXT_SEQUENCE value from the command output
   - session-name = argument `$1` if provided, otherwise auto-generate from session content
   - Example: `20251218-0015-journal-sequence-fix.md`

3. **Analyze Session Context**: Review the conversation history to understand what was accomplished

4. **Generate Journal Content** with these sections:
   - Session metadata (date, session ID matching filename sequence, type)
   - Overview of session goals
   - User requests and prompts made during the session
   - Technical work performed with commands and outputs
   - Files created/modified with key changes
   - Results and accomplishments
   - Next steps and recommendations
   - Session metrics (duration estimate, commands run, files changed)

5. **Write the file** to `docs/journal/{filename}`

## Session Name
Session name: "$1" (if provided) or auto-generate from session activities

## Content Guidelines
- Focus on documenting the "what", "why", and "how" of the session
- Include actual commands run and their outputs
- Document any problems encountered and how they were resolved
- Provide context that would be valuable for future reference
- Use proper markdown formatting with clear headings and code blocks

## CRITICAL RULES

1. **ALWAYS verify working directory FIRST (Step 0)** - Ensure you're in project root before any other steps
2. **ALWAYS run the sequence command (Step 2)** - Never assume or guess sequence numbers
3. **Use NEXT_SEQUENCE from command output** - This is the only correct value
4. **Session ID in metadata must match** - e.g., if filename is `20260105-0015-*`, Session ID is `20260105-0015`
5. **Never reuse sequence numbers** - The command handles this automatically
6. **If sequence command returns wrong results** - You're probably in the wrong directory (see Step 0)
7. **If command fails, debug it** - Don't fall back to hardcoded values
8. **Keep it concise but thorough** - Balance detail with readability

## Common Pitfalls to Avoid

❌ **WRONG**: Running sequence command from subdirectory (e.g., `docs/journal/`)
- Results in: Sequence number `0001` when it should be `0009`
- Fix: Always run from project root (see Step 0)

❌ **WRONG**: Assuming next sequence without checking
- Results in: Duplicate session IDs
- Fix: Always run Step 2 command

✅ **CORRECT**: Follow Steps 0, 1, 2, 3 in order every time
- Step 0: Verify/navigate to project root
- Step 1: Get date
- Step 2: Find max sequence for today
- Step 3: Calculate next sequence
