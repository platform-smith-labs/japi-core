---
type: capability
title: "Git open-PR node"
tags: [git, pull-request, github-app, workflow-node, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "The one privileged git Conductor node — opens a GitHub PR via an org-installed App token; all other git ops stay in the coding-agent session"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/git_open_pr.go
  - internal/workers/nodes/git_open_pr_opener.go
  - internal/platform/git_pr.go
  - internal/workers/nodes/common.go
see_also:
  - {repo: orchestrator, capability: "GitHub App installation PR-token mint", intent: "resolves the App installation for a repo and mints the create-only PR token this node calls with", descriptive: true}
---

# Git open-PR node

**What it does.** Opens a GitHub pull request as a first-class workflow step. It is the **one and only**
git operation modelled as a Conductor node: clone, commit, push, and test all happen *inside* the coding-agent
session (they belong in the agent's prompt, not the workflow graph). PR-creation is a node on its own only
because it needs an **org-installed GitHub App** credential that the per-session git token cannot mint — so it
is done under a centrally-minted create-only token rather than by the agent.

**How a peer interacts.** Model a Conductor task of type `git-open-pr` with a `_ps` annotation. It runs
synchronously and terminally (no park / no async completion). Identity (originating user + company) is taken
from `_ps` and forwarded to the token seam.

**Contract.**
- Inputs (`_ps`): `repo` (required — `owner/name`, or an https/ssh GitHub URL the node normalizes),
  `head` (required — the branch the agent already pushed in-session), `title` (required),
  `base` (optional, default `main`), `body` (optional).
- Outputs (task output map): `repo` (echoed) and `created` (bool) are always present. `pr_url` is present
  **only when** `created` is true; `skipped_reason` is present **only when** `created` is false (a non-fatal skip).
- Errors: task FAILED if `repo`, `head`, or `title` is missing, or if the PR-open call itself errors.

**Observable behavior.** On success the node mints an org-App create-only token for the repo (via the platform
seam → orchestrator), calls the GitHub create-PR API, and completes with `created:true` + `pr_url`. If the org
App is not installed on the target repo, the node completes **successfully** with `created:false` +
`skipped_reason:"app_not_installed"` — a skip, not a failure. If a PR for the same head/base already exists,
GitHub returns an existing URL (→ `created:true`, idempotent) or, when it omits the URL, the node completes
with `created:false` + `skipped_reason:"pr_exists"`.

**Invariants.** The token used is always the org-installed App's **create-only** credential, never the
per-session git token. "App must be installed on this repo" is **not enforced here** — it is decided upstream
by the token-mint owner (orchestrator), which signals not-installed; this node forwards that as a skip. A
missing installation is therefore an upstream condition, not a local check.

**Failure modes.** Missing required input or an errored open-call → task FAILED. App-not-installed and
already-exists-without-URL are **completions with a skip reason**, not failures — designed so a fan-out over
N repos still succeeds for the installed ones.

**Gotchas.**
- **This is the ONLY git node.** Do not expect nodes for commit / push / test — those are prompt steps in the
  coding-agent session. Only PR-opening is a node.
- **Branch on `created`, never assume a PR exists.** A completed `git-open-pr` task can mean "PR opened",
  "already existed", or "App not installed — nothing opened." A peer's trailing logic (e.g. a notify step)
  must read `created` and `skipped_reason`; `pr_url` is absent on any skip.
- **Environment-gated.** The node ships behind `GIT_OPEN_PR_LIVE`. When off (or the PR-opener is unwired) it
  returns an honest **NOT_LIVE** terminal state with `{repo, created:false}` and a reason — it never claims a
  PR was opened. A peer integrating against a stack where this is not yet live will see NOT_LIVE, not a PR.

**See also.** orchestrator — *GitHub App installation PR-token mint*: owns the GitHub App credentials and the
resolve-installation + mint-create-only-token surface this node depends on; whether the repo's App is
installed is decided there.
