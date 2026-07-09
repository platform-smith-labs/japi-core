---
type: capability
title: "Secrets and integration credentials"
tags: [secrets, integrations, credentials, encryption, multi-tenancy]
timestamp: 2026-07-09T10:35:01Z
description: "Two subsystems: orchestrator-proxied secret stores, and DB-direct integration connections with server-side AEAD encryption"
repo: ps-api
commit_sha: a4683c0
evidence:
  - cmd/handlers/secret_stores.go
  - cmd/handlers/secret_schemas.go
  - cmd/handlers/agent_definition_secret_refs.go
  - cmd/handlers/integrations.go
  - cmd/db/integration_connection_operations.go
  - cmd/db/channel_operations.go
  - cmd/server/main.go
  - pkg/crypto/aead/crypter.go
  - pkg/crypto/aead/keyprovider.go
  - pkg/crypto/aead/aad.go
  - docs/dev/decisions/kek-provider-openbao-transit.md
see_also:
  - {repo: ps-api, capability: "Auth and identity gateway", intent: "supplies the validated identity every secret/integration call is scoped by"}
  - {repo: orchestrator, capability: "Secret store management", intent: "owns all secret-store business logic behind the verbatim proxy", descriptive: true}
  - {repo: orchestrator, capability: "Coding-agent credential resolution at launch", intent: "decrypts credential_enc and picks the connection a runtime instance freezes to", descriptive: true}
---

# Secrets and integration credentials

Two distinct subsystems share this surface. **(A) Secret stores** — generic user secrets at
company/workspace/project scope — are pure proxy: ps-api forwards verbatim to orchestrator, which owns
all logic. **(B) Integration connections** — provider credentials such as claude_code/codex coding-agent
tokens — are handled DB-direct in ps-api, with server-side AEAD encryption of secret fields.

## A. Secret stores (proxied to orchestrator)

**How a peer interacts.** REST under the gateway (auth required, identity headers injected):
create/list per scope (`/api/v1/company/secrets`, `/api/v1/workspaces/{workspace_uuid}/secrets`,
`/api/v1/projects/{project_uuid}/secrets`); scope-agnostic item ops GET/PUT/DELETE
`/api/v1/secrets/{secret_store_uuid}` plus `/usage` (agent definitions referencing the secret),
`/descendants` (more-specific-scope secrets shadowing this one), `/value` (decrypted values), and
`GET /api/v1/secrets/search`. Secret *type* schemas: `GET /api/v1/secret-schemas[/{secret_type}]`.
Agent-definition secret dependencies: `/api/v1/agent-definitions/{agent_definition_uuid}/secret-refs`
(create/list/delete, and `/status` for resolution state).

**Observable behavior.** Requests and responses pass through unchanged — status codes, bodies, and
errors are orchestrator's. ps-api only authenticates, resolves a project's owning workspace for the
project-scoped routes, and injects trusted identity. `/value` returns decrypted values for visible
secrets and 403 for hidden ones. `/descendants` is metadata-only (each row carries `secret_policy`);
an empty result is an empty list, an absent or cross-tenant secret is 404.

**Gotcha.** Nothing about secret-store semantics (scoping, shadowing, hiding, policies) is decided in
ps-api — a peer questioning those behaviors must ask orchestrator's secret-store capability.

## B. Integration connections (DB-direct, encrypted in ps-api)

**What it does.** Stores a company's integration credentials (per provider + auth type), encrypts the
secret fields at write time, and manages lifecycle and workspace assignment. Any authenticated company
member may operate them (no admin-role gating yet).

**How a peer interacts.** Global catalog: `GET /api/v1/integrations/providers` and
`/providers/{provider_uuid}/auth-types` (each auth type carries a declarative `field_schema` +
`config_schema`). Connections: POST/GET `/api/v1/integrations/connections`, GET/PATCH/DELETE
`/connections/{connection_uuid}`, plus `/disable` and `/enable`. Workspace assignment:
GET/POST `/api/v1/workspaces/{workspace_uuid}/integrations`, DELETE `…/integrations/{connection_uuid}`.

**Observable behavior.** Create validates required fields against the auth type's `field_schema`
(400 listing missing fields; 404 for an unknown provider/auth-type pair), encrypts server-side, and
returns the connection *without* secret material — no read ever exposes it; reads expose only a
`has_credential` boolean. `personal_only` auth types are always owned by the requesting user
(enforced server-side, not caller-trusted); a caller may also request personal ownership of a shared
auth type. List returns company-shared connections plus the caller's own personal ones (`owned_by_me`
flag). Disable pauses (status=disabled, secret retained, `has_credential` stays true; idempotent);
enable reactivates without token re-entry, but a revoked connection cannot be enabled (409). Revoke
(DELETE) is a soft-revoke: secret material is zeroed, status=revoked, `has_credential` false.
PATCH updates only non-secret fields — there is no re-key/rotate-secret endpoint; replace the
connection to change a credential.

**Invariants.** Encryption is AES-256-GCM envelope: a fresh per-row data key, wrapped by a
key-encryption key from an injected provider; the authenticated data binds company + connection +
auth type, so a ciphertext cannot be replayed across tenants or rows. ps-api encrypts on write;
orchestrator decrypts at launch using the same shared scheme — the blob is opaque to everyone else.
Duplicate-name or single-connection-per-provider cardinality violations → 409; wrong tenant or
missing rows → 404.

**Failure modes / boot gate.** The only key provider *wired at boot* is a dev-only plaintext stub
gated by `PS_ALLOW_PLAINTEXT_SECRETS=1`. Without that env var the gateway still boots, but every
integration-credential write fails closed with a clear error — it never silently encrypts under the
stub. An OpenBao Transit provider implementation exists in-tree but is not yet wired (ADR "KEK
provider for AEAD envelope crypto is OpenBao Transit").

**Gotchas.** A coding-agent credential is resolved at launch and the session/runtime-instance is
frozen to that connection — creating, revoking, or swapping a credential affects only *subsequently
launched* runtimes. A session frozen to a *personal* credential may only be continued by its owner
(fail-closed when the owner is unknown); a workspace service principal resolves only *shared*
(non-personal) credentials, so alert/service flows cannot ride a teammate's personal token. The
launch path itself does not pre-check credential resolvability — an unresolvable credential surfaces
later as the agent failing ("not logged in"), not as a launch error.
