---
type: capability
title: "In-pod image build via builder pod"
tags: [build, builder-pod, image, launch, docker, fire-and-forget]
timestamp: 2026-07-09T11:13:06Z
description: "How the orchestrator drives a custom-image build inside a disposable builder pod and observes its outcome"
repo: controller
commit_sha: 4e237d3
evidence:
  - src/docker/image_builder.rs
  - src/docker/sandbox.rs
  - src/orchestrator/executor.rs
  - src/websocket/server.rs
  - src/orchestrator/websocket_client.rs
---

# In-pod image build via builder pod

**What it does.** Builds a customer/runtime image *inside a disposable "builder pod"* rather than
on the controller host. The controller spawns the builder pod (a container that itself runs
`docker build` against the host Docker daemon), then forwards the build recipe to it. The actual
image `FROM` and Dockerfile are authored by the orchestrator and built in-pod — the controller only
provisions the pod and pipes work + events.

**How a peer interacts.** Two launch-family commands, sent in order:
- `spawn_builder` — provision the builder pod. Payload: `runtime_name` + a **required**
  `instance_uuid` (the correlation key).
- `build_image` — forward the `.platform-smith/` recipe fileset to that pod (targets it by
  `runtime_name`).

**Observable behavior.**
- Both commands are **launch family**: they carry **no `task_id`** and correlate by
  `instance_uuid`. The controller's immediate reply (a synthetic ACK / spawn ack) is **suppressed
  at the source** and never reaches the orchestrator — do not await a correlated response.
- `spawn_builder` returns internally once the pod is created+started; that is **not** build
  readiness.
- The build outcome arrives **asynchronously**, as a stream of forwarded runtime events originating
  from the builder pod:
  `launch_builder_ready` → `launch_build_started` → `launch_build_complete` (success) or
  `launch_failed` (failure). Consume these to drive the build lifecycle.
- The controller is a **thin pipe** for `launch_*` events: it never originates, synthesizes,
  dedups, or reorders them. Ordering is whatever the single ordered builder→controller connection
  delivered. The controller adds `instance_uuid` to event metadata for observability only; the
  orchestrator should correlate off the typed event `data`.

**Contract.**
- `spawn_builder` in: `{runtime_name, instance_uuid}` (instance_uuid required, non-empty).
- `build_image` in: `{runtime_name, <recipe fileset payload>}`, forwarded verbatim.
- Out (both): suppressed ACK — treat as no synchronous result.
- Real outputs: the `launch_*` event family (see above), referenced by name.

**Invariants.**
- The builder pod is the **only** container that gets `/var/run/docker.sock` bind-mounted (so it can
  run `docker build`); product runtimes **never** mount it. The host socket's owning GID is probed
  once at controller startup and attached as a supplementary group so the non-root `psruntime` user
  can use the socket.
- The builder runs in a dedicated `builder` mode (`PS_RUNTIME_MODE=builder`); it also runs the
  authoring Claude session, so it is provisioned Claude credentials like a product runtime.
- Build **failure is signalled only by `launch_failed`** from the pod. If the builder pod *dies*
  without emitting it, that is the orchestrator's own **BUILDING-phase timeout** to detect — the
  controller will **not** synthesize a `launch_failed`.

**Failure modes.**
- Builder image cannot be built/ensured → `spawn_builder` fails with a structured spawn error
  (returned internally; since ACK is suppressed, the peer observes this via absence of `launch_*`
  progress, not a correlated error).
- **Same-name builder collision:** an existing builder pod for the same `runtime_name` is inspected
  by uptime. If **older** than `CONTROLLER_BUILDER_STALE_AFTER_SECS` (default 60s) it is assumed
  leaked, reaped, and replaced. If **younger**, it is assumed to be an active build and the new
  `spawn_builder` is **rejected** (refuses to disrupt a live build).

**Gotchas.**
- Do not block on a request/response for either command — the whole flow is asynchronous via
  `launch_*` events; the immediate reply is intentionally dropped on the wire.
- The **builder image is content-keyed on the runtime binary bytes**
  (`platform-smith-builder:bld-<hash>`): an unchanged binary hits cache and builds once; a changed
  binary (e.g. a controller rebuild shipping a new runtime) auto-forces a rebuild, and the tag never
  collides across parallel stacks. Superseded builder images are best-effort pruned to keep one.
- If two `spawn_builder`s for the same runtime arrive within the stale window, the second is
  rejected, not queued — the orchestrator must not race duplicate builds for one `instance_uuid`.
- `RUNTIME_CONTAINER_PREFIX` scopes builder/pod naming and reaping per stack; sibling stacks on one
  daemon do not cross-reap.
