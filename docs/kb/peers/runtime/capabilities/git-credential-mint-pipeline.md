---
type: capability
title: "Git credential mint pipeline"
tags: [git, credentials, token-minting, uds, credential-helper]
timestamp: 2026-07-09T10:42:29Z
description: "How in-pod git obtains short-lived tokens: helper binary → UDS → WS mint request, correlated response, nothing persisted"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/cred_server/mod.rs
  - src/credential_helper/main.rs
  - src/credential_helper/uds.rs
  - src/credential_helper/protocol.rs
  - src/main.rs
  - src/core/router/mod.rs
  - Cargo.toml
  - docs/dev/decisions/git-connection-resolved-server-side.md
  - docs/dev/decisions/verify-wire-contracts-across-both-repos.md
see_also:
  - {repo: controller, capability: "Message bridging (runtime <-> orchestrator)", intent: "relays the mint request/response between runtime and orchestrator"}
  - {repo: orchestrator, capability: "Git connection resolution and token minting", intent: "resolves the git connection and mints the actual token server-side"}
---

# Git credential mint pipeline

**What it does.** Gives `git` inside a pod short-lived, least-privilege access tokens on demand,
with no credential ever stored on disk. The crate ships a **second binary**,
`ps-git-credential-helper`, implementing the standard git credential-helper protocol; the runtime
(PID 1) transports each request to the platform and returns the minted token to git.

**How a peer interacts.** Not called by peers directly — `git` invokes the helper. On `get`, the
helper reads git's key=value block from stdin, derives a repository slug from the URL path, and
sends a mint request (`request_id`, `repositories`, `permissions`) as newline-terminated JSON
over a Unix domain socket to the runtime (default `/var/run/platform-smith/cred.sock`, overridable
via `PS_CRED_SOCKET`). `store` and `erase` are deliberate no-ops (drain stdin, exit 0) — tokens are
never persisted. The platform side sees this as the WS command `git_mint_token_request` (runtime →
controller → orchestrator) answered by `git_mint_token_response`.

**Observable behavior.** The runtime binds the UDS at boot in **every** mode (socket mode 0666 —
single-tenant container); bind failure is boot-fatal (process exits). Each helper connection is
handled independently, so concurrent git operations don't queue behind each other. The runtime forwards the
request over the live WS link (runtime metadata — runtime name + instance UUID — is injected into
the envelope) and parks the connection until the correlated `git_mint_token_response` arrives, then
writes the response back. On success the helper prints the standard git credential block —
`protocol=https`, `host=<from git's input, default github.com>`, `username=x-access-token`,
`password=<token>` — and exits 0. On any error it prints the error to stderr and exits 1, which
git surfaces as an auth failure.

**Contract.** UDS request key fields: `request_id` (UUID, correlation key), `repositories`
(slugs derived from the URL path; may be empty), `permissions`. Permission scope is chosen by
runtime role: builder mode (`PS_RUNTIME_MODE=builder`) requests read-only (`contents:read`,
`metadata:read` — a builder only clones); product mode requests `contents:write`, `metadata:read`,
and `pull_requests:write` (the agent pushes branches and opens PRs). UDS response key fields:
`request_id`, `token?`, `expires_at?`, `error?`. The request deliberately carries **no**
connection identifier: the orchestrator resolves the target git connection server-side from the
runtime instance UUID in the WS metadata (ADR: git-connection-resolved-server-side). Do not
re-introduce a `connection_uuid` field.

**Invariants.**
- Tokens are never persisted (no `store`) and token values are never logged — responses are
  logged by `request_id` only.
- Correlation is strictly by `request_id`; concurrent mints route independently.
- Least privilege by mode: a builder pod never **requests** write scopes (enforcement of the
  minted token's scope is the orchestrator's).
- This repo only **transports**: authorization, connection resolution, and actual token minting
  are the orchestrator's job; the controller relays.

**Failure modes.** WS link down when the request arrives → immediate
`"controller not connected"` error to git (no queueing, no retry). Socket missing (helper run
outside a Platform Smith pod) → "credential socket not found" error. Response for an unknown or
already-answered `request_id` → warned and dropped. Runtime shutdown mid-mint → helper sees the
socket close without a response and fails the git operation.

**Gotchas.**
- Git is wired to use the helper by git config **baked into the pod image by the controller** —
  nothing in this repo's source sets `credential.helper` (ADRs: git-connection-resolved-server-side,
  verify-wire-contracts-across-both-repos). A pod built outside that path has a working socket but
  git won't call the helper.
- There is no in-runtime retry or token cache: every git network operation triggers a fresh mint.
- `expires_at` is advisory to the caller; the runtime does not track or refresh expiry.
- How the orchestrator authorizes the requested scopes and mints the token: UNKNOWN here —
  owned by the orchestrator.

**See also.** controller — message bridging (relays `git_mint_token_request`/`_response`);
orchestrator — git connection resolution and token minting (the authorization + mint owner).
