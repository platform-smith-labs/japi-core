When you learn something worth remembering — a rule, convention, correction, or architectural decision — do NOT store it in Claude's auto-memory system. Instead, create or update a markdown file in `docs/dev/decisions/` so it is git-tracked and shared across all developers.

## When to create a decision document

- User corrects your approach ("don't do X", "always do Y")
- A non-obvious convention is established during implementation
- An architectural trade-off is made with specific reasoning
- A bug or failure reveals a rule that should be codified

## File naming

Use kebab-case descriptive names: `one-statement-per-no-transaction-migration.md`, `always-use-build-sh.md`

## Template

Every decision document MUST follow this format:

```markdown
# Decision: <Short Descriptive Title>

**Date**: YYYY-MM-DD
**Status**: Accepted | Deprecated | Superseded by [link]
**Context**: <What prompted this decision — a bug, audit, feature, or discussion>

## Decision

<1-2 sentence summary of the rule or convention being established.>

## Rules

### 1. <Rule name>

<Explanation of the rule.>

\`\`\`sql/go/bash
-- CORRECT: <description>
<correct example>

-- WRONG: <description>
<incorrect example>
\`\`\`

### 2. <Rule name>

<Repeat for each rule. Use numbered sub-sections. Always include CORRECT/WRONG code examples where applicable.>

## Rationale

<Why this decision was made. Cover the key reasons: security, performance, consistency, developer experience, or past incidents. Each reason can be a paragraph or a sub-heading.>

## Exceptions

<Numbered list of cases where this rule does not apply, with brief justification for each.>

1. **<Exception>** — <why it's acceptable>

## Enforcement

<How this rule is checked — code review, linting, CI, CLAUDE.md, etc.>

## See Also

- [Related Decision](./related-decision.md)
- [Related Work Item](../../work/work-NNNN/...)
```

## Rules for writing decisions

1. **One decision per file** — do not combine unrelated decisions
2. **Code examples are mandatory** — every rule must show CORRECT and WRONG patterns
3. **Rationale explains "why"** — not just "what". Future readers need to understand the reasoning to judge edge cases
4. **Exceptions are explicit** — if there are none, omit the section entirely rather than writing "None"
5. **Update, don't duplicate** — if an existing decision covers the topic, update it instead of creating a new file
6. **Keep it scannable** — developers should be able to read a decision in under 2 minutes
