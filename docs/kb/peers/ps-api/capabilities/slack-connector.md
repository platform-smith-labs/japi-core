---
type: capability
title: "Slack connector"
tags: [slack, chat-ingress, identity, sessions, alerts, gateway]
timestamp: 2026-07-09T10:35:01Z
description: "The Slack ingress — a second trusted gateway that maps Slack users to platform users and drives coding sessions from Slack (@mention, alerts, /smith)"
repo: ps-api
commit_sha: a4683c0
evidence:
  - cmd/slack/canonical.go
  - cmd/slack/config.go
  - cmd/slack/ingress.go
  - cmd/slack/identity.go
  - cmd/slack/processor.go
  - cmd/slack/routing.go
  - cmd/slack/orchestrate.go
  - cmd/slack/smith.go
  - cmd/slack/socketmode.go
  - cmd/slack/oauth.go
  - cmd/slack/sessionoutput.go
  - cmd/handlers/slack_admin.go
  - cmd/handlers/slack_links.go
  - cmd/handlers/slack_bindings.go
  - cmd/db/channel_operations.go
  - cmd/server/main.go
  - docs/dev/decisions/slack-connector-interim-rbac.md
  - docs/dev/decisions/slack-mention-binding-default-dm-dropped.md
see_also:
  - {repo: ps-api, capability: "Auth and identity gateway", intent: "the primary trusted gateway; the connector asserts the same trusted identity headers toward orchestrator"}
  - {repo: orchestrator, capability: "Gateway-trusted identity headers", intent: "trusts the (company, user) identity the connector asserts and enforces that user's real RBAC", descriptive: true}
  - {repo: orchestrator, capability: "Session launch and input", intent: "the /launch and session-input APIs the connector drives; session_event rows it polls for output", descriptive: true}
---

# Slack connector

**What it does.** Makes Slack a client of the platform: a "second trusted gateway" that maps a Slack
workspace to a company and a Slack user to a platform user, then drives coding sessions from Slack —
@mention launches a session in a bound channel, alert messages in bound channels spawn investigation
sessions, and the `/smith` command opens a launcher modal. It is a thin client with zero domain logic:
every backend action is an existing orchestrator API call made with the resolved identity asserted via
the same trusted headers the primary gateway uses.

**How a peer interacts.** Two planes:
- *Slack ingress* — gated on `SLACK_TRANSPORT`: unset ⇒ connector fully off (no routes, no scan);
  `socket_mode` (dev) ⇒ managed outbound WebSocket to Slack per installed workspace (app-level token,
  no public URL); `https` (prod) ⇒ HMAC-verified webhooks `POST /webhooks/slack/{events,interactivity,commands}`.
  Both normalize into one canonical event, so behavior is transport-identical. Config env: `SLACK_TRANSPORT`,
  `SLACK_OAUTH_CLIENT_ID`/`SLACK_OAUTH_CLIENT_SECRET`, `SLACK_OAUTH_REDIRECT_URI`, `SLACK_SIGNING_SECRET`,
  `SLACK_APP_TOKEN`, `SLACK_UI_REDIRECT_BASE` (key vars; per-install secrets are never env-configured).
- *Admin plane* (JWT-authenticated, for the frontend) under `/api/v1/integrations/slack/`: OAuth install
  start (`oauth/start`), GET/DELETE `installation`, user links (GET/PUT/DELETE `links`), channel
  bindings CRUD (`bindings`, plus GET `channels` for the picker). `GET /api/v1/users` serves the link
  map-picker. Exception: `oauth/callback` is NOT JWT-authenticated — Slack redirects the browser to
  it; trust is a signed single-use CSRF state minted by `oauth/start`.

**Observable behavior.** Every inbound event flows: resolve install (team→company) → durable dedup claim →
fail-closed identity resolution → act. @mention in a bound channel launches a session at that channel's
binding default target as the *mapped user* and streams the reply into the thread (ordered messages: narration
blocks + per-tool status messages, with an "Open session" deep link); a thread reply routes to the bound
session as a new turn. @mention in an unbound channel and any DM get an actionable notice — DMs never spawn
(decision: `slack-mention-binding-default-dm-dropped`). A channel message in a bound alert channel is evaluated
by first-match routing rules (route/drop) merged over the binding default and spawns an investigation as the
*workspace service principal*; repeats of the same normalized alert within the dedup window (default 5m) only
bump an occurrence count. `/smith` opens a modal whose project/environment/agent-def options are enumerated
server-side, company-scoped; submission launches as the mapped user. Webhooks ACK 200 within ~3s and process
asynchronously with bounded concurrency.

**Contract.** Ingress auth is Slack's: timestamp replay window + constant-time HMAC over the raw body using
the per-install signing secret; any failure is a generic 401 with nothing enqueued. Admin endpoints use the
gateway's normal JWT auth (except the OAuth callback — single-use-state trust, see above). Rejected humans
(unlinked/unverified) are DM'd the reason.

**Invariants.** Asserted identity derives *only* from server-side bindings — never from event fields — and
fails closed: unknown workspace, unlinked user, or a `proposed` (unconfirmed) link yields no identity, never a
default. Only `link_status = verified` authorizes acting; email auto-match creates `proposed` links and only
the target user can self-confirm (interim RBAC: any member may manage bindings — decision:
`slack-connector-interim-rbac`). RBAC itself is enforced by orchestrator on the mapped user. The OAuth install
binds a workspace to the *initiating admin's* company via single-use CSRF state, never a callback parameter.
The target environment always comes from bound config (binding/rule/modal picker), never parsed from message
text. Socket Mode must run single-replica; event dedup is the backstop.

**Failure modes.** Signature failure → 401, no action. Unknown workspace event → silently dropped. Unlinked
user → DM'd reject, no action. Event flood → bounded dispatch sheds load rather than growing unbounded.
Alert whose service principal has no shared coding-agent credential → pre-flight notice in channel, no spawn.
A session running on another user's personal credential cannot be continued from Slack (owner-gate notice).

**Gotchas.** Unset `SLACK_TRANSPORT` means the webhook and callback routes are not registered at all (404).
Per-install secrets (bot token, signing secret, app token) live encrypted in `integration_connection` and are
resolved per install — rotating one takes effect via cache invalidation, not restart. Slash commands arrive on
their own webhook (`commands`), separate from interactivity. ACK ≠ processed: Slack sees 200 before any action
runs. Session output is read by polling (~2s cadence, 15m cap per turn), so Slack replies trail the session.

**Business-critical data.** `channel_installation` (workspace→company + bot-token connection — the trust
anchor), `channel_user_link` (Slack user→platform user; `link_status` gates acting), `channel_conversation_binding`
+ `channel_routing_rule` (per-channel default target and alert routing), `channel_thread_binding` (Slack
thread→session continuity), `channel_event_dedup` (idempotency by Slack event id), `channel_alert_fingerprint`
(alert storm suppression). All are provider-discriminated for future non-Slack adapters.
