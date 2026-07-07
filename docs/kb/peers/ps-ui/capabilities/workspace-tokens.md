---
type: capability
title: "Workspace tokens"
tags: [workspace-tokens, controller, auth, readiness]
timestamp: 2026-07-07T06:27:35Z
description: "How ps-ui mints the workspace token a self-hosted controller uses to register with the orchestrator"
repo: ps-ui
commit_sha: 1f5f197
evidence:
  - src/api/workspace-tokens.ts
  - src/components/readiness/panels/connect-commands.ts
see_also:
  - {repo: ps-ui, capability: "Runtime readiness polling", intent: "the Connect-runner panel mints the token when a workspace needs a controller"}
  - {repo: orchestrator, capability: "Controller registration", intent: "hosts the token mint and validates the token a controller registers with", descriptive: true}
  - {repo: ps-api, capability: "Gateway request proxy", intent: "passthrough-proxies the mint to the orchestrator", descriptive: true}
---

# Workspace tokens

**What it does.** A workspace token is the identity a self-hosted controller registers with — it
becomes the controller's `CONTROLLER_TOKEN`. ps-ui mints one from the Connect-runner panel so a user's
controller can dial back into the orchestrator and bind the workspace's environment.

**Backend contracts consumed** (ps-api :9004):
- `POST /v1/workspace-tokens` → `{ workspace_token_uuid, name, token, created_at }`. Body:
  `{ workspace_uuid, name }`.

`workspace_uuid` is carried in the **body**, not the path. The response keys on
`workspace_token_uuid` (not an integer id). The raw `token` (`pst_…`) is the value emitted into the
generated `docker run` command as `CONTROLLER_TOKEN`.

**Observable behavior.** The raw `token` is returned **once** and is never re-readable — only its hash
is stored server-side. Tokens are long-lived until revoked. The mint is hosted by the orchestrator;
ps-api passthrough-proxies the call.

**Failure modes.** `409` means the `name` is already in use for this `(company, workspace)` pair; the
UI surfaces it as a name-field message and offers "Regenerate" (which simply mints a fresh token).

**Gotchas.**
- Because the raw token is shown only once, ps-ui cannot re-display an existing token — regeneration
  mints a new one rather than recovering the old value.
- There is **no list or revoke endpoint** consumed by ps-ui at P1 — only `create` exists here. "List"
  and "revoke" are UNKNOWN from this repo's surface.

**See also.** ps-ui — Runtime readiness (the panel that triggers the mint); orchestrator — Controller
registration (validates the token when a controller connects).
