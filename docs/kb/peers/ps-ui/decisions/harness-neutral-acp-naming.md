---
type: decision
title: "ACP transcript layer is harness-neutral"
tags: [decision, acp, sessions]
timestamp: 2026-07-07T06:27:35Z
description: "ps-ui renders session frames via harness-agnostic ACP, not a per-agent path"
repo: ps-ui
commit_sha: 1f5f197
evidence: [docs/dev/decisions/harness-neutral-acp-naming.md]
---

# ACP transcript layer is harness-neutral

**Consequence for a peer.** ps-ui renders a coding session's streamed output through a single
**harness-neutral ACP (Agent Client Protocol)** transcript layer — it looks identical whether the
session's agent is `claude_code` or `codex_cli`. A backend peer should deliver session frames in the
shared ACP `session_event` vocabulary regardless of harness; ps-ui does **not** branch rendering on the
harness. (See the coding-sessions capability for the event kinds.)
