---
description: Create a concise learning note from mistakes/discoveries
argument-hint: [topic-name]
allowed-tools: Bash(*), Read(*), Write(*)
---

# Learning Notes Generator

Create ultra-concise one-liner notes documenting mistakes and fixes for future reference.

## MANDATORY FIRST STEP

**Run these commands FIRST:**

```bash
# Step 1: Get date
date +%Y%m%d

# Step 2: Find max sequence for today
ls docs/learnings/ 2>/dev/null | grep "^YYYYMMDD-" | sed 's/^YYYYMMDD-\([0-9]*\).*/\1/' | sort -n | tail -1
```
(Replace YYYYMMDD with actual date, e.g., `grep "^20260103-"`)

**Calculate NEXT_SEQUENCE:**
- Empty result → use `0001`
- Got number → add 1, zero-pad to 4 digits

## Format Requirements

**Filename**: `docs/learnings/{DATE}-{SEQ}-{topic}.md`

**Topic**: Use `$1` if provided, else auto-generate from session (e.g., "postgresql-nullif-casting")

## Content Structure

```markdown
# {Short Topic Title}

**Date**: YYYY-MM-DD
**Tags**: tag1, tag2, tag3

## One-Liner Summary
Single sentence capturing the core learning.

## Mistakes → Fixes

### {Issue 1}
❌ **Don't**: What I did wrong (one line)
✅ **Do**: Correct approach (one line)
**Why**: Brief reason (one line)

### {Issue 2}
❌ **Don't**: ...
✅ **Do**: ...
**Why**: ...

## Code Snippets (if needed)

Keep to 3-5 lines max per snippet. Only include if essential.

## Quick Reference
- Bullet point reminders
- Maximum 5 items

## Related
- Link to detailed journal if exists
- Related learning docs
```

## CRITICAL RULES

1. **Maximum 100 lines total** - Keep it SHORT
2. **One-liners only** - No paragraphs, no explanations
3. **❌/✅ format** - Clear wrong vs right
4. **Code snippets: 3-5 lines max** - Only if absolutely necessary
5. **No duplication** - If it's in the journal, link it, don't repeat it

## Content Guidelines

**What to include:**
- Specific mistakes made
- Concrete fixes applied
- Technical gotchas discovered
- Commands that worked

**What to exclude:**
- Session narrative
- Detailed explanations (that's for journal)
- Multiple code examples
- Background context

**Tone**: Direct notes to future self

## Examples of Good One-Liners

✅ Good:
```
❌ Don't: Use current_setting(...)::uuid directly
✅ Do: Use NULLIF(current_setting(...), '')::uuid
Why: current_setting returns '' not NULL when unset
```

❌ Too verbose:
```
When working with PostgreSQL's current_setting function, we discovered
that it returns an empty string instead of NULL when the setting is not
configured. This causes issues when casting to UUID type because...
```

## Topic Naming

Use kebab-case, be specific:
- ✅ `postgresql-nullif-casting`
- ✅ `drizzle-migration-journal`
- ✅ `github-actions-cache-invalidation`
- ❌ `database-issues`
- ❌ `bugs-fixed`
- ❌ `learnings`

## CLAUDE.md Integration

After creating learning doc, optionally suggest adding key points to `.claude/CLAUDE.md`:

```markdown
## Learnings: {Topic}
- Point 1
- Point 2
- See: docs/learnings/{filename}
```
