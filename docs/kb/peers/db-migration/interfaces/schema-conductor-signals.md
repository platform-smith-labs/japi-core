---
type: interface
title: "Schema: Conductor signals, schedules & webhook triggers"
tags: [schema, postgres, workflow, conductor, signal, schedule, webhook]
timestamp: 2026-07-09T10:37:36Z
description: "Final-state reference for the signal correlation store, cron schedule registry, and webhook trigger tables"
repo: db-migration
commit_sha: a9ad8ea
evidence:
  - migrations/0056_signal.sql
  - migrations/0057_schedule.sql
  - migrations/0058_webhook_trigger.sql
provides_interfaces:
  - {name: "conductor trigger/signal tables", kind: postgres-schema, intent: "workflow signal correlation store, cron schedule registry and webhook trigger credentials"}
---

# Schema: Conductor signals, schedules & webhook triggers

The three tables backing the Conductor node-catalog expansion: a durable signal-correlation store a workflow parks on, a tenant-scoped cron registry that starts workflows, and webhook trigger credentials that start workflows over HTTP. All three back ps-workflow.

### signal

Durable, tenant-scoped Model-B signal correlation store. A workflow parks on a `(company_id, correlation_id)` token and is later unparked from one of four sources (human, agent via `ps_signal`, a2a count-of-N, webhook). Mirrors `session_task_correlation`'s exactly-once CAS discipline but keys on the high-entropy `correlation_id` token instead of a session name. The terminal transition is a single atomic `UPDATE signal SET status=… WHERE company_id=$1 AND correlation_id=$2 AND status='open' RETURNING …`, and `UNIQUE (company_id, correlation_id)` is the exactly-once backstop. Tenant is **part of the key** — a `correlation_id` collision across tenants must never cross.

| column | type | null | default |
|---|---|---|---|
| signal_id | SERIAL | no | (PK) |
| signal_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| correlation_id | TEXT | no | — |
| workflow_id | TEXT | yes | — |
| task_ref_name | TEXT | yes | — |
| kind | TEXT | no | 'signal' |
| status | signal_status | no | 'open' |
| payload | JSONB | no | '{}' |
| count_target | INTEGER | no | 1 |
| count_seen | INTEGER | no | 0 |
| deadline | TIMESTAMPTZ | yes | — |
| poll_deadline | TIMESTAMPTZ | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| terminal_at | TIMESTAMPTZ | yes | — |

**Constraints:**
- PK: `signal_id`; UNIQUE: `signal_uuid`
- FK `company_id` → company(company_id)
- UNIQUE `(company_id, correlation_id)` — the exactly-once backstop; also serves the hot Get / terminal-CAS / count-increment path via its implicit index (no separate index)
- CHECK `signal_terminal_at_check`: `(status = 'open') = (terminal_at IS NULL)` — status and terminal_at stay in lockstep
- CHECK `signal_count_check`: `count_target >= 1 AND count_seen >= 0`

**Indexes (all partial):**
- `idx_signal_terminal_at` (terminal_at) WHERE status <> 'open' — TTL cleanup sweep (terminal rows only)
- `idx_signal_open_deadline` (deadline) WHERE status = 'open' AND deadline IS NOT NULL — await-signal timeout sweep
- `idx_signal_open_poll_deadline` (poll_deadline) WHERE status = 'open' AND poll_deadline IS NOT NULL — poll re-arm / restart reconciliation

Semantics: `correlation_id` is a high-entropy opaque token (NOT a uuid, NOT an FK). `workflow_id` (Conductor instance id) and `task_ref_name` are **nullable** — a signal may be landed by an external source (e.g. webhook) *before* any workflow parks on it; the later park binds the engine task. `kind` is a free-form source discriminator (e.g. human, agent, a2a, webhook) — TEXT, not an enum, so ps-workflow owns the vocabulary without a migration. `count_target`/`count_seen` drive N-of-M awaits (default 1 = single-shot). `deadline` FAILs an open row past that instant; `poll_deadline` re-arms poll-driven kinds on restart. `payload` accumulates for N-of-M and is delivered to the resumed task.

### schedule

Tenant-scoped cron trigger registry. Each row fires `workflow_name` on `cron`. The firing loop is exactly-once across pods/restarts via a DB-CAS claim: a due row is claimed with a single `UPDATE schedule SET next_fire_at=<next tick>, claimed_at=NOW() WHERE schedule_id = (SELECT … WHERE enabled AND next_fire_at <= NOW() ORDER BY next_fire_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING …`, so exactly one pod advances `next_fire_at` and fires under concurrent pollers. The claim is intentionally **cross-tenant** (any pod fires any due schedule); tenant isolation is preserved by `company_id` on the row, not by scoping the claim.

| column | type | null | default |
|---|---|---|---|
| schedule_id | SERIAL | no | (PK) |
| schedule_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workflow_name | TEXT | no | — |
| cron | TEXT | no | — |
| enabled | BOOLEAN | no | TRUE |
| input | JSONB | no | '{}' |
| next_fire_at | TIMESTAMPTZ | no | — |
| claimed_at | TIMESTAMPTZ | yes | — |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |

**Constraints:**
- PK: `schedule_id`; UNIQUE: `schedule_uuid`
- FK `company_id` → company(company_id)
- UNIQUE `(company_id, workflow_name)` — one schedule per workflow per tenant; the natural upsert key for schedule registration via ON CONFLICT

**Indexes:**
- `idx_schedule_due` (next_fire_at) WHERE enabled = TRUE (partial) — serves the due-scan and the FOR UPDATE SKIP LOCKED claim; disabled schedules are excluded

Semantics: `workflow_name` is the definition to launch on each fire; `cron` is parsed service-side; `input` is JSON passed to each launched run. `next_fire_at` is advanced atomically by the claiming pod so exactly one fires. `claimed_at` records the last claim (stale-claim / last-fired observability); NULL before the first fire. Note: there is no composite `(company_id, schedule_id)` unique — this table is not a composite-FK target.

### webhook_trigger

P4 webhook trigger: a server-minted bearer credential that starts a run of a **published** workflow definition. The public URL carries the non-secret `webhook_trigger_uuid` as a path segment (`{PUBLIC_BASE}/api/v1/webhooks/{webhook_trigger_uuid}`); the token rides the `X-Webhook-Token` header and is stored HMAC-hashed (never plaintext), shown once at mint/rotate. Ingress looks up by the unique `webhook_trigger_uuid`, constant-time-compares `token_hash`, and resolves `(company_id, workflow_definition_id)` **from the row** — company is never taken from the caller.

| column | type | null | default |
|---|---|---|---|
| webhook_trigger_id | SERIAL | no | (PK) |
| webhook_trigger_uuid | UUID | no | gen_random_uuid() |
| company_id | INTEGER | no | — |
| workflow_definition_id | INTEGER | no | — |
| token_hash | BYTEA | no | — |
| token_prefix | TEXT | no | — |
| enabled | BOOLEAN | no | TRUE |
| created_at | TIMESTAMPTZ | no | NOW() |
| updated_at | TIMESTAMPTZ | no | NOW() |
| rotated_at | TIMESTAMPTZ | yes | — |
| last_fired_at | TIMESTAMPTZ | yes | — |

**Constraints:**
- PK: `webhook_trigger_id`; UNIQUE: `webhook_trigger_uuid`
- FK `company_id` → company(company_id)
- Composite FK `(company_id, workflow_definition_id)` → workflow_definition(company_id, workflow_definition_id) — the referenced definition must belong to the same company

**Indexes:**
- `idx_webhook_trigger_company_workflow_definition` (company_id, workflow_definition_id) — tenant-scoped listing + FK-integrity support; the ingress lookup by `webhook_trigger_uuid` is served by the unique constraint's implicit index

Semantics: `webhook_trigger_uuid` is the non-secret public id (URL path segment; safe to display). `token_hash` is `HMAC-SHA256(token, server_pepper)` — verified by constant-time compare, overwritten on rotate (old token immediately 401s). `token_prefix` is the first ~8 chars for UI identification (the list endpoint never returns the token). `enabled = FALSE` → ingress 404s (runs already started are unaffected). Note: no composite `(company_id, webhook_trigger_id)` unique — not a composite-FK target. There is no separate table for the SC-2 webhook *unpark* source — that reuses the `signal` table above.

## Enum types

- **signal_status**: `open` (parked/awaiting), `completed`, `failed` — `open`→terminal is one-way and exactly-once via the conditional UPDATE. `schedule.enabled` and `webhook_trigger.enabled` are booleans, not enums; `signal.kind` and `webhook_trigger` state are otherwise TEXT.

## See also

- The `workflow_definition` a `webhook_trigger` references (and its `published_at` publish gate) live in [/self/interfaces/schema-workflow.md](/self/interfaces/schema-workflow.md); a parked `signal`'s `(workflow_id, task_ref_name)` mirror the correlation shape of `session_task_correlation` in [/self/interfaces/schema-task-session.md](/self/interfaces/schema-task-session.md).
