---
type: capability
title: "Coding-agent credential dispatch"
tags: [credentials, auth-type, claude, codex, bedrock, vertex, fire-and-forget]
timestamp: 2026-07-06T23:40:38Z
description: "auth_type-dispatched provisioning of coding-agent credentials: file-writers (Claude creds file, Codex auth.json with provenance) and env-carriers staged for the next spawn — fire-and-forget, frozen per instance"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/session/credential_dispatch.rs
  - src/session/claude/credentials.rs
  - src/session/codex/credentials.rs
  - src/runtime_state.rs
  - src/core/router/handlers.rs
  - docs/dev/decisions/codex-credential-provenance-marker.md
see_also:
  - {repo: runtime, capability: "coding-agent-sessions", intent: "the sessions these credentials authenticate; a bad/missing credential surfaces only as that agent's auth failure"}
  - {repo: orchestrator, capability: "Runtime container lifecycle", intent: "credentials are frozen per runtime instance — a credential change requires launching a fresh instance"}
---

# Coding-agent credential dispatch

**What it does.** Installs the workspace's coding-agent credential (resolved and decrypted upstream,
blind-forwarded by the controller) into the pod so the Claude or Codex CLI can authenticate. The
runtime owns the routing: it branches on the credential's **`auth_type`** — never on its category —
and writes a credential file or stages environment variables accordingly.

**How a peer interacts.** Send the `setup_coding_agent_credential` command carrying a decrypted
IntegrationCredential — key fields: `auth_type`, `fields` (secret map), `config` (non-secret
region/project map), `connection_uuid`. Delivered once, post-registration, before the first agent
session spawns. Six auth types are handled:

| auth_type | effect |
|---|---|
| `claude_api_key` (`fields.api_key`) | written into the home Claude settings file (env.ANTHROPIC_API_KEY), mode 0600 |
| `claude_oauth_setup_token` (`fields.oauth_token`) | written to the home Claude credentials file (`~/.claude/.credentials.json`), mode 0600 |
| `codex_chatgpt_subscription` (`fields.auth_json`) | opaque bundle written verbatim to `$CODEX_HOME/auth.json` (default `~/.codex`), mode 0600, with a provenance marker |
| `codex_api_key` | env-carrier: stages `OPENAI_API_KEY` (+ `CODEX_API_KEY`) |
| `claude_bedrock` | env-carrier: stages `CLAUDE_CODE_USE_BEDROCK=1` + AWS key/secret/region |
| `claude_vertex` | secret service-account JSON → a 0600 creds file; stages `CLAUDE_CODE_USE_VERTEX=1`, `GOOGLE_APPLICATION_CREDENTIALS`, project + region |

**Observable behavior.** **Fire-and-forget: there is no reply on success OR failure.** A failed
write (unknown auth_type, missing field, filesystem error) is logged inside the pod and otherwise
silent — the failure surfaces later, as the coding agent failing fast at auth on its next session.
A peer must not wait for, or infer anything from, the absence of a response.

Env-carrier types take effect at the **next** session spawn: the staged vars are merged **under**
the spawn's per-session env, so an explicit per-session env var wins on key collision. Sessions
already running are unaffected.

**Invariants.**
- **Frozen per instance:** the credential is resolved at instance launch and pushed once; changing
  the workspace's credential requires a fresh runtime instance — a re-push to a live instance is not
  part of the normal flow.
- Dispatch is by `auth_type` only; an unknown `auth_type` is a (silent) dispatch failure, not a
  fallback.
- Codex `auth.json` is seed-aware: Codex rewrites it in place on token refresh, so an existing
  refreshed bundle is authoritative. With a `connection_uuid` present, a provenance marker
  distinguishes token refresh (bundle kept) from admin reassignment to a *different* connection
  (bundle rewritten); without one, delivery is seed-only-if-missing.
- Secrets are never logged and never appear in error text (errors name only the missing key +
  auth_type; Claude token logging shows a truncated prefix/suffix only).

**Failure modes.** Missing required `fields`/`config` key, empty Codex bundle, unknown auth_type, or
a write error ⇒ logged warning in the pod, no event upstream; the observable symptom is the agent's
auth error on its next turn. Diagnosis requires pod logs or the agent's own error output.

**Gotchas.**
- Never model this as request/response — success is invisible.
- An env-carrier credential pushed *after* a session spawned does nothing for that session; it
  applies from the next spawn.
- Older, narrower commands still exist: `setup_claude_credentials` (Claude file-writer; unlike the
  generic path it **does** reply with a `claude_credentials_setup` success/failure message) and
  `setup_codex_credentials` (fire-and-forget, legacy seed-only-if-missing — no provenance, so it
  cannot handle reassignment). New integrations should use the generic command.
- A static `auth_token` on `setup_devcontainer` is a separate, optional legacy Claude path;
  readiness is never gated on any coding-agent credential.
