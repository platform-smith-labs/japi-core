---
type: capability
title: "Workflow task catalog"
tags: [task-catalog, visual-builder, palette, node-schema, availability, tenant-blind]
timestamp: 2026-07-09T10:49:10Z
description: "Deployment-static, authenticated-but-tenant-blind catalog of every node/task type — schemas, typed handles, live gating — for the ps-ui v2 visual workflow builder"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - cmd/handlers/workflow_task_catalog.go
  - internal/taskcatalog/catalog.go
  - cmd/server/main.go
  - docs/dev/decisions/system-tasks-carry-no-tenant-context.md
see_also:
  - {repo: ps-workflow, capability: "Custom node catalog (Conductor task types)", intent: "describes the executable node types this catalog advertises", descriptive: false}
  - {repo: ps-ui, capability: "Visual workflow builder", intent: "renders the palette / node-config panels / ${...} wiring from this body", descriptive: true}
---

# Workflow task catalog

**What it does.** Serves a single, deployment-static description of every available node/task type — its
palette category, per-field input schema, output-field shape, and per-task availability — so the ps-ui v2
visual builder can render its node palette and node-config panels, and wire `${...}` references between
node outputs and downstream inputs, without hard-coding any of it.

**How a peer interacts.** `GET /api/v1/workflow-task-catalog`. Authenticated like every read via trusted
gateway headers (`X-User-UUID` + `X-Company-UUID`); returns the full catalog body.

**Observable behavior.** The body is built **once at startup** from the deployment's env gates (the same
flags the workers use) and served **verbatim**. It is **identical for every tenant on a deployment** —
the caller's tenant is authenticated but never used to filter or shape the response. The UI reads it to
populate the palette (grouped by `category`), drive each node's config form (`fields[]`, each field's
`type`/`required`/`source`), draw typed output handles (`output_schema` fields, the `${...}` targets), and
grey out or warn on nodes whose `availability.live` is false on this stack.

**Contract.** In: no body; gateway auth headers only. Out: `{version, tasks[]}`. Each task entry, key
fields: `task` (type name), `category`, `version` (per-entry schema version the UI pins per placed node),
`description`, `input_schema.fields[]`, `output_schema[]`, and `availability`. A field carries `source` =
`user` (rendered/caller-supplied) or `context` (hidden, auto-injected from `_ps` — never rendered), plus
`type`, `required`, and for context fields a `context_key`. Some entries also carry `presets[]` (palette
authoring shortcuts over the same task-type), `allows_additional_inputs` (UI shows a raw-params panel),
`deprecated`, or `experimental`. Errors: 401 on missing/malformed gateway headers **or** a user↔company
mismatch (the mismatch check is delegated to a DB validation, not a header parse).

**Availability shape.** `availability` = `{live, gated_by_env, experimental?}`. `live` reflects **server
env for this stack**, not the caller — for an env-gated node it mirrors that node's live flag
(runtime-stop, collect-result, git-open-pr, await-signal, llm, session-prompt), for a seam-gated node it
is on/off by whether the seam is wired, and it is a plain true for always-on nodes. `gated_by_env` names
the env var behind a gated node (null when ungated). A false `live` means the node exists in the palette
but is not executable on this deployment.

**Invariants.** Deployment-static (rebuilds only on service restart with new env); tenant-blind by
construction (no serving-path code filters by the caller's tenant); authenticated (no anonymous read).
Worker-backed **platform** tasks carry `company_uuid`/`user_uuid` context fields; engine-executed
**system** tasks (HTTP, WAIT, INLINE, JSON_JQ_TRANSFORM, SET_VARIABLE) carry **no** context field at all —
they run in the Conductor engine with nothing to assert tenant against, so isolation for them is a
service-layer property, not a per-node field.

**Failure modes.** No gateway headers, malformed UUIDs, or a user not belonging to the company → 401. There
is no per-tenant "not found" or partial catalog — the body is all-or-nothing and the same for everyone.

**Gotchas.** The trap is reading this as tenant-scoped: it is authenticated but **not** tenant-filtered —
two different tenants on the same deployment get byte-identical bodies. Equally, `availability.live` is a
property of the **stack's env**, not the caller — a node greyed out for one user is greyed out for all.
The catalog only *describes* nodes; it does not execute them, and a `live: false` entry is intentionally
present (so saved workflows referencing it don't strand) but not runnable here. System-task entries
deliberately omit `company_uuid` — do not treat a missing context field on HTTP/WAIT/INLINE/JQ/SET_VARIABLE
as a defect.

**See also.** ps-workflow **Custom node catalog (Conductor task types)** — the executable node types this
catalog advertises and whose per-node behavior the schemas mirror. ps-ui **visual workflow builder** — the
sole consumer that renders the palette, node-config panels, and `${...}` wiring from this body.
