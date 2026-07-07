---
type: gotcha
title: "This KB may lag the live schema — check the last-synced marker first"
tags: [freshness, staleness, schema, kb]
timestamp: 2026-07-07T01:10:52Z
description: "The schema changes rapidly; the KB reflects it only up to the last-synced commit in log.md — newer migrations (including whole new tables) are absent until regeneration"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - docs/kb/log.md
  - migrations/
---
The database schema changes rapidly — new migrations land continuously — but this KB is
regenerated on demand, not on every merge. Everything under `self/` describes the schema
**only up to the last-synced commit**: the newest entry in `log.md` (date + commit sha),
mirrored in each concept's `commit_sha` frontmatter.

The trap: scoping a task against this reference when migrations newer than the last-synced
marker exist. Those changes — new tables, new columns, altered constraints, extended enums —
are **completely absent** from the KB, and nothing inside a concept will hint at what's
missing.

How to detect drift: compare the last-synced marker against the repo's current state (any
migration added after the marker's commit is undocumented). When in doubt, ask the owning
team for a KB regeneration before trusting schema details for recently-changed domains.
