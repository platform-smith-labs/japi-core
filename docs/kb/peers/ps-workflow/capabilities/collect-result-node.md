---
type: capability
title: "collect-result node"
tags: [workflow-node, artifacts, session, gated, not-live, conductor-worker]
timestamp: 2026-07-07T06:49:45Z
description: "Workflow node that harvests a completed session's result artifacts for downstream workflow steps"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - internal/workers/nodes/collect_result.go
  - internal/platform/db_platform.go
see_also:
  - {repo: orchestrator, capability: "Session artifact read", intent: "owns GET /api/v1/sessions/{uuid}/artifacts — the gated read route this node consumes", descriptive: true}
---

# collect-result node

**What it does.** A custom Conductor worker node that harvests a coding session's result
artifacts (e.g. `diff.patch`, `summary.md`) so a downstream workflow step (typically a SWITCH)
can branch on what the agent produced. It reads the session's `scope=session, kind=result`
artifacts, with content pre-resolved to bytes.

**How a peer interacts.** Reference the Conductor task type `collect-result` in a workflow
definition and supply `inputParameters._ps.session_id` (required). The node runs when the
workflow reaches the step; it is not a peer-callable HTTP endpoint.

**Observable behavior — GATED / may be NOT_LIVE.** The node's read depends on an orchestrator
route (`GET /api/v1/sessions/{uuid}/artifacts`) that ships per-stack behind the
`CollectResultLive` flag. Until that route is deployed on a stack, the node returns a NOT_LIVE
terminal state (surfaced as a Conductor FAILED task carrying a `not_live: true` output marker)
— an honest failure, never a false empty harvest. A peer authoring or running a workflow on an
un-deployed stack must expect this not-live outcome. When live, the node completes
synchronously (not a park node).

**Contract.** In (`_ps`): `session_id` (req). Out (when live): `{session_id, found, count,
artifacts[]}`. Each artifact — `key fields:` `name`, `kind`, `content` (bytes, pre-resolved),
`content_type`, `artifact_version_uuid`, `status` (contract-fixed to `published`). Missing
`session_id` fails the node. NOT_LIVE output: `{session_id, not_live: true}`.

**Invariants.** Session resolution is company-scoped: an unknown or cross-tenant session
resolves to `found=false` with NO downstream HTTP call — never a cross-tenant read. A
`found=false` result (unknown/cross-tenant session OR zero artifacts) is a clean COMPLETED
RESULT for a downstream SWITCH, NOT a failure. A genuine transport/5xx error from the read
route DOES fail the node (no false success).

**Failure modes.** Missing `session_id` → fail. Route not deployed → NOT_LIVE (FAILED +
`not_live` marker). Non-404 error from the artifact route → fail. Empty or unknown session →
COMPLETED with `found=false`, `count=0`.

**Gotchas.** The `_ps.session_id` input value is a session NAME, not a session UUID — the node
resolves it to the session's UUID internally before reading. NOT_LIVE presents as a Conductor
FAILED task; distinguish it from a real failure by the `not_live: true` output marker, not by
the status alone. `found=false` does not mean an error — check it explicitly in the workflow.

**See also / peers.** The artifact read route and the result-artifact write path are owned by
the orchestrator; ps-workflow only consumes them.
