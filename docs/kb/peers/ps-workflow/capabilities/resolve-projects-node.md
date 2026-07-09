---
type: capability
title: "Resolve-projects fan-out node"
tags: [conductor-node, fork-join-dynamic, workspace, projects, fan-out]
timestamp: 2026-07-09T10:49:10Z
description: "Expands a workspace into its project array so a downstream Conductor FORK_JOIN_DYNAMIC can fan an agent across every repo"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/resolve_projects.go
  - internal/workers/nodes/common.go
  - internal/platform/platform.go
see_also:
  - {repo: ps-workflow, capability: "Runtime lifecycle nodes", intent: "the per-project fan-out targets launched for each resolved project", descriptive: false}
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "run an agent session per resolved project inside the fork", descriptive: false}
  - {repo: orchestrator, capability: "Project data", intent: "owns the workspace→project data this node reads", descriptive: true}
  - {repo: db-migration, capability: "Project schema", intent: "owns the project/workspace table shape read here", descriptive: true}
---

# Resolve-projects fan-out node

**What it does.** Expands one workspace into the array of projects (repos) it contains, so a
downstream fan-out step can run the same agent work once per project — the flagship "run an agent
across every repo in a workspace" pattern.

**How a peer interacts.** A workflow definition places a Conductor task of type `resolve-projects`,
annotated with `_ps.workspace_uuid`. Optionally set `_ps.only_with_repo: true` to keep only projects
that have a git-repo binding (the "loop every repo" case). It is a custom PS worker task, not a
Conductor system task.

**Observable behavior.** Synchronous and terminal — it completes in one step with no parking or
polling. It emits `projects` (the fan-out list) and `count`. Order this task **before** the
`FORK_JOIN_DYNAMIC` step that consumes `projects`: the dynamic fork reads the resolved array to
decide how many parallel branches to spawn.

**Contract.** In (via `_ps`): `workspace_uuid` (required), `only_with_repo` (optional bool). Out:
`projects` — an array whose per-item shape is `{project, name, repo, branch}` (project UUID, project
name, git repo URL, default branch); and `count` — the length of that array. An empty workspace
yields `projects: []` and `count: 0`.

**Invariants.** The workspace read is tenant-scoped by the originating company (see context.md);
`only_with_repo` filtering is enforced here, in this node. The node forwards the originating identity
(user + company) to the project read; the read is DB-direct and gated on company scope, so an
unknown or cross-tenant workspace surfaces as an empty result, never another tenant's projects.

**Failure modes.** The task result is FAILED when `_ps.workspace_uuid` is absent, or when the
underlying project listing errors. An empty or unknown workspace is **not** a failure — it returns an
empty array and a zero count, so a downstream dynamic fork simply spawns no branches and the
workflow proceeds.

**Gotchas.** Empty ≠ error: a peer must not treat `count: 0` as a fault; the correct downstream
behavior is "do no work." `only_with_repo` narrows the array — a pure-sandbox project with no repo
binding is dropped when it is set, so `count` can be smaller than the workspace's true project total.

## See also / peers

- ps-workflow — Runtime lifecycle nodes / Agent session nodes: the per-project work the dynamic fork
  fans out to, one branch per entry in `projects`.
- orchestrator / db-migration: own the workspace→project data this node reads.
