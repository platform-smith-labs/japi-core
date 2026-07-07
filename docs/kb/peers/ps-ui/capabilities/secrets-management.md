---
type: capability
title: "Secrets & secret references"
tags: [secrets, secret-refs, credentials, agent-definitions, scoping]
timestamp: 2026-07-07T06:27:35Z
description: "How the UI consumes the secrets store + agent-definition secret-refs, and how it keeps secret values out of the client"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/secrets.ts
  - src/api/secret-refs.ts
  - src/components/secrets/resolution.ts
  - src/components/secrets/EffectiveValue.tsx
  - src/components/secrets/UpdateSecretValueDialog.container.tsx
  - src/components/secrets/CreateSecretDialog.container.tsx
  - src/components/secrets/SecretDetail.tsx
see_also:
  - {repo: ps-api, capability: "Secrets & secret-ref API", intent: "owns the endpoints this UI calls; enforces value masking + write fail-closed", descriptive: true}
  - {repo: orchestrator, capability: "Secret injection at launch", intent: "resolves a secret-ref by name within a scope chain and injects the value into a runtime", descriptive: true}
---

# Secrets & secret references

**What it does.** The UI is a pure consumer of two distinct backend concepts. A **secret** (a
`SecretStore`) is a stored credential value plus metadata, owned at one scope level — `company`,
`workspace`, or `project` — and inherited downward. A **secret-ref** (an `AgentDefinitionSecretRef`)
is a *pointer* attached to an agent definition that names a secret and declares how to inject it at
runtime launch. A ref carries **no value**: it references a secret **by name + type**, and resolution
(name → concrete stored value, within the launch scope chain) happens server-side at launch, not here.

**Backend contracts consumed** (paths are what this UI calls; ps-api owns them).

Secrets (scope-keyed list/create):
- `GET /v1/company/secrets` · `GET /v1/workspaces/{ws}/secrets` · `GET /v1/projects/{proj}/secrets`
  — list; append `?include_inherited=true` to fold in ancestor-scope definitions. Returns a flat
  `SecretStore[]` (or `null` when empty).
- `POST` on those same three paths — create at that scope.
- `GET /v1/secrets/{uuid}` · `PUT /v1/secrets/{uuid}` · `DELETE /v1/secrets/{uuid}` — scope-agnostic
  read / merge-update / delete.
- `GET /v1/secrets/{uuid}/value` — the only endpoint returning values (`{values: {field → string}}`);
  secret-typed fields are **masked** when the secret's `visibility` is `hidden`, config fields
  (string/url) returned.
- `GET /v1/secrets/{uuid}/usage` — where this secret is referenced (agent definitions).
- `GET /v1/secrets/{uuid}/descendants` — lower-scope definitions that override this one by name.
- `GET /v1/secrets/search` — key params `q`, `workspace_uuid`, `project_uuid`, `secret_type`,
  `agent_definition_uuid`, `exclude_usage_context`.
- `GET /v1/secret-schemas` · `GET /v1/secret-schemas/{type}` — per-type field shapes.

Secret-refs (attached to an agent definition):
- `GET` / `POST /v1/agent-definitions/{def}/secret-refs` — list / create.
- `DELETE /v1/agent-definitions/{def}/secret-refs/{refUuid}`.
- `GET /v1/agent-definitions/{def}/secret-refs/status` — key params `workspace_uuid`,
  `project_uuid`, `environment_uuid`; returns each ref with a `resolution` telling whether it
  currently resolves under that scope.

**Key fields.**
- `SecretStore` (key fields): `name`, `secret_type` (oauth · bearer_token · api_key · ssh_key ·
  basic_auth · env_vars), `scope_level`, `provision_status` (provisioned · placeholder · revoked),
  `secret_policy` (open · locked), `visibility` (hidden · visible), `has_secret` (boolean presence
  flag — never the value), `fulfillment_scope`, `placeholder_hint`, `is_system_provided`.
- `AgentDefinitionSecretRef` (key fields): `secret_name`, `secret_type`, `usage_context` (mcp_server ·
  skill · command · hook · env · template), `injection_method` (template · env_var · file),
  `inject_as`, `required`.
- Ref `resolution` (key fields): `status` (resolved · placeholder · missing), `resolved_scope`,
  `provision_status`, `secret_store_uuid`.

**Observable behavior.** A secret is created/edited at one scope and cascades to every scope below it.
Effective-value resolution the UI performs client-side over the returned list: at a standpoint scope,
only definitions at-or-above apply; a `locked` definition at a broader scope **wins over** any
more-specific one and **blocks** overriding below it — otherwise the most-specific definition wins. A
secret-ref's `status` endpoint is the readiness signal a peer would poll: `resolved` = a provisioned
secret backs the name at that scope, `placeholder` = the name is declared but has no value yet,
`missing` = nothing resolves.

**Failure modes.**
- Create name collision at a scope → HTTP **409**; surfaced inline on the name field
  ("A secret with this name already exists").
- Update that clears a **required** field (validated against the merged result) → HTTP **422**;
  surfaced as an error toast.
- **Write fail-closed without server-side KMS** — value writes can be rejected server-side (ps-api's
  fail-closed posture when plaintext secrets are disallowed). The UI has no special branch for this:
  it surfaces the backend's returned error message via a generic error toast. The exact status/shape
  is UNKNOWN from this repo — a peer should not assume 409/422.

**Gotchas.**
- **Secret ≠ secret-ref.** The secret is the stored value at a scope; the ref is a named pointer on an
  agent definition. Deleting/creating one does not touch the other; a ref can be `missing`/`placeholder`
  even though it exists, when no secret backs its name.
- **Values are never returned by list/get.** `SecretStore.has_secret` only signals presence. Values
  come *only* from `GET /secrets/{uuid}/value`, and secret-typed fields there are masked when
  `visibility=hidden`. The UI never renders a value unless `visibility=visible` and the user explicitly
  reveals it (otherwise shown as `••••••••`); it sets query cache `gcTime/staleTime: 0` and clears form
  state on dialog close so plaintext is not retained client-side.
- **Refs resolve by NAME, not UUID.** "Copy ref" yields a `{{secret:NAME}}` template token pasted into
  an agent-definition file; resolution keys off the name within the launch scope chain. `SecretStore`
  and refs both expose a `secret_store_uuid`, but the UI groups/keys by `name` across scopes.
- **Update is merge-by-key.** `PUT /secrets/{uuid}` overwrites only the keys sent (empty string
  clears; omitted keys are preserved) — the UI sends only fields the user edited.

**See also.** ps-api — *Secrets & secret-ref API* (owns these endpoints and the masking / write
fail-closed enforcement). orchestrator — *Secret injection at launch* (resolves a ref by name within
the scope chain and injects the value into the runtime).
