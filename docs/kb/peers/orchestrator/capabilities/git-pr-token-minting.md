---
type: capability
title: "Git PR-token minting"
tags: [git, github-app, installation-token, pull-request, rest-api, workflow]
timestamp: 2026-07-09T10:40:45Z
description: "How a peer mints a scoped, short-lived GitHub App installation token for a single repo to open a pull request"
repo: orchestrator
commit_sha: 2fa8172
evidence:
  - cmd/handlers/git_pr_token.go
  - cmd/db/git_mint.go
see_also:
  - {repo: ps-workflow, capability: "Git open-PR node", intent: "the sole caller; mints this token to open a PR, maps installed:false to an app_not_installed skip", descriptive: true}
---

# Git PR-token minting

**What it does.** Mints a scoped, short-lived GitHub App **installation token** for one repository so a
caller can open a pull request. The orchestrator is the single place this happens: it is the only
service holding the GitHub App private key, so token minting is centralized behind the gateway. The
intended caller is a workflow's git-open-PR step, which holds only an `owner`/`repo` (no runtime).

**How a peer interacts.** `POST /api/v1/git/installations/pr-token` with `{owner, repo,
permissions?}`. `permissions` is the scoped installation-token permission set; when omitted it
defaults to a create-only PR scope (`pull_requests:write` + `contents:read`). Company identity comes
from the trusted gateway header — no token or company field in the body.

**Observable behavior.** Synchronous. The orchestrator resolves the company's active GitHub App
installation for `owner`, then mints a fresh installation token scoped to the single `owner/repo` with
the requested permissions. The response carries the token, its expiry, and an `installed` flag. The
token is short-lived (GitHub installation-token lifetime) — mint one per use rather than caching.

**Contract.**
- In: `{owner, repo, permissions?}` — `permissions` is a `{scope: level}` map (e.g.
  `{"pull_requests":"write","contents":"read"}`).
- Out: `{token, installed, expires_at}`.
  - `installed:true` + `token` + `expires_at` — a usable token.
  - `installed:false` + empty `token` — **the app is not installed for this owner in this company**.
    This is the deliberate **non-fatal "skip" signal** (HTTP `200`, not an error): a caller maps it to
    an "app not installed" skip and continues, rather than failing.
- Errors: a genuine resolution/mint failure returns a server error; the "no installation" case is
  explicitly **not** an error and does not leak whether the owner exists elsewhere.

**Invariants.**
- **Company-scoped.** The installation is resolved for the caller's company (trusted gateway header);
  an owner with no active installation *for this company* returns `installed:false`, never another
  tenant's installation.
- **Single-repo scope.** The minted token is scoped to just the requested `owner/repo`, not the whole
  installation.
- **Create-only default.** Absent `permissions`, the token can open a PR and read contents — nothing
  broader.

**Failure modes.**
- No active app installation for the owner in this company → `200 {installed:false}` (skip, not error).
- Missing/invalid gateway headers → rejected (unauthenticated), like every orchestrator route.

**Gotchas.**
- `installed:false` is a success response, not a failure — do not treat the empty token as an error;
  branch on `installed`.
- The token is short-lived and single-repo; re-mint per operation instead of holding it.

**See also / peers.** The caller is a workflow git-open-PR step (in ps-workflow), which mints this
token to open a PR and treats `installed:false` as an `app_not_installed` skip.
</content>
