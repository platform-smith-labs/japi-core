---
type: interface
title: "Schema: task + session"
tags: [schema, postgres, task, session]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for task, session and correlation tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0013_task.sql
  - migrations/0014_session.sql
  - migrations/0020_session_event.sql
  - migrations/0023_session_instance_scoped_name.sql
  - migrations/0025_agent_definitions_and_secrets.sql
  - migrations/0026_session_coding_agent_type.sql
  - migrations/0027_session_agent_session_id.sql
  - migrations/0030_session_task_correlation.sql
  - migrations/0033_session_task_correlation_turn.sql
  - migrations/0034_session_task_correlation_poll_cursor.sql
  - migrations/0035_session_task_correlation_lifecycle_kinds.sql
  - migrations/0041_session_integration_connection.sql
  - migrations/0043_session_add_conversation_id.sql
  - migrations/0001_enums.sql
provides_interfaces:
  - {name: "task/session tables", kind: postgres-schema, intent: "tasks, coding-agent sessions, session events and task-session correlation"}
---

# Schema: task + session domain

Final-state reference for the five tables backing task dispatch, coding-agent sessions, session transcripts, and workflow park/resume correlation.

### task
Internal dispatch-queue detail for controller-bound work (not the user-facing lifecycle, which lives on the runtime).

| column | type | null | default |
|---|---|---|---|
| task_id | SERIAL (PK) | no | auto |
| task_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| user_id | INTEGER | no | — |
| controller_id | INTEGER | yes | — |
| type | TEXT | no | — |
| status | TEXT | no | 'pending' |
| runtime_name | TEXT | yes | — |
| controller_name | TEXT | yes | — |
| payload | JSONB | no | '{}' |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| sent_at | TIMESTAMPTZ | yes | — |
| completed_at | TIMESTAMPTZ | yes | — |

**Constraints:** PK `task_id`; `task_uuid` UNIQUE. FKs: `company_id` → company; composite `(company_id, user_id)` → users; composite `(company_id, controller_id)` → controller. Unique index `(company_id, task_id)` (composite-FK target). Checks: `type` IN (spawn_runtime, send_message, execute_command, execute_claude, spawn_claude_session, claude_session_input, close_claude_session, shell_session_start, brownfield_build, terminate_runtime); `status` IN (pending, sent, completed, failed).

**Indexes:** `(company_id)`; `(company_id, status)`; `(company_id, runtime_name)` WHERE runtime_name IS NOT NULL; `(created_at DESC)`; `(company_id, controller_name)` WHERE controller_name IS NOT NULL; `(company_id, controller_id)` WHERE controller_id IS NOT NULL; UNIQUE `(company_id, task_id)`.

### task_response
One response row per task, carrying success/error/output of task execution (spawn_error holds a structured spawn-failure payload from the controller).

| column | type | null | default |
|---|---|---|---|
| task_response_id | SERIAL (PK) | no | auto |
| task_response_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| task_id | INTEGER | no | — |
| success | BOOLEAN | no | FALSE |
| response | TEXT | yes | — |
| error | TEXT | yes | — |
| stdout | TEXT | yes | — |
| stderr | TEXT | yes | — |
| exit_code | INTEGER | no | 0 |
| spawn_error | JSONB | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:** PK `task_response_id`; `task_response_uuid` UNIQUE. FKs: `company_id` → company; composite `(company_id, task_id)` → task. Uniques: `(company_id, task_response_id)` (composite-FK target); `(company_id, task_id)` (one response per task).

**Indexes:** `(company_id)`; `(created_at DESC)`; `(company_id, task_id)`.

### session
A coding-agent (or shell) session running inside a runtime instance; identity is scoped to the owning runtime instance so a reused session name can never reattach a dead instance's transcript.

| column | type | null | default |
|---|---|---|---|
| session_id | SERIAL (PK) | no | auto |
| session_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| user_id | INTEGER | no | — |
| runtime_instance_id | INTEGER | no | — |
| session_name | TEXT | no | — |
| runtime_name | TEXT | no | — |
| display_name | TEXT | no | — |
| session_type | TEXT | no | 'claude' |
| state | TEXT | no | 'pending' |
| pid | INTEGER | yes | — |
| model | TEXT | yes | — |
| error | TEXT | yes | — |
| exit_code | INTEGER | yes | — |
| coding_agent_type | coding_agent_type (enum) | yes | — |
| agent_session_id | TEXT | yes | — |
| integration_connection_id | INTEGER | yes | — |
| is_personal_integration | BOOLEAN | no | FALSE |
| conversation_id | INTEGER | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| started_at | TIMESTAMPTZ | yes | — |
| closed_at | TIMESTAMPTZ | yes | — |

Semantics: `coding_agent_type` is the harness frozen at spawn (NULL = shell/legacy, consumers default to claude_code); `agent_session_id` is an opaque agent-native resume token; `integration_connection_id` is the credential that authenticated the session, frozen at first insert (NULL = legacy/static key); `is_personal_integration` is denormalized at spawn to gate message/continue to the owner; `conversation_id` binds the session to a conversation (nullable — most sessions have none).

**Constraints:** PK `session_id`; `session_uuid` UNIQUE. FKs: `company_id` → company; composite `(company_id, user_id)` → users; composite `(company_id, runtime_instance_id)` → runtime_instance; composite `(company_id, integration_connection_id)` → integration_connection; composite `(company_id, conversation_id)` → conversation. Uniques: `(company_id, runtime_instance_id, session_name)` — session-name uniqueness is instance-scoped, NOT company-global; unique index `(company_id, session_id)` (composite-FK target). Checks: `state` IN (pending, started, failed, closed, crashed); `session_type` IN (claude, shell).

**Indexes:** `(company_id)`; `(company_id, state)`; `(company_id, runtime_name)`; `(created_at DESC)`; `(company_id, runtime_instance_id)` WHERE runtime_instance_id IS NOT NULL; `(company_id, session_type)`; UNIQUE `(company_id, session_id)`; `(company_id, conversation_id)` WHERE conversation_id IS NOT NULL.

### session_event
Append-only session transcript store and SSE/poll source; rows are never updated or deleted (no retention janitor). `session_event_id` (BIGSERIAL) is the internal keyset cursor and never appears on the wire; `session_event_uuid` is the wire cursor, resolved via the company-scoped unique.

| column | type | null | default |
|---|---|---|---|
| session_event_id | BIGSERIAL (PK) | no | auto |
| session_event_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| session_id | INTEGER | no | — |
| event_type | TEXT | no | — |
| severity | TEXT | no | 'info' |
| phase | TEXT | yes | — |
| data | JSONB | no | '{}' |
| created_at | TIMESTAMPTZ | no | NOW() |

No `updated_at` (append-only). Gotcha: inside the `data` envelope, the `"data"` key is a STRING (the inner agent NDJSON line), not nested JSON — consumers depend on this.

**Constraints:** PK `session_event_id`. FKs: `company_id` → company; composite `(company_id, session_id)` → session. Unique: `(company_id, session_event_uuid)` — deliberately company-scoped, not globally unique; cursor-resolution queries always carry company_id. Check: `severity` IN (info, warn, error).

**Indexes:** `(company_id)`; `(company_id, session_id, session_event_id)` (keyset pagination).

### session_task_correlation
Durable tenant-scoped map from an orchestrator session NAME to a Conductor `(workflow_id, task_ref_name)` — the exactly-once park/resume store for workflow waits. `session_name` is an opaque correlation token, NOT an FK to session. Lookup identity is `(company_id, session_name)`; the terminal transition (open → completed/failed) is a single atomic conditional update.

| column | type | null | default |
|---|---|---|---|
| session_task_correlation_id | SERIAL (PK) | no | auto |
| session_task_correlation_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| session_name | TEXT | no | — |
| workflow_id | TEXT | no | — |
| task_ref_name | TEXT | no | — |
| status | session_task_correlation_status (enum) | no | 'open' |
| kind | TEXT | no | 'session_close' |
| poll_deadline | TIMESTAMPTZ | yes | — |
| poll_after_event_id | BIGINT | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| terminal_at | TIMESTAMPTZ | yes | — |

Semantics: `kind` says how a parked row completes — session_close (inbound close bridge, default), turn_poll (per-turn reply wait), launch_poll (runtime start until READY), session_start_poll (session start until started). Poll kinds are bounded by `poll_deadline`; rows past deadline are failed by a startup reconciliation sweep. `poll_after_event_id` is the session_event_id baseline captured at prompt-send time for turn_poll (a prior turn's result frame can never complete the current turn); NULL for session_close rows.

**Constraints:** PK `session_task_correlation_id`; uuid UNIQUE (global). FK: `company_id` → company. Unique: `(company_id, session_name)` — tenant is part of the key; this implicit index also serves the hot lookup. Checks: `kind` IN (session_close, turn_poll, launch_poll, session_start_poll); `(status = 'open') = (terminal_at IS NULL)` (status and terminal_at in lockstep).

**Indexes (all partial):** `(terminal_at)` WHERE status <> 'open' (cleanup sweep); `(poll_deadline)` WHERE status = 'open' AND kind = 'turn_poll'; `(poll_deadline)` WHERE status = 'open' AND kind = 'launch_poll'; `(poll_deadline)` WHERE status = 'open' AND kind = 'session_start_poll'.

## Enum types used

- **coding_agent_type**: `claude_code`, `codex_cli`, `cursor_cli`, `opencode`, `gemini_cli` (shared with the agent-definition tables).
- **session_task_correlation_status**: `open`, `completed`, `failed`.

Note: `task.type`, `task.status`, `session.state`, `session.session_type`, and `session_task_correlation.kind` are TEXT columns constrained by CHECK, not PostgreSQL enums; their value sets are listed under each table's constraints above.
