---
name: land-work
description: >-
  Given a work item, land ALL of its changes to main across every repo it touches. For the parent
  work item and each of its child (sub-work) repos, it: commits every change to a new work branch,
  pushes it to remote, merges origin/main into the branch (resolving conflicts, stopping for the
  human on anything non-trivial), then merges the branch into main and pushes. Use when the user says
  "land the work item", "commit and merge all repos to main", "ship this work", "merge <work-id> to
  main across repos", or wants a finished multi-repo work item integrated to main in one pass. This is
  a monorepo-root conductor operation (crosses repos by design). NOT for a single uncommitted change
  in one repo — use /commit for that.
argument-hint: "<work-id> [--force] [--pr] [--no-verify]"
---

# Land Work

Integrate a completed work item to `main` across **every repo it spans**. A parent work item's
execution is distributed across child work items in different repos (see `/conduct`); this skill walks
that set and lands each repo's changes to `main` with a uniform, safe git flow.

```bash
/land-work <work-id>            # Plan across all repos, CONFIRM, then land each to main
/land-work <work-id> --force    # Skip the confirmation gate (unattended); still stops on real conflicts
/land-work <work-id> --pr       # Instead of merging to main directly, open a PR per repo (protected-main safe)
/land-work <work-id> --no-verify # Skip the per-repo build/test quality gate before merging to main
```

`--force` (alias `auto`) is for autonomous use — it skips only the *plan confirmation*, NOT the
conflict/quality safeguards (those always stop and surface).

## ⚠️ This is destructive and outward-facing — treat it that way

Pushing to `main` is hard to reverse and visible to the whole team. Therefore:

- **Always PLAN then CONFIRM first** (unless `--force`). Show every repo, its branch, the files that
  will be committed, and the exact main it will push to. Get an explicit "yes".
- **Never blindly resolve merge conflicts** (`-X ours/theirs` is banned as a default). Resolve only
  conflicts you fully understand; for anything non-trivial, STOP and hand the specific conflict to the
  human.
- **Never force-push.** No `git push --force` to `main` or shared branches, ever.
- If `main` is **branch-protected** (direct push rejected), fall back to a PR for that repo and say so
  (same as `--pr`). Do not try to circumvent protection.
- This runs at the **monorepo root** (it legitimately spans repos, like `/conduct`). It is the one
  sanctioned cross-repo git operation — but it still only *commits what each repo already changed*; it
  never edits another repo's source to make it land.

## Monorepo root resolution

Run from the monorepo root, or resolve it: `./docs/work/` + `./repos/` present ⇒ CWD is the root; else
if `../repos/` exists ⇒ root is `..`. Repo aliases resolve via the root `CLAUDE.md` `## Repo Aliases`
table (`solution` → `.`, everything else → `repos/<alias>`).

## Step 1 — Resolve the work item + discover the repo set

1. **Resolve `<work-id>`** by glob (never arithmetic): exact `docs/work/<id>` or
   `repos/*/docs/work/<id>`, else `*<id>*`. Error on zero/multiple.
2. **Build the repo set** = the union of:
   - the work item's **own repo** (its `created` event `repo=`), and
   - **every child (sub-work) repo**: scan all `docs/work/*/work.jsonl` +
     `repos/*/docs/work/*/work.jsonl` for a `created`/`meta_changed` with `parent=<work-id>` (recurse
     for N-level nesting — a child may itself be a parent), and take each one's `repo=`.
   Dedup. Map each alias → dir (`solution` → root `.`; else `repos/<alias>`).
   > This mirrors `scripts/conduct-board.sh` discovery — you can also read the parent manifest's board
   > (between the BOARD anchors) to get the child work-item list, then take their repos.
3. **Filter to repos with something to land**: for each repo dir, keep it only if it has uncommitted
   changes (`git status --porcelain` non-empty) **or** an existing unmerged work branch (see Step 3).
   Skip clean repos and say so.

## Step 2 — Plan and confirm

For each repo in the set, gather and present:
- repo alias + dir, current branch, default branch (see Step 3), the **work branch name**,
- `git status --porcelain` summary (files to commit) and `git diff --stat`,
- the proposed conventional-commit subject.

Print the full plan (all repos) and **ask for confirmation** — unless `--force`. Refuse to continue if
any repo is mid-rebase/mid-merge (`.git/MERGE_HEAD`, `rebase-merge/`) — tell the human to finish it
first.

## Step 3 — Per-repo landing flow (iterate; a failure in one repo stops the run for that repo, reports, and moves on only if the user opts to continue)

For each repo dir, run this exact sequence. **`$DEF`** = the repo's default branch, detected robustly:
```bash
git -C "$DIR" remote set-head origin -a >/dev/null 2>&1 || true
DEF=$(git -C "$DIR" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@')
DEF=${DEF:-main}   # fall back to main, then master if main is absent
git -C "$DIR" show-ref --verify --quiet "refs/remotes/origin/$DEF" || DEF=master
BR="work/<work-id>"   # e.g. work/work-2607090251-a2a-delivery-tracking
```

1. **Branch.** Create the work branch from the current HEAD (which carries the uncommitted changes),
   or reuse it if it already exists:
   ```bash
   git -C "$DIR" rev-parse --verify --quiet "$BR" \
     && git -C "$DIR" switch "$BR" \
     || git -C "$DIR" switch -c "$BR"
   ```
   (If HEAD was on `$DEF` with dirty changes, `switch -c` carries them onto `$BR` — intended.)
2. **Commit all changes.** Stage everything and commit with a Conventional Commit referencing the work
   item. Craft the subject from the actual diff (type `feat`/`fix`/`docs`/`refactor` as fits); do NOT
   invent scope. Include the `Co-Authored-By` trailer this environment requires.
   ```bash
   git -C "$DIR" add -A
   git -C "$DIR" commit -m "<type>(<scope>): <summary> (<work-id>)" -m "<body: what + why>" \
     -m "Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
   ```
   If there is nothing to commit (already committed on `$BR`), skip the commit and continue.
   *(For richer messages you may run `/commit --force` inside `$DIR` instead of hand-crafting — but
   keep it non-interactive so the loop doesn't stall.)*
3. **Push the branch.**
   ```bash
   git -C "$DIR" push -u origin "$BR"
   ```
4. **Integrate `origin/$DEF` into the branch** (so the eventual merge to main is clean/up-to-date):
   ```bash
   git -C "$DIR" fetch origin "$DEF"
   git -C "$DIR" merge --no-edit "origin/$DEF"
   ```
5. **Conflicts (if step 4 stops with a conflict).** Handle deliberately, never blindly:
   - Inspect `git -C "$DIR" diff --name-only --diff-filter=U` and each conflicted hunk.
   - Resolve **only** conflicts whose correct resolution is unambiguous (e.g. both sides added the
     same generated file, or disjoint additions in a list/doc). Edit the file, `git add` it.
   - For **any** conflict involving real logic, competing edits to the same lines, or that you are not
     certain about → **STOP**: `git -C "$DIR" merge --abort`, report the exact files/hunks, and ask the
     human to resolve (or approve a specific resolution). Do not push a guessed merge.
   - After a clean resolution: `git -C "$DIR" commit --no-edit` then re-push `$BR` (step 3's push).
6. **Quality gate (unless `--no-verify`).** Run the repo's fast build/test if one is obvious and cheap
   (`cargo build`+`cargo test` for Rust, `go build ./...`+`go vet` for Go, `pnpm build` for ps-ui).
   If it fails, STOP for that repo and report — do not merge broken code to main. (Skip only when no
   quick check exists; note that.)
7. **Merge the branch into `$DEF` and push.**
   ```bash
   git -C "$DIR" switch "$DEF"
   git -C "$DIR" pull --ff-only origin "$DEF"     # abort/redo integration if this fast-forwards past step 4
   git -C "$DIR" merge --no-ff "$BR" -m "Merge $BR into $DEF (<work-id>)"
   git -C "$DIR" push origin "$DEF"
   ```
   - If `pull --ff-only` shows `$DEF` moved since step 4, re-run steps 4–6 (re-integrate) before
     merging — don't merge a stale branch.
   - If `push origin "$DEF"` is **rejected by branch protection**, do NOT force. Fall back to a PR:
     `gh pr create --base "$DEF" --head "$BR" --title "…" --body "…(<work-id>)"` and record the PR URL
     for that repo. (`--pr` takes this path for every repo up front.)
8. **Leave the branch** on the remote (don't delete) unless the user asks — it's the PR/merge record.

## Step 4 — Report

Summarize per repo: branch, commit sha(s), whether main was pushed or a PR was opened (+URL), any repo
skipped (clean) or **stopped** (conflict/quality gate) with exactly what the human must do. If every
repo landed, say so plainly; if any stopped, the run is **not** complete — list the blockers.

## Notes & guarantees

- **Idempotent-ish**: re-running reuses the existing `work/<id>` branch and only does the pending
  steps. It never creates duplicate commits for an already-committed tree.
- **One writer for work-item state**: this skill commits *files*; it does not fabricate work-log
  events. If landing should also flip the work item to a settled/committed state, do that separately
  via `/work` / `wlog.sh` (the append-only event log), not here.
- **Order**: process repos in dependency order when known (e.g. `db-migration` → `orchestrator` →
  runtime/UI) so a schema/API a peer depends on lands first; otherwise repo order is fine since each
  repo's main is independent.
- **Submodule pointer**: if the monorepo root (`solution`) tracks these repos as submodules and the
  user wants the root's submodule pointers bumped too, treat `solution` as one more repo in the set
  (its "change" is the updated submodule SHAs) — but only if the user asks; default is per-repo only.
