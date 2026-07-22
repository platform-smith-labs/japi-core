---
type: capability
title: "Git clone with checkout-or-create branch resolution"
tags: [git, clone, branch-resolution, pod-manifest, expose-ports, async-completion]
timestamp: 2026-07-09T10:42:29Z
description: "setup_git_clone: N-repo background clone with 3-tier branch checkout-or-create, HEAD-SHA resolution, POD.md emission, Dockerfile EXPOSE scan, and a single async git_clone_complete event"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/router/handlers.rs
  - src/dockerfile.rs
  - src/core/protocol/payload.rs
  - docs/dev/decisions/pod-md-schema.md
see_also:
  - {repo: runtime, capability: "Git credential mint pipeline", descriptive: false, intent: "supplies the GitHub token during clone via the in-pod credential helper — blocking-the-read-loop would deadlock this, hence the background task"}
  - {repo: runtime, capability: "In-pod image build (builder mode)", descriptive: false, intent: "the build the orchestrator may dispatch after seeing platform_smith_dockerfile_present=true"}
  - {repo: runtime, capability: "Platform-Smith file materialisation (.platform-smith/ batch delivery)", descriptive: false, intent: "the relaunch path that re-seeds .platform-smith/ files into the clone this capability produced"}
---

# Git clone with checkout-or-create branch resolution

**What it does.** Clones one or more customer repositories into the pod at orchestrator-chosen paths,
resolving which branch to check out — creating and pushing the branch when it doesn't exist yet —
then emits the pod manifest and reports everything back in a single completion event.

**How a peer interacts.** Send the `setup_git_clone` command with `repos[]`, each
`{url, clone_path, branch?, base_branch?, clone_depth?}` (`clone_path` absolute in-pod; omitted
`branch` = remote default; omitted `clone_depth` = full clone). The command is **accepted
immediately** — the handler returns before any git work runs. Completion arrives asynchronously as
exactly **one** `git_clone_complete` event carrying the same `request_id`; that event is the only
done-signal, there is no polling read.

**Observable behavior.**
- Repos are cloned **serially** in a background task. On the plain no-branch path a single
  **15-minute timeout** covers the clone and a timed-out clone's partial directory is removed; on
  the branch-resolution path the 15-minute timeout applies **per git operation**
  (probe/fetch/checkout/push — total wall time can exceed 15 minutes), and a failure removes only
  the partial `.git` (working tree kept).
- **Idempotent skip:** if `<clone_path>/.git` already exists the network clone is skipped, but HEAD
  SHA and current branch are still resolved and reported as a success.
- **3-tier branch resolution** (when `branch` is set): (1) target exists on the remote → checked
  out; (2) target missing + `base_branch` set → branch is **created** from `base_branch` (which must
  exist, else hard-fail); (3) target missing, no `base_branch` → created from the repo's default
  branch. A created branch is immediately **pushed** to origin; if the push loses a concurrent-create
  race the clone reconciles onto the now-remote branch and reports it as tier-1 (not created).
  Branch names are validated before any remote contact; invalid names fail the repo.
- Builder-mode pods **refuse branch creation** (their token is read-only) with a per-repo failure.
- On all-success: the pod manifest is written to `/var/run/platform-smith/POD.md` (slug-only fields —
  workspace, repo, mode, branch (currently always the literal `default`), clone_path, cloned_at;
  **no UUIDs** by design), and an
  image-staged runbook (if present) is copied into the first repo's `.platform-smith/runbook.md`
  unless the repo already tracks one (repo content wins).
- `EXPOSE` directives are parsed from the first repo's Dockerfile —
  `.platform-smith/Dockerfile.platformsmith` wins over root `Dockerfile`; deduplicated,
  `/tcp`/`/udp` suffixes normalised, ARG-substituted/invalid tokens silently dropped — and emitted
  so the orchestrator can seed port bindings.

**Contract.** `git_clone_complete` — key fields: `success` (true only if **all** repos succeeded),
`repos[]` per-repo `{url, clone_path, success, error?, git_sha?, resolved_branch, created_from_base,
created_from?}`, plus top-level **first-repo mirrors**: `git_sha`, `resolved_branch`,
`created_from_base`, `created_from`, `platform_smith_dockerfile_present`,
`exposed_container_ports[]`, and an `instance_uuid` correlation echo (omitted when empty).
Per-repo `git_sha` is a full 40-char HEAD SHA and never empty on success — a post-clone SHA
resolution failure demotes that repo to `success=false`.

**Invariants.**
- One `git_clone_complete` per command, always sent — even on total failure (side-effect errors like
  POD.md or EXPOSE parsing never suppress it).
- A failed branch materialisation removes its partial `.git` so a retry re-clones cleanly (it will
  not be mistaken for a completed clone by the idempotency check). Exception: a SHA-resolution
  failure after a completed materialisation keeps the repo intact so a retry takes the idempotent
  path.
- Errors from branch resolution, remote probes, and branch materialisation are **secret-redacted**
  (URL-embedded credentials and GitHub token material masked); the plain no-branch clone path
  reports git's stderr as-is (its URLs carry no embedded credentials — tokens arrive via the
  credential helper).

**Failure modes.** Any repo failing → `success=false` with the redacted reason in that repo's
`error`; the top-level mirrors then collapse to empty/false. Timeout, unreachable remote, missing
`base_branch` (tier 2), invalid branch name, and builder-mode create attempts are all per-repo
failures, not silent skips.

**Gotchas.**
- The top-level mirror fields reflect **`repos[0]` only** — on a multi-repo request, per-repo facts
  for the rest live solely in `repos[]`. Consumers must not read the mirrors as aggregates.
- Top-level `git_sha` is the empty string on failure — and by cross-repo contract, an empty SHA on a
  Dockerfile-present path is treated **upstream as a loud build failure** (missing-SHA class), never
  a soft default.
- Acceptance ≠ completion: nothing about the clone is known until `git_clone_complete` arrives;
  clones of large repos can legitimately take minutes.
- A reconciled create-race reports `created_from_base=false` / `created_from=null` even though this
  pod attempted a create — the branch is reported as it exists on the remote.

**See also / peers.** runtime — *git-credential-mint-pipeline* (mints the clone token mid-clone over
the same WebSocket); runtime — *in-pod-image-build* (consumes
`platform_smith_dockerfile_present` + `git_sha` downstream); runtime —
*platformsmith-file-materialisation* (relaunch re-seed into this clone).
