---
type: capability
title: "Slack integration"
tags: [slack, integrations, oauth, bindings, routing]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui connects a Slack workspace, links identities, and manages alert-channel bindings"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/slack.ts
  - src/routes/_auth/company-settings/_layout/slack.tsx
see_also:
  - {repo: orchestrator, capability: "Alert session launch", intent: "a matched Slack binding launches an agent session downstream", descriptive: true}
  - {repo: ps-ui, capability: "Coding sessions (ACP streaming chat)", intent: "a binding names the project + agent definition a session runs as"}
---

# Slack integration

**What it does.** The company-settings Slack admin plane: connect a Slack workspace via OAuth, view
the install, link Slack posters to Platform Smith users, and define alert-channel **bindings** that
route inbound Slack messages to a project + agent definition. All calls go to ps-api only.

**Backend contracts consumed** (ps-api :9004, under `/v1/integrations/slack`):
- `GET /oauth/start` → `{ authorize_url }`; ps-ui redirects the browser to it (OAuth state is
  server-owned). The callback returns to the app at `/company-settings/slack?install=ok&team=…` or
  `?install=error&reason=…`.
- `GET /installation` → `{ status, team_name, reason }` where `status` ∈ `connected | not_connected |
  error`. This read is the **authoritative** install state (the `install=…` URL param is only a hint).
- `DELETE /installation` — admin teardown.
- `GET /links` → `{ users: [...] }` (ps-ui normalizes to `links`). Row key fields: `external_user_ref`
  (opaque natural key, never rendered), `status` ∈ `proposed | verified | revoked`, `link_method` ∈
  `admin | email | oauth`, `linked_user { user_uuid, name, email }`.
- `PUT /links/{external_user_ref}` — upsert a link; body `{ user_uuid }`.
- `DELETE /links/{external_user_ref}`.
- `GET /channels` → `{ channels: [{ conversation_ref, name }] }` (bot-visible channels; picker source).
- `GET /bindings` → `{ bindings: [...] }`; `GET /bindings/{binding_id}` hydrates the edit form.
- `POST /bindings` → a binding. **May return an empty 201 body** — do not assume the created binding
  is echoed back.
- `PUT /bindings/{binding_id}` — **full** update: replaces every field including the entire ordered
  `routing_rules` array.
- `DELETE /bindings/{binding_id}`.

Binding read carries display names alongside opaque keys (project/environment/agent-definition names +
uuids, `conversation_name`, resolved `who_may_trigger` names). Binding **write** sends bare scalars:
`conversation_ref`, `workspace_uuid`, `project_uuid` (required), `environment_uuid` (nullable ⇒
workspace-default), `agent_definition_uuid`, `identity_mode` (`user | service_principal`),
`allowed_agent_definitions[]`/`allowed_environments[]` (uuid[]), `who_may_trigger[]`
(external_user_ref[]), `on_no_match` (`default_target | drop`), and an ordered `routing_rules[]`.

**Observable behavior.** Routing-rule **priority is server-derived from array order** (ascending =
first-match-wins); the write body carries no `priority`/`rule_id`. A rule's `target` is present iff its
`action === 'route'` (null for `drop`, a terminal deny).

**Failure modes.**
- `PUT /links/{external_user_ref}` is **self-link only** at M0: `body.user_uuid` must be the caller's
  own PS user, else `403`.
- Binding writes are subject to a server-side envelope guard that rejects out-of-set choices with a
  `4xx` (e.g. a rule target env not in `allowed_environments`). Exact error shape is owned by ps-api.

**Gotchas.**
- `installation` (the read) is the source of truth — never trust the OAuth `?install=…` redirect param
  beyond an initial hint; ps-ui strips it from the URL after consumption.
- Opaque keys (`external_user_ref`, `conversation_ref`, `binding_id`, `rule_id`) are path/body params
  only and are never displayed.
- `POST /bindings` returning an empty body means a caller must re-fetch the list rather than read the
  response.

**See also.** orchestrator — Alert session launch (a matched binding drives a downstream agent
session); ps-ui — Coding sessions (the project + agent definition a binding targets).
