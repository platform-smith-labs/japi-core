---
type: capability
title: "Agent definitions & profiles"
tags: [agent-definitions, agent-profiles, coding-agent, scope-inheritance, file-policy, consumer]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui reads/writes scoped coding-agent config bundles (definitions), the read-only profile registry, and the company→workspace→project inheritance/override chain — endpoints and key fields a backend peer serves."
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/agent-definitions.ts
  - src/api/agent-profiles.ts
  - src/contexts/agent-profiles-context.tsx
  - src/hooks/use-agent-profiles.ts
  - src/components/agent-definitions/OverrideEditor.container.tsx
  - src/components/agent-definitions/FileTreeEditorPage.container.tsx
see_also:
  - {repo: orchestrator, capability: "Agent definitions & file resolution", intent: "owns the definition/file store, the resolved-files merge, and profile registry ps-ui proxies through ps-api", descriptive: true}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "proxies the /v1/agent-definitions and /v1/agent-profiles routes to the orchestrator verbatim", descriptive: true}
---

# Agent definitions & profiles

**What it does.** ps-ui is the authoring UI for **agent definitions** — scoped, versioned config
bundles (a name + a set of files: instructions, settings, skills, commands, MCP servers, hooks, etc.)
that the platform injects into a coding agent (Claude Code / Codex) when a runtime session spawns.
A definition exists at one of three **scopes** — `company` / `workspace` / `project` — and more
specific scopes inherit and can **override** files from broader ones. ps-ui is a **pure consumer**:
it reads and writes definitions through REST; the orchestrator (via the ps-api gateway) owns the
store, the inheritance merge, and the profile registry.

**Backend contracts consumed.** All paths below are relative to the gateway API base (`…/api`); ps-api
proxies them to the orchestrator. (Bearer-JWT auth applies as everywhere — see context.md.)

Definitions — scoped list/create:
- `GET  /v1/company/agent-definitions` · `GET /v1/workspaces/{workspaceId}/agent-definitions` ·
  `GET /v1/projects/{projectId}/agent-definitions` — each takes `?include_archived=true`.
- `POST /v1/company/agent-definitions` · `POST /v1/workspaces/{workspaceId}/agent-definitions` ·
  `POST /v1/projects/{projectId}/agent-definitions`.

Definitions — scope-agnostic (any definition UUID):
- `GET|PUT|DELETE /v1/agent-definitions/{definitionId}`
- `POST /v1/agent-definitions/{definitionId}/reset-to-default` — restores a seeded system definition
  to its frozen baseline.
- Files: `GET|POST /v1/agent-definitions/{definitionId}/files` ·
  `GET|PUT|DELETE /v1/agent-definitions/{definitionId}/files/{fileId}`.
- `GET /v1/agent-definitions/{definitionId}/resolved-files` — the full inheritance chain merged.
- `DELETE /v1/agent-definitions/{definitionId}/override` (drop this scope's whole override) ·
  `DELETE /v1/agent-definitions/{definitionId}/files/{fileId}/override` (drop one file's override).
- `GET /v1/projects/{projectId}/agent-definitions/preview` — one call returning the merged file tree
  with per-scope provenance (replaces fetching each scope and merging client-side).

Profiles (read-only reference registry):
- `GET /v1/agent-profiles` → `{ profiles: [...] }` · `GET /v1/agent-profiles/{type}` (e.g. `claude_code`).

**Definition — key fields** (list/get response, not exhaustive): `agent_definition_uuid`,
`company_uuid`, `project_uuid` (null above project scope), `scope_type`, `coding_agent_type`, `name`,
`description`, `version`, `is_archived`, `is_system_provided`, `is_mandatory`, `default_file_policy`,
and on list endpoints `has_local_override` / `override_count`, plus optional `concepts` / `file_count`.

**Definition file — key fields:** `agent_definition_file_uuid`, `concept`, `file_path`, `content`,
`merge_strategy`, `file_policy`, `ordering`.

**Resolved-file — key fields:** `file_path`, `concept`, `effective_policy`, `merge_strategy`,
`source` (`inherited` | `overridden` | `local_only`), `sources[]` (per-scope contributions ordered
least→most specific), optional `local_override`, and `merged_content` — the final combined content
that is injected at spawn. `definition_chain[]` lists the definitions that participate.

**Observable behavior.**
- **Scope resolution.** For a project, the effective config is company → workspace → project, each
  layer's file merged into the next by that file's `merge_strategy`: `append` (concatenate),
  `replace` (most-specific wins whole), or `deep_merge` (recursive JSON merge, most-specific key wins).
- **Override flow.** A file inherited from a broader scope is displayed read-only until the user
  creates a **local override** at the current scope; saving writes a file row at that scope, and the
  merged result re-resolves. Reverting deletes the override and the file falls back to the inherited
  version. ps-ui projects the merged preview client-side for the editor, but **the backend is
  authoritative at spawn** — the resolved-files `merged_content` is the source of truth.
- **File policy** governs whether a downstream scope may change an inherited file: `open` (freely),
  `append_only` (may only add), `locked` (immutable downstream), or `inherit` (no explicit per-file
  policy → falls back to the nearest directory-row policy, then the definition's `default_file_policy`,
  which itself defaults to `open` and is never `inherit`).
- **Profile registry** is fetched once per session (static, `staleTime: Infinity`) and distributed via
  a React context (`AgentProfilesProvider`). A profile describes a `coding_agent_type`: its
  `instructions_file`, `merge_format`, `concepts[]` (each with a `path_hint`, `path_kind`,
  `default_merge_strategy`, `authoring` mode), and `templates[]` (new-file scaffolds). Concept
  ids are backend-driven (open string) — a new profile concept needs no ps-ui change; human labels
  are client-side presentation only.

**Failure modes.** `reset-to-default` returns 422 if the target is not system-provided, 409 on a
name collision, 404 if not found (errors surfaced from the orchestrator, not ps-ui). `getProfile()`
returns `undefined` for an unknown/legacy `coding_agent_type` and while the registry is loading —
callers must tolerate an absent profile rather than assume presence.

**Gotchas.**
- **Definition vs profile vs override are three distinct things.** A *definition* is editable, scoped,
  versioned config the platform stores. A *profile* is the read-only catalog describing a coding-agent
  type's shape (concepts/templates) — it is reference data, never written by ps-ui. An *override* is a
  more-specific scope supplying its own version of an inherited file.
- **Workspace-scoped definitions have `project_uuid = null`** and apply to *any* project in the
  workspace, so listing only the project endpoint misses them; a "all definitions usable for this
  project" view must merge the project list with the workspace list.
- **`file_policy: 'inherit'` is sent as an omitted/empty value on the wire** — it means "no explicit
  policy," not a literal enum value; resolve the effective policy through the directory-row →
  `default_file_policy` fallback.
- **`merged_content` is what actually runs**, not any single scope's `content`; never assume the
  project-scope file alone is the injected config.

**See also / peers.** orchestrator — *Agent definitions & file resolution* (owns the store, the
resolved-files merge, and the profile registry); ps-api — *Gateway request proxy* (proxies these
routes verbatim). Both `see_also` names are unverified from this repo and marked descriptive.
