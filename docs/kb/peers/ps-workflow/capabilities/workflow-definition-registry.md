---
type: capability
title: "Workflow definition registry"
tags: [workflow-definition, registry, scopes, conductor, clone-to-override, multi-tenant]
timestamp: 2026-07-07T06:49:45Z
description: "Scoped CRUD + clone + publish for workflow definitions; PS DB is canonical, Conductor is a publish-time derived cache"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_definitions.go
  - cmd/handlers/workflow_definitions_scoped.go
  - cmd/db/workflow_definition.go
  - cmd/db/workflow_definition_scoped.go
  - cmd/db/workflow_definition_mandatory.go
  - cmd/models/workflow_definition.go
see_also:
  - {repo: ps-workflow, capability: "Workflow definition secret refs", intent: "binds secret_store refs to a definition; a separate surface from CRUD"}
  - {repo: ps-workflow, capability: "Workflow execution API", intent: "starts an execution from a published definition, resolving name-to-scope at launch"}
  - {repo: orchestrator, capability: "Agent definition registry", intent: "the sibling scoped-definition model this mirrors", descriptive: true}
---

# Workflow definition registry

**What it does.** The system of record for workflow definitions — a tenant's authored
Conductor-JSON workflow docs, scoped at **company / workspace / project** (mirrors the agent
definition model). Canonical records live in the PS database; Conductor is a **derived cache** that
is written only when a definition is *published*. There is **no merging** across scopes — to
specialise a broader definition you **clone it to a narrower scope** and edit the copy.

**How a peer interacts.** HTTP on port 9005. Create/list per scope:
`POST|GET /api/v1/company/workflow-definitions`,
`POST|GET /api/v1/workspaces/{workspace_uuid}/workflow-definitions`,
`POST|GET /api/v1/projects/{project_uuid}/workflow-definitions`. Scope-agnostic item ops by UUID:
`GET|PUT|DELETE /api/v1/workflow-definitions/{workflow_definition_uuid}`,
`POST .../{uuid}/clone`, `POST .../{uuid}/publish`. `ps-api`/`ps-ui` author definitions through
these routes. The `_uuid` in a create path is the scope entity (workspace/project); the company
comes from the tenant headers, never the path.

**Observable behavior.** Create returns 201 with the full definition including its new
`workflow_definition_uuid` and `version` (starts at 1). Scope is fixed by *which* create route you
call — it is never read from the body. List is ordered by name and excludes archived definitions
unless `include_archived=true`. **Publish** validates the stored `conductor_json` then registers the
**tenant-namespaced** doc into the engine (idempotent upsert), returning the `namespaced_name`,
`version`, and `published=true`; nothing runs in the engine until a definition is published. Delete
is a **soft archive** (204), reversible by a PUT that sets `is_archived=false`.

**Contract.** Create in — key fields: `{name (required, ≤128), conductor_json (required), description?,
ps_metadata?}`. Update in — any subset of `{name, description, conductor_json, ps_metadata,
is_archived, is_mandatory}` (COALESCE patch; an empty body is 400). Clone in — **exactly one** of
`{target_workspace_uuid, target_project_uuid}` (neither/both → 400). Out — a `WorkflowDefinition`;
key fields: `workflow_definition_uuid`, `scope_type`, `company/workspace/project_uuid` (only the
scope-relevant ones populated), `version`, `cloned_from_uuid?`, `is_archived`, `is_mandatory`,
`update_available?`. The internal integer key is never exposed — every identifier is a UUID.

**Invariants.** Every call is tenant-scoped by company; a definition is only ever visible/mutable
within its owning company (cross-tenant clone/read is impossible). Name uniqueness is enforced **per
scope** (duplicate → 409). A clone target must be **strictly more specific** than the source
(company→workspace→project); a same-or-broader target → 422. At most **one mandatory lineage per
company** — turning on `is_mandatory` while a differently-named lineage is already mandatory → 409
(application-layer guard, best-effort under concurrency; launch-time resolution is the deterministic
backstop). The mandatory flag may repeat across scopes for the *same* lineage.

**Failure modes.** Unknown/foreign scope entity or definition → 404. Name clash at the scope → 409.
Malformed `conductor_json` (not a JSON object, or missing a non-empty `name`) is rejected at
**publish** with 422 before the engine is touched. If the engine registration itself fails, publish
returns **502** and the definition stays unpublished (PS DB record is unchanged).

**Gotchas.** Publish is the *only* thing that reaches Conductor — CRUD alone never touches the
engine, so a freshly created/edited definition is not runnable until re-published. Editing a
definition does **not** auto-republish. A published name is **tenant-namespaced** inside the engine,
so the engine-visible name differs from the `name` you supplied. A **clone** pins the source's
version as its `template_version`; a later read sets `update_available=true` once the source lineage
advances past it — this is an advisory signal only, clones are never auto-refreshed. Per-scope
**node-type gating** is design intent: a capability gate is consulted at publish but is a
**permissive stub** in this version, so it does not yet reject disallowed node types.

**Business-critical data.** The `workflow_definition` table holds the canonical record: scope keys,
`name`, `version`, `conductor_json` (the runnable doc), `ps_metadata` (PS annotations),
`cloned_from`/`template_version` (clone provenance + update signal), `is_archived`, `is_mandatory`,
`is_system_provided`. (Tenant scoping applies as everywhere — see context.md.)

**See also / peers.** Secret binding for a definition is a **separate** surface (see *Workflow
definition secret binding* in this repo). Starting a run from a published definition — including the
name→most-specific-scope resolution — belongs to *Workflow execution start*.
