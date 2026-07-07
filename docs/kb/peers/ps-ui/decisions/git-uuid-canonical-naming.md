---
type: decision
title: "Git UUIDs use full canonical prefixed names"
tags: [decision, git, identifiers]
timestamp: 2026-07-07T06:27:35Z
description: "ps-ui expects fully-prefixed git UUID field names, never abbreviated"
repo: ps-ui
commit_sha: 1f5f197
evidence: [docs/dev/decisions/git-uuid-canonical-naming.md]
---

# Git UUIDs use full canonical prefixed names

**Consequence for a peer.** ps-ui expects git-entity UUID fields under their **full prefixed names** —
`git_connection_uuid`, `git_installation_uuid`, `git_installation_repo_uuid` — which are **distinct,
non-interchangeable** entities. A backend response that abbreviates one to a bare `connection_uuid` /
`installation_uuid` invites the UI to conflate two different entities. Keep the canonical names on the
wire. (See the git-connections capability for the one endpoint that historically abbreviated it.)
