---
type: interface
title: "Schema: conversation"
tags: [schema, postgres, conversation]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for conversation, participant and message tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0040_conversation.sql
  - migrations/0041_conversation_participant.sql
  - migrations/0042_conversation_message.sql
  - migrations/0044_conversation_participant_add_agent_definition.sql
  - migrations/0054_conversation_add_git_branch.sql
  - migrations/0055_conversation_add_git_base_branch.sql
  - migrations/0001_enums.sql
provides_interfaces:
  - {name: "conversation tables", kind: postgres-schema, intent: "conversations, their participants and messages"}
---

# Schema: conversation domain

Cross-pod agent-to-agent (A2A) messaging: a conversation is the coordination unit; participants are projects; messages are a persist-first replay log. Only `*_uuid` columns are exposed externally.

### conversation

Coordination unit for cross-pod agent A2A messaging.

| column | type | null | default |
|---|---|---|---|
| conversation_id | SERIAL | NOT NULL | auto (PK) |
| conversation_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NOT NULL | — |
| status | TEXT | NOT NULL | 'active' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| git_branch | TEXT | NULL | — |
| git_base_branch | TEXT | NULL | — |

`status` is a freeform lifecycle label (e.g. active|closed) — intentionally no CHECK; vocabulary is owned by the orchestrator. `git_branch`: optional per-conversation target git branch every pod spawned for the conversation checks out (NULL → per-project default). `git_base_branch`: optional base the target branch is created from if missing (NULL → repo default).

**Constraints:**
- PK: `conversation_id`; UNIQUE: `conversation_uuid`
- FK `company_id` → `company(company_id)`
- Composite FK `(company_id, workspace_id)` → `workspace(company_id, workspace_id)`
- UNIQUE `(company_id, conversation_id)` (composite-FK anchor; referenced by conversation_participant, conversation_message, session, sandbox_autospawn_claim)

**Indexes:**
- `idx_conversation_company_id` on `(company_id)`

### conversation_participant

Membership / address book for a conversation. Addressing is by project; `role` / `capabilities` / `status` are descriptive only.

| column | type | null | default |
|---|---|---|---|
| conversation_participant_id | SERIAL | NOT NULL | auto (PK) |
| conversation_participant_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| conversation_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NOT NULL | — |
| role | TEXT | NULL | — |
| capabilities | JSONB | NOT NULL | '{}' |
| status | TEXT | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| joined_at | TIMESTAMPTZ | NULL | — |
| agent_definition_id | INTEGER | NULL | — |

`workspace_id` is intentionally omitted (derivable via project). `agent_definition_id`: optional per-(conversation, participant) harness selection — NULL means the code-level default harness; non-NULL pins the harness of the referenced agent_definition.

**Constraints:**
- PK: `conversation_participant_id`; UNIQUE: `conversation_participant_uuid`
- FK `company_id` → `company(company_id)`
- Composite FK `(company_id, conversation_id)` → `conversation(company_id, conversation_id)`
- Composite FK `(company_id, project_id)` → `project(company_id, project_id)`
- FK `agent_definition_id` → `agent_definition(agent_definition_id)` ON DELETE SET NULL — deliberately single-column (SET NULL is incompatible with a composite FK on NOT NULL company_id); same-company integrity is enforced in the application layer
- UNIQUE `(company_id, conversation_participant_id)` (composite-FK anchor)
- UNIQUE `(company_id, conversation_id, project_id)` — one participant row per project per conversation

**Indexes:**
- `idx_conversation_participant_company_id` on `(company_id)`
- `idx_conversation_participant_agent_definition` on `(agent_definition_id)` WHERE `agent_definition_id IS NOT NULL` (partial)

### conversation_message

Persist-first A2A message log. `seq` (BIGSERIAL) is the authoritative replay cursor — consumers ORDER BY `seq`, never `created_at`; `seq` may gap or commit out of order under concurrency. No `updated_at` column (append-oriented log).

| column | type | null | default |
|---|---|---|---|
| conversation_message_id | SERIAL | NOT NULL | auto (PK) |
| conversation_message_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| conversation_id | INTEGER | NOT NULL | — |
| from_project_id | INTEGER | NOT NULL | — |
| to_project_id | INTEGER | NOT NULL | — |
| seq | BIGSERIAL | NOT NULL | auto |
| message_id | UUID | NOT NULL | — |
| in_reply_to | UUID | NULL | — |
| type | TEXT | NULL | — |
| data | JSONB | NOT NULL | '{}' |
| delivery_state | TEXT | NOT NULL | 'pending' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |

`message_id` is the upstream-supplied idempotency key (distinct from the internal `conversation_message_uuid`); redelivery of the same key within a conversation is an idempotent no-op via the unique constraint. `in_reply_to` is an optional correlation handle holding a `message_id` — intentionally NO FK, dangling values are valid. `type` is a freeform vocabulary (e.g. plan/assign/result). `data` holds the message payload. `created_at` is display-only.

**Constraints:**
- PK: `conversation_message_id`; UNIQUE: `conversation_message_uuid`
- FK `company_id` → `company(company_id)`
- Composite FK `(company_id, conversation_id)` → `conversation(company_id, conversation_id)`
- Composite FK `(company_id, from_project_id)` → `project(company_id, project_id)`
- Composite FK `(company_id, to_project_id)` → `project(company_id, project_id)`
- UNIQUE `(company_id, conversation_message_id)` (composite-FK anchor)
- UNIQUE `(company_id, conversation_id, message_id)` — idempotency guard
- CHECK `delivery_state IN ('pending', 'delivered')` — TEXT+CHECK (not boolean) by design; the vocabulary is expected to grow (e.g. failed/expired)

**Indexes:**
- `idx_conversation_message_company_id` on `(company_id)`
- `idx_conversation_message_conversation_seq` on `(company_id, conversation_id, seq)`

### ENUM types

None. These tables use no PostgreSQL ENUM types — all state columns are TEXT (freeform or CHECK-constrained as listed above).
