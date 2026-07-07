---
type: glossary
title: "Glossary"
tags: [glossary]
timestamp: 2026-07-07T06:27:35Z
description: "Domain terms a peer needs to read ps-ui's KB"
repo: ps-ui
commit_sha: 1f5f197
evidence: [src/api/sessions.ts, src/api/launch.ts, src/api/readiness.ts, src/stores/active-workspace-store.ts]
---

# Glossary

- **ps-api gateway** — the backend ps-ui talks to (`VITE_API_URL`, default `:9004/api`); authenticates
  and proxies to the orchestrator. ps-ui contacts nothing else.
- **Workspace** — the tenant-scoped container a user operates in; selected workspace is persisted in
  the browser. Most resources are keyed by `workspace_uuid`.
- **Project** — a repo-backed unit within a workspace.
- **Runtime / sandbox** — an isolated container a user launches to run a coding agent. A "sandbox" is a
  runtime with `kind=sandbox` (no git). Readiness is observed via a status field, not assumed on launch.
- **Runtime instance** — the concrete launched instance of a runtime; platform-wide, readiness is
  tracked on the **instance**, not the parent runtime record.
- **Coding session** — an interactive Claude/Codex chat bound to a runtime. Addressed by **name**
  (`sessionName`), not UUID, in ps-ui routes.
- **ACP (Agent Client Protocol)** — the event vocabulary ps-ui applies to render streaming agent
  output inside a session (delivered over SSE; see the coding-sessions capability).
- **Integration credential** — a coding-agent auth credential (Claude OAuth setup token or API key,
  Codex) resolved at runtime launch and frozen into the runtime instance.
- **Agent definition / profile** — a reusable coding-agent configuration and its scoped binding
  (company / workspace / project) with an override hierarchy.
- **Workspace token** — the identity credential a controller registers with the platform under
  (`CONTROLLER_TOKEN`).
- **Secret vs secret-ref** — a secret is the stored value; a secret-ref is a pointer used to inject it
  into a runtime without exposing the value.
- **Readiness chain** — the ordered launch pre-flight (`NEEDS_CODING_AGENT_CREDENTIAL → NEEDS_GIT_CONNECTION
  → NEEDS_ENVIRONMENT → NEEDS_RUNNER`) ps-ui polls before a runtime is launchable, terminal at `READY`.
