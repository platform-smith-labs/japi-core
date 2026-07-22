---
type: capability
title: "Platform-Smith file materialisation (.platform-smith/ batch delivery)"
tags: [file-delivery, platform-smith-dir, path-confinement, relaunch, sanitised-errors]
timestamp: 2026-07-09T10:42:29Z
description: "materialise_platformsmith_files: writes an orchestrator-supplied file batch strictly inside <clone_path>/.platform-smith/, ACKing _complete{files_written} or a sanitised _failed"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/router/handlers.rs
  - src/core/protocol/payload.rs
  - docs/dev/decisions/dockerfile-platformsmith-naming.md
see_also:
  - {repo: runtime, capability: "Agent-config and secret file materialisation", descriptive: false, intent: "the separate spawn-inline seam that delivers agent_files/secret_files at session spawn — a different channel with different confinement roots"}
  - {repo: runtime, capability: "Git clone with checkout-or-create branch resolution", descriptive: false, intent: "produces the clone this capability writes into; the orchestrator triggers materialisation from that flow's completion event"}
  - {repo: runtime, capability: "In-pod image build (builder mode)", descriptive: false, intent: "the build the orchestrator dispatches AFTER the materialisation ACK — materialisation itself never starts a build"}
---

# Platform-Smith file materialisation

**What it does.** Writes an orchestrator-supplied batch of files (typically a prior attempt's
authored artifacts, replayed on relaunch of an already-bootstrapped project) into the customer
clone — confined strictly to the `<clone_path>/.platform-smith/` subtree.

**How a peer interacts.** Send the `materialise_platformsmith_files` command with
`{clone_path, files[]}` — `clone_path` is the **absolute** in-pod path of the customer clone;
each file is `{path, content}` where `path` is relative to `clone_path` and **must** start with
`.platform-smith/` (e.g. `.platform-smith/Dockerfile.platformsmith` — the canonical Platform-Smith
Dockerfile name; there is no bare-`Dockerfile` PS path).

**Observable behavior.** The batch is handled synchronously and answered with exactly one ACK:
`materialise_platformsmith_files_complete{files_written}` on full success, or
`materialise_platformsmith_files_failed{error_message, partial_files_written}` on the **first**
failing file (processing stops there — no rollback of files already written). Each file is written
atomically (temp-then-rename, mode 0640, parent dirs created as needed), so a re-sent batch simply
overwrites in place. An empty batch succeeds with `files_written: 0`.

**Contract.** Rejected before any write (whole batch untouched):
- non-absolute `clone_path`;
- a pre-existing `.platform-smith/` that is a symlink, not a directory, or contains **any** symlink
  anywhere in its tree (recursive walk — git preserves customer-committed symlinks on clone, and a
  write through one could land outside the repo).
Each file's path is validated just before its own write — empty, absolute, `..`/drive-prefix/NUL,
or not starting with `.platform-smith/` fails the batch **at that file** (earlier files are already
on disk; see `partial_files_written`).
A missing `.platform-smith/` is fine — it is created on write.

**Invariants.**
- No write ever lands outside `<clone_path>/.platform-smith/`.
- `error_message` is sanitised by design: relative, customer-recognisable paths only — never the
  absolute clone path, errno text, or raw IO error detail (those stay in runtime logs).
- After the success ACK the orchestrator issues its build command separately; this capability never
  initiates a build.

**Failure modes.** Payload parse failure, confinement rejection, symlink-tree rejection, or a write
error each produce the `_failed` ACK with a generic reason. On symlink rejection the message names
the offending `.platform-smith/…` entry so a human can fix the repo.

**Gotchas.**
- `partial_files_written` is informational only — the consumer must **not** treat a partial tree as
  durable or usable; a half-materialised `.platform-smith/` is worse than none. Retry the whole
  batch.
- This is a distinct seam from the spawn-inline `agent_files`/`secret_files` delivery, which rides
  the session-spawn command with its own (different) confinement roots — do not conflate the two.

**See also / peers.** runtime — *agent-config-and-secret-materialisation* (the spawn-inline file
seam); runtime — *git-clone-checkout-or-create* (produces the target clone); runtime —
*in-pod-image-build* (the follow-on build dispatched after this ACK).
