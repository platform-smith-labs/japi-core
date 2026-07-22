---
type: interface
title: "Boot environment contract"
tags: [boot, env-vars, configuration, lifecycle, controller]
timestamp: 2026-07-09T10:42:29Z
description: "The env vars a controller must inject when creating a runtime container, and the conditions that make boot fatal"
repo: runtime
commit_sha: 6f27e3b
evidence:
  - src/config.rs
  - src/main.rs
  - src/session/codex/credentials.rs
  - src/session/codex/session_type.rs
  - src/session/credential_dispatch.rs
  - src/session/claude/mod.rs
  - src/credential_helper/main.rs
  - src/cred_server/mod.rs
  - src/mcp_server/mod.rs
consumes_interfaces:
  - {name: "boot-env-injection", kind: env-vars, peer: controller, intent: "the controller sets these on the container at create time; the runtime reads them once at process start"}
see_also:
  - {repo: runtime, capability: "Controller WebSocket protocol", descriptive: false, intent: "the env-selected launch path determines which readiness event the runtime emits"}
---

# Boot environment contract

What the **controller must inject** into a runtime container's environment at create time. The
runtime (the image ENTRYPOINT, running as PID 1) reads these once at process start; they are
immutable for the process's lifetime. Docker `CMD` argv becomes the supervised customer process
(empty `CMD` → the runtime serves WS only; builder mode never supervises a customer CMD).
`PLATFORM_SMITH_*` variables are stripped from coding-agent sessions, spawned daemons, and the
supervised CMD child; one-shot `execute_command` children are the exception — they inherit the
full environment.

## Required (missing → exit 1, container dies immediately)

- **`PLATFORM_SMITH_WS_URL`** — the controller's WebSocket endpoint the runtime dials.
  Must start with `ws://` or `wss://`; anything else (including `http://`) is a fatal config
  error at boot. After boot, connection failures are NOT fatal — the runtime retries forever
  with backoff.
- **`PLATFORM_SMITH_RUNTIME_NAME`** — the runtime's identity string. Echoed in `registration`
  and auto-injected into every outbound message's metadata.

## Launch-path selector

- **`PLATFORM_SMITH_INSTANCE_UUID`** (optional) — the orchestrator-minted instance UUID.
  - **Present and non-empty** → unified launch path: the value is echoed at registration and the
    readiness event is `launch_ready {instance_uuid}` (legacy `runtime_ready` is suppressed).
  - **Absent or empty-string** (empty is treated as absent) → legacy path: the runtime
    self-mints a UUID and emits `runtime_ready` instead. The self-minted value is used for
    metadata observability only — it is **NEVER echoed for correlation** (correlation fields go
    empty rather than carry a value the orchestrator cannot match).
  A peer that needs to correlate launch events MUST inject this variable.

## Mode

- **`PS_RUNTIME_MODE`** (optional) — `greenfield` (default when absent) or `builder`.
  Any unknown value — including retired legacy modes — logs a WARN and falls back to
  `greenfield`; it never crashes and never silently becomes builder. `builder` selects the
  in-pod image-build pod: no customer-CMD supervision, readiness event `launch_builder_ready`,
  and it is the only mode that accepts `build_image`.

## Optional knobs (defaults apply when unset)

- **`HOME`** — Claude-session paths fall back to `/home/psruntime` when unset; Codex and Vertex
  credential paths instead fail their (non-fatal) credential setup, so `HOME` should always be
  set. Anchors the `$HOME`-relative defaults below.
- **`CODEX_HOME`** — where the Codex auth bundle (`auth.json`) is written; default `$HOME/.codex`.
- **`CODEX_BIN`** — the Codex CLI binary to exec; default `codex` resolved on `PATH`.
- **`PS_CRED_SOCKET`** — read by the git-credential-helper **child binary** (not the runtime
  itself) to locate the credential UDS; default `/var/run/platform-smith/cred.sock`. The runtime
  always binds the UDS at that fixed default path.
- **`PS_VERTEX_CREDS_DIR`** — where Vertex service-account credentials are materialised;
  default `$HOME/.config/gcloud`.
- **`RUST_LOG`** — standard tracing filter; default `platform_smith_runtime=info`.

## Boot-fatal conditions beyond env (each → exit 1 before the WS connects)

1. Credential UDS bind failure (`/var/run/platform-smith/cred.sock` — e.g. the directory is
   missing or not writable in the image).
2. MCP per-session config-root sweep/create failure (without it every later session spawn would
   silently lose its MCP config).
3. MCP HTTP server bind failure on loopback port **9099** (eager-bind at boot to fail fast on a
   port conflict).

A container that keeps restarting immediately with exit code 1 and no `registration` on the wire
indicates one of the two required env vars is missing/invalid or one of the three binds failed —
not a controller-connectivity problem.

## Gotchas

- Registration is sent before the command router exists; only the readiness event
  (`launch_ready` / `launch_builder_ready` / `runtime_ready`) means the runtime is safe to
  command. Which one fires is decided entirely by this env contract
  (`PS_RUNTIME_MODE` + `PLATFORM_SMITH_INSTANCE_UUID`).
- Coding-agent credentials are NOT part of the boot env — they arrive post-registration over the
  WS and are frozen per instance; changing a credential requires a fresh container with a fresh
  instance UUID.
- Do not rely on `PLATFORM_SMITH_*` values being visible to code running inside sessions or
  spawned processes — they are stripped; per-session identity uses `PS_SESSION_ID` instead.
