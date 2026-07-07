---
type: decision
title: "User-facing terminology: runtime→agent, session→conversation"
tags: [decision, terminology, vocabulary]
timestamp: 2026-07-07T06:27:35Z
description: "Schema entities runtime/session are shown to users as agent/conversation"
repo: ps-ui
commit_sha: 1f5f197
evidence: [docs/dev/decisions/terminology-runtime-to-agent.md]
---

# User-facing terminology: runtime→agent, session→conversation

**Consequence for a peer.** ps-ui deliberately renders the schema entities `runtime` and `session` to
users as **"agent"** and **"conversation"**. Field names, URL paths, error codes, and wire identifiers
are **out of scope** and stay `runtime`/`session`. So when reading ps-ui's UI vocabulary against your
backend contracts: "agent" in the UI == a `runtime`, "conversation" == a `session`. Keep your contract
field names unchanged — the rename is presentation-only.
