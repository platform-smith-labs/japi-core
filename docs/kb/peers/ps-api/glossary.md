---
type: glossary
title: "ps-api domain glossary"
tags: [glossary, domain-terms, ps-api]
timestamp: 2026-07-07T03:33:49Z
description: "Non-obvious domain terms a peer needs to reason about ps-api's API surface"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/models/workspace.go
  - cmd/models/project.go
  - cmd/models/environment.go
  - cmd/models/runtime.go
  - cmd/models/launch.go
  - cmd/models/session.go
  - cmd/models/integration.go
  - cmd/models/platformsmith.go
  - cmd/models/workspace_token.go
  - cmd/handlers/agent_profiles.go
  - cmd/handlers/verbatim.go
  - cmd/handlers/passthrough.go
---

# Glossary

- **company** — the tenant. Internal scoping identifier only; never on the wire (see context).
- **workspace** — company-scoped organizational unit; container for projects, tokens, and integration assignments.
- **project** — workspace-scoped unit of work; carries build inputs (base image, devcontainer ref) and declared port bindings.
- **environment** — the seat a controller occupies (topology, e.g. `docker_pods`); 1:1 with a controller via a stable controller identity, with a `last_seen_at` heartbeat. May be pre-connect (no controller bound yet).
- **controller** — the container-manager service instance running user containers. Runtimes and sessions are addressed by `controller_name` + `runtime_name`.
- **runtime** — a named container. Its wire `status` is a derived liveness (`active`|`inactive`) collapsed from connected/ready flags — it does NOT carry launch progress.
- **runtime instance** — one incarnation of a runtime. **Launch lifecycle and readiness live here**, not on the runtime: `launch_status` (`requested`…`ready`|`failed`) plus `failed_phase`. Poll the instance, not the parent runtime.
- **launch** — the latest runtime instance of a runtime, keyed on the wire by `instance_uuid`. Has its own SSE timeline stream.
- **launch attempt** — a runtime-scoped history entry, one per build/author try; `succeeded` is true iff that launch reached READY.
- **session** — a coding-agent (Claude/Codex/shell) session inside a runtime. **Name-keyed on the API** (`/api/v1/sessions/{name}`); created with a client-supplied `session_id` + `runtime_name` (`controller_name` optional — omitted means attach to an already-connected runtime). `coding_agent_type` is frozen at spawn by the orchestrator; NULL means the legacy default (claude_code).
- **agent definition** — project- or workspace-scoped configuration selecting which coding agent a session runs (with attached files / secret refs). ps-api forwards its UUID; the orchestrator resolves and freezes it at spawn.
- **agent profile** — global registry entry keyed by `coding_agent_type` (e.g. `claude_code`). Reference data, no tenant scoping.
- **integration connection** — company-scoped credential instance for a provider (auth type per a provider catalog). Credential material is write-only: reads expose only a computed `has_credential`. May be personal (user-owned, `owned_by_me`) or company-shared.
- **workspace token** — workspace-scoped bearer credential. Raw value is returned exactly once at creation, never recoverable after. (Controllers are minted identity via a distinct controller-token surface; whether that is a workspace token underneath is UNKNOWN from this repo.)
- **platformsmith attempt** — one launch try of a Platform Smith-generated project: git SHA, `launch_succeeded`, optional PR URL, plus an attached generated-file store.
- **raw passthrough** — proxy flavor 1: authenticated reverse-proxy byte stream to the orchestrator; the orchestrator's response IS the wire contract (no gateway model).
- **verbatim relay** — proxy flavor 2: a typed gateway route whose upstream response — including non-2xx error bodies — is forwarded byte-for-byte (preserves upstream error discriminators).
- **typed proxy** — proxy flavor 3: ps-api owns the wire model and maps to/from orchestrator payloads (e.g. the unified task-creation request).
