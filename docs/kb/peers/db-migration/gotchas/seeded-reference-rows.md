---
type: gotcha
title: "Some reference tables are seeded by migrations, not by services"
tags: [migrations, seed-data, reference-data]
timestamp: 2026-07-07T01:02:42Z
description: "A few migrations insert reference rows (git providers, integration providers including Slack, default agent definitions) at migration time; peers must not assume these tables start empty or re-seed them"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0004_git.sql
  - migrations/0025_agent_definitions_and_secrets.sql
  - migrations/0038_integration_provider.sql
  - migrations/0049_slack_provider_seed.sql
---
Migrations here are not schema-only: a handful also **insert reference rows** at migration
time — known git providers, integration providers (including Slack) with their supported
auth types, and default agent definitions.

Traps for a peer service:
- Do not assume these tables start empty on a fresh database — the seed rows exist as soon
  as migrations have run.
- Do not re-seed or "bootstrap" these rows from application code; duplicates or conflicting
  values fight the migration-owned data. Extending the set is done with a **new migration**
  in this repo, not an app-level insert.
- The exact row contents are owned by the migrations; query the tables at runtime rather
  than hard-coding assumptions about which providers exist.
