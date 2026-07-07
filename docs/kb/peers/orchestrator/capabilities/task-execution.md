---
type: capability
title: "Task execution and polling"
tags: [tasks, async, polling, runtime, cross-repo]
timestamp: 2026-07-07T00:00:00Z
description: "How a peer submits a command/message/claude task to a runtime and polls for the result"
repo: orchestrator
commit_sha: 6843154
evidence:
  - REST_API.md
  - cmd/handlers/tasks.go
  - cmd/models/task.go
  - cmd/models/task_response.go
---

# Task execution and polling

**What it does.** Lets a peer run a unit of work inside a live runtime container — a shell
command, a plain message, or a Claude prompt — and retrieve the result. This is the async
"do something in a runtime and get the output back" channel.

**How a peer interacts.** Submit is one POST per task kind, each targeting a runtime by name:
- `POST /api/v1/tasks/command` — run a shell command (`command`, optional `workdir`).
- `POST /api/v1/tasks/message` — send a plain message (`message`).
- `POST /api/v1/tasks/claude` — run a Claude prompt (`prompt`).

All three also require `controller_name` and `runtime_name` — the string names of the target
controller and runtime, not UUIDs. If you launched the runtime via `POST /api/v1/launch` (which
returns only `runtime_uuid`/`instance_uuid`), obtain the names by reading the runtime **instance**
record (it carries `runtime_name` + `controller_name`) or by using `POST /api/v1/tasks/spawn` (which
returns them inline) — see the runtime-lifecycle capability. The submit returns immediately with
a Task record (its `task_uuid` and `status`) — it does **not** carry the result. The peer then
polls for the outcome:
- `GET /api/v1/tasks/{uuid}` — the current Task (its status).
- `GET /api/v1/tasks/{uuid}/response` — the stored result, once one exists.

**Observable behavior.** Submission is fire-and-forget: the task is persisted as `pending`,
then dispatched to the target controller. The peer learns the outcome only by polling `{uuid}`
for a terminal status and reading `{uuid}/response` for the payload. A task moves through
`pending → sent → completed | failed`. The response endpoint returns an HTTP 400 error while the task
is still `pending` or `sent` (no result recorded yet); a peer polls until a result appears or a
terminal status is reached.

**Contract.**
- In (command): `{controller_name, runtime_name, command, workdir?}`. In (message):
  `{controller_name, runtime_name, message}`. In (claude): `{controller_name, runtime_name, prompt}`.
- Out (submit): a Task — `{task_uuid, type, status, runtime_name, controller_name, created_at, …}`,
  `status="pending"`.
- Out (response): a TaskResponse — `{response_uuid, task_uuid, success, response, error?, stdout?,
  stderr?, exit_code, spawn_error?, created_at}`. `success=false` corresponds to a `failed` task;
  `error`/`stderr` carry the failure detail; `spawn_error` (JSONB) carries a structured spawn failure
  when the task never ran.
- Errors: submit rejects a body missing required fields (validation error). Response GET returns
  an HTTP 400 error when no result has been recorded yet (not a 404).

**Invariants.** A task's terminal state is `completed` (success) or `failed`; `success=true` in the
response iff the task completed. Tasks and responses are company-scoped — a caller only sees tasks
belonging to its own tenant. `task_uuid` is the stable correlation key between the submit, the
status poll, and the response.

**Failure modes.** Command/prompt errors inside the runtime surface as a `failed` task with a
TaskResponse carrying `error` and a nonzero `exit_code` — the submit itself still succeeds, so a
peer must inspect the response to distinguish success from failure. If the target controller is not
connected, the task stays un-progressed rather than erroring at submit; the peer observes it never
leaving `pending`/`sent`.

**Gotchas.**
- No result is returned synchronously — waiting for completion is the caller's job (submit-then-poll).
  Do not treat a 2xx submit as "the work ran."
- Response GET returning an HTTP 400 error means "not ready yet," not a malformed request — keep
  polling until a terminal status. (Do not key the retry on 404; the not-ready error is a 400.)
- The orchestrator's own `GET /api/v1/tasks/{uuid}` and `/response` handlers are deprecated: ps-api
  now owns the task read path DB-direct (its route for responses is `/tasks/{id}/responses`, plural).
  A peer behind ps-api reads through ps-api, not these orchestrator endpoints.
- These endpoints run a runtime task; they are distinct from the interactive Claude *session*
  channel (spawn/input/close) used for multi-turn conversations.

**Business-critical data.** A submitted task is persisted as a `task` row (`task_uuid`, `type`,
`status`, `runtime_name`, `controller_name`, kind-specific `payload`); the controller's result is
recorded as a `task_response` row keyed by `task_uuid` (`success`, `response`, `error`, `stdout`,
`stderr`, `exit_code`). The poll reads join these by `task_uuid`. (Tenant scoping applies as
everywhere — see context.md.)
