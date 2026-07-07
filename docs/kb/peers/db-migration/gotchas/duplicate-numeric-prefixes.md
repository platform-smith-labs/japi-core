---
type: gotcha
title: "Duplicate numeric prefixes exist; order is strict filename sort"
tags: [migrations, ordering, naming]
timestamp: 2026-07-07T01:02:42Z
description: "Some numeric prefixes repeat across parallel feature strands; execution order is alphabetical filename order, so a new migration must sort after everything merged"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0037_integration_enums.sql
  - migrations/0037_mcp_tool_seam.sql
  - migrations/0038_artifact_plane.sql
  - migrations/0038_integration_provider.sql
  - pkg/migration/migration.go
---
The runner executes migrations in plain ascending **filename string order** — there is no
sequence table or gap detection. Parallel feature strands have merged migrations sharing the
same numeric prefix (and some prefixes carry letter suffixes for addenda that must run
immediately after a base file). Both are safe *once merged*, because the full filename still
sorts deterministically.

The trap: adding a migration whose name sorts **before** any already-applied file. It would
still run (tracking is per-file, not high-water-mark), but on a fresh database it executes in
a different relative position than on existing databases — ordering-sensitive DDL can then
diverge between environments.

Avoid it by always picking the **next unused number after everything already merged**, and
treating a duplicated prefix as an accident of parallel merges, not a pattern to imitate.
