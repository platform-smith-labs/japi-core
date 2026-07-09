---
type: gotcha
title: "Node identifiers: session_id is a name, runtime_uuid keys the instance"
tags: [nodes, identifiers, session-id, runtime-uuid, correlation-store]
timestamp: 2026-07-09T10:49:10Z
description: "Two node identifier fields carry a value that diverges from what the name implies — session_id is a session name, runtime_uuid resolves to the runtime instance"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/session_start.go
  - internal/workers/nodes/sendprompt.go
  - internal/workers/nodes/reads.go
  - internal/workers/nodes/runtime_start.go
  - internal/workers/nodes/runtime_stop.go
---

# Node identifiers: session_id is a name, runtime_uuid keys the instance

**The trap.** Two `_ps` node fields have names that suggest one identity but carry another. An
author who treats them by their name (a UUID lookup, "the runtime") mis-wires the workflow.

**What is true.**

- **`_ps.session_id` is the session NAME, not a UUID.** Every session node (`session-start`,
  `session-send-prompt`, `session-status`, `session-get-last-message`, `session-stop`,
  `collect-result`) takes `session_id` as a caller-supplied, human/stable **session name** — known
  before the session is created, which is exactly why it can be threaded through the chain and used
  as the one-outstanding-turn guard key. It is *not* the session's `session_uuid` (that is a
  separate value some nodes return in their output).

- **`runtime_uuid` is the stable runtime identity that resolves to the newest runtime
  *instance*.** `runtime-start` emits `runtime_uuid` as the stable identity; `runtime-stop`
  consumes it and the platform seam resolves it to the newest `runtime_instance` for the actual
  mutation. So the value keys a runtime whose *instance* is what gets acted on — the readiness of a
  runtime tracks the instance, not the parent record.

- **The durable correlation store's key column is overloaded.** For a running-turn row it holds the
  session **name**; for a runtime-launch row (`runtime-start`) it holds the **`runtime_uuid`**. The
  same field carries a session identifier in one row kind and a runtime identifier in another — do
  not assume a value read from that store is always a session name.

**What a peer/author must do.** Supply `_ps.session_id` as the agreed session **name** consistently
across every session step (the ordering chain relies on it matching). Pass `runtime_uuid` (the
value `runtime-start` output) to `runtime-stop`, not a runtime name or instance id. Read a node's
returned `session_uuid` / `runtime_name` from its output when you actually need those distinct
values.
