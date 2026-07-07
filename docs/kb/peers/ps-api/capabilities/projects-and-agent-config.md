---
type: capability
title: "Projects and agent configuration"
tags: [projects, agent-definitions, agent-profiles, port-bindings, platformsmith-attempts, gateway]
timestamp: 2026-07-07T03:33:49Z
description: "Project workbench surface: project CRUD + import-and-launch, per-project port bindings, agent definitions across scopes, agent profiles, and Platform Smith attempt history"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/projects.go
  - cmd/handlers/port_bindings.go
  - cmd/handlers/agent_definitions.go
  - cmd/handlers/agent_profiles.go
  - cmd/handlers/platformsmith.go
  - cmd/db/project_operations.go
see_also:
  - {repo: ps-api, capability: "Auth and identity gateway", intent: "every route here requires its JWT; identity headers it injects are what orchestrator trusts"}
  - {repo: ps-api, capability: "Runtime launch gateway", intent: "import-and-launch chains into launching a runtime for the new project"}
  - {repo: orchestrator, capability: "Agent definition scope merge", intent: "owns scope-precedence, override, and secret-detection logic this gateway proxies verbatim", descriptive: true}
  - {repo: orchestrator, capability: "Port binding allocation", intent: "owns port-conflict resolution and dockerfile defaults behind the port-binding proxy", descriptive: true}
---

# Projects and agent configuration

**What it does.** The project workbench surface: create and manage projects inside a workspace,
configure how their runtimes are exposed (port bindings) and how the coding agent behaves in them
(agent definitions layered at project/workspace/company scope, plus a global agent-profile
registry), and review the history of AI-generated change attempts ("Platform Smith attempts").
Projects CRUD is answered from the gateway's own DB; everything else is a verbatim proxy to
orchestrator (status codes and bodies pass through byte-for-byte).

**How a peer interacts.** All routes require the gateway JWT.
- Projects: list/create under `/api/v1/workspaces/{workspace_uuid}/projects`
  (`?include_archived=true` to include archived); get/patch/archive at
  `/api/v1/projects/{project_uuid}`; composite `POST …/projects/import-and-launch` creates a
  GitHub-linked project and immediately launches a runtime (proxied).
- Port bindings: CRUD + `…/reset` under `/api/v1/projects/{project_uuid}/port-bindings` (proxied).
- Agent definitions: created/listed at one of three scopes — project
  (`/projects/{uuid}/agent-definitions`), workspace (`/workspaces/{uuid}/agent-definitions`), or
  company (`/company/agent-definitions`); once created, item + file operations are scope-agnostic at
  `/api/v1/agent-definitions/{uuid}` (all proxied). Extras: `…/preview` (merged result across scopes
  with provenance), `…/detected-secrets`, `…/resolved-files` (inheritance provenance),
  `…/reset-to-default`, and override reverts (`DELETE …/override`, per-definition or per-file).
- Agent profiles: read-only registry at `/api/v1/agent-profiles[/{type}]`, keyed by
  `coding_agent_type` (e.g. `claude_code`). Global reference data, no tenant scoping.
- Attempts: `GET /api/v1/projects/{uuid}/platformsmith-attempts` (history),
  `GET …/latest/files` (files of the latest successful attempt),
  `POST …/{attempt_uuid}/pr-url` (record a manually raised PR URL; fire-and-forget — nothing
  fetches or polls the URL).

**Observable behavior.** Project responses always carry a `port_bindings` array: the list view
emits `[]` per row (deliberately, to avoid per-row lookups) while the single-project read inlines
the real bindings — a peer needing bindings must fetch the detail route. Delete is a soft archive
(`is_archived=true`); archived projects vanish from default lists and 404 on further mutation.
Patch is partial (only `name`/`description`, omitted fields unchanged). Attempt lists are
newest-first with a per-row `file_count`; an existing project with zero attempts returns an empty
list (200, not 404). Import-and-launch is create-project + spawn-runtime in one call; its success
body is the orchestrator's verbatim response — that is where the launch identifiers
(`instance_uuid` etc.) needed for readiness tracking come from, and its exact shape is owned by
the orchestrator (ask its KB). On partial failure the proxied error body includes `project_uuid`
and a `retry_hint` so the caller can resume.

**Contract.** Project in: `name` (1–60 chars, required) + optional `description` (≤255); out: the
project (key fields: `project_uuid`, `name`, `description`, `is_archived`, `default_workdir`,
`base_image`, `devcontainer_ref`, timestamps, `port_bindings`). Port-binding create: `host_port`
may be omitted for dynamic allocation. Latest-attempt files: `{attempt, files}` where each file is
a path/content/sha256 tuple (for drift detection). Agent-definition and profile body shapes are
owned by orchestrator — the gateway does not validate them.

**Invariants.** Every read/write is company-scoped; tenant mismatch is indistinguishable from
not-found (404 or empty list — no existence leak). Project names are unique per workspace among
live (non-archived) rows only. Recording the same `pr_url` twice is idempotent (204). For
project-scoped proxied routes the gateway resolves the project's workspace from its own DB first,
so an unknown/foreign project fails at the gateway before reaching orchestrator.

**Failure modes.** Create project: 409 duplicate live name, 404 unknown workspace. Port bindings:
409 `port_conflict` with the full holder object + hint forwarded verbatim (a UI diagnostic depends
on that exact body); reset → 422 `no_dockerfile_default` when the container port left the image's
EXPOSE list. Reset-to-default → 422 unless the target is the system-provided default definition.
Unknown `coding_agent_type` → 404. PR-url: 400 invalid/oversized URL (validated by orchestrator).

**Gotchas.** `port_bindings: []` in a project *list* does not mean "no bindings" — only the detail
read is authoritative. Updating a binding's `host_port` flips its `source` from `dockerfile` to
`operator`, taking it off the dockerfile-default track until reset. Archive is irreversible via
this API (no unarchive route). Agent-definition merge/precedence semantics live in orchestrator;
use the `preview`/`resolved-files` reads rather than assuming an order.

**Business-critical data.** The gateway's own `project` table (joined to workspace/company) backs
project CRUD and the project→workspace resolution used before proxying; `is_archived` plus a
partial unique index on live names enforces the naming invariant. Port bindings, agent
definitions/files, and attempt records are owned by orchestrator's store, not this repo.
