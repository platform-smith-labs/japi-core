---
type: decision
title: "Runtime ADR registry — peer-facing consequences"
tags: [adr, mcp, credentials, builder, git, pid1, wire-contracts]
timestamp: 2026-07-09T10:42:29Z
description: "One line per load-bearing runtime architecture decision, phrased as what it means for a peer repo; full text lives in this repo's docs/dev/decisions/"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - docs/dev/decisions/mcp-transport-http-sse.md
  - docs/dev/decisions/codex-mcp-transport.md
  - docs/dev/decisions/generalized-mcp-tool-seam-spawn-seed.md
  - docs/dev/decisions/agent-config-inline-delivery-over-relay.md
  - docs/dev/decisions/codex-credential-provenance-marker.md
  - docs/dev/decisions/builder-pod-registration-not-forwarded.md
  - docs/dev/decisions/brownfield-runtime-modes.md
  - docs/dev/decisions/pod-md-schema.md
  - docs/dev/decisions/dockerfile-platformsmith-naming.md
  - docs/dev/decisions/git-connection-resolved-server-side.md
  - docs/dev/decisions/reaper-only-reaps-tracked-pids.md
  - docs/dev/decisions/verify-wire-contracts-across-both-repos.md
---

# Runtime ADR registry — what each decision means for you, a peer repo

Each entry points at the authoritative ADR under this repo's `docs/dev/decisions/`. Read the ADR
before designing against or amending the decision; the lines below are routing, not substitutes.

- **`mcp-transport-http-sse.md`** — All runtime-hosted platform tools live on ONE pod-loopback
  HTTP/SSE MCP server; the agent's session identity comes from the `X-PS-Session-ID` header set
  in per-session config (never from tool args, which are agent-spoofable), and every tool call
  forwarded to the orchestrator carries `originating_session_id`. New tool families must join
  this server, never spawn another.
- **`codex-mcp-transport.md`** — Codex reaches the same single MCP server via its own config file
  with the session header sourced from the `PS_SESSION_ID` env var (fresh each spawned turn);
  MCP wiring failure is fail-open — the session still spawns, just tool-blind.
- **`generalized-mcp-tool-seam-spawn-seed.md`** — The per-session tool list is SEEDED in the spawn
  payload (the runtime never fetches grants over HTTP), `tools/call` forwards any tool name
  generically (new platform tools need zero runtime code), and the runtime never enforces
  grants — the orchestrator is the sole gate on `call`; unseeded sessions advertise an empty
  tool set.
- **`agent-config-inline-delivery-over-relay.md`** — Agent-config and secret files are delivered
  INLINE in the spawn command for both Claude and Codex and materialized before exec; there is
  no separate materialise command anywhere in the pipeline, and a materialization failure fails
  the spawn loudly rather than starting a misconfigured agent.
- **`codex-credential-provenance-marker.md`** — Codex `auth.json` rewrites are keyed on the
  resolved integration-connection UUID via a non-secret sidecar marker: same connection →
  preserve the on-disk (possibly refreshed) bundle; different or ambiguous provenance → APPLY.
  A subscription reassignment is never silently dropped; credential contents are never logged.
- **`builder-pod-registration-not-forwarded.md`** — A builder pod's `registration` is consumed
  at the controller, never forwarded upstream (it now echoes the injected instance UUID for
  correlation, defensive-empty only if unset); the orchestrator only ever sees the ordered
  `launch_*` event family from a builder. Do not expect a builder registration or
  `runtime_ready` to reach the orchestrator.
- **`brownfield-runtime-modes.md`** — Runtime behavior is selected by the `PS_RUNTIME_MODE` env
  var at spawn (the spawner controls it); unknown values WARN and default to greenfield — the
  runtime never crashes on a mode it doesn't know. The ADR's reader modes are retired (modes
  collapsed to greenfield/builder); any legacy reader-mode string now degrades to greenfield —
  treat the ADR's reader sections as historical.
- **`pod-md-schema.md`** — The pod identity file `POD.md` (under the pod's runtime state dir)
  carries ONLY human-readable slugs — UUIDs are banned from it by invariant — and is atomically
  overwritten on each successful clone. Do not add fields or expect UUIDs there.
- **`dockerfile-platformsmith-naming.md`** — The Platform-Smith-authored Dockerfile is named
  `.platform-smith/Dockerfile.platformsmith` in EVERY repo that authors, stores, transfers, or
  builds it; the customer's own root `Dockerfile` is never renamed. A bare
  `.platform-smith/Dockerfile` reference is a build-breaking regression.
- **`git-connection-resolved-server-side.md`** — Git-credential mint requests carry NO
  `connection_uuid`; the orchestrator derives the git connection server-side from the trusted
  runtime-instance identity. Never re-introduce a client-supplied connection field — it would
  be ignored (or a tampering surface).
- **`reaper-only-reaps-tracked-pids.md`** — The PID-1 zombie reaper collects only the runtime's
  own tracked children, never `waitpid(-1)`; a blanket reaper steals async-runtime children's
  exit statuses (breaking in-pod builds and session exit collection). Untracked orphan
  grandchildren may linger as zombies — accepted for ephemeral pods.
- **`verify-wire-contracts-across-both-repos.md`** — Process rule that binds peers too: any claim
  that a runtime↔peer wire contract is broken or needs a change must be verified against BOTH
  sides' code (and the relaying controller, which drops unknown metadata fields) before it goes
  into research, relays, or plans.
