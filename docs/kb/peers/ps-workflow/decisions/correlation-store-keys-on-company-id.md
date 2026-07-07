---
type: decision
title: "Correlation store keys on company_id (int), durable across restarts"
tags: [decision, correlation-store, tenancy, durability, park]
timestamp: 2026-07-07T06:49:45Z
description: "The session→task correlation store anchors on integer company_id and survives restarts, so parked tasks are re-armed after a redeploy"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - docs/dev/decisions/correlation-store-keys-on-company-id.md
  - internal/workers/host.go
---

# Correlation store keys on company_id (int), durable across restarts

**Consequence for a peer.** The durable session→Conductor-task correlation (the record that lets a
parked task be completed out-of-band) is tenant-keyed on the platform-standard integer
`company_id` — the composite `(company_id, session_name)` is the lookup/uniqueness key. Because the store is durable, a parked
task (running turn, pending approval, in-flight launch) **survives a ps-workflow restart**: on
startup a reconciliation sweep re-arms live rows and expires past-deadline ones. A peer can rely on
parks not being silently lost across a redeploy — but completion events are still matched only
within the owning tenant, so a completion for an unknown or cross-tenant session is a benign no-op.
