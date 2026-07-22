---
type: capability
title: "In-pod image build (builder mode)"
tags: [builder, build-image, docker-build, launch-pipeline, unified-launch]
timestamp: 2026-07-09T10:42:29Z
description: "Builder-mode build_image: in-pod docker build with strictly ordered launch_* events and a single-build-per-pod invariant"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/core/router/handlers.rs
  - src/core/router/mod.rs
  - src/config.rs
  - src/runtime_state.rs
  - src/websocket/client.rs
  - src/constants.rs
  - src/main.rs
  - docs/dev/decisions/builder-pod-registration-not-forwarded.md
  - docs/dev/decisions/dockerfile-platformsmith-naming.md
see_also:
  - {repo: controller, capability: "build-pipeline", intent: "spawns the builder pod, filters its registration, and injects runtime_uuid when forwarding launch_* events upstream"}
---

# In-pod image build (builder mode)

**What it does.** A builder pod (runtime started with `PS_RUNTIME_MODE=builder`) runs one `docker build` inside itself and reports the outcome as launch lifecycle events. This is the unified launch pipeline's image-baking step: the orchestrator ships the build inputs down; the pod assembles a build context, builds the image tag, and terminates its useful life.

**How a peer interacts.** Send the `build_image` command over the runtime's WS link after `launch_builder_ready`. Payload key fields: `image_tag`, `base_image` (the `FROM` image; required — omitting it is a malformed payload), `files[]` ({path, content} pairs forming/overlaying the build context), optional `context_clone_path` (absolute path of an already-cloned repo dir to use as the context root). The command is accepted **only in builder mode**; any other mode drops it with a WARN log — no `error_response`, nothing on the wire.

**Observable behavior.** The handler returns immediately and the build runs off the read loop (the link stays responsive during a multi-minute build). Events are strictly ordered: `launch_build_started` before the build, then exactly one terminal event — `launch_build_complete{image_tag}` on success, or `launch_failed{phase:"building", error_message}` on failure. `launch_failed` may legally arrive with **no** preceding `launch_build_started` — on a context-assembly failure or a malformed payload. Events carry `role` and an echoed `instance_uuid` (omitted if none was injected); they carry **no** `runtime_uuid` — that enrichment is controller-owned on forward.

**Contract.**
- Context assembly, base-image mode (no/empty `context_clone_path`): a fresh runtime-owned temp dir; `files[]` are the whole context; the dir is deleted after the terminal event.
- Context assembly, repo-import mode (`context_clone_path` set): the clone dir IS the context root; `files[]` are overlaid and must be confined to `.platform-smith/`; the clone is never deleted (only runtime-staged bins are cleaned up). Empty `files[]` in this mode fails the build.
- In both modes the runtime stages its own trusted binaries into `.platform-smith/bins/` inside the context (a `files[]` entry targeting that path is rejected), and the canonical dockerfile is `.platform-smith/Dockerfile.platformsmith` (naming decision: `Dockerfile.platformsmith`, not `.ps`).
- The build shells out to `docker build` against the pod's bind-mounted Docker socket (socket provisioning is the controller's concern).
- Errors: unsafe `files[]` paths (absolute, `..`, reserved bins path, non-`.platform-smith/` overlay targets), an invalid/missing/symlinked `context_clone_path`, or a failed `docker build` all surface as `launch_failed` with a sanitized `error_message` (build stderr tail included).

**Invariants.**
- Single build per pod: the first `build_image` claims the pod's only build slot; a second `build_image` is **silently ignored** (WARN log only — prevents a duplicate `launch_build_started` upstream).
- Exactly one terminal event per accepted build; `launch_build_started` never repeats.
- The customer's clone is never removed, on success or failure (repo-import mode).

**Failure modes.**
- Malformed `build_image` payload → terminal `launch_failed{phase:"building"}` immediately, not an `error_response` (so the orchestrator fails fast instead of waiting out a phase timeout).
- `build_image` outside builder mode → dropped with WARN; the sender observes nothing.
- Duplicate `build_image` in the same pod → dropped silently; the ongoing/finished first build's events are all the peer sees.
- Docker daemon unreachable or build error → `launch_failed` with the docker error tail.

**Gotchas.**
- A builder pod is not a normal runtime: it skips the image-CMD supervision entirely (build-only pod), registers with an echoed — possibly empty-string — `instance_uuid` (the key is always present), and its registration is filtered at the controller, never forwarded upstream. Its readiness signal is `launch_builder_ready`, not `launch_ready`/`runtime_ready`.
- Silence is a defined outcome twice over (wrong mode, duplicate build) — a peer that waits for an error reply will wait forever. Correlate by `instance_uuid` on the events, not by `request_id` (readiness events carry none).
- The runtime does not sequence clone→build: a `context_clone_path` that exists but is mid-clone is out of its contract — sequencing is the sender's job.

**See also / peers.** The flow starts and ends in the controller's build pipeline (repo: controller) — it spawns the builder pod with the Docker socket, filters the builder registration, and injects `runtime_uuid` into forwarded `launch_*` events.
