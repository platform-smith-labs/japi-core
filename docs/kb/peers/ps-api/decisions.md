---
type: decision
title: "Peer-relevant ADR summaries"
tags: [decisions, adr, gateway, proxy, slack, crypto]
timestamp: 2026-07-07T03:33:49Z
description: "One-line summaries of the architectural decisions whose consequences a peer repo observes"
repo: ps-api
commit_sha: f8157e0
evidence:
  - docs/dev/decisions/thin-gateway-default.md
  - docs/dev/decisions/no-raw-harness-envelopes-on-the-wire.md
  - docs/dev/decisions/verify-proxy-contract-against-shipped-handler.md
  - docs/dev/decisions/slack-connector-interim-rbac.md
  - docs/dev/decisions/slack-mention-binding-default-dm-dropped.md
  - docs/dev/decisions/kek-provider-openbao-transit.md
  - docs/dev/decisions/integration-audit-via-slog.md
---

# Peer-relevant ADR summaries

- **thin-gateway-default** — UI-facing read endpoints are either DB-direct (pure
  DB projection, no orchestrator hop) or byte-for-byte reverse-proxied to
  orchestrator; no per-endpoint mirror mapping, so for proxied routes
  orchestrator's wire IS ps-api's wire and upstream field changes flow through
  automatically.
- **no-raw-harness-envelopes-on-the-wire** — both the SSE stream and the REST
  session-events read project stored rows through the same projection onto ACP
  frames; a client never receives raw coding-agent output, and telemetry/noise
  rows are omitted rather than returned raw.
- **verify-proxy-contract-against-shipped-handler** — the authoritative contract
  for any proxied route is the upstream service's shipped handler code, not
  relay/requirements prose; peers changing an upstream handler change ps-api's
  effective contract.
- **slack-connector-interim-rbac** — Slack connector admin endpoints are gated
  only by company membership (no platform roles exist yet); identity-link PUT is
  self-link only, and remaining escalation surfaces await platform RBAC.
- **slack-mention-binding-default-dm-dropped** — a Slack @mention in a bound
  channel launches a session at that channel binding's default target as the
  mapped user; DMs no longer spawn or continue sessions (users are pointed to
  `/smith` or an @mention).
- **kek-provider-openbao-transit** — the chosen real KEK provider is OpenBao
  Transit (KEK never exported); self-hosted deployments will need a reachable
  OpenBao once it is wired in — today only the gated dev-only plaintext provider
  runs (managed-KMS clouds may add sibling providers).
- **integration-audit-via-slog** — integration connection create/revoke/assign
  events are audited as structured log lines only; peers must not expect durable,
  queryable audit rows for these mutations.
