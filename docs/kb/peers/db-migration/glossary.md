---
type: glossary
title: "Domain glossary for the platform_smith schema"
tags: [db-migration, glossary, domain-vocabulary]
timestamp: 2026-07-09T10:37:36Z
description: "One-line definitions of the domain entities a peer needs to read the schema reference"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - migrations/0002_foundation.sql
  - migrations/0003_controllers.sql
  - migrations/0005_project.sql
  - migrations/0007_environment.sql
  - migrations/0009_runtime.sql
  - migrations/0010_launch_recipe.sql
  - migrations/0011_runtime_instance.sql
  - migrations/0013_task.sql
  - migrations/0014_session.sql
  - migrations/0015_audit_event.sql
  - migrations/0025_agent_definitions_and_secrets.sql
  - migrations/0029_workflow_definitions_and_secret_refs.sql
  - migrations/0030_session_task_correlation.sql
  - migrations/0037_mcp_tool_seam.sql
  - migrations/0038_artifact_plane.sql
  - migrations/0038_integration_provider.sql
  - migrations/0039_integration_connection.sql
  - migrations/0040_conversation.sql
  - migrations/0050_slack_channel_tables.sql
  - migrations/0051_users_service_principal.sql
  - migrations/0056_signal.sql
  - migrations/0056_b2_primary_session_routing.sql
  - migrations/0057_schedule.sql
  - migrations/0058_webhook_trigger.sql
---

# Glossary

- **company** — the tenant root; every tenant-scoped row anchors to a company.
- **users** — people (and service principals) within a company; email is globally unique.
- **service principal** — a non-human `users` row (`is_service_principal`), workspace-scoped,
  at most one per (company, workspace); acts as the actor identity for automated work.
- **workspace** — named grouping of resources within a tenant company; archivable, has an owner user.
- **workspace token** — hashed token letting apps/controllers self-register against a workspace;
  multiple active tokens per workspace; also serves as a controller's identity.
- **project** — workspace-scoped unit of work; either a base-image sandbox or a git-repo project
  (`source_type` picks the launch path; promotion is a flip of that field).
- **environment** — deployment environment, bound 1:1 to a stable controller; a runtime's
  environment is derived through its controller, never stored on the runtime.
- **controller** — logical controller definition (one per controller name); the process that
  manages containers for an environment.
- **controller instance** — one incarnation of a controller process; tracks reconnects vs restarts.
- **runtime** — a launched instance of a project on one controller; carries a derived
  `launch_status` projection (the launch lifecycle HEAD lives on the runtime instance).
- **runtime instance** — each container incarnation of a runtime (one active at a time); carries
  per-incarnation launch status and the attempt it was built from. Readiness is read here, not on
  the parent runtime.
- **launch attempt** — one author/build try for a project; `succeeded` means the launch reached
  READY, which anchors recipe-cache reuse. Its file-set is the content-addressed recipe.
- **launch event** — append-only launch timeline (and SSE source); its BIGSERIAL id is the
  authoritative cursor; `runtime.status` is a projection derived from it.
- **task** — internal dispatch-queue row (with `task_response`) for work sent to controllers.
- **session** — a coding-agent (Claude/Codex) or shell session running inside a runtime instance.
- **session-task correlation** — durable map from an orchestrator session *name* (opaque token,
  not an FK) to a Conductor (workflow_id, task_ref_name); backs park/resume with exactly-once
  terminal marking.
- **conversation** — coordination unit for cross-pod agent-to-agent (A2A) messaging, with
  participants (addressed by project) and a persist-first message log.
- **channel binding** — authorization of an external chat channel (Slack) to a target session,
  scoped to a workspace + pinned project; thread bindings map individual Slack threads to sessions
  for in-thread streaming.
- **integration provider** — deployment-wide catalog entry for a third-party provider (coding
  agents now; cloud/SaaS later) with per-provider auth types; non-secret catalog data only.
- **integration connection** — company-scoped credential instance for a provider (encrypted
  credential material); assignable to workspaces.
- **agent definition** — named configuration template for a coding agent, with files materialized
  into the agent workdir at session start and name-based secret references.
- **secret store** — generic secret store with hierarchical scoping (company/workspace/project),
  envelope encryption, and a placeholder/contract pattern for deferred provisioning.
- **workflow definition** — scoped canonical record of a workflow (runnable Conductor definition +
  platform annotations); Conductor itself stays tenant-blind.
- **workflow run context** — per-execution sidecar (one row per Conductor workflow instance) recording
  the workspace/project/execution_context a run targeted plus its owning definition, for the run inbox.
- **signal** — durable tenant-scoped correlation token a workflow parks on (keyed by high-entropy
  `correlation_id`) and is unparked from one of four sources; exactly-once terminal transition.
- **schedule** — tenant-scoped cron trigger registry row; a DB-CAS claim fires the named workflow
  exactly-once across pods.
- **webhook trigger** — server-minted bearer credential (HMAC-hashed token) that starts a run of a
  published workflow definition over HTTP; public URL carries the non-secret `webhook_trigger_uuid`.
- **primary session (B2 routing)** — the elected session for a (conversation, project), referenced by
  durable **logical name** (not a physical `session_id`) so it survives pod incarnations; a `session_role`
  of `secondary` marks a read-only judge/aux session.
- **artifact** — one logical named artifact per scope (session/project/workspace) with immutable
  versions over a content-addressed blob store.
- **MCP tool grant** — per-session, tenant-scoped grant of a tool from the global platform MCP
  tool registry; created before spawn, retained for audit (soft revoke).
- **audit event** — immutable audit log of workspace-scoped actions; rows never updated or deleted.
- **script_log** — the migration-tracking system table (which migration files have been applied);
  owned by the runner, not application data.
