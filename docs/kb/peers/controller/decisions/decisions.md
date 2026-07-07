---
type: decision
title: "Controller architecture decisions — peer index"
tags: [controller, decisions, adr-index, thin-bridge, isolation, credentials]
timestamp: 2026-07-07T00:00:00Z
description: "One line per controller ADR, stated as the consequence for a peer repo; full ADRs live in docs/dev/decisions/."
repo: controller
commit_sha: 3412b7d
evidence:
  - docs/dev/decisions/controller-thin-bridge.md
  - docs/dev/decisions/relay-pipeline-pattern.md
  - docs/dev/decisions/generic-passthrough-default.md
  - docs/dev/decisions/spawn-error-contract.md
  - docs/dev/decisions/host-container-isolation.md
  - docs/dev/decisions/configurable-runtime-container-prefix.md
  - docs/dev/decisions/tenant-isolated-builds.md
  - docs/dev/decisions/bootstrap-pod-docker-sock-group-add.md
  - docs/dev/decisions/bootstrap-pod-docker-socket-mount.md
  - docs/dev/decisions/codex-api-key-controller-held-container-env.md
  - docs/dev/decisions/a2a-metadata-fields-survive-passthrough.md
---

# Controller architecture decisions — peer index

One line per controller ADR, phrased as "what it means for you, a peer repo."
Each references its file under `docs/dev/decisions/`; read that file for the full
rule, examples, and rationale. This index never restates the ADR body.

- **controller-thin-bridge.md** — The controller blind-forwards payloads it does not
  itself act on and never fabricates defaults; don't rely on it to interpret,
  normalize, or fill in your fields — the producer (you) must send them explicitly.

- **relay-pipeline-pattern.md** — Runtime-directed commands are exactly one of relay
  (one correlated `request_id` reply), fire-and-forget (synthetic ACK only), or an
  uncorrelated event stream; classify a new command up front — a fire-and-forget or
  stream command will hang if a peer awaits it as a relay.

- **generic-passthrough-default.md** — Unrecognised runtime→orchestrator wire commands
  are forwarded by default, not dropped; a new cross-bridge command usually needs no
  controller change and pays no controller tax.

- **spawn-error-contract.md** — Spawn failures arrive as a structured
  `spawn_error: Option<SpawnErrorData>` sibling alongside the legacy `error` string
  (absent from JSON when none); read the typed variant for class-specific handling,
  and a new variant is a coordinated wire-contract change.

- **host-container-isolation.md** — The controller only ever observes/acts through the
  Docker socket and never bypasses host namespace isolation; don't expect it to see or
  act on host processes, and host-side visibility features are forbidden by default.

- **configurable-runtime-container-prefix.md** — Spawn naming, reaping, and eviction are
  all scoped to `RUNTIME_CONTAINER_PREFIX`; parallel stacks on one daemon must set
  distinct prefixes or they cross-reap each other's containers.

- **tenant-isolated-builds.md** — Tenancy is enforced solely by the orchestrator
  validating the workspace-scoped `CONTROLLER_TOKEN` at WebSocket connect; there is no
  per-task company assertion, and (V1 closed-beta) customer build `RUN` lines are not
  sandboxed.

- **bootstrap-pod-docker-sock-group-add.md** — Bootstrap pods get docker.sock access via
  a supplementary GID (probed from the host socket), never by running as root; a peer
  relying on bootstrap self-validation should not expect a uid-0 pod.

- **bootstrap-pod-docker-socket-mount.md** — Only `mode == "bootstrap"` runtimes mount
  the Docker socket (for image self-validation); greenfield and brownfield-active
  runtimes never do, so send exactly the string `"bootstrap"` to enable it.

- **codex-api-key-controller-held-container-env.md** — Codex credentials are
  controller-held and provisioned to product runtimes (API key via container env, or a
  ChatGPT-subscription `auth.json` written by the runtime); the orchestrator is not in
  the Codex credential path.

- **a2a-metadata-fields-survive-passthrough.md** — Metadata survives the controller only
  as explicitly-named fields on its typed `Metadata` struct; any new required cross-hop
  metadata key (e.g. a2a `to_project`) must be added there or it is silently dropped —
  a coordinated runtime + controller + orchestrator change.
