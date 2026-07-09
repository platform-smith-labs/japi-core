---
type: overview
title: "Orchestrator — central API, WebSocket hub, task queue"
tags: [overview, orchestrator, api, websocket]
timestamp: 2026-07-09T10:40:45Z
repo: orchestrator
commit_sha: 2fa8172
---
# Orchestrator

The orchestrator is Platform Smith's central control plane (Go, HTTP on port 9003). Peers use it to
spawn and drive runtime containers, run tasks and interactive sessions, exchange agent-to-agent
messages across projects, retrieve session artifacts, mint scoped git tokens, and bridge in-session
agent events to the ps-workflow engine.

**What peers rely on it for.** A REST API (consumed chiefly by the ps-api gateway) for runtimes,
tasks, sessions, conversations, artifacts, git connections, and single-repo PR-token minting; a
WebSocket hub that the controller connects to, over which runtime lifecycle and agent traffic flow;
and an outbound bridge that forwards agent signals and session-lifecycle events to **ps-workflow** so
parked workflow steps complete.

**Role for a peer's agent.** If your task needs to create a runtime, run a command or Claude session,
message another project, read what a session produced, mint a PR token, or complete a parked workflow
signal, you interact with the orchestrator — via its REST API (through ps-api) or, for
controllers/runtimes, its WebSocket protocol. If you are the workflow engine, the orchestrator is the
service that calls *you* with session/agent completion events.
