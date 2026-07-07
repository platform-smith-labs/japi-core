---
type: overview
title: "Orchestrator — central API, WebSocket hub, task queue"
tags: [overview, orchestrator, api, websocket]
timestamp: 2026-07-07T00:00:00Z
repo: orchestrator
commit_sha: 6843154
---
# Orchestrator

The orchestrator is Platform Smith's central control plane (Go, HTTP on port 9003). Peers use it to
spawn and drive runtime containers, run tasks and interactive sessions, exchange agent-to-agent
messages across projects, and retrieve session artifacts.

**What peers rely on it for.** A REST API (consumed chiefly by the ps-api gateway) for runtimes,
tasks, sessions, conversations, artifacts, and git connections; and a WebSocket hub that the
controller connects to, over which runtime lifecycle and agent traffic flow.

**Role for a peer's agent.** If your task needs to create a runtime, run a command or Claude session,
message another project, or read what a session produced, you interact with the orchestrator — via
its REST API (through ps-api) or, for controllers/runtimes, its WebSocket protocol.
