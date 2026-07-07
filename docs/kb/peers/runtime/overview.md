---
type: overview
title: "Platform Smith Runtime — the in-pod execution engine (PID 1)"
tags: [runtime, pid1, executor, websocket, coding-agent]
timestamp: 2026-07-06T23:40:38Z
description: "What the runtime is and its role for peer repos: the Rust executor that runs as PID 1 in every Platform Smith pod"
repo: runtime
commit_sha: 33f85d5
evidence:
  - src/main.rs
  - src/lib.rs
  - src/config.rs
  - README.md
  - CLAUDE.md
  - Cargo.toml
  - docs/kb/kb-config.yaml
---

# Platform Smith Runtime

The runtime is a Rust binary that runs as **PID 1 inside every Platform Smith pod**. It is the
platform's **executor**: it owns the real business logic of running things inside a customer
container — one-shot shell commands, long-lived daemon processes, interactive coding-agent
(Claude and Codex) sessions with streamed output, git clone/checkout, in-pod image builds, file
and secret materialisation, and an in-pod MCP tool server.

Contrast with the **controller**, which is a thin bridge: the controller creates/destroys
containers and forwards messages between orchestrator and runtime, but it does not interpret
most of them. If a peer wants something to *happen inside a pod*, the runtime is the component
that does it; the controller and orchestrator are the path to reach it.

## Role in the platform

- **Init process**: as PID 1 it supervises the image's original CMD (if any), handles Unix
  signals, and reaps zombies. A pod with an empty CMD runs the runtime as a pure
  command-server. On a `shutdown` command it acks and then exits — the pod terminates.
- **Single outbound WebSocket** to the controller is its only control channel. It registers
  itself on connect and then announces readiness; only after the readiness event is it safe to
  command (see context).
- **Coding-agent host**: spawns and manages Claude (retained process) and Codex
  (spawn-per-turn) sessions, streams their output upstream, injects credentials, and serves
  each session an MCP tool endpoint whose tools bridge back to the orchestrator.
- **Two operating modes**: `greenfield` (default; normal product pod) and `builder` (build-only
  pod that runs an in-pod `docker build` and emits ordered `launch_*` events; no customer CMD
  is supervised).

## What the crate ships

Two binaries from one crate:

1. **platform-smith-runtime** — the PID 1 init process described above.
2. **ps-git-credential-helper** — a small git credential helper installed inside the pod; git
   invokes it, and it talks to the runtime's local Unix socket to mint short-lived git
   credentials on demand (the runtime relays the mint request upstream).

## What peers should expect

- A full roster of executor capabilities (command execution, sessions, clone, build, MCP/A2A,
  credential pipeline, file materialisation) — each documented as its own capability concept.
- Several command families are deliberately **fire-and-forget** (no success reply); never model
  them as request/response. Which ones are listed in the cross-cutting gotchas concept; the two
  session-ID namespaces every session-related message uses are stated once in the context concept.
- Nothing inside a runtime survives a pod restart; durable state lives with the orchestrator.
