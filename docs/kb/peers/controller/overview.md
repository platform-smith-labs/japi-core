---
type: overview
title: "Controller — thin WebSocket bridge + Docker lifecycle manager"
tags: [controller, websocket, docker, runtime-lifecycle, thin-bridge]
timestamp: 2026-07-07T00:00:00Z
description: "What the controller is and its role for peer repos: a stateful message relay and container lifecycle manager that owns no payload semantics"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/main.rs
  - src/lib.rs
  - README.md
  - CLAUDE.md
  - docs/dev/decisions/controller-thin-bridge.md
  - docs/dev/decisions/relay-pipeline-pattern.md
  - src/websocket/registry.rs
---

# Controller

The **controller** is the one component that turns an orchestrator's intent into
running containers. It is a Rust service that sits between the orchestrator and
the per-user **runtime** containers, and it does two jobs and only two jobs:

1. **Message routing** — it is a bidirectional bridge. Work from the orchestrator
   flows down to a named runtime; events and replies from runtimes flow back up.
2. **Docker lifecycle + host arbitration** — it spawns, builds for, and terminates
   runtime containers on its Docker host; arbitrates host ports; evicts stale
   containers; and provisions coding-agent credentials into a container at spawn.

## The thin-bridge principle (read this before scoping any task here)

The controller is deliberately **thin**: it moves messages and manages containers,
but it **does not own the *meaning* of any payload**. If a task concerns *what a
message says* or *what a runtime should do with it* — command semantics, session
logic, credential resolution policy, task scheduling — that logic lives in the
**orchestrator** or the **runtime**, not here. The controller forwards most
runtime-directed payloads verbatim and only interprets the thin routing/lifecycle
envelope around them.

Practical consequence for a peer: if you are adding a new *behavior* a runtime
performs, the controller change is usually just "carry one more field through" or
"relay one more command name" — the decision-making belongs upstream or in the
runtime. Do not push payload semantics into the controller.

## What it owns

- **Container lifecycle** — spawn a product runtime, spawn a builder pod, build a
  customer image, terminate a named runtime.
- **Host-port arbitration & eviction** — it decides which host ports a container
  gets and reaps stale/conflicting containers, strictly within an allowlist keyed
  on the runtime-name prefix (see context).
- **Credential-at-spawn provisioning** — it holds shared coding-agent credentials
  (Claude/Codex) and freezes them into a runtime as it comes up; a credential
  change requires a fresh runtime.
- **Correlation** — it matches runtime replies back to the originating request.

## What it does NOT own

- Payload/command *semantics* — orchestrator or runtime.
- Task scheduling, user/session/workspace state — orchestrator.
- In-container execution (running commands, coding-agent sessions) — runtime.

## No database — state is in-memory and transient

The controller has **no database**. Its knowledge of live runtimes is an
**in-memory registry** that is **lost on restart**. This is safe by design:
surviving runtime containers reconnect to a fresh controller on their own, and the
orchestrator holds the durable record. A peer must never assume the controller
remembers anything across a restart — if you need durable state, it lives upstream.

## How peers reach it

The controller is not called over REST. It maintains a client connection **up** to
the orchestrator (it dials the orchestrator, not the reverse) and a WebSocket
server **down** for runtime containers. See `context.md` for the two surfaces, the
authentication contract, the container-naming/eviction contract, and the message
envelope — the ubiquitous facts the rest of this KB relies on.
