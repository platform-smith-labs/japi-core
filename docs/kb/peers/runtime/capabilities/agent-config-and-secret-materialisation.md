---
type: capability
title: "Agent-config and secret file materialisation"
tags: [agent-config, secrets, spawn, file-delivery, path-confinement, symlink-guard]
timestamp: 2026-07-09T10:42:29Z
description: "Spawn-inline delivery of agent_files and secret_files into the pod before the coding-agent process execs — all-or-nothing, path-confined, symlink-guarded"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/router/handlers.rs
  - src/util/fs.rs
  - docs/dev/decisions/agent-config-inline-delivery-over-relay.md
see_also:
  - {repo: runtime, capability: "Coding-agent sessions (Claude + Codex)", descriptive: false, intent: "the spawn this delivery is inlined into; a materialise failure surfaces as that session's claude_session_failed"}
  - {repo: orchestrator, capability: "Interactive session management", intent: "the peer that composes the spawn payload carrying agent_files/secret_files"}
---

# Agent-config and secret file materialisation

**What it does.** Writes platform-injected agent configuration (Claude/Codex settings, CLAUDE.md,
MCP config, hooks) and secret files into the pod's filesystem so the coding agent starts already
configured. Delivery is **inline on session spawn** — by design there is no separate
"deliver files" command or relay (decision: agent-config-inline-delivery-over-relay); this is
race-free because the CLI only reads its config when it execs, which happens strictly after the
writes.

**How a peer interacts.** Attach two optional arrays to the `spawn_claude_session` command payload:
- `agent_files` — key fields: `file_path`, `content`. Written mode `0640`.
- `secret_files` — key fields: `path`, `content`, `mode` (octal string, e.g. `"0600"`).

Both empty (or absent, from an older orchestrator) ⇒ the spawn behaves exactly as before. The same
mechanism serves Claude and Codex spawns; for spawn-per-turn Codex, files materialised on the first
spawn persist across turns.

**Observable behavior.** Files are written **before** the agent process execs, so the agent's very
first read sees them. On success there is no separate acknowledgement — the normal
`claude_session_started` implies the files were delivered. On **any** failure the whole spawn fails:
the runtime emits `claude_session_failed{session_id, error}` and no agent process starts (the
deliberate opposite of the runtime's own non-fatal MCP-config write).

**Contract (path confinement).** Each file targets one of three confinement roots, enforced by a
positive allowlist:
- workspace-relative `agent_files` (joined under the session `working_dir`, default `/workspace`):
  only `CLAUDE.md`, `.mcp.json`, or anything under `.claude/` (nesting allowed);
- `~/`-prefixed `agent_files`: only under the home `.claude/`;
- `secret_files` (root-relative): only under `run/secrets/` (`run/secrets-evil/x` is rejected).

Rejected regardless of root: empty path, absolute path, NUL byte, any `..` component. A malformed
`mode` on a secret silently falls back to `0600` (restrictive default), not an error.

**Invariants.**
- **Whole-batch, all-or-nothing ACK semantics:** the first invalid path or write failure aborts the
  batch and fails the spawn — a peer must treat any failure as "nothing durably delivered" and never
  assume partial delivery. (Files written before the failing one may exist on disk, but the contract
  gives no per-file accounting.)
- Writes are atomic (tmp-file + rename, with open flags that refuse to follow a symlink at the tmp
  path), and every ancestor directory of a target is checked to not be a symlink — a
  customer-committed symlink directory in the clone cannot redirect a platform write.
- Files are owned by the runtime user by construction (the runtime writes them; no chown), so the
  agent process — same user — can read and rewrite them.
- Config wins on collision: an injected file overwrites a same-path file already in the workspace.

**Failure modes.** Path outside the allowlist / traversal attempt / symlink ancestor / filesystem
write error → `claude_session_failed` with a sanitised, customer-relative error message (no absolute
pod paths, no errno text, never secret content). The peer's recovery is to fix the payload and
re-spawn.

**Gotchas.**
- There is no standalone file-delivery command: to update agent config or secrets, spawn (or
  re-spawn) a session carrying the new files.
- Secret content is never logged or echoed; errors reference the secret's *path* only.
- Home-root delivery covers `~/.claude/**` only — this channel cannot write arbitrary dotfiles
  (e.g. `~/.ssh`); workspace-root cannot write arbitrary workspace files either (contrast the
  separate `.platform-smith/`-confined batch delivery capability).
