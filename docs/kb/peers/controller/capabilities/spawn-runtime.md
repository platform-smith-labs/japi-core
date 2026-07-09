---
type: capability
title: "Spawn product runtime"
tags: [spawn-runtime, runtime-lifecycle, launch-family, docker, ports, launch-failed]
timestamp: 2026-07-09T11:13:06Z
description: "Create and start a product runtime container from a prebuilt image; success correlates by instance events, failure is delivered as a controller-origin launch_failed"
repo: controller
commit_sha: 4e237d3
evidence:
  - src/orchestrator/executor.rs
  - src/protocol/orchestrator.rs
  - src/orchestrator/websocket_client.rs
  - src/docker/sandbox.rs
---

# Spawn product runtime

**What it does.** Creates and starts one product runtime container from a
prebuilt image the orchestrator has already had built, publishing the requested
ports and injecting the launch's secret env. The controller no longer builds or
wraps product images inline — that moved to the builder-pod pipeline; this
capability only runs the container.

**How a peer interacts.** Send the WebSocket command `spawn_runtime`. It is
**launch-family**: it carries **no `task_id`** and is correlated by
`instance_uuid` (the orchestrator-minted runtime-instance UUID it threads into
the container). Do **not** wait for a `task_response` — the controller
suppresses the synthetic ACK for launch-family commands. Instead:

- **Success** → await the runtime's later `registration` / `runtime_ready`
  events (unchanged).
- **Failure** → the controller **originates a `launch_failed` event**
  (work-2607070349) with `data: {instance_uuid, phase: "starting_runtime",
  error_message}`. The `error_message` is composed from the structured spawn
  error as `"{class}: {raw error} — {hint}"` (e.g. `port_in_use: … — Stopped
  'x' to free port 9002…`), so the failure class and operator hint arrive in
  string form within seconds — a peer must NOT rely solely on its own launch
  timeout anymore (the timeout remains a backstop).

**Observable behavior.** Build/pull + create + start can take 60–120s, so the
controller runs the spawn on a background task and keeps heartbeats/pongs
flowing on the orchestrator connection meanwhile; the connection is not frozen.
On success the runtime comes up and announces itself through its later
`registration` / `runtime_ready` events. A same-name runtime that already exists
from a prior launch is stopped and removed first, so exactly one instance per
`runtime_name` is ever active ("sequential instance, one active").
The former `port_mapping` readback of Docker's actually-assigned host ports was
**retired** (work-2607070349): it rode the suppressed `task_response` and was
also discarded orchestrator-side, so it was never delivered. A peer that needs
the host ports still cannot obtain them from the spawn result.

**Contract.** In: `SpawnRuntimeData` — requires a non-empty `runtime_name`,
`base_image`, and a prebuilt `image_tag` (the in-pod-built image spawned from
directly). Also carries `exposed_ports` (typed port-binding specs, each with a
required protocol and an optional pinned host port), `secret_env_vars` (a flat
`{KEY: VALUE}` map, already decrypted/final — appended to the container env
verbatim, **never logged**), optional `git_clone` config, `instance_uuid`,
`project_uuid` / `environment_uuid`, and an optional `coding_agent_credential`
(forwarded to the runtime verbatim post-registration — the controller never
parses it). Out: **success delivers nothing** (suppressed ACK; observe
registration/ready events); **failure delivers a `launch_failed` event**
correlated by `instance_uuid` carrying the composed `error_message`. The
structured `spawn_error` object itself still does not ride the wire for a
launch-family spawn (only its class + hint, folded into the string).

**Invariants.** One active container per `runtime_name` (relaunch replaces).
Container naming follows the ubiquitous `{prefix}-{runtime_name}` contract (see
context.md). Secret env is appended verbatim with no controller-side expansion.
Launch-family correlation is by `instance_uuid`, never a `task_id`.
`launch_failed` is emitted **only on failure** — the success path never emits a
synthetic ACK or event (epic-0065 preserved).

**Failure modes.**
- Empty `image_tag` or empty `base_image` → immediate `InvalidConfig`
  spawn error, before any Docker call; nothing is stored/started. Delivered as
  `launch_failed` (`invalid_config: …`) when an `instance_uuid` is present.
- Host-port conflict on a pinned port → the controller attempts a **one-shot
  eviction**: if the port is held by a sibling `{prefix}-*` runtime it stops
  that peer, waits up to 5s for the port to free, and retries the spawn once;
  otherwise (infra/unknown/host-process holder, or eviction failed) it refuses
  with a `PortInUse` spawn error carrying an operator hint — delivered as
  `launch_failed` (`port_in_use: … — {hint}`). A conflict on an auto-resolved
  (unpinned) port is refused, not evicted.
- Any other Docker create/start error → classified spawn error, delivered as
  `launch_failed`; the half-created container and registry metadata are cleaned
  up.
- A spawn that fails **without an `instance_uuid`** (no correlation key) cannot
  be reported — the controller logs a warning and the peer falls back to its
  launch timeout. This is the only remaining timeout-dependent failure path.

**Gotchas.**
- The eviction path only ever stops `{prefix}-*` containers; on parallel dev
  stacks the prefix scopes this so sibling stacks never cross-reap. Platform
  infrastructure and host processes are never auto-stopped.
- Coding-agent auth and secret **files** are no longer materialized by the
  controller — the runtime writes them itself at session start; only
  `secret_env_vars` are injected into the container at spawn.
- Codex: `CODEX_API_KEY` is injected into the container env **only when** the
  controller holds no ChatGPT-subscription `auth.json` bundle (the bundle takes
  precedence and is pushed to the runtime post-registration instead). An
  explicit same-named secret in `secret_env_vars` wins over the injected key.
- `launch_failed` from a spawn failure and `launch_failed` forwarded from a
  builder pod ride the same upstream pipe and the same wire shape; the spawn
  one names `phase: "starting_runtime"`.
