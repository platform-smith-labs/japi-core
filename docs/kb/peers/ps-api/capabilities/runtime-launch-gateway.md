---
type: capability
title: "Runtime launch gateway"
tags: [launch, runtime, gateway, proxy, tasks]
timestamp: 2026-07-07T03:45:26Z
description: "How a peer launches a runtime container through ps-api and tracks it to readiness"
repo: ps-api
commit_sha: f8157e0
evidence:
  - cmd/handlers/launch.go
  - cmd/handlers/verbatim.go
  - cmd/handlers/launches.go
  - cmd/handlers/launch_sse.go
  - cmd/handlers/launch_sse_transform.go
  - cmd/handlers/runtimes.go
  - cmd/handlers/tasks.go
  - cmd/db/launch_operations.go
  - cmd/db/runtime_operations.go
see_also:
  - {repo: orchestrator, capability: "Runtime launch engine", intent: "owns launch validation, per-kind rules, and the launch state machine ps-api fronts", descriptive: true}
  - {repo: ps-api, capability: "Realtime streams", intent: "SSE plumbing (auth, cursor replay, heartbeats) shared by the launch event stream"}
---

# Runtime launch gateway

**What it does.** Fronts the platform's runtime-container lifecycle for authenticated frontend/API
clients: submit a launch, watch it progress to ready, list/inspect running runtimes, and
stop/restart/delete them. Launch writes are proxied to the orchestrator (the owner of launch
logic); launch/runtime reads are answered directly from the shared database with no orchestrator
round-trip.

**How a peer interacts.** All routes require a user JWT; everything is company-scoped.
- `POST /api/v1/launch` — start a runtime. Body is a discriminated union keyed on `kind`
  (`service` | `sandbox`), forwarded byte-for-byte to the orchestrator; ps-api does not validate
  or transform it (ask the orchestrator's launch capability for the per-kind body shape).
- `GET /api/v1/launches` / `GET /api/v1/launches/{instance_uuid}` /
  `…/{instance_uuid}/attempts` — launch list (latest incarnation per runtime, newest-first),
  single-launch detail (with declared ports + live port mapping), and attempt history.
- `GET /api/v1/launches/{instance_uuid}/events/stream` — SSE push alternative to polling.
- `GET /api/v1/runtimes[/{id}]` — runtime instance list/get; `POST …/{id}/stop`,
  `POST …/{id}/restart`, `DELETE …/{id}` — lifecycle actions, proxied to the orchestrator.
- `POST /api/v1/tasks` — legacy typed task surface (spawn_runtime, execute_command,
  send_message, execute_claude), routed to the orchestrator's per-type task endpoints;
  `GET /tasks/{id}` and `…/responses` read task state/results DB-direct; `…/wait` blocks via
  the orchestrator (a proxied blocking wait), so it fails when the orchestrator is unreachable,
  unlike the other two reads.

**Observable behavior — async readiness.** Launching is asynchronous: `POST /launch` returns
immediately with the orchestrator's response (key fields: `runtime_uuid`, `session_uuid?`,
`status`, `instance_uuid`) — upstream status and body are relayed verbatim, including errors.
The concrete readiness signal is the **runtime instance** status, never the parent runtime record
(which lags behind): poll `GET /api/v1/launches/{instance_uuid}` and read `status` until it is
`ready` (success) or `failed` (terminal failure; `failed_phase` says where). The SSE stream gives
the same signal push-style: an initial `launch_status` snapshot, raw launch events (client
reconnect replays from the `Last-Event-ID` cursor), 15s heartbeats, and a `stream_end` event when
the launch reaches `ready`/`failed`.

**Identity hand-off (UUIDs → names).** The legacy task routes key runtimes by **name**
(`controller_name` + `runtime_name`), while `/launch` and the launch reads deal in **UUIDs**. The
bridge is the launch read surface: every row from `GET /api/v1/launches` carries the UUIDs and
`runtime_name` + `controller_name` for the same entity. `GET /api/v1/runtimes` rows also carry
both names, but beware: their `runtime_uuid` wire field is actually the runtime *instance* UUID
(it matches `/launches`' `instance_uuid`, not its `runtime_uuid`) — correlate across the two
lists by names or instance UUID, never by that field name. A failed `spawn_runtime` task surfaces a structured `spawn_error` (with a class-specific
HTTP status) on the per-task GET/wait reads.

**Invariants.** Multi-tenancy is enforced in SQL on every read; unknown and cross-tenant IDs both
return 404 (no existence leak). The launch list shows exactly one row per runtime — its newest
incarnation. Launch reads never contact the orchestrator; launch/lifecycle writes always do, with
the caller's identity forwarded as derived user/company headers.

**Failure modes.** Orchestrator rejections (e.g. 422 for a launch with no bound controller) pass
through verbatim with the upstream body intact. Orchestrator unreachable/timeout → 503/504 from
the gateway. A launch that dies mid-flight terminates as `status=failed` + `failed_phase`;
per-attempt errors are in the attempts read.

**Gotchas.**
- Readiness must be read from the instance-keyed launch reads above; the parent runtime record's
  status is not a readiness signal.
- Two distinct UUID handles both travel as "instance UUID": the launch list emits the gateway DB
  key, which the REST detail/attempts routes accept, while the SSE stream (and the
  `instance_uuid` the orchestrator returns from `POST /launch`) use the orchestrator's canonical
  launch key — the DB layer states these are different columns. Whether the two values coincide
  in practice is UNKNOWN — safest is: use list-read values for REST reads, the `POST /launch`
  `instance_uuid` for the SSE stream.
- `GET /runtimes` derives its `status` (`active`/`inactive`) from connectivity, separate from the
  launch-progress `launch_status` field also on the row — don't conflate them.
- The `/tasks` surface is legacy; `POST /launch` is the consolidated launch entry point.

**Business-critical data.** `runtime_instance` (one row per launch incarnation: status,
failed_phase, connected/ready, port_mapping) is the source of truth for all reads; `launch_event`
(cursor-ordered log) feeds the SSE timeline; `launch_attempt` holds per-attempt build/spawn
outcomes; `task` + `task_response` back the legacy task reads.
