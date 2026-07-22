---
type: decision
title: "Engine workflow name is Namespace(company, definition name)"
tags: [decision, publish, namespacing, execution, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "Publish registers a definition under its PS name (company-namespaced), not the conductor_json's internal name; start resolves the same name"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - docs/dev/decisions/engine-workflow-name-is-namespaced-def-name.md
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_executions.go
see_also:
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "the publish path this rule governs"}
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "start resolves the run by the same namespaced name"}
---

# Engine workflow name is Namespace(company, definition name)

**Consequence for a peer.** The engine-visible workflow name is always derived from the PS
`workflow_definition.name` (company-namespaced), **never** from the `conductor_json`'s own internal
`name`. Publish overrides the doc's top-level `name` to the PS definition name before registering, so
publish and start always agree. A peer therefore keys everything by `workflow_definition_uuid` and
never needs (or should trust) the `conductor_json.name`; the `namespaced_name` a start returns is the
engine-internal identity, useful for observability only, not as a lookup key. If a definition's
authored `conductor_json.name` differs from its PS `name`, that difference is invisible at runtime —
the platform owns the engine identity. (Sub-workflow/SUB_WORKFLOW composition is out of scope for v1
and would need separate name-rewriting.)
