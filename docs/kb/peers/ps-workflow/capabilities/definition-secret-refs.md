---
type: capability
title: "Workflow definition secret refs"
tags: [secrets, workflow-definition, secret-store, execution, multi-tenant]
timestamp: 2026-07-07T06:49:45Z
description: "Declare a definition's secret dependencies as refs into secret_store; resolved (not embedded) at execution time"
repo: ps-workflow
commit_sha: 6b13ca9
evidence:
  - cmd/handlers/workflow_definition_secret_refs.go
  - cmd/db/secret_resolution.go
  - cmd/db/workflow_definition_secret_ref.go
  - cmd/models/secret_resolution.go
  - cmd/models/workflow_definition_secret_ref.go
see_also:
  - {repo: ps-workflow, capability: "Workflow definition registry", intent: "the definition a ref is attached to; refs share its scope"}
---

# Workflow definition secret refs

**What it does.** Binds the secrets a workflow definition needs — by *reference*, not
value. A ref names a secret in the platform `secret_store` (by name + type) and how it
should be delivered at run time. The definition itself never embeds secret material;
values are looked up and decrypted only when an execution starts, then discarded.

**How a peer interacts.** Four HTTP endpoints, all under a parent definition UUID:
- `POST /api/v1/workflow-definitions/{uuid}/secret-refs` — attach a ref.
- `GET  /api/v1/workflow-definitions/{uuid}/secret-refs` — list the definition's refs.
- `GET  /api/v1/workflow-definitions/{uuid}/secret-refs/status` — per-ref resolution
  readiness (see below); never returns a decrypted value.
- `DELETE /api/v1/workflow-definitions/{uuid}/secret-refs/{secret_ref_uuid}` — remove a
  ref (`204`).

**Observable behavior.** Attaching a ref just records the dependency; nothing is
resolved or decrypted at attach time. Actual resolution happens when a workflow
*execution* starts: this service scope-walks `secret_store` across the company →
workspace → project hierarchy, picks the most-specific provisioned match per secret name
(a `locked`-policy secret pins resolution to its own, possibly broader, scope), decrypts,
and injects. The status endpoint runs the same scope walk *without* decrypting, so a peer
can check readiness ahead of an execution.

**Contract.** Attach body — `key fields:` `secret_name`, `secret_type`
(`oauth|bearer_token|api_key|ssh_key|basic_auth|env_vars`), `usage_context`
(`node|workflow_input|template|env`), `injection_method` (`template|env_var|file`),
optional `context_identifier`, `inject_as`, `file_mode`, and `required` (defaults **true**
when omitted). Returns the created ref keyed by `secret_ref_uuid`. Errors: `404` if the
definition is not found in the tenant; `409` if a ref with the same
(definition, secret_name, usage_context, context_identifier) already exists;
`400` on a malformed UUID.

Status rows — `key fields:` `secret_ref_uuid`, `secret_name`, `secret_type`, `required`,
`status`, and `resolved_scope` (the scope level a winner was found at, when any). `status`
is one of:
- `resolved` — a provisioned matching secret exists in scope.
- `missing` — no candidate found at any scope for that name.
- `placeholder` — a candidate exists but is unprovisioned (no value yet).
- `type_mismatch` — a value resolves, but its stored type differs from the ref's declared
  `secret_type`.

**Invariants.** Refs, decryption, and status are all tenant-scoped by the parent
definition's company; a ref never resolves against another tenant's `secret_store`.
Secret values are never persisted on the ref and never logged (only names appear in logs).
The int primary key never leaks — refs are addressed by UUID only.

**Failure modes.** At execution start, if **any** `required` secret is unresolved
(missing, unprovisioned, or fails to decrypt), the start is **blocked**: the caller
receives a `422`-class error naming the unresolved secrets, and **no** workflow is started
in the engine — there is no partial run. Unresolved **optional** (`required:false`) secrets
do not block; the execution starts *degraded* with a warning, and a `type_mismatch` on any
secret is likewise a warning, not a block. A peer checking `/secret-refs/status` before
starting can see `missing`/`placeholder`/`type_mismatch` and provision before running.

**What a peer must provision first.** Before an execution that uses secrets can succeed,
the referenced secret must exist in `secret_store` at (or above) the definition's scope,
be **provisioned** (not a `placeholder`), match the declared `secret_type`, and — if the
definition scopes to an environment — match that environment (or be environment-agnostic).
`secret_store` provisioning is owned outside this service (the platform secret store /
orchestrator), not by these endpoints.

**Gotchas.** Attaching a ref does **not** validate that the secret exists — a ref can sit
`missing`/`placeholder` indefinitely and only bites at execution start. `required` defaults
to **true**, so an omitted flag means an unprovisioned secret hard-blocks the run. The
status endpoint walks at the *definition's own* workspace/project scope with no environment
filter, so its readiness view can differ slightly from an execution-time resolution that
carries a specific environment.

**Business-critical data.** `workflow_definition_secret_ref` stores each declared ref
(name, type, usage context, injection method, `required`) linked to its
`workflow_definition`; the referenced secret material lives only in `secret_store`,
resolved transiently at run time. (Tenant scoping applies as everywhere — see context.md.)

**See also.** ps-workflow **Workflow definition registry** — owns the parent definition
these refs attach to and whose scope they inherit.
