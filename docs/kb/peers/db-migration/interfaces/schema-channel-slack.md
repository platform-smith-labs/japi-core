---
type: interface
title: "Schema: channel (Slack)"
tags: [schema, postgres, channel, slack]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for the Slack channel binding/routing tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0050_slack_channel_tables.sql
  - migrations/0052_slack_routing_engine.sql
  - migrations/0053_drop_slack_concurrency_cap.sql
  - migrations/0048_slack_auth_type_enum.sql
  - migrations/0037_integration_enums.sql
provides_interfaces:
  - {name: "channel tables", kind: postgres-schema, intent: "Slack workspace installations, channel/thread-conversation bindings, routing and dedup"}
---

# Schema: channel (Slack)

Connector mapping/dedup/routing plane for the Slack integration. All tables except
`channel_routing_rule` carry a `provider` discriminator (currently only `'slack'` allowed) so other providers can reuse them later. Secrets
are NOT stored here — the bot token lives in `integration_connection`. No `ON DELETE CASCADE`
anywhere. Three uniques are intentionally GLOBAL (no `company_id`) because they key on identifiers
Slack itself guarantees globally unique: `(provider, tenant_ref)`, `(provider, event_id)`, and
`(provider, conversation_ref, thread_ref)`. All other uniqueness is tenant-scoped.

### channel_installation
Per-(provider, tenant_ref) install anchor mapping a Slack workspace (team_id) to a company and the
stored bot-token connection.

| column | type | null | default |
|---|---|---|---|
| channel_installation_id | SERIAL | NOT NULL | auto |
| channel_installation_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| integration_connection_id | INTEGER | NOT NULL | — |
| provider | TEXT | NOT NULL | — |
| tenant_ref | TEXT | NOT NULL | — |
| provider_meta | JSONB | NOT NULL | '{}' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_installation_id`; UNIQUE `channel_installation_uuid`;
FK `company_id` → `company(company_id)`;
composite FK `(company_id, integration_connection_id)` → `integration_connection(company_id, integration_connection_id)`;
UNIQUE `(company_id, channel_installation_id)`; UNIQUE `(provider, tenant_ref)` (intentional global — a Slack workspace belongs to exactly one company);
CHECK `provider IN ('slack')`.
**Indexes:** `idx_channel_installation_company_id` on `(company_id)`.

### channel_user_link
Maps a Slack user id to a Platform Smith user within a company (actor identity for human-initiated
work). The same Slack user may appear across multiple companies.

| column | type | null | default |
|---|---|---|---|
| channel_user_link_id | SERIAL | NOT NULL | auto |
| channel_user_link_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| user_id | INTEGER | NOT NULL | — |
| provider | TEXT | NOT NULL | — |
| external_user_ref | TEXT | NOT NULL | — |
| link_status | TEXT | NOT NULL | 'proposed' |
| link_method | TEXT | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_user_link_id`; UNIQUE `channel_user_link_uuid`;
FK `company_id` → `company(company_id)`;
composite FK `(company_id, user_id)` → `users(company_id, user_id)`;
UNIQUE `(company_id, channel_user_link_id)`; UNIQUE `(provider, external_user_ref, company_id)`;
CHECK `provider IN ('slack')`; CHECK `link_status IN ('proposed','verified','revoked')`;
CHECK `link_method IN ('admin','email','oauth')`.
**Indexes:** `idx_channel_user_link_company_id` on `(company_id)`.

### channel_conversation_binding
Authorizes a Slack channel to a target (currently a session) scoped to a workspace + pinned project,
with an optional investigation agent definition for alert channels, an explicit environment target,
and routing controls. `identity_mode` selects human actor vs workspace service principal.

| column | type | null | default |
|---|---|---|---|
| channel_conversation_binding_id | SERIAL | NOT NULL | auto |
| channel_conversation_binding_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| workspace_id | INTEGER | NOT NULL | — |
| project_id | INTEGER | NULL | — |
| agent_definition_id | INTEGER | NULL | — |
| provider | TEXT | NOT NULL | — |
| conversation_ref | TEXT | NOT NULL | — |
| target_type | TEXT | NOT NULL | 'session' |
| identity_mode | TEXT | NOT NULL | — |
| allowed_agent_definitions | JSONB | NOT NULL | '[]' |
| who_may_trigger | JSONB | NOT NULL | '[]' |
| environment_id | INTEGER | NULL | — |
| on_no_match | TEXT | NOT NULL | 'default_target' |
| allowed_environments | JSONB | NOT NULL | '[]' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_conversation_binding_id`; UNIQUE `channel_conversation_binding_uuid`;
FK `company_id` → `company(company_id)`;
composite FKs: `(company_id, workspace_id)` → `workspace`, `(company_id, project_id)` → `project`,
`(company_id, agent_definition_id)` → `agent_definition`, `(company_id, environment_id)` → `environment`
(nullable FK columns skip enforcement when NULL);
UNIQUE `(company_id, channel_conversation_binding_id)`; UNIQUE `(provider, conversation_ref, company_id)`;
CHECK `provider IN ('slack')`; CHECK `target_type IN ('session')`;
CHECK `identity_mode IN ('user','service_principal')`; CHECK `on_no_match IN ('default_target','drop')`.
**Indexes:** `idx_channel_conv_binding_company_id` on `(company_id)`;
`idx_channel_conv_binding_company_workspace` on `(company_id, workspace_id)`.

Notes: `environment_id` NULL means the workspace default environment. `allowed_environments` is a
JSON array of environment_uuids a routing rule may target (empty = only the binding's own
environment); the in-set guard is enforced in ps-api, not the DB. `allowed_agent_definitions` is a
JSON array of agent_definition_uuids permitted from this channel. There is NO per-binding
concurrency-cap column (a prior cap was removed).

### channel_thread_binding
Binds a Slack thread (thread_ts) to a session for in-thread streaming; binds to sessions only.
`session_uuid` is carried for external joins; the composite FK to `session` proves same-company.

| column | type | null | default |
|---|---|---|---|
| channel_thread_binding_id | SERIAL | NOT NULL | auto |
| channel_thread_binding_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| session_id | INTEGER | NOT NULL | — |
| session_uuid | UUID | NOT NULL | — |
| provider | TEXT | NOT NULL | — |
| conversation_ref | TEXT | NOT NULL | — |
| thread_ref | TEXT | NOT NULL | — |
| target_type | TEXT | NOT NULL | 'session' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_thread_binding_id`; UNIQUE `channel_thread_binding_uuid`;
FK `company_id` → `company(company_id)`;
composite FK `(company_id, session_id)` → `session(company_id, session_id)`;
UNIQUE `(company_id, channel_thread_binding_id)`; UNIQUE `(provider, conversation_ref, thread_ref)`
(intentional global — a Slack thread is globally unique);
CHECK `provider IN ('slack')`; CHECK `target_type IN ('session')`.
**Indexes:** `idx_channel_thread_binding_company_id` on `(company_id)`;
`idx_channel_thread_binding_company_session` on `(company_id, session_id)`.

### channel_event_dedup
Idempotency ledger for inbound Slack events (Slack retries ~3x): the connector INSERTs
`(provider, event_id)`; a unique violation means duplicate → no-op (no double-spawn). Retention /
pruning is an ops concern, not enforced here.

| column | type | null | default |
|---|---|---|---|
| channel_event_dedup_id | SERIAL | NOT NULL | auto |
| channel_event_dedup_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| provider | TEXT | NOT NULL | — |
| event_id | TEXT | NOT NULL | — |
| received_at | TIMESTAMPTZ | NOT NULL | NOW() |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_event_dedup_id`; UNIQUE `channel_event_dedup_uuid`;
FK `company_id` → `company(company_id)`;
UNIQUE `(company_id, channel_event_dedup_id)`; UNIQUE `(provider, event_id)` (intentional global —
Slack event ids are globally unique); CHECK `provider IN ('slack')`.
**Indexes:** `idx_channel_event_dedup_company_id` on `(company_id)`.

### channel_routing_rule
Per-binding ordered routing rules for alert channels. Evaluated priority ASC, first-match-wins;
duplicate priorities are allowed and broken deterministically by `channel_routing_rule_id` ASC.
`action=route` resolves a (project, environment, agent-def) target override; `action=drop` is a
terminal deny. `target_environment_id` must be within the binding's `allowed_environments` —
enforced in ps-api, not the DB.

| column | type | null | default |
|---|---|---|---|
| channel_routing_rule_id | SERIAL | NOT NULL | auto |
| channel_routing_rule_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| channel_conversation_binding_id | INTEGER | NOT NULL | — |
| target_project_id | INTEGER | NULL | — |
| target_environment_id | INTEGER | NULL | — |
| target_agent_definition_id | INTEGER | NULL | — |
| priority | INTEGER | NOT NULL | — |
| action | TEXT | NOT NULL | — |
| match_conditions | JSONB | NOT NULL | '{}' |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_routing_rule_id`; UNIQUE `channel_routing_rule_uuid`;
FK `company_id` → `company(company_id)`;
composite FKs: `(company_id, channel_conversation_binding_id)` → `channel_conversation_binding`,
`(company_id, target_project_id)` → `project`, `(company_id, target_environment_id)` → `environment`,
`(company_id, target_agent_definition_id)` → `agent_definition`;
UNIQUE `(company_id, channel_routing_rule_id)` (no unique on priority — duplicates intended);
CHECK `action IN ('route','drop')`.
**Indexes:** `idx_channel_routing_rule_company_id` on `(company_id)`;
`idx_channel_routing_rule_binding` on `(company_id, channel_conversation_binding_id, priority)`.

Note: `match_conditions` holds poster allowlist + literal/substring text/label conditions (no regex).

### channel_alert_fingerprint
Content-fingerprint storm suppression for alerts (one spawn per fingerprint per time window),
tenant-scoped and per-binding — distinct from `channel_event_dedup` (global per-event at-most-once
delivery). The connector UPSERTs on the window unique, incrementing `occurrence_count` on a window
hit; `window_bucket` is the received-at timestamp truncated to the suppression window (5 min
default, configurable in the connector). Window length / retention is an ops concern, not enforced
here.

| column | type | null | default |
|---|---|---|---|
| channel_alert_fingerprint_id | SERIAL | NOT NULL | auto |
| channel_alert_fingerprint_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| channel_conversation_binding_id | INTEGER | NOT NULL | — |
| provider | TEXT | NOT NULL | — |
| fingerprint | TEXT | NOT NULL | — |
| window_bucket | TIMESTAMPTZ | NOT NULL | — |
| occurrence_count | INTEGER | NOT NULL | 1 |
| first_seen_at | TIMESTAMPTZ | NOT NULL | NOW() |
| last_seen_at | TIMESTAMPTZ | NOT NULL | NOW() |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:** PK `channel_alert_fingerprint_id`; UNIQUE `channel_alert_fingerprint_uuid`;
FK `company_id` → `company(company_id)`;
composite FK `(company_id, channel_conversation_binding_id)` → `channel_conversation_binding`;
UNIQUE `(company_id, channel_alert_fingerprint_id)`;
UNIQUE `(company_id, channel_conversation_binding_id, provider, fingerprint, window_bucket)`
(tenant-scoped — a content fingerprint is not globally unique);
CHECK `provider IN ('slack')`.
**Indexes:** `idx_channel_alert_fingerprint_company_id` on `(company_id)`;
`idx_channel_alert_fingerprint_window` on `(company_id, window_bucket)`.

## ENUM types

None of the seven channel tables has an ENUM-typed column — all constrained-value fields are
TEXT + CHECK (value sets listed per table above).

Related enum on the referenced `integration_connection` table (its `auth_type` column;
`channel_installation` FKs to that table): `integration_auth_type`, final value set —
`claude_api_key`, `claude_bedrock`, `claude_vertex`, `claude_oauth_setup_token`, `codex_api_key`,
`codex_chatgpt_subscription`, `linear_api_key`, `jira_api_token`, `notion_api_key`,
`github_actions_pat`, `aws_iam_keys`, `aws_assume_role`, `azure_service_principal`,
`gcp_service_account`, `cloudflare_api_token`, `e2b_api_key`, `modal_token`, `daytona_api_key`,
`slack_bot_token`. The Slack connector's bot token uses `slack_bot_token`.
