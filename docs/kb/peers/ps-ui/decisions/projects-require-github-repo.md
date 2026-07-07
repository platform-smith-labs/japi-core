---
type: decision
title: "Every project requires a GitHub repo"
tags: [decision, projects, git]
timestamp: 2026-07-07T06:27:35Z
description: "A project is always created by importing a GitHub repo — no create-from-scratch"
repo: ps-ui
commit_sha: 1f5f197
evidence: [docs/dev/decisions/projects-require-github-repo.md]
---

# Every project requires a GitHub repo

**Consequence for a peer.** ps-ui has no create-from-scratch project path — a project is **always**
created by importing a GitHub repo (create + git-link, as one flow). A backend peer should assume every
project ps-ui creates carries a linked git repo; there is no UI route that produces a repo-less project.
