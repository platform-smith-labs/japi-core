---
type: capability
title: "Run-command node"
tags: [workflow-node, conductor, command-exec, deterministic, switch-branch]
timestamp: 2026-07-09T10:49:10Z
description: "Deterministic shell exec in a target runtime; returns exit code + output so a downstream SWITCH can branch on success"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/run_command.go
  - internal/workers/nodes/run_command_runner.go
  - internal/workers/nodes/common.go
see_also:
  - {repo: ps-workflow, capability: "Runtime lifecycle nodes", intent: "emits the runtime_name this node consumes", descriptive: false}
  - {repo: orchestrator, capability: "Command-task exec path", intent: "owns the in-runtime command exec + result read", descriptive: true}
---

# Run-command node

**What it does.** A Conductor workflow node that runs one deterministic shell command inside a target
runtime — the "run the tests / run the build" step. It is *not* the agent node: there is no LLM, no
session, no autonomy. It executes exactly the command given and reports the outcome, so a downstream
Conductor SWITCH can branch on whether it succeeded.

**How a peer interacts.** Author a Conductor task of type `run-command` and supply, under the node's
`_ps` annotation, the command to run and the runtime to run it in. The workflow service picks up the
task, executes, and reports the result back to the engine.

**Observable behavior.** Synchronous and terminal — the node runs the command to completion and
returns in one shot (no parking, no polling). The command runs as the originating user of the
workflow. On success the node completes with the captured outcome; the typical wiring feeds
`success` (or `exit_code`) into a SWITCH node to fork the workflow into pass/fail branches.

**Contract.**
- Inputs (`_ps` annotation, all required): `command` — the shell command string; `runtime_name` —
  the **name** of the target runtime (not its UUID); `user_uuid` — the originating user (required for
  any mutating/exec node routed through the orchestrator).
- Outputs (node result): `exit_code` (int), `success` (bool, true iff `exit_code == 0`), `stdout`
  (string), `stderr` (string).
- Errors: the node FAILS if `command` or `runtime_name` is missing, or if the underlying exec
  errors. The exec runs as the originating user; that user must belong to the company, a check
  enforced by the **orchestrator** command-task path (rejected 401 there) — this node forwards the
  identity, it does not enforce membership itself.

**Invariants.** Deterministic: the same command is run verbatim, no LLM interpretation. The command
executes under the originating user's identity, never a service or elevated principal. Honest
liveness: if the exec path is not wired on a given deployment the node reports NOT_LIVE rather than
fabricating an exit code.

**Failure modes.** Missing `command`/`runtime_name` → node FAILED (authoring error). Exec error
(runtime unreachable, orchestrator rejects the identity, etc.) → node FAILED with the underlying
reason. A command that runs but returns a non-zero exit code is **not** a node failure — the node
COMPLETES successfully with `success=false` and a non-zero `exit_code`; branching on that outcome is
the workflow author's job (a SWITCH), not an engine error.

**Gotchas.**
- `runtime_name` is a runtime **NAME**, not a UUID. When a runtime-lifecycle node earlier in the
  workflow emits a runtime name, feed *that* value in — passing a UUID will not resolve.
- Non-zero exit is a normal completion, not a failure. Do not rely on the engine's failure handling
  to catch a failing test suite; wire a SWITCH on `success`/`exit_code`.
- No git/PR/commit here — this node runs one command. Multi-step agentic work (edit, test, commit)
  belongs in the agent session node, not a chain of run-command nodes.

**See also / peers.** ps-workflow *Runtime lifecycle nodes* produce the `runtime_name` this node
consumes. The orchestrator owns the actual in-runtime command exec and the result read; this node is
a thin, identity-forwarding adapter over that path.
